// File: internal/monitor/poller.go
package monitor

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sirupsen/logrus"
	"github.com/smartdevs17/rsk-event-listener/internal/connection"
	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
)

// BlockPoller handles block polling operations
type BlockPoller struct {
	connectionManager connection.Manager
	config            *MonitorConfig
	logger            *logrus.Logger

	mu           sync.RWMutex
	lastPollTime time.Time
	pollCount    uint64
	errorCount   uint64
}

// BlockInfo contains information about a polled block
type BlockInfo struct {
	Number    uint64    `json:"number"`
	Hash      string    `json:"hash"`
	Timestamp time.Time `json:"timestamp"`
	TxCount   int       `json:"tx_count"`
}

// NewBlockPoller creates a new block poller
func NewBlockPoller(connectionManager connection.Manager, config *MonitorConfig) *BlockPoller {
	return &BlockPoller{
		connectionManager: connectionManager,
		config:            config,
		logger:            utils.GetLogger(),
	}
}

// GetLatestBlock gets the latest block from the blockchain
func (bp *BlockPoller) GetLatestBlock(ctx context.Context) (*BlockInfo, error) {
	bp.mu.Lock()
	bp.pollCount++
	bp.lastPollTime = time.Now()
	bp.mu.Unlock()

	client, err := bp.connectionManager.GetClientWithContext(ctx)
	if err != nil {
		bp.recordError()
		return nil, utils.NewAppError(utils.ErrCodeConnection, "Failed to get client", err.Error())
	}

	// Get latest block number
	blockNumber, err := client.BlockNumber(ctx)
	if err != nil {
		bp.recordError()
		return nil, utils.NewAppError(utils.ErrCodeBlockchain, "Failed to get block number", err.Error())
	}

	// Get block details
	block, err := client.BlockByNumber(ctx, big.NewInt(int64(blockNumber)))
	if err != nil {
		bp.recordError()
		return nil, utils.NewAppError(utils.ErrCodeBlockchain, "Failed to get block", err.Error())
	}

	return &BlockInfo{
		Number:    blockNumber,
		Hash:      block.Hash().Hex(),
		Timestamp: time.Unix(int64(block.Time()), 0),
		TxCount:   len(block.Transactions()),
	}, nil
}

// GetBlock gets a specific block by number
func (bp *BlockPoller) GetBlock(ctx context.Context, blockNumber uint64) (*types.Block, error) {
	client, err := bp.connectionManager.GetClientWithContext(ctx)
	if err != nil {
		bp.recordError()
		return nil, utils.NewAppError(utils.ErrCodeConnection, "Failed to get client", err.Error())
	}

	block, err := client.BlockByNumber(ctx, big.NewInt(int64(blockNumber)))
	if err != nil {
		bp.recordError()
		return nil, utils.NewAppError(utils.ErrCodeBlockchain, "Failed to get block", err.Error())
	}

	return block, nil
}

// GetBlockRange gets a range of blocks
func (bp *BlockPoller) GetBlockRange(ctx context.Context, fromBlock, toBlock uint64) ([]*types.Block, error) {
	if fromBlock > toBlock {
		return nil, utils.NewAppError(utils.ErrCodeValidation, "Invalid block range", "fromBlock > toBlock")
	}

	blocks := make([]*types.Block, 0, toBlock-fromBlock+1)

	for blockNum := fromBlock; blockNum <= toBlock; blockNum++ {
		block, err := bp.GetBlock(ctx, blockNum)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}

	return blocks, nil
}

// GetStats returns poller statistics
func (bp *BlockPoller) GetStats() map[string]interface{} {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	return map[string]interface{}{
		"poll_count":     bp.pollCount,
		"error_count":    bp.errorCount,
		"last_poll_time": bp.lastPollTime,
	}
}

func (bp *BlockPoller) recordError() {
	bp.mu.Lock()
	bp.errorCount++
	bp.mu.Unlock()
}
