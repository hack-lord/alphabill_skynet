package blockchain

import (
	"context"
	"sync"
	"time"

	"gitdc.ee.guardtime.com/alphabill/alphabill/internal/errors/errstr"

	"gitdc.ee.guardtime.com/alphabill/alphabill/internal/eventbus"

	"gitdc.ee.guardtime.com/alphabill/alphabill/internal/errors"
	log "gitdc.ee.guardtime.com/alphabill/alphabill/internal/logger"
)

const (
	idle          phase = iota
	synchronizing       // TODO rename it to recovering? Needs input from Risto.
	proposing
	finalizing
	closing
)

var logger = log.CreateForPackage()
var ErrNilParameter = errors.New(errstr.NilArgument)

type Transaction interface{}

type State interface {
	Process(tx Transaction) error
	GetRootHash() []byte
	Rollback()
}
type phase int

// Blockchain proposes and finalizes blockchain blocks.
type Blockchain struct {
	ctx                 context.Context
	ctxCancel           context.CancelFunc
	conf                *Configuration
	phase               phase
	mutex               sync.Mutex
	transactionsInputCh <-chan interface{} // channel for incoming transactions
	blockchainInputCh   <-chan interface{} // channel for incoming blockchain events
	eventbus            *eventbus.EventBus
	state               State // blockchain state tree
	blockProposeTimer   *time.Timer
	currentBlock        *Block
	previousBlock       *Block
}

func New(ctx context.Context, eventbus *eventbus.EventBus, state State, configuration *Configuration) (*Blockchain, error) {
	if ctx == nil {
		return nil, ErrNilParameter
	}
	if eventbus == nil {
		return nil, ErrNilParameter
	}
	if state == nil {
		return nil, ErrNilParameter
	}
	if configuration == nil {
		return nil, ErrNilParameter
	}
	transactionsCh, ErrNilParameter := eventbus.Subscribe(TransactionBufferOutputChannelID, configuration.InputChannelCapacity)
	if ErrNilParameter != nil {
		return nil, ErrNilParameter
	}

	// create channel for listening consensus events
	consensusCh, ErrNilParameter := eventbus.Subscribe(InputChannelID, configuration.InputChannelCapacity)
	if ErrNilParameter != nil {
		return nil, ErrNilParameter
	}

	// block proposal timeout
	timer := time.NewTimer(configuration.BlockProposalTimeout)
	timer.Stop()

	context, cancelFunc := context.WithCancel(ctx)

	bc := &Blockchain{
		ctx:                 context,
		ctxCancel:           cancelFunc,
		eventbus:            eventbus, // used to change messages with other components
		transactionsInputCh: transactionsCh,
		blockchainInputCh:   consensusCh,
		state:               state,
		blockProposeTimer:   timer,
		conf:                configuration,
	}

	// TODO ledger synchronization. Needs input from Risto.

	bc.currentBlock = &Block{
		blockNr:   uint64(0),
		stateRoot: state.GetRootHash(),
	}
	bc.phase = idle

	// start main blockchain loop
	go bc.loop()

	return bc, nil
}

func (b *Blockchain) Close() error {
	b.mutex.Lock()
	b.phase = closing
	b.mutex.Unlock()
	b.ctxCancel()
	return nil
}

func (b *Blockchain) loop() {
	for {
		select {
		case <-b.ctx.Done():
			logger.Info("Exiting blockchain component main loop")
			return
		case e := <-b.blockchainInputCh:
			logger.Info("Handling blockchain input event %v", e)
			b.handleBlockchainEvent(e)
		case tx := <-b.transactionsInputCh:
			logger.Info("Handling tx event %v", tx)
			b.handleTxEvent(tx)
		case <-b.blockProposeTimer.C:
			logger.Info("Handling propose timeout")
			b.endPropose()
		}
	}
}

func (b *Blockchain) handleBlockchainEvent(event interface{}) {
	switch event.(type) {
	case StartBlockProposeEvent:
		b.startPropose(event.(StartBlockProposeEvent))
	default:
		logger.Warning("Invalid event: %v", event)
	}
}

func (b *Blockchain) startPropose(e StartBlockProposeEvent) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if b.phase == closing {
		logger.Info("Ignoring StartBlockProposeEvent. Blockchain component is closing.")
		return
	}
	if b.phase == synchronizing {
		// ledger synchronization isn't completed. Ignore StartBlockPropose event.
		logger.Info("Ignoring StartBlockProposeEvent. Ledger isn't synchronized.")
		return
	}

	// TODO what happens if b.phase == finalizing? Rollback and Sync? Needs more input from Risto.

	if b.phase == proposing {
		if (b.previousBlock.blockNr + 1) < e.blockNr {
			// current node does not have the latest block. Rollback the state and start synchronization.
			b.state.Rollback()
			b.phase = synchronizing
			b.eventbus.Submit(TransactionBufferInputChannelID, StopSendingTransactionsEvent{})
		}
		// TODO check if the given event is a duplicate. This part needs more input from Risto.
		return
	}

	b.phase = proposing
	b.previousBlock = b.currentBlock
	// create new empty block
	b.currentBlock = &Block{
		blockNr: e.blockNr,
	}
	// start timer
	b.blockProposeTimer.Reset(b.conf.BlockProposalTimeout)
	// start receiving transactions from tx-buffer
	b.eventbus.Submit(TransactionBufferInputChannelID, StartSendingTransactionsEvent{})
}

func (b *Blockchain) handleTxEvent(event interface{}) {

	switch event.(type) {
	case TransactionEvent:
		b.handleTx(event.(TransactionEvent))
	default:
		logger.Warning("Invalid event: %v", event)
	}
}

func (b *Blockchain) handleTx(e TransactionEvent) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if b.phase == closing {
		logger.Info("Ignoring StartBlockProposeEvent. Blockchain component is closing.")
		return
	}
	if b.phase != proposing {
		logger.Info("Received a transaction after proposing phase was ended")
		b.eventbus.Submit(TransactionBufferInputChannelID, e)
		return
	}
	err := b.state.Process(e.tx)
	if err != nil {
		logger.Info("Failed to update state. Error '%v', transaction '%v'", err, e.tx)
		return
	}
	b.currentBlock.appendTx(e.tx)
}

func (b *Blockchain) endPropose() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if b.phase != proposing {
		return
	}
	b.phase = finalizing
	b.eventbus.Submit(TransactionBufferInputChannelID, StopSendingTransactionsEvent{})
	b.currentBlock.stateRoot = b.state.GetRootHash()
	b.eventbus.Submit(OutputChannelID, NewBlockProposalEvent{block: b.currentBlock})
}

func (b *Blockchain) finalize() (*Block, error) {
	// TODO AB-62
	return nil, errors.ErrNotImplemented
}
