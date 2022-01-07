package blockchain

import "time"

type Configuration struct {
	BlockProposalTimeout time.Duration
	InputChannelCapacity uint
}
