package blockchain

import "gitdc.ee.guardtime.com/alphabill/alphabill/internal/txbuffer"

const (
	InputChannelID  = "blockchain-input"
	OutputChannelID = "blockchain-output"
)

// StartBlockProposeEvent is written to InputChannelID channel.
type StartBlockProposeEvent struct {
	blockNr uint64
	// TODO needs more information from Risto.
}

// NewBlockProposalEvent is written to OutputChannelID channel.
type NewBlockProposalEvent struct {
	block *Block
}

//FinalizeBlockEvent is written to InputChannelID channel
type FinalizeBlockEvent struct {
	block *Block
}

// TODO move transaction related events to tx-buffer package
const (
	TransactionBufferInputChannelID  = "transaction-buffer-input"
	TransactionBufferOutputChannelID = "transaction-buffer-output"
)

// StartSendingTransactionsEvent is written to TransactionBufferInputChannelID channel.
type StartSendingTransactionsEvent struct {
}

// StopSendingTransactionsEvent is written to TransactionBufferInputChannelID channel.
type StopSendingTransactionsEvent struct {
}

// TransactionEvent is written to TransactionBufferInputChannelID or TransactionBufferOutputChannelID channel.
type TransactionEvent struct {
	tx txbuffer.Transaction
}
