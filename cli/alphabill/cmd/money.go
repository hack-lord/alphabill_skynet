package cmd

import (
	"context"
	"crypto"
	"crypto/sha256"
	"fmt"

	"gitdc.ee.guardtime.com/alphabill/alphabill/internal/errors"
	"gitdc.ee.guardtime.com/alphabill/alphabill/internal/logger"
	"gitdc.ee.guardtime.com/alphabill/alphabill/internal/script"
	"gitdc.ee.guardtime.com/alphabill/alphabill/internal/txsystem"
	"gitdc.ee.guardtime.com/alphabill/alphabill/internal/txsystem/money"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/anypb"
)

type (
	moneyNodeConfiguration struct {
		baseNodeConfiguration
		Node      *startNodeConfiguration
		RPCServer *grpcServerConfiguration
		// The value of initial bill in Alphabills.
		InitialBillValue uint64 `validate:"gte=0"`
		// The initial bill owner's public key in HEX
		InitialBillOwner string
		// The initial value of Dust Collector Money supply.
		DCMoneySupplyValue uint64 `validate:"gte=0"`
	}

	// moneyNodeRunnable is the function that is run after configuration is loaded.
	moneyNodeRunnable func(ctx context.Context, nodeConfig *moneyNodeConfiguration) error
)

const (
	defaultInitialBillValue   = 1000000
	defaultDCMoneySupplyValue = 1000000
	defaultInitialBillId      = 1
)

var log = logger.CreateForPackage()

// newMoneyNodeCmd creates a new cobra command for the shard component.
//
// nodeRunFunc - set the function to override the default behaviour. Meant for tests.
func newMoneyNodeCmd(ctx context.Context, baseConfig *baseConfiguration, nodeRunFunc moneyNodeRunnable) *cobra.Command {
	config := &moneyNodeConfiguration{
		baseNodeConfiguration: baseNodeConfiguration{
			Base: baseConfig,
		},
		Node:      &startNodeConfiguration{},
		RPCServer: &grpcServerConfiguration{},
	}
	var nodeCmd = &cobra.Command{
		Use:   "money",
		Short: "Starts a money node",
		Long:  `Starts a money partition's node, binding to the network address provided by configuration.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if nodeRunFunc != nil {
				return nodeRunFunc(ctx, config)
			}
			return runMoneyNode(ctx, config)
		},
	}

	nodeCmd.Flags().Uint64Var(&config.InitialBillValue, "initial-bill-value", defaultInitialBillValue, "the initial bill value for new node.")
	nodeCmd.Flags().StringVar(&config.InitialBillOwner, "initial-bill-owner", "", "the initial bill owner's public key in HEX. If empty then owner is set to always true predicate.")
	nodeCmd.Flags().Uint64Var(&config.DCMoneySupplyValue, "dc-money-supply-value", defaultDCMoneySupplyValue, "the initial value for Dust Collector money supply. Total money sum is initial bill + DC money supply.")
	nodeCmd.Flags().StringVarP(&config.Node.Address, "address", "a", "/ip4/127.0.0.1/tcp/26652", "node address in libp2p multiaddress-format")
	nodeCmd.Flags().StringVarP(&config.Node.RootChainAddress, "rootchain", "r", "/ip4/127.0.0.1/tcp/26662", "root chain address in libp2p multiaddress-format")
	nodeCmd.Flags().StringToStringVarP(&config.Node.Peers, "peers", "p", nil, "a map of partition peer identifiers and addresses. must contain all genesis validator addresses")
	nodeCmd.Flags().StringVarP(&config.Node.KeyFile, keyFileCmd, "k", "", "path to the key file (default: $AB_HOME/vd/keys.json)")
	nodeCmd.Flags().StringVarP(&config.Node.Genesis, "genesis", "g", "", "path to the partition genesis file : $AB_HOME/vd/partition-genesis.json)")

	config.RPCServer.addConfigurationFlags(nodeCmd)
	return nodeCmd
}

func runMoneyNode(ctx context.Context, cfg *moneyNodeConfiguration) error {
	pg, err := loadPartitionGenesis(cfg.Node.Genesis)
	if err != nil {
		return errors.Wrapf(err, "failed to read genesis file %s", cfg.Node.Genesis)
	}

	initialBillOwner, err := cfg.getInitialBillOwner()
	if err != nil {
		return err
	}
	fmt.Println("node owner: " + string(initialBillOwner))
	ib := &money.InitialBill{
		ID:    uint256.NewInt(defaultInitialBillId),
		Value: cfg.InitialBillValue,
		Owner: initialBillOwner,
	}

	hashAlgorithm := crypto.SHA256
	genesisBlock, err := NewMoneyGenesisBlock(&MoneyGenesisBlockConfig{
		initialBillValue:          cfg.InitialBillValue,
		initialBillOwnerPubKeyHex: cfg.InitialBillOwner,
		systemIdentifier:          []byte{0, 0, 0, 0},
		hashAlgo:                  hashAlgorithm,
		unicityCertificate:        pg.Certificate,
	})
	if err != nil {
		return err
	}
	cfg.Node.genesisBlock = genesisBlock

	txs, err := money.NewMoneyTxSystem(
		hashAlgorithm,
		ib,
		cfg.DCMoneySupplyValue,
		money.SchemeOpts.SystemIdentifier(pg.GetSystemDescriptionRecord().GetSystemIdentifier()),
	)
	if err != nil {
		return errors.Wrapf(err, "failed to start money transaction system")
	}
	return defaultNodeRunFunc(ctx, "money node", txs, cfg.Node, cfg.RPCServer)
}

func (cfg *moneyNodeConfiguration) getInitialBillOwner() ([]byte, error) {
	if cfg.InitialBillOwner != "" {
		ownerPubKey, err := hexutil.Decode(cfg.InitialBillOwner)
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

func (cfg *moneyNodeConfiguration) getGenesisTransactions() ([]*txsystem.Transaction, error) {
	initialBillId := uint256.NewInt(defaultInitialBillId).Bytes32()
	initialBillTx, err := cfg.initialBillTx()
	if err != nil {
		return nil, err
	}
	return []*txsystem.Transaction{
		{
			SystemId:              []byte{0, 0, 0, 0},
			UnitId:                initialBillId[:],
			TransactionAttributes: initialBillTx,
			Timeout:               1,
		},
	}, nil
}

func (cfg *moneyNodeConfiguration) initialBillTx() (*anypb.Any, error) {
	initialBillOwner, err := cfg.getInitialBillOwner()
	if err != nil {
		return nil, err
	}
	return anypb.New(&money.TransferOrder{
		TargetValue: cfg.InitialBillValue,
		NewBearer:   initialBillOwner,
		Backlink:    nil, // TODO what backlink to use?
	})
}
