package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartdevs17/rsk-event-listener/internal/config"
	"github.com/smartdevs17/rsk-event-listener/internal/models"
	"github.com/smartdevs17/rsk-event-listener/internal/storage"
	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
)

func TestSQLiteStorage(t *testing.T) {
	// Setup test database
	testDB := "./test_events.db"
	defer os.Remove(testDB)

	cfg := &config.StorageConfig{
		Type:             "sqlite",
		ConnectionString: testDB,
		MaxConnections:   10,
		MaxIdleTime:      time.Minute * 15,
	}

	// Initialize logger
	utils.InitLogger("info", "text", "stdout", "")

	// Create storage
	store, err := storage.NewStorage(cfg)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	// Test connection
	err = store.Connect()
	if err != nil {
		t.Fatalf("Failed to connect to storage: %v", err)
	}

	// Test migration
	err = store.Migrate()
	if err != nil {
		t.Fatalf("Failed to migrate storage: %v", err)
	}

	// Test ping
	err = store.Ping()
	if err != nil {
		t.Fatalf("Failed to ping storage: %v", err)
	}

	t.Logf("✓ Storage connection and migration successful")

	// Run tests
	t.Run("Event Operations", func(t *testing.T) { testEventOperations(t, store) })
	t.Run("Contract Operations", func(t *testing.T) { testContractOperations(t, store) })
	t.Run("Block Tracking", func(t *testing.T) { testBlockTracking(t, store) })
	t.Run("Notification Operations", func(t *testing.T) { testNotificationOperations(t, store) })
	t.Run("Statistics", func(t *testing.T) { testStatistics(t, store) })
}

func testEventOperations(t *testing.T, store storage.Storage) {
	ctx := context.Background()

	// Create test event
	event := &models.Event{
		ID:          "test-event-1",
		BlockNumber: 12345,
		BlockHash:   "0x1234567890abcdef",
		TxHash:      "0xabcdef1234567890",
		TxIndex:     0,
		LogIndex:    0,
		Address:     "0x1234567890123456789012345678901234567890",
		EventName:   "Transfer",
		EventSig:    "Transfer(address,address,uint256)",
		Data:        map[string]interface{}{"from": "0x123", "to": "0x456", "value": "1000"},
		Timestamp:   time.Now(),
		Processed:   false,
	}

	// Test save event
	err := store.SaveEvent(ctx, event)
	if err != nil {
		t.Fatalf("Failed to save event: %v", err)
	}
	t.Logf("✓ Event saved successfully")

	// Test get event
	retrievedEvent, err := store.GetEvent(ctx, event.ID)
	if err != nil {
		t.Fatalf("Failed to get event: %v", err)
	}
	if retrievedEvent == nil {
		t.Fatal("Event not found")
	}
	if retrievedEvent.ID != event.ID {
		t.Errorf("Expected event ID %s, got %s", event.ID, retrievedEvent.ID)
	}
	t.Logf("✓ Event retrieved successfully")

	// Test get events with filter
	filter := models.EventFilter{
		EventName: &event.EventName,
		Limit:     10,
	}
	events, err := store.GetEvents(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to get events: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("No events found")
	}
	t.Logf("✓ Events filtered successfully: found %d events", len(events))

	// Test event count
	count, err := store.GetEventCount(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to get event count: %v", err)
	}
	if count != int64(len(events)) {
		t.Errorf("Event count mismatch: expected %d, got %d", len(events), count)
	}
	t.Logf("✓ Event count verified: %d", count)

	// Test update event
	event.Processed = true
	now := time.Now()
	event.ProcessedAt = &now
	err = store.UpdateEvent(ctx, event)
	if err != nil {
		t.Fatalf("Failed to update event: %v", err)
	}
	t.Logf("✓ Event updated successfully")

	// Test batch save
	batchEvents := []*models.Event{
		{
			ID:          "batch-event-1",
			BlockNumber: 12346,
			BlockHash:   "0x2345678901bcdef0",
			TxHash:      "0xbcdef01234567890",
			TxIndex:     0,
			LogIndex:    0,
			Address:     "0x1234567890123456789012345678901234567890",
			EventName:   "Approval",
			EventSig:    "Approval(address,address,uint256)",
			Data:        map[string]interface{}{"owner": "0x123", "spender": "0x456", "value": "2000"},
			Timestamp:   time.Now(),
			Processed:   false,
		},
		{
			ID:          "batch-event-2",
			BlockNumber: 12346,
			BlockHash:   "0x2345678901bcdef0",
			TxHash:      "0xbcdef01234567891",
			TxIndex:     1,
			LogIndex:    0,
			Address:     "0x1234567890123456789012345678901234567890",
			EventName:   "Transfer",
			EventSig:    "Transfer(address,address,uint256)",
			Data:        map[string]interface{}{"from": "0x456", "to": "0x789", "value": "500"},
			Timestamp:   time.Now(),
			Processed:   false,
		},
	}

	err = store.SaveEvents(ctx, batchEvents)
	if err != nil {
		t.Fatalf("Failed to save batch events: %v", err)
	}
	t.Logf("✓ Batch events saved successfully: %d events", len(batchEvents))
}

func testContractOperations(t *testing.T, store storage.Storage) {
	ctx := context.Background()

	// Create test contract
	contract := &models.Contract{
		Address:    common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Name:       "TestToken",
		ABI:        `[{"type":"event","name":"Transfer"}]`,
		StartBlock: 12345,
		Active:     true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Test save contract
	err := store.SaveContract(ctx, contract)
	if err != nil {
		t.Fatalf("Failed to save contract: %v", err)
	}
	t.Logf("✓ Contract saved successfully")

	// Test get contract
	retrievedContract, err := store.GetContract(ctx, contract.Address.Hex())
	if err != nil {
		t.Fatalf("Failed to get contract: %v", err)
	}
	if retrievedContract == nil {
		t.Fatal("Contract not found")
	}
	if retrievedContract.Name != contract.Name {
		t.Errorf("Expected contract name %s, got %s", contract.Name, retrievedContract.Name)
	}
	t.Logf("✓ Contract retrieved successfully")

	// Test get contracts
	active := true
	contracts, err := store.GetContracts(ctx, &active)
	if err != nil {
		t.Fatalf("Failed to get contracts: %v", err)
	}
	if len(contracts) == 0 {
		t.Fatal("No contracts found")
	}
	t.Logf("✓ Contracts retrieved successfully: found %d contracts", len(contracts))

	// Test update contract
	contract.Name = "UpdatedTestToken"
	contract.UpdatedAt = time.Now()
	err = store.UpdateContract(ctx, contract)
	if err != nil {
		t.Fatalf("Failed to update contract: %v", err)
	}
	t.Logf("✓ Contract updated successfully")
}

func testBlockTracking(t *testing.T, store storage.Storage) {
	// Test latest processed block
	blockNumber := uint64(12345)
	err := store.SetLatestProcessedBlock(blockNumber)
	if err != nil {
		t.Fatalf("Failed to set latest processed block: %v", err)
	}

	retrievedBlock, err := store.GetLatestProcessedBlock()
	if err != nil {
		t.Fatalf("Failed to get latest processed block: %v", err)
	}
	if retrievedBlock != blockNumber {
		t.Errorf("Expected block number %d, got %d", blockNumber, retrievedBlock)
	}
	t.Logf("✓ Block tracking working: block %d", retrievedBlock)

	// Test block processing status
	status := &storage.BlockProcessingStatus{
		BlockNumber: 12346,
		Status:      "processing",
		StartedAt:   time.Now(),
		EventsFound: 5,
		EventsSaved: 5,
	}

	err = store.SetBlockProcessingStatus(status)
	if err != nil {
		t.Fatalf("Failed to set block processing status: %v", err)
	}

	retrievedStatus, err := store.GetBlockProcessingStatus(status.BlockNumber)
	if err != nil {
		t.Fatalf("Failed to get block processing status: %v", err)
	}
	if retrievedStatus == nil {
		t.Fatal("Block processing status not found")
	}
	if retrievedStatus.Status != status.Status {
		t.Errorf("Expected status %s, got %s", status.Status, retrievedStatus.Status)
	}
	t.Logf("✓ Block processing status working: block %d", retrievedStatus.BlockNumber)
}

func testNotificationOperations(t *testing.T, store storage.Storage) {
	ctx := context.Background()

	// Create test notification
	notification := &models.Notification{
		ID:        "test-notification-1",
		Type:      models.NotificationTypeWebhook,
		EventID:   "test-event-1",
		Title:     "Test Notification",
		Message:   "This is a test notification",
		Data:      map[string]interface{}{"test": true},
		Target:    "http://localhost:8080/webhook",
		Status:    "pending",
		Attempts:  0,
		CreatedAt: time.Now(),
	}

	// Test save notification
	err := store.SaveNotification(ctx, notification)
	if err != nil {
		t.Fatalf("Failed to save notification: %v", err)
	}
	t.Logf("✓ Notification saved successfully")

	// Test get pending notifications
	pending, err := store.GetPendingNotifications(ctx, 10)
	if err != nil {
		t.Fatalf("Failed to get pending notifications: %v", err)
	}
	if len(pending) == 0 {
		t.Fatal("No pending notifications found")
	}
	t.Logf("✓ Pending notifications retrieved: found %d", len(pending))

	// Test update notification status
	err = store.UpdateNotificationStatus(ctx, notification.ID, "sent", nil)
	if err != nil {
		t.Fatalf("Failed to update notification status: %v", err)
	}
	t.Logf("✓ Notification status updated successfully")
}

func testStatistics(t *testing.T, store storage.Storage) {
	ctx := context.Background()

	// Test storage stats
	stats, err := store.GetStorageStats()
	if err != nil {
		t.Fatalf("Failed to get storage stats: %v", err)
	}
	if stats.TotalEvents == 0 {
		t.Error("Expected some events in stats")
	}
	t.Logf("✓ Storage stats retrieved: %d events, %d contracts",
		stats.TotalEvents, stats.TotalContracts)

	// Test event stats
	fromTime := time.Now().Add(-24 * time.Hour)
	toTime := time.Now()
	eventStats, err := store.GetEventStats(ctx, fromTime, toTime)
	if err != nil {
		t.Fatalf("Failed to get event stats: %v", err)
	}
	t.Logf("✓ Event stats retrieved: %d events in last 24h", eventStats.TotalEvents)
}
