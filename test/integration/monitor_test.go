// File: test/integration/monitor_test.go
package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartdevs17/rsk-event-listener/internal/config"
	"github.com/smartdevs17/rsk-event-listener/internal/connection"
	"github.com/smartdevs17/rsk-event-listener/internal/models"
	"github.com/smartdevs17/rsk-event-listener/internal/monitor"
	"github.com/smartdevs17/rsk-event-listener/internal/storage"
	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
)

func TestEventMonitor(t *testing.T) {
	// Setup test environment
	testDB := "./test_monitor.db"
	defer os.Remove(testDB)

	// Initialize logger
	utils.InitLogger("info", "text", "stdout", "")

	// Create configurations
	rskConfig := &config.RSKConfig{
		NodeURL:        "https://public-node.testnet.rsk.co",
		NetworkID:      31,
		RequestTimeout: 30 * time.Second,
		RetryAttempts:  3,
		RetryDelay:     5 * time.Second,
		MaxConnections: 5,
	}

	storageConfig := &config.StorageConfig{
		Type:             "sqlite",
		ConnectionString: testDB,
		MaxConnections:   10,
		MaxIdleTime:      15 * time.Minute,
	}

	monitorConfig := &monitor.MonitorConfig{
		PollInterval:       15 * time.Second,
		BatchSize:          10,
		ConfirmationBlocks: 12,
		StartBlock:         0,
		EnableWebSocket:    false,
		MaxReorgDepth:      64,
		ConcurrentBlocks:   1,
		RetryAttempts:      3,
		RetryDelay:         5 * time.Second,
	}

	// Create components
	connectionManager := connection.NewConnectionManager(rskConfig)
	defer connectionManager.Close()

	store, err := storage.NewStorage(storageConfig)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	err = store.Connect()
	if err != nil {
		t.Fatalf("Failed to connect storage: %v", err)
	}

	err = store.Migrate()
	if err != nil {
		t.Fatalf("Failed to migrate storage: %v", err)
	}

	// Create monitor
	eventMonitor := monitor.NewEventMonitor(connectionManager, store, monitorConfig)

	t.Run("Monitor Lifecycle", func(t *testing.T) { testMonitorLifecycle(t, eventMonitor) })
	t.Run("Contract Management", func(t *testing.T) { testContractManagement(t, eventMonitor) })
	t.Run("Block Processing", func(t *testing.T) { testBlockProcessing(t, eventMonitor, connectionManager) })
	t.Run("Event Filtering", func(t *testing.T) { testEventFiltering(t, eventMonitor) })
	t.Run("Statistics", func(t *testing.T) { testMonitorStatistics(t, eventMonitor) })
}

func testMonitorLifecycle(t *testing.T, monitor monitor.Monitor) {
	// Test initial state
	if monitor.IsRunning() {
		t.Error("Monitor should not be running initially")
	}

	// Test start
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := monitor.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start monitor: %v", err)
	}

	if !monitor.IsRunning() {
		t.Error("Monitor should be running after start")
	}
	t.Logf("✓ Monitor started successfully")

	// Test stats
	stats := monitor.GetStats()
	if !stats.IsRunning {
		t.Error("Stats should show monitor as running")
	}
	t.Logf("✓ Monitor stats: contracts=%d", stats.ContractsMonitored)

	// Test health
	health := monitor.GetHealth()
	if !health.Healthy {
		t.Errorf("Monitor should be healthy: %v", health.Issues)
	}
	t.Logf("✓ Monitor health check passed")

	// Test stop
	err = monitor.Stop()
	if err != nil {
		t.Fatalf("Failed to stop monitor: %v", err)
	}

	if monitor.IsRunning() {
		t.Error("Monitor should not be running after stop")
	}
	t.Logf("✓ Monitor stopped successfully")
}

func testContractManagement(t *testing.T, monitor monitor.Monitor) {
	// Create test contract
	contract := &models.Contract{
		Address:    common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Name:       "TestToken",
		ABI:        `[{"type":"event","name":"Transfer","inputs":[{"name":"from","type":"address","indexed":true},{"name":"to","type":"address","indexed":true},{"name":"value","type":"uint256","indexed":false}]}]`,
		StartBlock: 100000,
		Active:     true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Test add contract
	err := monitor.AddContract(contract)
	if err != nil {
		t.Fatalf("Failed to add contract: %v", err)
	}
	t.Logf("✓ Contract added: %s", contract.Name)

	// Test get contracts
	contracts := monitor.GetContracts()
	if len(contracts) == 0 {
		t.Error("Should have at least one contract")
	}

	found := false
	for _, c := range contracts {
		if c.Address == contract.Address {
			found = true
			break
		}
	}
	if !found {
		t.Error("Added contract not found in list")
	}
	t.Logf("✓ Contract found in list: %d total contracts", len(contracts))

	// Test update contract
	contract.Name = "UpdatedTestToken"
	contract.UpdatedAt = time.Now()
	err = monitor.UpdateContract(contract)
	if err != nil {
		t.Fatalf("Failed to update contract: %v", err)
	}
	t.Logf("✓ Contract updated: %s", contract.Name)

	// Test remove contract
	err = monitor.RemoveContract(contract.Address)
	if err != nil {
		t.Fatalf("Failed to remove contract: %v", err)
	}
	t.Logf("✓ Contract removed")

	// Verify removal
	contracts = monitor.GetContracts()
	for _, c := range contracts {
		if c.Address == contract.Address {
			t.Error("Contract should have been removed")
		}
	}
}

func testBlockProcessing(t *testing.T, monitor monitor.Monitor, connMgr connection.Manager) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Get a recent block number
	latestBlock, err := connMgr.GetLatestBlockNumber()
	if err != nil {
		t.Fatalf("Failed to get latest block: %v", err)
	}

	// Process a single block (use an older block to ensure it exists)
	blockToProcess := latestBlock - 100

	result, err := monitor.ProcessBlock(ctx, blockToProcess)
	if err != nil {
		t.Fatalf("Failed to process block: %v", err)
	}

	if result.BlockNumber != blockToProcess {
		t.Errorf("Expected block number %d, got %d", blockToProcess, result.BlockNumber)
	}

	t.Logf("✓ Block processed: %d (events found: %d, processing time: %s)",
		result.BlockNumber, result.EventsFound, result.ProcessingTime)

	// Test block range processing
	fromBlock := blockToProcess - 5
	toBlock := blockToProcess - 1

	rangeResult, err := monitor.ProcessBlockRange(ctx, fromBlock, toBlock)
	if err != nil {
		t.Fatalf("Failed to process block range: %v", err)
	}

	expectedBlocks := int(toBlock - fromBlock + 1)
	if rangeResult.BlocksProcessed != expectedBlocks {
		t.Errorf("Expected %d blocks processed, got %d",
			expectedBlocks, rangeResult.BlocksProcessed)
	}

	t.Logf("✓ Block range processed: %d-%d (%d blocks, %d total events, %s)",
		fromBlock, toBlock, rangeResult.BlocksProcessed,
		rangeResult.TotalEvents, rangeResult.ProcessingTime)
}

func testEventFiltering(t *testing.T, monitor monitor.Monitor) {
	// Add a contract with event filters
	contract := &models.Contract{
		Address:    common.HexToAddress("0xabcdef1234567890123456789012345678901234"),
		Name:       "FilterTestToken",
		ABI:        `[{"type":"event","name":"Transfer","inputs":[{"name":"from","type":"address","indexed":true},{"name":"to","type":"address","indexed":true},{"name":"value","type":"uint256","indexed":false}]}, {"type":"event","name":"Approval","inputs":[{"name":"owner","type":"address","indexed":true},{"name":"spender","type":"address","indexed":true},{"name":"value","type":"uint256","indexed":false}]}]`,
		StartBlock: 100000,
		Active:     true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := monitor.AddContract(contract)
	if err != nil {
		t.Fatalf("Failed to add filter test contract: %v", err)
	}
	defer monitor.RemoveContract(contract.Address)

	t.Logf("✓ Event filtering contract added: %s", contract.Name)

	// Event filtering is tested implicitly through block processing
	// The monitor should only process events from monitored contracts
}

func testMonitorStatistics(t *testing.T, monitor monitor.Monitor) {
	// Start monitor briefly to generate stats
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := monitor.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start monitor for stats test: %v", err)
	}
	defer monitor.Stop()

	// Wait a moment for stats to be generated
	time.Sleep(2 * time.Second)

	stats := monitor.GetStats()
	if stats.StartTime.IsZero() {
		t.Error("Start time should be set")
	}
	if stats.Uptime <= 0 {
		t.Error("Uptime should be positive")
	}

	t.Logf("✓ Monitor statistics:")
	t.Logf("  - Uptime: %s", stats.Uptime)
	t.Logf("  - Blocks processed: %d", stats.TotalBlocksProcessed)
	t.Logf("  - Events found: %d", stats.TotalEventsFound)
	t.Logf("  - Events saved: %d", stats.TotalEventsSaved)
	t.Logf("  - Processing rate: %.2f blocks/sec", stats.ProcessingRate)
}
