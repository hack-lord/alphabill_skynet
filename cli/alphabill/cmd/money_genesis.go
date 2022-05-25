package cmd

import (
	"context"
	"crypto"
	"crypto/sha256"
	"os"
	"path"

	"gitdc.ee.guardtime.com/alphabill/alphabill/internal/errors"
	"gitdc.ee.guardtime.com/alphabill/alphabill/internal/partition"
	"gitdc.ee.guardtime.com/alphabill/alphabill/internal/script"
	"gitdc.ee.guardtime.com/alphabill/alphabill/internal/txsystem"
	"gitdc.ee.guardtime.com/alphabill/alphabill/internal/txsystem/money"
	moneytx "gitdc.ee.guardtime.com/alphabill/alphabill/internal/txsystem/money"
	"gitdc.ee.guardtime.com/alphabill/alphabill/internal/util"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/anypb"
)

const moneyPartitionDir = "money"

var defaultABMoneySystemIdentifier = []byte{0, 0, 0, 0}

type moneyGenesisConfig struct {
	Base               *baseConfiguration
	SystemIdentifier   []byte
	KeyFile            string
	Output             string
	ForceKeyGeneration bool
	InitialBillValue   uint64 `validate:"gte=0"`
	InitialBillOwner   string
	DCMoneySupplyValue uint64 `validate:"gte=0"`
}

// newMoneyGenesisCmd creates a new cobra command for the alphabill money partition genesis.
func newMoneyGenesisCmd(ctx context.Context, baseConfig *baseConfiguration) *cobra.Command {
	config := &moneyGenesisConfig{Base: baseConfig}
	var cmd = &cobra.Command{
		Use:   "money-genesis",
		Short: "Generates a genesis file for the Alphabill Money partition",
		RunE: func(cmd *cobra.Command, args []string) error {
			return abMoneyGenesisRunFun(ctx, config)
		},
	}

	cmd.Flags().BytesHexVarP(&config.SystemIdentifier, "system-identifier", "s", defaultABMoneySystemIdentifier, "system identifier in HEX format")
	cmd.Flags().BoolVarP(&config.ForceKeyGeneration, "force-key-gen", "f", false, "generates new keys for the node if the key-file does not exist")
	cmd.Flags().StringVarP(&config.KeyFile, keyFileCmd, "k", "", "path to the key file (default: $AB_HOME/money/keys.json). If key file does not exist and flag -f is present then new keys are generated.")
	cmd.Flags().StringVarP(&config.Output, "output", "o", "", "path to the output genesis file (default: $AB_HOME/money/node-genesis.json)")
	cmd.Flags().Uint64Var(&config.InitialBillValue, "initial-bill-value", defaultInitialBillValue, "the initial bill value")
	cmd.Flags().StringVar(&config.InitialBillOwner, "initial-bill-owner", "", "the initial bill owner's public key in HEX. If empty then owner is set to always true predicate.")
	cmd.Flags().Uint64Var(&config.DCMoneySupplyValue, "dc-money-supply-value", defaultDCMoneySupplyValue, "the initial value for Dust Collector money supply. Total money sum is initial bill + DC money supply.")
	return cmd
}

func abMoneyGenesisRunFun(_ context.Context, config *moneyGenesisConfig) error {
	moneyPartitionHomePath := path.Join(config.Base.HomeDir, moneyPartitionDir)
	if !util.FileExists(moneyPartitionHomePath) {
		err := os.MkdirAll(moneyPartitionHomePath, 0700) // -rwe------
		if err != nil {
			return err
		}
	}

	nodeGenesisFile := config.getNodeGenesisFileLocation(moneyPartitionHomePath)
	if util.FileExists(nodeGenesisFile) {
		return errors.Errorf("node genesis %s exists", nodeGenesisFile)
	}

	keys, err := LoadKeys(config.getKeyFileLocation(), config.ForceKeyGeneration)
	if err != nil {
		return errors.Wrapf(err, "failed to load keys %v", config.getKeyFileLocation())
	}
	peerID, err := peer.IDFromPublicKey(keys.EncryptionPrivateKey.GetPublic())
	if err != nil {
		return err
	}
	encryptionPublicKeyBytes, err := keys.EncryptionPrivateKey.GetPublic().Raw()
	if err != nil {
		return err
	}

	initialBillOwner, err := config.getInitialBillOwner()
	if err != nil {
		return err
	}
	ib := &money.InitialBill{
		ID:    uint256.NewInt(defaultInitialBillId),
		Value: config.InitialBillValue,
		Owner: initialBillOwner,
	}

	hashAlgorithm := crypto.SHA256
	genesisBlock, err := NewMoneyGenesisBlock(&MoneyGenesisBlockConfig{
		initialBillValue:          config.InitialBillValue,
		initialBillOwnerPubKeyHex: config.InitialBillOwner,
		systemIdentifier:          []byte{0, 0, 0, 0},
		hashAlgo:                  hashAlgorithm,
	})
	if err != nil {
		return err
	}

	txSystem, err := money.NewMoneyTxSystem(
		hashAlgorithm,
		ib,
		config.DCMoneySupplyValue,
		money.SchemeOpts.SystemIdentifier(config.SystemIdentifier),
	)
	nodeGenesis, err := partition.NewNodeGenesis(
		txSystem,
		partition.WithPeerID(peerID),
		partition.WithSigningKey(keys.SigningPrivateKey),
		partition.WithEncryptionPubKey(encryptionPublicKeyBytes),
		partition.WithSystemIdentifier(config.SystemIdentifier),
		partition.WithGenesisBlock(genesisBlock),
	)
	if err != nil {
		return err
	}
	return util.WriteJsonFile(nodeGenesisFile, nodeGenesis)
}

func (c *moneyGenesisConfig) getKeyFileLocation() string {
	if c.KeyFile != "" {
		return c.KeyFile
	}
	return path.Join(c.Base.HomeDir, moneyPartitionDir, keysFileName)
}

func (c *moneyGenesisConfig) getNodeGenesisFileLocation(home string) string {
	if c.Output != "" {
		return c.Output
	}
	return path.Join(home, vdGenesisFileName)
}

func (c *moneyGenesisConfig) getInitialBillOwner() ([]byte, error) {
	if c.InitialBillOwner != "" {
		ownerPubKey, err := hexutil.Decode(c.InitialBillOwner)
		if err != nil {
			return nil, err
		}
		hasher := sha256.New()
		hasher.Write(ownerPubKey)
		pubKeyHash := hasher.Sum(nil)
		// TODO use PredicatePayToPublicKeyHash instead i.e. use partition hash algo
		return script.PredicatePayToPublicKeyHashDefault(pubKeyHash), nil
	}
	return script.PredicateAlwaysTrue(), nil
}

func (c *moneyGenesisConfig) getGenesisTransactions() ([]*txsystem.Transaction, error) {
	initialBillId := uint256.NewInt(defaultInitialBillId).Bytes32()
	initialBillTx, err := c.initialBillTx()
	if err != nil {
		return nil, err
	}
	return []*txsystem.Transaction{
		{
			SystemId:              c.SystemIdentifier,
			UnitId:                initialBillId[:],
			TransactionAttributes: initialBillTx,
			Timeout:               1,
		},
	}, nil
}

func (c *moneyGenesisConfig) initialBillTx() (*anypb.Any, error) {
	initialBillOwner, err := c.getInitialBillOwner()
	if err != nil {
		return nil, err
	}
	return anypb.New(&moneytx.TransferOrder{
		TargetValue: c.InitialBillValue,
		NewBearer:   initialBillOwner,
		Backlink:    nil, // TODO what backlink to use?
	})
}
