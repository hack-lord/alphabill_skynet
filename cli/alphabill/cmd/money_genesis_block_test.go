package cmd

import (
	"crypto"
	"testing"

	"gitdc.ee.guardtime.com/alphabill/alphabill/internal/script"
	moneytx "gitdc.ee.guardtime.com/alphabill/alphabill/internal/txsystem/money"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestMoneyGenesisBlockWithTransaction(t *testing.T) {
	hashAlgo := crypto.SHA256
	pubKeyHex := "0x0212911c7341399e876800a268855c894c43eb849a72ac5a9d26a0091041c107f0"
	pubKey, _ := hexutil.Decode(pubKeyHex)
	pubKeyHash := hash(hashAlgo, pubKey)
	systemIdentifier := []byte{0, 0, 0, 0}
	initialBillId := uint256.NewInt(defaultInitialBillId).Bytes32()
	initialBillValue := uint64(10)

	b, err := NewMoneyGenesisBlock(&MoneyGenesisBlockConfig{
		initialBillValue:          initialBillValue,
		initialBillOwnerPubKeyHex: pubKeyHex,
		systemIdentifier:          systemIdentifier,
		hashAlgo:                  hashAlgo,
	})
	require.NoError(t, err)
	require.NotNil(t, b)

	require.EqualValues(t, systemIdentifier, b.SystemIdentifier)
	require.EqualValues(t, 1, b.BlockNumber)
	require.EqualValues(t, make([]byte, 32), b.PreviousBlockHash)
	require.Len(t, b.Transactions, 1)
	require.Nil(t, b.UnicityCertificate)

	tx := b.Transactions[0]
	require.EqualValues(t, systemIdentifier, tx.SystemId)
	require.EqualValues(t, initialBillId[:], tx.UnitId)
	require.EqualValues(t, 1, tx.Timeout)
	require.Nil(t, tx.OwnerProof)

	to := &moneytx.TransferOrder{}
	err = tx.TransactionAttributes.UnmarshalTo(to)
	require.NoError(t, err)
	require.EqualValues(t, script.PredicatePayToPublicKeyHashDefault(pubKeyHash), to.NewBearer)
	require.EqualValues(t, initialBillValue, to.TargetValue)
	require.Nil(t, to.Backlink)
}

func hash(hashAlgo crypto.Hash, b []byte) []byte {
	hasher := hashAlgo.New()
	hasher.Write(b)
	return hasher.Sum(nil)
}
