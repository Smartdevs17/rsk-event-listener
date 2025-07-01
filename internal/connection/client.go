package connection

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
)

// RSKClient wraps ethclient.Client with RSK-specific functionality
type RSKClient struct {
	client  *ethclient.Client
	manager Manager
	logger  *logrus.Logger
}

// NewRSKClient creates a new RSK client wrapper
func NewRSKClient(manager Manager) *RSKClient {
	return &RSKClient{
		manager: manager,
		logger:  utils.GetLogger(),
	}
}

// GetClient returns the underlying ethclient
func (rc *RSKClient) GetClient() (*ethclient.Client, error) {
	if rc.client == nil {
		client, err := rc.manager.GetClient()
		if err != nil {
			return nil, err
		}
		rc.client = client
	}
	return rc.client, nil
}

// GetBlockByNumber gets a block by number with retry logic
func (rc *RSKClient) GetBlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	client, err := rc.GetClient()
	if err != nil {
		return nil, err
	}

	block, err := client.BlockByNumber(ctx, number)
	if err != nil {
		rc.logger.Error("Failed to get block", "number", number, "error", err)
		return nil, utils.NewAppError(utils.ErrCodeBlockchain, "Failed to get block", err.Error())
	}

	return block, nil
}

// GetLatestBlock gets the latest block
func (rc *RSKClient) GetLatestBlock(ctx context.Context) (*types.Block, error) {
	return rc.GetBlockByNumber(ctx, nil)
}

// GetBlockRange gets a range of blocks
func (rc *RSKClient) GetBlockRange(ctx context.Context, fromBlock, toBlock uint64) ([]*types.Block, error) {
	var blocks []*types.Block

	for i := fromBlock; i <= toBlock; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		block, err := rc.GetBlockByNumber(ctx, big.NewInt(int64(i)))
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}

	return blocks, nil
}

// FilterLogs filters logs with retry logic
func (rc *RSKClient) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	client, err := rc.GetClient()
	if err != nil {
		return nil, err
	}

	logs, err := client.FilterLogs(ctx, query)
	if err != nil {
		rc.logger.Error("Failed to filter logs", "query", query, "error", err)
		return nil, utils.NewAppError(utils.ErrCodeBlockchain, "Failed to filter logs", err.Error())
	}

	rc.logger.Debug("Filtered logs", "count", len(logs), "query", query)
	return logs, nil
}

// GetTransactionReceipt gets transaction receipt with retry
func (rc *RSKClient) GetTransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	client, err := rc.GetClient()
	if err != nil {
		return nil, err
	}

	receipt, err := client.TransactionReceipt(ctx, txHash)
	if err != nil {
		rc.logger.Error("Failed to get transaction receipt", "tx_hash", txHash.Hex(), "error", err)
		return nil, utils.NewAppError(utils.ErrCodeBlockchain, "Failed to get transaction receipt", err.Error())
	}

	return receipt, nil
}

// WaitForConfirmations waits for a transaction to be confirmed
func (rc *RSKClient) WaitForConfirmations(ctx context.Context, txHash common.Hash, confirmations int) error {
	receipt, err := rc.GetTransactionReceipt(ctx, txHash)
	if err != nil {
		return err
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return utils.NewAppError(utils.ErrCodeBlockchain, "Transaction failed", txHash.Hex())
	}

	targetBlock := receipt.BlockNumber.Uint64() + uint64(confirmations)

	ticker := time.NewTicker(30 * time.Second) // RSK block time
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			currentBlock, err := rc.manager.GetLatestBlockNumber()
			if err != nil {
				rc.logger.Warn("Failed to get latest block number", "error", err)
				continue
			}

			if currentBlock >= targetBlock {
				rc.logger.Info("Transaction confirmed",
					"tx_hash", txHash.Hex(),
					"confirmations", confirmations,
					"current_block", currentBlock,
					"tx_block", receipt.BlockNumber.Uint64())
				return nil
			}

			rc.logger.Debug("Waiting for confirmations",
				"tx_hash", txHash.Hex(),
				"current_block", currentBlock,
				"target_block", targetBlock)
		}
	}
}
