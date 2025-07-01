// File: internal/monitor/reorg_handler.go
package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/smartdevs17/rsk-event-listener/internal/connection"
	"github.com/smartdevs17/rsk-event-listener/internal/models"
	"github.com/smartdevs17/rsk-event-listener/internal/storage"
	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
)

// ReorgHandler handles blockchain reorganizations
type ReorgHandler struct {
	connectionManager connection.Manager
	storage           storage.Storage
	config            *MonitorConfig
	logger            *logrus.Logger
}

// ReorgEvent represents a reorganization event
type ReorgEvent struct {
	DetectedAt    time.Time `json:"detected_at"`
	OldBlockHash  string    `json:"old_block_hash"`
	NewBlockHash  string    `json:"new_block_hash"`
	BlockNumber   uint64    `json:"block_number"`
	Depth         int       `json:"depth"`
	EventsRemoved int       `json:"events_removed"`
	EventsAdded   int       `json:"events_added"`
}

// NewReorgHandler creates a new reorganization handler
func NewReorgHandler(
	connectionManager connection.Manager,
	storage storage.Storage,
	config *MonitorConfig,
) *ReorgHandler {
	return &ReorgHandler{
		connectionManager: connectionManager,
		storage:           storage,
		config:            config,
		logger:            utils.GetLogger(),
	}
}

// DetectReorg detects if a reorganization has occurred
func (rh *ReorgHandler) DetectReorg(ctx context.Context, blockNumber uint64) (*ReorgEvent, error) {
	// Get current block hash from blockchain
	client, err := rh.connectionManager.GetClientWithContext(ctx)
	if err != nil {
		return nil, utils.NewAppError(utils.ErrCodeConnection, "Failed to get client", err.Error())
	}

	currentBlock, err := client.BlockByNumber(ctx, nil)
	if err != nil {
		return nil, utils.NewAppError(utils.ErrCodeBlockchain, "Failed to get current block", err.Error())
	}

	// Get stored block hash
	storedHash, err := rh.storage.GetBlockHash(blockNumber)
	if err != nil {
		// Block not found in storage, not a reorg
		return nil, nil
	}

	// Compare hashes
	if currentBlock.Hash().Hex() != storedHash {
		reorgEvent := &ReorgEvent{
			DetectedAt:   time.Now(),
			OldBlockHash: storedHash,
			NewBlockHash: currentBlock.Hash().Hex(),
			BlockNumber:  blockNumber,
			Depth:        1, // Will be calculated in HandleReorg
		}
		return reorgEvent, nil
	}

	return nil, nil
}

// HandleReorg handles a detected reorganization
func (rh *ReorgHandler) HandleReorg(ctx context.Context, reorgEvent *ReorgEvent) error {
	rh.logger.Warn("Handling blockchain reorganization",
		"block_number", reorgEvent.BlockNumber,
		"old_hash", reorgEvent.OldBlockHash,
		"new_hash", reorgEvent.NewBlockHash)

	// Find the depth of the reorganization
	depth, err := rh.findReorgDepth(ctx, reorgEvent.BlockNumber)
	if err != nil {
		return err
	}

	reorgEvent.Depth = depth

	if depth > rh.config.MaxReorgDepth {
		return utils.NewAppError(utils.ErrCodeBlockchain,
			"Reorganization too deep",
			fmt.Sprintf("depth: %d, max: %d", depth, rh.config.MaxReorgDepth))
	}

	// Remove events from reorganized blocks
	fromBlock := reorgEvent.BlockNumber - uint64(depth) + 1
	toBlock := reorgEvent.BlockNumber

	removedCount, err := rh.storage.DeleteEventsByBlockRange(ctx, fromBlock, toBlock)
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to remove reorganized events", err.Error())
	}

	reorgEvent.EventsRemoved = removedCount

	// Update latest processed block to trigger reprocessing
	if err := rh.storage.SetLatestProcessedBlock(fromBlock - 1); err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to update latest processed block", err.Error())
	}

	// Log reorganization event
	if err := rh.logReorgEvent(ctx, reorgEvent); err != nil {
		rh.logger.Warn("Failed to log reorganization event", "error", err)
	}

	rh.logger.Info("Reorganization handled successfully",
		"depth", depth,
		"events_removed", removedCount,
		"reprocess_from", fromBlock)

	return nil
}

// findReorgDepth finds the depth of the reorganization
func (rh *ReorgHandler) findReorgDepth(ctx context.Context, startBlock uint64) (int, error) {
	client, err := rh.connectionManager.GetClientWithContext(ctx)
	if err != nil {
		return 0, err
	}

	depth := 1
	currentBlock := startBlock

	for depth <= rh.config.MaxReorgDepth {
		if currentBlock == 0 {
			break
		}

		currentBlock--

		// Get current block hash from blockchain
		block, err := client.BlockByNumber(ctx, nil)
		if err != nil {
			return 0, utils.NewAppError(utils.ErrCodeBlockchain, "Failed to get block", err.Error())
		}

		// Get stored block hash
		storedHash, err := rh.storage.GetBlockHash(currentBlock)
		if err != nil {
			// Block not found, assume this is the end of reorg
			break
		}

		// If hashes match, we found the common ancestor
		if block.Hash().Hex() == storedHash {
			break
		}

		depth++
	}

	return depth, nil
}

// logReorgEvent logs the reorganization event
func (rh *ReorgHandler) logReorgEvent(ctx context.Context, reorgEvent *ReorgEvent) error {
	// Create reorg log entry
	logEntry := map[string]interface{}{
		"event_type":     "blockchain_reorg",
		"detected_at":    reorgEvent.DetectedAt,
		"block_number":   reorgEvent.BlockNumber,
		"old_block_hash": reorgEvent.OldBlockHash,
		"new_block_hash": reorgEvent.NewBlockHash,
		"depth":          reorgEvent.Depth,
		"events_removed": reorgEvent.EventsRemoved,
		"events_added":   reorgEvent.EventsAdded,
	}

	// Store in database (assuming we have a logs table)
	return rh.storage.LogEvent(ctx, "reorg", logEntry)
}

// GetReorgHistory returns reorganization history
func (rh *ReorgHandler) GetReorgHistory(ctx context.Context, limit int) ([]*ReorgEvent, error) {
	// This would query the logs table for reorg events
	logs, err := rh.storage.GetLogsByType(ctx, "reorg", limit)
	if err != nil {
		return nil, err
	}

	reorgs := make([]*ReorgEvent, 0, len(logs))
	for _, log := range logs {
		reorg := &ReorgEvent{
			DetectedAt:    log.CreatedAt,
			BlockNumber:   uint64(log.Data["block_number"].(float64)),
			OldBlockHash:  log.Data["old_block_hash"].(string),
			NewBlockHash:  log.Data["new_block_hash"].(string),
			Depth:         int(log.Data["depth"].(float64)),
			EventsRemoved: int(log.Data["events_removed"].(float64)),
			EventsAdded:   int(log.Data["events_added"].(float64)),
		}
		reorgs = append(reorgs, reorg)
	}

	return reorgs, nil
}

// Additional helper methods for monitor.go

// createEventFilters creates event filter specifications from contract ABI
func (em *EventMonitor) createEventFilters(contract *models.Contract) ([]EventFilterSpec, error) {
	return em.filter.CreateEventFilters(contract)
}

// loadContracts loads contracts from storage
func (em *EventMonitor) loadContracts(ctx context.Context) error {
	contracts, err := em.storage.GetAllContracts(ctx)
	if err != nil {
		return err
	}

	for _, contract := range contracts {
		em.contracts[contract.Address] = contract

		filters, err := em.createEventFilters(contract)
		if err != nil {
			em.logger.Warn("Failed to create event filters for contract",
				"address", contract.Address.Hex(), "error", err)
			filters = []EventFilterSpec{}
		}
		em.eventFilters[contract.Address] = filters
	}

	em.logger.Info("Loaded contracts from storage", "count", len(contracts))
	return nil
}

// ProcessBlockRange processes a range of blocks
func (em *EventMonitor) ProcessBlockRange(ctx context.Context, fromBlock, toBlock uint64) (*RangeResult, error) {
	startTime := time.Now()

	result := &RangeResult{
		FromBlock:    fromBlock,
		ToBlock:      toBlock,
		BlockResults: make(map[uint64]*BlockResult),
		Errors:       []error{},
	}

	for blockNum := fromBlock; blockNum <= toBlock; blockNum++ {
		blockResult, err := em.ProcessBlock(ctx, blockNum)
		if err != nil {
			result.Errors = append(result.Errors, err)
			em.logger.Error("Failed to process block", "block", blockNum, "error", err)
			continue
		}

		result.BlockResults[blockNum] = blockResult
		result.BlocksProcessed++
		result.TotalEvents += blockResult.EventsFound
	}

	result.ProcessingTime = time.Since(startTime)

	em.logger.Info("Block range processed",
		"from", fromBlock,
		"to", toBlock,
		"blocks_processed", result.BlocksProcessed,
		"total_events", result.TotalEvents,
		"processing_time", result.ProcessingTime)

	return result, nil
}

// BackfillEvents backfills events for a block range
func (em *EventMonitor) BackfillEvents(ctx context.Context, fromBlock, toBlock uint64) error {
	em.logger.Info("Starting event backfill", "from", fromBlock, "to", toBlock)

	batchSize := uint64(em.config.BatchSize)
	for start := fromBlock; start <= toBlock; start += batchSize {
		end := start + batchSize - 1
		if end > toBlock {
			end = toBlock
		}

		_, err := em.ProcessBlockRange(ctx, start, end)
		if err != nil {
			return err
		}

		// Update progress
		if err := em.storage.SetLatestProcessedBlock(end); err != nil {
			em.logger.Warn("Failed to update backfill progress", "error", err)
		}
	}

	em.logger.Info("Event backfill completed", "from", fromBlock, "to", toBlock)
	return nil
}

// GetStats returns monitor statistics
func (em *EventMonitor) GetStats() *MonitorStats {
	em.mu.RLock()
	defer em.mu.RUnlock()

	em.stats.Uptime = time.Since(em.stats.StartTime)
	em.stats.ContractsMonitored = len(em.contracts)

	return em.stats
}

// GetHealth returns monitor health status
func (em *EventMonitor) GetHealth() *HealthStatus {
	health := &HealthStatus{
		Healthy: true,
		Issues:  []string{},
	}

	// Check connection health
	_, err := em.connectionManager.GetLatestBlockNumber()
	health.ConnectionHealthy = err == nil
	if err != nil {
		health.Healthy = false
		health.Issues = append(health.Issues, "Connection unhealthy: "+err.Error())
	}

	// Check storage health
	_, err = em.storage.GetLatestProcessedBlock()
	health.StorageHealthy = err == nil
	if err != nil {
		health.Healthy = false
		health.Issues = append(health.Issues, "Storage unhealthy: "+err.Error())
	}

	// Check block lag
	if health.ConnectionHealthy {
		latestChain, _ := em.connectionManager.GetLatestBlockNumber()
		latestProcessed, _ := em.storage.GetLatestProcessedBlock()

		if latestChain > latestProcessed {
			blocksBehind := latestChain - latestProcessed
			health.BlockLag = time.Duration(blocksBehind) * 30 * time.Second // Assume 30s block time

			if blocksBehind > 10 {
				health.Issues = append(health.Issues,
					fmt.Sprintf("Block lag detected: %d blocks behind", blocksBehind))
			}
		}
	}

	return health
}

// updateStats updates monitor statistics
func (em *EventMonitor) updateStats(result *BlockResult) {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.stats.TotalBlocksProcessed++
	em.stats.TotalEventsFound += uint64(result.EventsFound)
	em.stats.TotalEventsSaved += uint64(result.EventsSaved)
	em.stats.LatestProcessedBlock = result.BlockNumber

	if result.Error != nil {
		em.stats.ErrorCount++
		errorStr := result.Error.Error()
		em.stats.LastError = &errorStr
		now := time.Now()
		em.stats.LastErrorTime = &now
	}

	// Calculate processing rate
	if em.stats.TotalBlocksProcessed > 0 {
		em.stats.ProcessingRate = float64(em.stats.TotalBlocksProcessed) /
			time.Since(em.stats.StartTime).Seconds()
	}
}

// recordError records an error in statistics
func (em *EventMonitor) recordError(err error) {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.stats.ErrorCount++
	errorStr := err.Error()
	em.stats.LastError = &errorStr
	now := time.Now()
	em.stats.LastErrorTime = &now
}
