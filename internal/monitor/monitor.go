// File: internal/monitor/monitor.go
package monitor

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sirupsen/logrus"
	"github.com/smartdevs17/rsk-event-listener/internal/connection"
	"github.com/smartdevs17/rsk-event-listener/internal/metrics"
	"github.com/smartdevs17/rsk-event-listener/internal/models"
	"github.com/smartdevs17/rsk-event-listener/internal/storage"
	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
)

// Monitor defines the event monitor interface
type Monitor interface {
	// Lifecycle management
	Start(ctx context.Context) error
	Stop() error
	IsRunning() bool

	// Contract management
	AddContract(contract *models.Contract) error
	RemoveContract(address common.Address) error
	GetContracts() []*models.Contract
	UpdateContract(contract *models.Contract) error

	// Event monitoring
	GetLatestProcessedBlock() (uint64, error)
	SetStartBlock(blockNumber uint64) error
	ProcessBlock(ctx context.Context, blockNumber uint64) (*BlockResult, error)
	ProcessBlockRange(ctx context.Context, fromBlock, toBlock uint64) (*RangeResult, error)

	// Backfill and recovery
	BackfillEvents(ctx context.Context, fromBlock, toBlock uint64) error
	HandleReorg(ctx context.Context, newBlock, oldBlock uint64) error

	// Statistics and monitoring
	GetStats() *MonitorStats
	GetHealth() *HealthStatus
}

// EventMonitor implements the Monitor interface
type EventMonitor struct {
	// Dependencies
	connectionManager connection.Manager
	storage           storage.Storage
	logger            *logrus.Logger

	// Configuration
	config *MonitorConfig

	// State management
	mu           sync.RWMutex
	running      bool
	contracts    map[common.Address]*models.Contract
	eventFilters map[common.Address][]EventFilterSpec
	stopChan     chan struct{}
	stopOnce     sync.Once
	wg           sync.WaitGroup

	// Components
	poller       *BlockPoller
	parser       *EventParser
	filter       *EventFilter
	reorgHandler *ReorgHandler

	// Statistics
	stats          *MonitorStats
	metricsManager *metrics.Manager
}

// MonitorConfig holds monitor configuration
type MonitorConfig struct {
	PollInterval       time.Duration `json:"poll_interval"`
	BatchSize          int           `json:"batch_size"`
	ConfirmationBlocks int           `json:"confirmation_blocks"`
	StartBlock         uint64        `json:"start_block"`
	EnableWebSocket    bool          `json:"enable_websocket"`
	MaxReorgDepth      int           `json:"max_reorg_depth"`
	ConcurrentBlocks   int           `json:"concurrent_blocks"`
	RetryAttempts      int           `json:"retry_attempts"`
	RetryDelay         time.Duration `json:"retry_delay"`
}

// EventFilterSpec defines an event filter specification
type EventFilterSpec struct {
	EventName    string   `json:"event_name"`
	EventSig     string   `json:"event_signature"`
	Topics       []string `json:"topics"`
	RequiredData []string `json:"required_data,omitempty"`
}

// BlockResult contains the result of processing a single block
type BlockResult struct {
	BlockNumber    uint64          `json:"block_number"`
	BlockHash      string          `json:"block_hash"`
	Timestamp      time.Time       `json:"timestamp"`
	EventsFound    int             `json:"events_found"`
	EventsSaved    int             `json:"events_saved"`
	ProcessingTime time.Duration   `json:"processing_time"`
	Events         []*models.Event `json:"events"`
	Error          error           `json:"error,omitempty"`
}

// RangeResult contains the result of processing a block range
type RangeResult struct {
	FromBlock       uint64                  `json:"from_block"`
	ToBlock         uint64                  `json:"to_block"`
	BlocksProcessed int                     `json:"blocks_processed"`
	TotalEvents     int                     `json:"total_events"`
	ProcessingTime  time.Duration           `json:"processing_time"`
	BlockResults    map[uint64]*BlockResult `json:"block_results"`
	Errors          []error                 `json:"errors,omitempty"`
}

// MonitorStats provides monitoring statistics
type MonitorStats struct {
	StartTime            time.Time     `json:"start_time"`
	Uptime               time.Duration `json:"uptime"`
	IsRunning            bool          `json:"is_running"`
	LatestProcessedBlock uint64        `json:"latest_processed_block"`
	TotalBlocksProcessed uint64        `json:"total_blocks_processed"`
	TotalEventsFound     uint64        `json:"total_events_found"`
	TotalEventsSaved     uint64        `json:"total_events_saved"`
	ContractsMonitored   int           `json:"contracts_monitored"`
	AverageBlockTime     time.Duration `json:"average_block_time"`
	ProcessingRate       float64       `json:"processing_rate"` // blocks per second
	ErrorCount           uint64        `json:"error_count"`
	LastError            *string       `json:"last_error,omitempty"`
	LastErrorTime        *time.Time    `json:"last_error_time,omitempty"`
}

// HealthStatus provides health information
type HealthStatus struct {
	Healthy            bool          `json:"healthy"`
	LastBlockProcessed time.Time     `json:"last_block_processed"`
	BlockLag           time.Duration `json:"block_lag"`
	ConnectionHealthy  bool          `json:"connection_healthy"`
	StorageHealthy     bool          `json:"storage_healthy"`
	Issues             []string      `json:"issues,omitempty"`
}

// NewEventMonitor creates a new event monitor
func NewEventMonitor(
	connectionManager connection.Manager,
	storage storage.Storage,
	config *MonitorConfig,
) *EventMonitor {

	monitor := &EventMonitor{
		connectionManager: connectionManager,
		storage:           storage,
		config:            config,
		logger:            utils.GetLogger(),
		contracts:         make(map[common.Address]*models.Contract),
		eventFilters:      make(map[common.Address][]EventFilterSpec),
		stopChan:          make(chan struct{}),
		stats: &MonitorStats{
			StartTime: time.Now(),
		},
	}

	// Initialize components
	monitor.poller = NewBlockPoller(connectionManager, config)
	monitor.parser = NewEventParser(config)
	monitor.filter = NewEventFilter(config)
	monitor.reorgHandler = NewReorgHandler(connectionManager, storage, config)

	return monitor
}

// Start starts the event monitor
func (em *EventMonitor) Start(ctx context.Context) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	if em.running {
		return utils.NewAppError(utils.ErrCodeInternal, "Monitor already running", "")
	}

	em.logger.Info("Starting event monitor")

	// Load contracts from storage
	if err := em.loadContracts(ctx); err != nil {
		return utils.NewAppError(utils.ErrCodeInternal, "Failed to load contracts", err.Error())
	}

	// Start monitoring
	em.running = true
	em.stats.StartTime = time.Now()
	em.stats.IsRunning = true

	// Start main monitoring loop
	em.wg.Add(1)
	go em.monitoringLoop(ctx)

	em.logger.Info("Event monitor started",
		"contracts", len(em.contracts),
		"poll_interval", em.config.PollInterval)

	return nil
}

// Stop stops the event monitor
func (em *EventMonitor) Stop() error {
	em.mu.Lock()
	defer em.mu.Unlock()

	if !em.running {
		return nil
	}

	em.logger.Info("Stopping event monitor")

	em.running = false
	em.stats.IsRunning = false

	em.stopOnce.Do(func() {
		close(em.stopChan)
	})

	// Wait for goroutines to finish
	em.wg.Wait()

	em.logger.Info("Event monitor stopped")
	return nil
}

// IsRunning returns whether the monitor is running
func (em *EventMonitor) IsRunning() bool {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.running
}

// AddContract adds a contract to monitor
func (em *EventMonitor) AddContract(contract *models.Contract) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	// Save to storage
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := em.storage.SaveContract(ctx, contract); err != nil {
		return err
	}

	// Add to memory
	em.contracts[contract.Address] = contract

	// Parse ABI and create event filters
	filters, err := em.createEventFilters(contract)
	if err != nil {
		em.logger.Warn("Failed to create event filters for contract",
			"address", contract.Address.Hex(), "error", err)
		// Continue without filters rather than failing
		filters = []EventFilterSpec{}
	}
	em.eventFilters[contract.Address] = filters

	em.logger.Info("Contract added to monitoring",
		"address", contract.Address.Hex(),
		"name", contract.Name,
		"filters", len(filters))

	return nil
}

// RemoveContract removes a contract from monitoring
func (em *EventMonitor) RemoveContract(address common.Address) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	// Remove from storage
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := em.storage.DeleteContract(ctx, address.Hex()); err != nil {
		return err
	}

	// Remove from memory
	delete(em.contracts, address)
	delete(em.eventFilters, address)

	em.logger.Info("Contract removed from monitoring", "address", address.Hex())
	return nil
}

// GetContracts returns all monitored contracts
func (em *EventMonitor) GetContracts() []*models.Contract {
	em.mu.RLock()
	defer em.mu.RUnlock()

	contracts := make([]*models.Contract, 0, len(em.contracts))
	for _, contract := range em.contracts {
		contracts = append(contracts, contract)
	}
	return contracts
}

// UpdateContract updates a monitored contract
func (em *EventMonitor) UpdateContract(contract *models.Contract) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	// Update in storage
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	contract.UpdatedAt = time.Now()
	if err := em.storage.UpdateContract(ctx, contract); err != nil {
		return err
	}

	// Update in memory
	em.contracts[contract.Address] = contract

	// Recreate event filters
	filters, err := em.createEventFilters(contract)
	if err != nil {
		em.logger.Warn("Failed to recreate event filters for contract",
			"address", contract.Address.Hex(), "error", err)
		filters = []EventFilterSpec{}
	}
	em.eventFilters[contract.Address] = filters

	em.logger.Info("Contract updated", "address", contract.Address.Hex(), "name", contract.Name)
	return nil
}

// GetLatestProcessedBlock returns the latest processed block number
func (em *EventMonitor) GetLatestProcessedBlock() (uint64, error) {
	return em.storage.GetLatestProcessedBlock()
}

// SetStartBlock sets the starting block for monitoring
func (em *EventMonitor) SetStartBlock(blockNumber uint64) error {
	return em.storage.SetLatestProcessedBlock(blockNumber)
}

// ProcessBlock processes a single block for events
func (em *EventMonitor) ProcessBlock(ctx context.Context, blockNumber uint64) (*BlockResult, error) {
	startTime := time.Now()

	result := &BlockResult{
		BlockNumber: blockNumber,
		Timestamp:   startTime,
		Events:      []*models.Event{},
	}

	// Get block data
	client, err := em.connectionManager.GetClientWithContext(ctx)
	if err != nil {
		result.Error = err
		return result, err
	}

	block, err := client.BlockByNumber(ctx, big.NewInt(int64(blockNumber)))
	if err != nil {
		result.Error = utils.NewAppError(utils.ErrCodeBlockchain, "Failed to get block", err.Error())
		return result, result.Error
	}

	result.BlockHash = block.Hash().Hex()
	result.Timestamp = time.Unix(int64(block.Time()), 0)

	// Create filter query for all monitored contracts
	addresses := make([]common.Address, 0, len(em.contracts))
	em.mu.RLock()
	for addr := range em.contracts {
		addresses = append(addresses, addr)
	}
	em.mu.RUnlock()

	if len(addresses) == 0 {
		// No contracts to monitor
		result.ProcessingTime = time.Since(startTime)
		return result, nil
	}

	// Filter logs for monitored contracts
	filterQuery := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(blockNumber)),
		ToBlock:   big.NewInt(int64(blockNumber)),
		Addresses: addresses,
	}

	logs, err := client.FilterLogs(ctx, filterQuery)
	if err != nil {
		result.Error = utils.NewAppError(utils.ErrCodeBlockchain, "Failed to filter logs", err.Error())
		return result, result.Error
	}

	result.EventsFound = len(logs)

	// Parse logs into events
	events := make([]*models.Event, 0, len(logs))
	for _, log := range logs {
		event, err := em.parser.ParseLog(log, em.contracts[log.Address])
		if err != nil {
			em.logger.Warn("Failed to parse log", "error", err, "tx_hash", log.TxHash.Hex())
			continue
		}

		if event != nil {
			events = append(events, event)
		}
	}

	result.Events = events
	result.EventsSaved = len(events)

	// Save events to storage
	if len(events) > 0 {
		if err := em.storage.SaveEvents(ctx, events); err != nil {
			result.Error = utils.NewAppError(utils.ErrCodeDatabase, "Failed to save events", err.Error())
			return result, result.Error
		}
	}

	// Update block processing status
	blockStatus := &storage.BlockProcessingStatus{
		BlockNumber:    blockNumber,
		Status:         "completed",
		StartedAt:      startTime,
		CompletedAt:    &[]time.Time{time.Now()}[0],
		EventsFound:    result.EventsFound,
		EventsSaved:    result.EventsSaved,
		ProcessingTime: &[]time.Duration{time.Since(startTime)}[0],
	}

	if err := em.storage.SetBlockProcessingStatus(blockStatus); err != nil {
		em.logger.Warn("Failed to update block processing status", "error", err)
	}

	// Update statistics
	em.updateStats(result)

	result.ProcessingTime = time.Since(startTime)

	em.logger.Debug("Block processed",
		"block", blockNumber,
		"events_found", result.EventsFound,
		"events_saved", result.EventsSaved,
		"processing_time", result.ProcessingTime)

	// Record metrics
	if em.metricsManager != nil {
		prometheus := em.metricsManager.GetPrometheusMetrics()

		// Record block processing metrics
		prometheus.RecordBlockProcessed()
		prometheus.RecordBlockProcessingDuration(time.Since(startTime))
		prometheus.UpdateLatestProcessedBlock(blockNumber)

		if result != nil {
			// Record event metrics
			for _, event := range result.Events {
				status := "success"
				if result.Error != nil {
					status = "error"
				}
				prometheus.RecordEventProcessed(
					event.Address,
					event.EventName,
					status,
				)
			}
		}

		if err != nil {
			// This could be expanded to track different error types
			prometheus.RecordConnectionError("unknown", "processing_error")
		}
	}

	return result, nil
}

// monitoringLoop is the main monitoring loop
func (em *EventMonitor) monitoringLoop(ctx context.Context) {
	defer em.wg.Done()

	ticker := time.NewTicker(em.config.PollInterval)
	defer ticker.Stop()

	em.logger.Info("Starting monitoring loop", "interval", em.config.PollInterval)

	for {
		select {
		case <-ctx.Done():
			em.logger.Info("Monitoring loop stopped by context")
			return
		case <-em.stopChan:
			em.logger.Info("Monitoring loop stopped by stop signal")
			return
		case <-ticker.C:
			if err := em.pollForNewBlocks(ctx); err != nil {
				em.logger.Error("Error polling for new blocks", "error", err)
				em.recordError(err)
			}
		}
	}
}

// pollForNewBlocks polls for new blocks and processes them
func (em *EventMonitor) pollForNewBlocks(ctx context.Context) error {
	// Get latest block from blockchain
	latestChainBlock, err := em.connectionManager.GetLatestBlockNumber()
	if err != nil {
		return utils.NewAppError(utils.ErrCodeConnection, "Failed to get latest block number", err.Error())
	}

	// Get latest processed block from storage
	latestProcessedBlock, err := em.storage.GetLatestProcessedBlock()
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to get latest processed block", err.Error())
	}

	// Calculate confirmation blocks
	confirmedBlock := latestChainBlock
	if latestChainBlock > uint64(em.config.ConfirmationBlocks) {
		confirmedBlock = latestChainBlock - uint64(em.config.ConfirmationBlocks)
	}

	// Process new blocks
	if confirmedBlock > latestProcessedBlock {
		fromBlock := latestProcessedBlock + 1
		toBlock := confirmedBlock

		// Limit batch size
		if toBlock-fromBlock+1 > uint64(em.config.BatchSize) {
			toBlock = fromBlock + uint64(em.config.BatchSize) - 1
		}

		em.logger.Debug("Processing block range",
			"from", fromBlock,
			"to", toBlock,
			"latest_chain", latestChainBlock,
			"confirmed", confirmedBlock)

		_, err := em.ProcessBlockRange(ctx, fromBlock, toBlock)
		if err != nil {
			return err
		}

		// Update latest processed block
		if err := em.storage.SetLatestProcessedBlock(toBlock); err != nil {
			em.logger.Error("Failed to update latest processed block", "error", err)
		}
	}

	return nil
}

// HandleReorg handles a blockchain reorganization (required by Monitor interface)
func (em *EventMonitor) HandleReorg(ctx context.Context, newBlock, oldBlock uint64) error {
	// You can delegate to your reorgHandler if you want
	reorgEvent, err := em.reorgHandler.DetectReorg(ctx, oldBlock)
	if err != nil {
		return err
	}
	if reorgEvent == nil {
		// No reorg detected
		return nil
	}
	return em.reorgHandler.HandleReorg(ctx, reorgEvent)
}

// Helper methods will be implemented in the next part...
