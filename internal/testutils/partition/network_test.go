package testpartition

import (
	"testing"

	"github.com/alphabill-org/alphabill/internal/crypto"
	test "github.com/alphabill-org/alphabill/internal/testutils"
	testtransaction "github.com/alphabill-org/alphabill/internal/testutils/transaction"
	testtxsystem "github.com/alphabill-org/alphabill/internal/testutils/txsystem"
	"github.com/alphabill-org/alphabill/internal/txsystem"
	"github.com/stretchr/testify/require"
)

var systemIdentifier = []byte{1, 2, 4, 1}

func TestNewNetwork_Ok(t *testing.T) {
	network, err := NewNetwork(3,
		func(_ map[string]crypto.Verifier) txsystem.TransactionSystem {
			return &testtxsystem.CounterTxSystem{}
		},
		systemIdentifier)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, network.Close())
	}()
	require.NotNil(t, network.RootNode)
	require.Equal(t, 3, len(network.Nodes))

	tx := testtransaction.NewTransaction(t, testtransaction.WithSystemID(systemIdentifier))
	require.NoError(t, network.SubmitTx(tx))
	require.Eventually(t, BlockchainContainsTx(tx, network), test.WaitDuration, test.WaitTick)

	tx = testtransaction.NewTransaction(t, testtransaction.WithSystemID(systemIdentifier))
	require.NoError(t, network.BroadcastTx(tx))
	require.Eventually(t, BlockchainContainsTx(tx, network), test.WaitDuration, test.WaitTick)
}
