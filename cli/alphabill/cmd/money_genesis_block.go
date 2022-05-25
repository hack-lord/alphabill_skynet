package cmd

import (
	gocrypto "crypto"

	"gitdc.ee.guardtime.com/alphabill/alphabill/internal/block"
	"gitdc.ee.guardtime.com/alphabill/alphabill/internal/certificates"
	"gitdc.ee.guardtime.com/alphabill/alphabill/internal/errors"
	"gitdc.ee.guardtime.com/alphabill/alphabill/internal/script"
	"gitdc.ee.guardtime.com/alphabill/alphabill/internal/txsystem"
	moneytx "gitdc.ee.guardtime.com/alphabill/alphabill/internal/txsystem/money"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
	"google.golang.org/protobuf/types/known/anypb"
)

type MoneyGenesisBlockConfig struct {
	initialBillValue          uint64
	initialBillOwnerPubKeyHex string
	systemIdentifier          []byte
	hashAlgo                  gocrypto.Hash
	unicityCertificate        *certificates.UnicityCertificate
}

func NewMoneyGenesisBlock(config *MoneyGenesisBlockConfig) (*block.Block, error) {
	txs, err := config.getGenesisTransactions()
	if err != nil {
		return nil, err
	}
	return &block.Block{
		SystemIdentifier:   config.systemIdentifier,
		BlockNumber:        1,
		PreviousBlockHash:  make([]byte, config.hashAlgo.Size()),
		Transactions:       txs,
		UnicityCertificate: config.unicityCertificate,
	}, nil
}

func (c *MoneyGenesisBlockConfig) getGenesisTransactions() ([]*txsystem.Transaction, error) {
	initialBillId := uint256.NewInt(defaultInitialBillId).Bytes32()
	initialBillTxAttributes, err := c.getInitialBillTxAttributes()
	if err != nil {
		return nil, err
	}
	return []*txsystem.Transaction{
		{
			SystemId:              c.systemIdentifier,
			UnitId:                initialBillId[:],
			TransactionAttributes: initialBillTxAttributes,
			Timeout:               1,
		},
	}, nil
}

func (c *MoneyGenesisBlockConfig) getInitialBillOwnerPredicate() ([]byte, error) {
	if c.initialBillOwnerPubKeyHex == "" {
		return script.PredicateAlwaysTrue(), nil
	}
	hashAlgByte, err := c.getHashAlgoByte(c.hashAlgo)
	if err != nil {
		return nil, err
	}
	pkHash, err := c.getOwnerPubKeyHash(c.initialBillOwnerPubKeyHex, c.hashAlgo)
	if err != nil {
		return nil, err
	}
	return script.PredicatePayToPublicKeyHash(hashAlgByte, pkHash, script.SigSchemeSecp256k1), nil
}

func (c *MoneyGenesisBlockConfig) getOwnerPubKeyHash(ownerPubKeyHex string, hashAlgo gocrypto.Hash) ([]byte, error) {
	ownerPubKey, err := hexutil.Decode(ownerPubKeyHex)
	if err != nil {
		return nil, err
	}
	hasher := hashAlgo.New()
	hasher.Write(ownerPubKey)
	pkHash := hasher.Sum(nil)
	return pkHash, nil
}

func (c *MoneyGenesisBlockConfig) getHashAlgoByte(hashAlgo gocrypto.Hash) (byte, error) {
	if hashAlgo == gocrypto.SHA256 {
		return script.HashAlgSha256, nil
	} else if hashAlgo == gocrypto.SHA512 {
		return script.HashAlgSha512, nil
	}
	return 0, errors.New("invalid hashalgo: cannot convert from gocrypto.Hash to script.hashalgo byte")
}

func (c *MoneyGenesisBlockConfig) getInitialBillTxAttributes() (*anypb.Any, error) {
	initialBillOwner, err := c.getInitialBillOwnerPredicate()
	if err != nil {
		return nil, err
	}
	return anypb.New(&moneytx.TransferOrder{
		TargetValue: c.initialBillValue,
		NewBearer:   initialBillOwner,
		Backlink:    nil, // TODO what backlink to use?
	})
}
