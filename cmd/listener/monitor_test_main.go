// File: cmd/listener/monitor_test_main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartdevs17/rsk-event-listener/internal/config"
	"github.com/smartdevs17/rsk-event-listener/internal/connection"
	"github.com/smartdevs17/rsk-event-listener/internal/models"
	"github.com/smartdevs17/rsk-event-listener/internal/monitor"
	"github.com/smartdevs17/rsk-event-listener/internal/storage"
	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
)

func main() {
	fmt.Println("RSK Event Listener - Event Monitor Test")

	// Initialize logger
	err := utils.InitLogger("info", "text", "stdout", "")
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Setup test database
	testDB := "./test_monitor.db"
	defer os.Remove(testDB)

	// Load configurations
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Override storage for testing
	cfg.Storage.ConnectionString = testDB

	// Create monitor configuration
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

	fmt.Println("\n=== Testing Event Monitor ===")

	// Test 1: Create Components
	fmt.Println("Creating components...")

	// Connection Manager
	connectionManager := connection.NewConnectionManager(&cfg.RSK)
	defer connectionManager.Close()

	// Storage
	store, err := storage.NewStorage(&cfg.Storage)
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	err = store.Connect()
	if err != nil {
		log.Fatalf("Failed to connect storage: %v", err)
	}

	err = store.Migrate()
	if err != nil {
		log.Fatalf("Failed to migrate storage: %v", err)
	}

	// Event Monitor
	eventMonitor := monitor.NewEventMonitor(connectionManager, store, monitorConfig)
	fmt.Printf("✓ Components created successfully\n")

	// Test 2: Contract Management
	fmt.Println("\nTesting contract management...")

	testContract := &models.Contract{
		Address:    common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Name:       "TestToken",
		ABI:        `[{"type":"event","name":"Transfer","inputs":[{"name":"from","type":"address","indexed":true},{"name":"to","type":"address","indexed":true},{"name":"value","type":"uint256","indexed":false}]}]`,
		StartBlock: 100000,
		Active:     true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err = eventMonitor.AddContract(testContract)
	if err != nil {
		log.Fatalf("Failed to add contract: %v", err)
	}
	fmt.Printf("✓ Contract added: %s\n", testContract.Name)

	contracts := eventMonitor.GetContracts()
	fmt.Printf("✓ Total contracts: %d\n", len(contracts))

	// Test 3: Block Processing
	fmt.Println("\nTesting block processing...")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Get latest block
	latestBlock, err := connectionManager.GetLatestBlockNumber()
	if err != nil {
		log.Fatalf("Failed to get latest block: %v", err)
	}
	fmt.Printf("✓ Latest blockchain block: %d\n", latestBlock)

	// Process a recent block
	blockToProcess := latestBlock - 100
	fmt.Printf("Processing block: %d\n", blockToProcess)

	result, err := eventMonitor.ProcessBlock(ctx, blockToProcess)
	if err != nil {
		log.Fatalf("Failed to process block: %v", err)
	}

	fmt.Printf("✓ Block processed successfully:\n")
	fmt.Printf("  - Block: %d\n", result.BlockNumber)
	fmt.Printf("  - Hash: %s\n", result.BlockHash[:10]+"...")
	fmt.Printf("  - Events found: %d\n", result.EventsFound)
	fmt.Printf("  - Events saved: %d\n", result.EventsSaved)
	fmt.Printf("  - Processing time: %s\n", result.ProcessingTime)

	// Test 4: Block Range Processing
	fmt.Println("\nTesting block range processing...")

	fromBlock := blockToProcess - 5
	toBlock := blockToProcess - 1

	rangeResult, err := eventMonitor.ProcessBlockRange(ctx, fromBlock, toBlock)
	if err != nil {
		log.Fatalf("Failed to process block range: %v", err)
	}

	fmt.Printf("✓ Block range processed:\n")
	fmt.Printf("  - Range: %d to %d\n", fromBlock, toBlock)
	fmt.Printf("  - Blocks processed: %d\n", rangeResult.BlocksProcessed)
	fmt.Printf("  - Total events: %d\n", rangeResult.TotalEvents)
	fmt.Printf("  - Processing time: %s\n", rangeResult.ProcessingTime)

	// Test 5: Monitor Lifecycle
	fmt.Println("\nTesting monitor lifecycle...")

	// Start monitor
	startCtx, startCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer startCancel()

	err = eventMonitor.Start(startCtx)
	if err != nil {
		log.Fatalf("Failed to start monitor: %v", err)
	}
	fmt.Printf("✓ Monitor started\n")

	// Check status
	if !eventMonitor.IsRunning() {
		log.Fatal("Monitor should be running")
	}
	fmt.Printf("✓ Monitor is running\n")

	// Get statistics
	time.Sleep(2 * time.Second) // Let it run briefly
	stats := eventMonitor.GetStats()
	fmt.Printf("✓ Monitor statistics:\n")
	fmt.Printf("  - Uptime: %s\n", stats.Uptime)
	fmt.Printf("  - Contracts monitored: %d\n", stats.ContractsMonitored)
	fmt.Printf("  - Is running: %v\n", stats.IsRunning)

	// Get health status
	health := eventMonitor.GetHealth()
	fmt.Printf("✓ Monitor health:\n")
	fmt.Printf("  - Healthy: %v\n", health.Healthy)
	fmt.Printf("  - Connection healthy: %v\n", health.ConnectionHealthy)
	fmt.Printf("  - Storage healthy: %v\n", health.StorageHealthy)
	if len(health.Issues) > 0 {
		fmt.Printf("  - Issues: %v\n", health.Issues)
	}

	// Stop monitor
	err = eventMonitor.Stop()
	if err != nil {
		log.Fatalf("Failed to stop monitor: %v", err)
	}
	fmt.Printf("✓ Monitor stopped\n")

	// Test 6: Latest Block Tracking
	fmt.Println("\nTesting block tracking...")

	// Set start block
	err = eventMonitor.SetStartBlock(blockToProcess)
	if err != nil {
		log.Fatalf("Failed to set start block: %v", err)
	}

	// Get latest processed block
	latestProcessed, err := eventMonitor.GetLatestProcessedBlock()
	if err != nil {
		log.Fatalf("Failed to get latest processed block: %v", err)
	}
	fmt.Printf("✓ Latest processed block: %d\n", latestProcessed)

	// Clean up contract
	err = eventMonitor.RemoveContract(testContract.Address)
	if err != nil {
		log.Fatalf("Failed to remove contract: %v", err)
	}
	fmt.Printf("✓ Test contract removed\n")

	fmt.Println("\n🎉 All event monitor tests passed! Ready for Step 5.")
}
