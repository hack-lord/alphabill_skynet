package blockchain

import (
	"context"
	"testing"
	"time"

	testtransaction "gitdc.ee.guardtime.com/alphabill/alphabill/internal/testutils/transaction"

	test "gitdc.ee.guardtime.com/alphabill/alphabill/internal/testutils"
	"github.com/stretchr/testify/require"

	"gitdc.ee.guardtime.com/alphabill/alphabill/internal/eventbus"
)

const DefaultChannelCapacity = 10

type mockState struct {
	rollback bool
}

func TestNew_Ok(t *testing.T) {
	eb := eventbus.New()
	ctx := context.Background()
	bc, err := New(ctx, eb, newState(), defaultConf())

	require.NoError(t, err)
	require.NotNil(t, bc)
}

func TestNew_ContextIsNil(t *testing.T) {
	eb := eventbus.New()
	_, err := New(nil, eb, newState(), defaultConf())

	require.ErrorIs(t, err, ErrNilParameter)
}

func TestNewEventBusIsNil(t *testing.T) {
	ctx := context.Background()
	_, err := New(ctx, nil, newState(), defaultConf())

	require.ErrorIs(t, err, ErrNilParameter)
}

func TestNewStateIsNil(t *testing.T) {
	eb := eventbus.New()
	ctx := context.Background()
	_, err := New(ctx, eb, nil, defaultConf())

	require.ErrorIs(t, err, ErrNilParameter)
}

func TestNew_ConfigurationIsNil(t *testing.T) {
	eb := eventbus.New()
	ctx := context.Background()
	_, err := New(ctx, eb, newState(), nil)

	require.ErrorIs(t, err, ErrNilParameter)
}

func TestBlockchain_HandleStartBlockProposeEvent(t *testing.T) {
	eb := eventbus.New()
	ctx := context.Background()
	bc, err := New(ctx, eb, newState(), defaultConf())

	require.NoError(t, err)

	err = eb.Submit(InputChannelID, StartBlockProposeEvent{})

	require.NoError(t, err)
	require.Eventually(t, func() bool { return bc.phase == proposing }, test.WaitDuration, test.WaitTick)
}

func TestBlockchain_HandleStartBlockProposeEventMultipleTimesWithSameHeight(t *testing.T) {
	eb := eventbus.New()
	ctx := context.Background()
	bc, err := New(ctx, eb, newState(), defaultConf())

	require.NoError(t, err)

	err = eb.Submit(InputChannelID, StartBlockProposeEvent{blockNr: 1})
	require.NoError(t, err)
	require.Eventually(t, func() bool { return bc.phase == proposing }, test.WaitDuration, test.WaitTick)

	err = eb.Submit(InputChannelID, StartBlockProposeEvent{blockNr: 1})
	require.NoError(t, err)
	require.Eventually(t, func() bool { return bc.phase == proposing }, test.WaitDuration, test.WaitTick)
}

func TestBlockchain_HandleStartBlockProposeEventMultipleTimesWithDifferentHeight(t *testing.T) {
	eb := eventbus.New()
	ctx := context.Background()
	state := newState()
	bc, err := New(ctx, eb, state, defaultConf())

	require.NoError(t, err)

	err = eb.Submit(InputChannelID, StartBlockProposeEvent{blockNr: 1})
	require.NoError(t, err)
	require.Eventually(t, func() bool { return bc.phase == proposing }, test.WaitDuration, test.WaitTick)

	err = eb.Submit(InputChannelID, StartBlockProposeEvent{blockNr: 3})
	require.NoError(t, err)
	require.Eventually(t, func() bool { return bc.phase == synchronizing }, test.WaitDuration, test.WaitTick)
	require.True(t, state.rollback)
}

func TestHandleTransactionEvent_PhaseNotProposing(t *testing.T) {
	eb := eventbus.New()
	txInChannel, err := eb.Subscribe(TransactionBufferInputChannelID, DefaultChannelCapacity)
	require.NoError(t, err)

	ctx := context.Background()
	state := newState()
	_, err = New(ctx, eb, state, defaultConf())
	require.NoError(t, err)

	tx := testtransaction.NewGenericTransaction(testtransaction.RandomBillTransfer())
	err = eb.Submit(TransactionBufferOutputChannelID, TransactionEvent{
		tx: tx,
	})
	require.Eventually(t, func() bool {
		proposal := <-txInChannel
		switch proposal.(type) {
		case TransactionEvent:
			p := proposal.(TransactionEvent)
			require.Equal(t, tx, p.tx)
			return true
		default:
			return false
		}
	}, test.WaitDuration, test.WaitTick)
}

func TestBlockchain_BlockProposeTimeout(t *testing.T) {
	eb := eventbus.New()

	blockChannel, err := eb.Subscribe(OutputChannelID, DefaultChannelCapacity)
	require.NoError(t, err)

	ctx := context.Background()
	state := newState()
	bc, err := New(ctx, eb, state, defaultConf())

	require.NoError(t, err)

	err = eb.Submit(InputChannelID, StartBlockProposeEvent{blockNr: 1})
	require.NoError(t, err)
	require.Eventually(t, func() bool { return bc.phase == proposing }, test.WaitDuration, test.WaitTick)

	err = eb.Submit(TransactionBufferOutputChannelID, TransactionEvent{
		tx: testtransaction.NewGenericTransaction(testtransaction.RandomBillTransfer()),
	})

	require.NoError(t, err)
	require.Eventually(t, func() bool {
		proposal := <-blockChannel
		switch proposal.(type) {
		case NewBlockProposalEvent:
			p := proposal.(NewBlockProposalEvent)
			require.Equal(t, uint64(1), p.block.blockNr)
			require.Equal(t, 1, len(p.block.transactions))
			require.Equal(t, []byte{0, 0, 0, 0, 1}, p.block.stateRoot)
			return true
		default:
			return false
		}
	}, test.WaitDuration, test.WaitTick)
	require.False(t, state.rollback)

	require.Equal(t, uint64(1), bc.currentBlock.blockNr)
	require.Equal(t, 1, len(bc.currentBlock.transactions))
	require.Equal(t, []byte{0, 0, 0, 0, 1}, bc.currentBlock.stateRoot)
}

func TestEndProposal_PhaseNotProposing(t *testing.T) {
	eb := eventbus.New()
	ctx := context.Background()
	state := newState()
	bc, err := New(ctx, eb, state, defaultConf())

	require.NoError(t, err)
	bc.phase = closing

	bc.endPropose()
	require.Equal(t, closing, bc.phase)
}

func TestBlockchain_Close(t *testing.T) {
	eb := eventbus.New()
	ctx := context.Background()
	state := newState()
	bc, err := New(ctx, eb, state, defaultConf())
	require.NoError(t, err)
	err = bc.Close()
	require.NoError(t, err)
	require.Equal(t, closing, bc.phase)
}

func (m *mockState) Process(tx Transaction) error {
	logger.Info("\t MockState \tProcessing tx %v", tx)
	return nil
}

func (m *mockState) GetRootHash() []byte {
	logger.Info("\t MockState \tReturn root")
	return []byte{0, 0, 0, 0, 1}
}

func (m *mockState) Rollback() {
	logger.Info("\t MockState \tRollback")
	m.rollback = true
}

func newState() *mockState {
	return &mockState{}
}

func defaultConf() *Configuration {
	return &Configuration{
		BlockProposalTimeout: time.Millisecond * 500,
		InputChannelCapacity: DefaultChannelCapacity,
	}
}
