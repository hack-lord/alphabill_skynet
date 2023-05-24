package clientmock

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/alphabill-org/alphabill/internal/block"
	"github.com/alphabill-org/alphabill/internal/rpc/alphabill"
	"github.com/alphabill-org/alphabill/internal/txsystem"
)

type (
	MockAlphabillClient struct {
		m                        sync.Mutex
		recordedTxs              []*txsystem.Transaction
		txResponse               error
		maxBlockNumber           uint64
		maxRoundNumber           uint64
		shutdown                 bool
		blocks                   map[uint64]*block.Block
		txListener               func(tx *txsystem.Transaction)
		incrementOnFetch         atomic.Bool // if true, maxBlockNumber will be incremented on each GetBlocks call
		lastRequestedBlockNumber uint64
	}
	Option func(c *MockAlphabillClient)
)

func NewMockAlphabillClient(options ...Option) *MockAlphabillClient {
	mockClient := &MockAlphabillClient{blocks: map[uint64]*block.Block{}}
	for _, o := range options {
		o(mockClient)
	}
	return mockClient
}

func WithMaxBlockNumber(blockNumber uint64) Option {
	return func(c *MockAlphabillClient) {
		c.SetMaxBlockNumber(blockNumber)
	}
}

func WithMaxRoundNumber(roundNumber uint64) Option {
	return func(c *MockAlphabillClient) {
		c.maxRoundNumber = roundNumber
	}
}

func WithBlocks(blocks map[uint64]*block.Block) Option {
	return func(c *MockAlphabillClient) {
		c.blocks = blocks
	}
}

func (c *MockAlphabillClient) SendTransaction(ctx context.Context, tx *txsystem.Transaction) error {
	c.m.Lock()
	c.recordedTxs = append(c.recordedTxs, tx)
	rsp := c.txResponse
	cbf := c.txListener
	c.m.Unlock()

	if cbf != nil {
		cbf(tx)
	}
	return rsp
}

func (c *MockAlphabillClient) GetBlock(ctx context.Context, blockNumber uint64) (*block.Block, error) {
	if c.incrementOnFetch.Load() {
		defer c.SetMaxBlockNumber(blockNumber + 1)
	}
	c.m.Lock()
	defer c.m.Unlock()

	if c.blocks != nil {
		b := c.blocks[blockNumber]
		return b, nil
	}
	return nil, nil
}

func (c *MockAlphabillClient) GetBlocks(ctx context.Context, blockNumber, blockCount uint64) (*alphabill.GetBlocksResponse, error) {
	if c.incrementOnFetch.Load() {
		defer c.SetMaxBlockNumber(blockNumber + 1)
	}
	c.m.Lock()
	defer c.m.Unlock()

	c.lastRequestedBlockNumber = blockNumber
	batchMaxBlockNumber := blockNumber
	if blockNumber <= c.maxBlockNumber {
		var blocks []*block.Block
		b, f := c.blocks[blockNumber]
		if f {
			blocks = []*block.Block{b}
			batchMaxBlockNumber = b.UnicityCertificate.InputRecord.RoundNumber
		} else {
			blocks = []*block.Block{}
		}
		return &alphabill.GetBlocksResponse{
			MaxBlockNumber:      c.maxBlockNumber,
			MaxRoundNumber:      c.maxRoundNumber,
			Blocks:              blocks,
			BatchMaxBlockNumber: batchMaxBlockNumber,
		}, nil
	}
	return &alphabill.GetBlocksResponse{
		MaxBlockNumber:      c.maxBlockNumber,
		MaxRoundNumber:      c.maxRoundNumber,
		Blocks:              []*block.Block{},
		BatchMaxBlockNumber: batchMaxBlockNumber,
	}, nil
}

func (c *MockAlphabillClient) GetRoundNumber(ctx context.Context) (uint64, error) {
	c.m.Lock()
	defer c.m.Unlock()
	return c.maxRoundNumber, nil
}

func (c *MockAlphabillClient) Shutdown() error {
	c.m.Lock()
	defer c.m.Unlock()
	c.shutdown = true
	return nil
}

func (c *MockAlphabillClient) SetTxResponse(txResponse error) {
	c.m.Lock()
	defer c.m.Unlock()
	c.txResponse = txResponse
}

func (c *MockAlphabillClient) SetMaxBlockNumber(blockNumber uint64) {
	c.m.Lock()
	c.maxBlockNumber = blockNumber
	roundNumber := c.maxRoundNumber
	c.m.Unlock()
	if blockNumber > roundNumber {
		c.SetMaxRoundNumber(blockNumber)
	}
}

func (c *MockAlphabillClient) SetMaxRoundNumber(roundNumber uint64) {
	c.m.Lock()
	defer c.m.Unlock()
	if c.maxBlockNumber > roundNumber {
		panic(fmt.Sprintf("round number (%d) cannot be behind the block number (%d)", roundNumber, c.maxBlockNumber))
	}
	c.maxRoundNumber = roundNumber
}

func (c *MockAlphabillClient) SetBlock(b *block.Block) {
	c.m.Lock()
	defer c.m.Unlock()
	c.blocks[b.UnicityCertificate.InputRecord.RoundNumber] = b
}

func (c *MockAlphabillClient) GetRecordedTransactions() []*txsystem.Transaction {
	c.m.Lock()
	defer c.m.Unlock()
	return c.recordedTxs
}

func (c *MockAlphabillClient) ClearRecordedTransactions() {
	c.m.Lock()
	defer c.m.Unlock()
	c.recordedTxs = make([]*txsystem.Transaction, 0)
}

func (c *MockAlphabillClient) SetTxListener(txListener func(tx *txsystem.Transaction)) {
	c.m.Lock()
	defer c.m.Unlock()
	c.txListener = txListener
}

func (c *MockAlphabillClient) SetIncrementOnFetch(incrementOnFetch bool) {
	c.incrementOnFetch.Store(incrementOnFetch)
}

func (c *MockAlphabillClient) GetLastRequestedBlockNumber() uint64 {
	c.m.Lock()
	defer c.m.Unlock()
	return c.lastRequestedBlockNumber
}
