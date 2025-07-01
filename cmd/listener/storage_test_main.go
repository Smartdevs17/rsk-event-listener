// package main

// import (
// 	"context"
// 	"fmt"
// 	"log"
// 	"os"
// 	"time"

// 	"github.com/ethereum/go-ethereum/common"
// 	"github.com/smartdevs17/rsk-event-listener/internal/config"
// 	"github.com/smartdevs17/rsk-event-listener/internal/models"
// 	"github.com/smartdevs17/rsk-event-listener/internal/storage"
// 	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
// )

// func main() {
// 	fmt.Println("RSK Event Listener - Storage Layer Test")

// 	// Initialize logger
// 	err := utils.InitLogger("info", "text", "stdout", "")
// 	if err != nil {
// 		log.Fatalf("Failed to initialize logger: %v", err)
// 	}

// 	// Setup test database
// 	testDB := "./test_storage.db"
// 	defer os.Remove(testDB)

// 	cfg := &config.StorageConfig{
// 		Type:             "sqlite",
// 		ConnectionString: testDB,
// 		MaxConnections:   10,
// 		MaxIdleTime:      time.Minute * 15,
// 	}

// 	fmt.Println("\n=== Testing Storage Layer ===")

// 	// Test 1: Storage Creation and Connection
// 	fmt.Println("Testing storage creation and connection...")
// 	store, err := storage.NewStorage(cfg)
// 	if err != nil {
// 		log.Fatalf("Failed to create storage: %v", err)
// 	}
// 	defer store.Close()

// 	err = store.Connect()
// 	if err != nil {
// 		log.Fatalf("Failed to connect to storage: %v", err)
// 	}
// 	fmt.Printf("✓ Connected to %s storage\n", cfg.Type)

// 	// Test 2: Database Migration
// 	fmt.Println("\nTesting database migration...")
// 	err = store.Migrate()
// 	if err != nil {
// 		log.Fatalf("Failed to migrate database: %v", err)
// 	}
// 	fmt.Printf("✓ Database migration completed\n")

// 	// Test 3: Basic Operations
// 	fmt.Println("\nTesting basic operations...")
// 	err = store.Ping()
// 	if err != nil {
// 		log.Fatalf("Failed to ping storage: %v", err)
// 	}
// 	fmt.Printf("✓ Storage ping successful\n")

// 	// Test 4: Event Operations
// 	fmt.Println("\nTesting event operations...")
// 	ctx := context.Background()

// 	// Create and save test event
// 	event := &models.Event{
// 		ID:          "test-event-storage-1",
// 		BlockNumber: 12345,
// 		BlockHash:   "0x1234567890abcdef",
// 		TxHash:      "0xabcdef1234567890",
// 		TxIndex:     0,
// 		LogIndex:    0,
// 		Address:     "0x1234567890123456789012345678901234567890",
// 		EventName:   "Transfer",
// 		EventSig:    "Transfer(address,address,uint256)",
// 		Data:        map[string]interface{}{"from": "0x123", "to": "0x456", "value": "1000"},
// 		Timestamp:   time.Now(),
// 		Processed:   false,
// 	}

// 	err = store.SaveEvent(ctx, event)
// 	if err != nil {
// 		log.Fatalf("Failed to save event: %v", err)
// 	}
// 	fmt.Printf("✓ Event saved: %s\n", event.ID)

// 	// Retrieve event
// 	retrievedEvent, err := store.GetEvent(ctx, event.ID)
// 	if err != nil {
// 		log.Fatalf("Failed to get event: %v", err)
// 	}
// 	if retrievedEvent == nil {
// 		log.Fatal("Event not found")
// 	}
// 	fmt.Printf("✓ Event retrieved: %s\n", retrievedEvent.EventName)

// 	// Test 5: Contract Operations
// 	fmt.Println("\nTesting contract operations...")
// 	contract := &models.Contract{
// 		Address:    common.HexToAddress("0x1234567890123456789012345678901234567890"),
// 		Name:       "TestToken",
// 		ABI:        `[{"type":"event","name":"Transfer"}]`,
// 		StartBlock: 12345,
// 		Active:     true,
// 		CreatedAt:  time.Now(),
// 		UpdatedAt:  time.Now(),
// 	}

// 	err = store.SaveContract(ctx, contract)
// 	if err != nil {
// 		log.Fatalf("Failed to save contract: %v", err)
// 	}
// 	fmt.Printf("✓ Contract saved: %s\n", contract.Name)

// 	retrievedContract, err := store.GetContract(ctx, contract.Address.Hex())
// 	if err != nil {
// 		log.Fatalf("Failed to get contract: %v", err)
// 	}
// 	if retrievedContract == nil {
// 		log.Fatal("Contract not found")
// 	}
// 	fmt.Printf("✓ Contract retrieved: %s\n", retrievedContract.Name)

// 	// Test 6: Block Tracking
// 	fmt.Println("\nTesting block tracking...")
// 	blockNumber := uint64(12345)
// 	err = store.SetLatestProcessedBlock(blockNumber)
// 	if err != nil {
// 		log.Fatalf("Failed to set latest processed block: %v", err)
// 	}

// 	retrievedBlock, err := store.GetLatestProcessedBlock()
// 	if err != nil {
// 		log.Fatalf("Failed to get latest processed block: %v", err)
// 	}
// 	fmt.Printf("✓ Block tracking working: block %d\n", retrievedBlock)

// 	// Test 7: Statistics
// 	fmt.Println("\nTesting statistics...")
// 	stats, err := store.GetStorageStats()
// 	if err != nil {
// 		log.Fatalf("Failed to get storage stats: %v", err)
// 	}
// 	fmt.Printf("✓ Storage stats:\n")
// 	fmt.Printf("  - Total Events: %d\n", stats.TotalEvents)
// 	fmt.Printf("  - Total Contracts: %d\n", stats.TotalContracts)
// 	fmt.Printf("  - Latest Block: %d\n", stats.LatestBlock)

// 	// Test 8: Batch Operations
// 	fmt.Println("\nTesting batch operations...")
// 	batchEvents := []*models.Event{
// 		{
// 			ID:          "batch-1",
// 			BlockNumber: 12346,
// 			BlockHash:   "0x2345678901bcdef0",
// 			TxHash:      "0xbcdef01234567890",
// 			TxIndex:     0,
// 			LogIndex:    0,
// 			Address:     "0x1234567890123456789012345678901234567890",
// 			EventName:   "Approval",
// 			EventSig:    "Approval(address,address,uint256)",
// 			Data:        map[string]interface{}{"owner": "0x123", "spender": "0x456", "value": "2000"},
// 			Timestamp:   time.Now(),
// 			Processed:   false,
// 		},
// 		{
// 			ID:          "batch-2",
// 			BlockNumber: 12346,
// 			BlockHash:   "0x2345678901bcdef0",
// 			TxHash:      "0xbcdef01234567891",
// 			TxIndex:     1,
// 			LogIndex:    0,
// 			Address:     "0x1234567890123456789012345678901234567890",
// 			EventName:   "Transfer",
// 			EventSig:    "Transfer(address,address,uint256)",
// 			Data:        map[string]interface{}{"from": "0x456", "to": "0x789", "value": "500"},
// 			Timestamp:   time.Now(),
// 			Processed:   false,
// 		},
// 	}

// 	err = store.SaveEvents(ctx, batchEvents)
// 	if err != nil {
// 		log.Fatalf("Failed to save batch events: %v", err)
// 	}
// 	fmt.Printf("✓ Batch events saved: %d events\n", len(batchEvents))

// 	// Final statistics
// 	finalStats, err := store.GetStorageStats()
// 	if err != nil {
// 		log.Fatalf("Failed to get final storage stats: %v", err)
// 	}
// 	fmt.Printf("\nFinal storage statistics:\n")
// 	fmt.Printf("  - Total Events: %d\n", finalStats.TotalEvents)
// 	fmt.Printf("  - Total Contracts: %d\n", finalStats.TotalContracts)
// 	fmt.Printf("  - Database Size: %d bytes\n", finalStats.DatabaseSize)

// 	fmt.Println("\n🎉 All storage layer tests passed! Ready for Step 4.")
// }
