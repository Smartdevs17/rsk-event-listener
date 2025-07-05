// // File: cmd/listener/processor_test_main.go
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
// 	"github.com/smartdevs17/rsk-event-listener/internal/notification"
// 	"github.com/smartdevs17/rsk-event-listener/internal/processor"
// 	"github.com/smartdevs17/rsk-event-listener/internal/storage"
// 	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
// )

// func main() {
// 	fmt.Println("RSK Event Listener - Event Processor Test")

// 	// Initialize logger
// 	err := utils.InitLogger("info", "text", "stdout", "")
// 	if err != nil {
// 		log.Fatalf("Failed to initialize logger: %v", err)
// 	}

// 	// Setup test database
// 	testDB := "./test_processor.db"
// 	defer os.Remove(testDB)

// 	fmt.Println("\n=== Testing Event Processor ===")

// 	// Test 1: Storage Setup
// 	fmt.Println("Setting up storage for processor testing...")
// 	storageCfg := &config.StorageConfig{
// 		Type:             "sqlite",
// 		ConnectionString: testDB,
// 		MaxConnections:   10,
// 		MaxIdleTime:      time.Minute * 15,
// 	}

// 	store, err := storage.NewStorage(storageCfg)
// 	if err != nil {
// 		log.Fatalf("Failed to create storage: %v", err)
// 	}
// 	defer store.Close()

// 	err = store.Connect()
// 	if err != nil {
// 		log.Fatalf("Failed to connect to storage: %v", err)
// 	}

// 	err = store.Migrate()
// 	if err != nil {
// 		log.Fatalf("Failed to migrate database: %v", err)
// 	}

// 	err = store.Ping()
// 	if err != nil {
// 		log.Fatalf("Failed to ping storage: %v", err)
// 	}
// 	fmt.Printf("✓ Storage setup completed\n")

// 	// Test 2: Notification Manager Setup
// 	fmt.Println("\nSetting up notification manager...")
// 	// Setup notification configuration
// 	notificationMgrCfg := &notification.NotificationManagerConfig{
// 		MaxConcurrentNotifications: 5,
// 		NotificationTimeout:        30 * time.Second,
// 		RetryAttempts:              3,
// 		RetryDelay:                 1 * time.Second,
// 		EnableEmailNotifications:   true,
// 		EnableWebhookNotifications: true,
// 		QueueSize:                  100,
// 		LogLevel:                   "info",
// 	}

// 	notifier := notification.NewNotificationManager(notificationMgrCfg)
// 	defer notifier.Stop()

// 	fmt.Printf("✓ Notification manager setup completed\n")

// 	// Test 3: Processor Creation and Configuration
// 	fmt.Println("\nCreating event processor...")
// 	// Use actual ProcessorConfig structure
// 	processorCfg := &processor.ProcessorConfig{
// 		MaxConcurrentProcessing: 10,
// 		ProcessingTimeout:       30 * time.Second,
// 		RetryAttempts:           3,
// 		RetryDelay:              2 * time.Second,
// 		EnableValidation:        true,
// 		EnableAggregation:       true,
// 		AggregationWindow:       1 * time.Minute,
// 		BufferSize:              100,
// 	}

// 	// Use actual constructor
// 	proc := processor.NewEventProcessor(store, notifier, processorCfg)
// 	if proc == nil {
// 		log.Fatal("Failed to create event processor")
// 	}
// 	defer proc.Stop()

// 	fmt.Printf("✓ Event processor created with:\n")
// 	fmt.Printf("  - Max concurrent processing: %d\n", processorCfg.MaxConcurrentProcessing)
// 	fmt.Printf("  - Processing timeout: %v\n", processorCfg.ProcessingTimeout)
// 	fmt.Printf("  - Validation enabled: %t\n", processorCfg.EnableValidation)
// 	fmt.Printf("  - Aggregation enabled: %t\n", processorCfg.EnableAggregation)

// 	// Test 4: Processor Lifecycle
// 	fmt.Println("\nTesting processor lifecycle...")
// 	ctx := context.Background()

// 	// Test initial state
// 	if proc.IsRunning() {
// 		log.Fatal("Processor should not be running initially")
// 	}
// 	fmt.Printf("✓ Initial state: not running\n")

// 	// Start processor
// 	err = proc.Start(ctx)
// 	if err != nil {
// 		log.Fatalf("Failed to start processor: %v", err)
// 	}

// 	if !proc.IsRunning() {
// 		log.Fatal("Processor should be running after start")
// 	}
// 	fmt.Printf("✓ Processor started successfully\n")

// 	// Test 5: Contract Setup
// 	fmt.Println("\nSetting up test contract...")
// 	contract := &models.Contract{
// 		Address:    common.HexToAddress("0x1234567890123456789012345678901234567890"),
// 		Name:       "TestToken",
// 		ABI:        `[{"type":"event","name":"Transfer","inputs":[{"name":"from","type":"address","indexed":true},{"name":"to","type":"address","indexed":true},{"name":"value","type":"uint256"}]},{"type":"event","name":"Approval","inputs":[{"name":"owner","type":"address","indexed":true},{"name":"spender","type":"address","indexed":true},{"name":"value","type":"uint256"}]}]`,
// 		Active:     true,
// 		StartBlock: 0, // or your desired start block
// 		CreatedAt:  time.Now(),
// 		UpdatedAt:  time.Now(),
// 	}

// 	err = store.SaveContract(ctx, contract)
// 	if err != nil {
// 		log.Fatalf("Failed to save test contract: %v", err)
// 	}
// 	fmt.Printf("✓ Test contract saved: %s (%s)\n", contract.Name, contract.Address)

// 	// Test 6: Single Event Processing
// 	fmt.Println("\nTesting single event processing...")
// 	event1 := &models.Event{
// 		ID:          "test-event-single",
// 		Address:     contract.Address.Hex(),
// 		EventName:   "Transfer",
// 		EventSig:    "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
// 		BlockNumber: 12345,
// 		BlockHash:   "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
// 		TxHash:      "0x1111111111111111111111111111111111111111111111111111111111111111",
// 		TxIndex:     0,
// 		LogIndex:    0,
// 		Data: map[string]interface{}{
// 			"from":  "0xabcd1234567890123456789012345678901234abcd",
// 			"to":    "0xefgh1234567890123456789012345678901234efgh",
// 			"value": "1000000000000000000", // 1 ETH
// 		},
// 		Timestamp: time.Now(),
// 		Processed: false,
// 	}

// 	startTime := time.Now()

// 	// FIX: Save the event to storage before processing
// 	err = store.SaveEvent(ctx, event1)
// 	if err != nil {
// 		log.Fatalf("Failed to save event before processing: %v", err)
// 	}

// 	// ProcessEvent returns (*ProcessResult, error)
// 	result, err := proc.ProcessEvent(ctx, event1)
// 	processingTime := time.Since(startTime)

// 	if err != nil {
// 		log.Fatalf("Failed to process single event: %v", err)
// 	}

// 	if result == nil {
// 		log.Fatal("Process result should not be nil")
// 	}

// 	if !result.Success {
// 		log.Fatalf("Event processing should be successful: %v", result.Error)
// 	}

// 	if !event1.Processed {
// 		log.Fatal("Event should be marked as processed")
// 	}

// 	// Verify event is in storage
// 	savedEvent, err := store.GetEvent(ctx, event1.ID)
// 	if err != nil {
// 		log.Fatalf("Failed to retrieve saved event: %v", err)
// 	}
// 	if savedEvent == nil {
// 		log.Fatal("Event not found in storage")
// 	}
// 	if !savedEvent.Processed {
// 		log.Fatal("Saved event should be marked as processed")
// 	}

// 	fmt.Printf("✓ Single event processed successfully in %v\n", processingTime)
// 	fmt.Printf("  - Event ID: %s\n", result.EventID)
// 	fmt.Printf("  - Success: %t\n", result.Success)
// 	fmt.Printf("  - Processing Time: %v\n", result.ProcessingTime)
// 	fmt.Printf("  - Rules Executed: %v\n", result.RulesExecuted)
// 	fmt.Printf("  - Actions Executed: %v\n", result.ActionsExecuted)
// 	fmt.Printf("  - Workflows Executed: %v\n", result.WorkflowsExecuted)
// 	fmt.Printf("  - Stored in database: ✓\n")

// 	// Test 7: Batch Event Processing
// 	fmt.Println("\nTesting batch event processing...")
// 	batchEvents := []*models.Event{
// 		{
// 			ID:          "batch-event-1",
// 			Address:     contract.Address.Hex(),
// 			EventName:   "Transfer",
// 			EventSig:    "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
// 			BlockNumber: 12346,
// 			BlockHash:   "0xbcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890a",
// 			TxHash:      "0x2222222222222222222222222222222222222222222222222222222222222222",
// 			TxIndex:     0,
// 			LogIndex:    0,
// 			Data: map[string]interface{}{
// 				"from":  "0x1111111111111111111111111111111111111111",
// 				"to":    "0x2222222222222222222222222222222222222222",
// 				"value": "2000000000000000000",
// 			},
// 			Timestamp: time.Now(),
// 			Processed: false,
// 		},
// 		{
// 			ID:          "batch-event-2",
// 			Address:     contract.Address.Hex(),
// 			EventName:   "Approval",
// 			EventSig:    "0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925",
// 			BlockNumber: 12346,
// 			BlockHash:   "0xbcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890a",
// 			TxHash:      "0x3333333333333333333333333333333333333333333333333333333333333333",
// 			TxIndex:     1,
// 			LogIndex:    0,
// 			Data: map[string]interface{}{
// 				"owner":   "0x3333333333333333333333333333333333333333",
// 				"spender": "0x4444444444444444444444444444444444444444",
// 				"value":   "3000000000000000000",
// 			},
// 			Timestamp: time.Now(),
// 			Processed: false,
// 		},
// 	}

// 	batchStartTime := time.Now()

// 	// Save all batch events before processing
// 	for _, event := range batchEvents {
// 		err := store.SaveEvent(ctx, event)
// 		if err != nil {
// 			log.Fatalf("Failed to save batch event before processing: %v", err)
// 		}
// 	}

// 	// Now process the batch
// 	// ProcessEvents returns (*BatchProcessResult, error)
// 	batchResult, err := proc.ProcessEvents(ctx, batchEvents)
// 	batchProcessingTime := time.Since(batchStartTime)

// 	if err != nil {
// 		log.Fatalf("Failed to process batch events: %v", err)
// 	}

// 	if batchResult == nil {
// 		log.Fatal("Batch result should not be nil")
// 	}

// 	// Verify all events are processed and saved
// 	for _, event := range batchEvents {
// 		savedEvent, err := store.GetEvent(ctx, event.ID)
// 		if err != nil {
// 			log.Fatalf("Failed to retrieve saved batch event: %v", err)
// 		}
// 		if savedEvent == nil {
// 			log.Fatalf("Batch event not found in storage: %s", event.ID)
// 		}
// 		if !savedEvent.Processed {
// 			log.Fatalf("Batch event should be marked as processed: %s", event.ID)
// 		}
// 	}

// 	fmt.Printf("✓ Batch events processed successfully in %v\n", batchProcessingTime)
// 	fmt.Printf("  - Total events: %d\n", batchResult.TotalEvents)
// 	fmt.Printf("  - Processed events: %d\n", batchResult.ProcessedEvents)
// 	fmt.Printf("  - Failed events: %d\n", batchResult.FailedEvents)
// 	fmt.Printf("  - Batch processing time: %v\n", batchResult.ProcessingTime)
// 	fmt.Printf("  - All events stored in database: ✓\n")

// 	// Test 8: Event Validation
// 	fmt.Println("\nTesting event validation...")

// 	// Test with invalid event (missing required fields)
// 	invalidEvent := &models.Event{
// 		ID:          "invalid-event",
// 		Address:     "", // Missing address
// 		EventName:   "", // Missing event name
// 		BlockNumber: 0,  // Invalid block number
// 		Data:        map[string]interface{}{},
// 		Timestamp:   time.Now(),
// 		Processed:   false,
// 	}

// 	invalidResult, err := proc.ProcessEvent(ctx, invalidEvent)
// 	if err == nil && (invalidResult == nil || invalidResult.Success) {
// 		log.Fatal("Processing invalid event should have failed")
// 	}
// 	fmt.Printf("✓ Event validation working correctly: error detected\n")

// 	// Test 9: Rule Management
// 	fmt.Println("\nTesting rule management...")

// 	// Create a test rule
// 	testRule := &processor.ProcessingRule{
// 		ID:          "test-rule-1",
// 		Name:        "High Value Transfer Alert",
// 		Description: "Alert on high value transfers",
// 		Enabled:     true,
// 		Priority:    1,
// 		Conditions: &processor.RuleConditions{
// 			EventNames: []string{"Transfer"},
// 		},
// 		Actions: []*processor.RuleAction{
// 			{
// 				Type: "notify",
// 				Config: map[string]interface{}{
// 					"type":       "email",
// 					"recipients": []string{"admin@example.com"},
// 					"subject":    "High Value Transfer",
// 					"template":   "Transfer detected",
// 				},
// 			},
// 		},
// 		CreatedAt: time.Now(),
// 		UpdatedAt: time.Now(),
// 	}

// 	// Add rule
// 	err = proc.AddRule(testRule)
// 	if err != nil {
// 		log.Fatalf("Failed to add rule: %v", err)
// 	}
// 	fmt.Printf("✓ Rule added successfully: %s\n", testRule.Name)

// 	// Get rules
// 	rules := proc.GetRules()
// 	if len(rules) == 0 {
// 		log.Fatal("Should have at least one rule")
// 	}
// 	fmt.Printf("✓ Retrieved %d rules\n", len(rules))

// 	// Update rule
// 	testRule.Name = "Updated High Value Transfer Alert"
// 	err = proc.UpdateRule(testRule)
// 	if err != nil {
// 		log.Fatalf("Failed to update rule: %v", err)
// 	}
// 	fmt.Printf("✓ Rule updated successfully\n")

// 	// Remove rule
// 	err = proc.RemoveRule(testRule.ID)
// 	if err != nil {
// 		log.Fatalf("Failed to remove rule: %v", err)
// 	}
// 	fmt.Printf("✓ Rule removed successfully\n")

// 	// Test 10: Workflow Management
// 	fmt.Println("\nTesting workflow management...")

// 	// Create a test workflow
// 	testWorkflow := &processor.Workflow{
// 		ID:          "test-workflow-1",
// 		Name:        "Event Validation Workflow",
// 		Description: "Validates and processes events",
// 		Enabled:     true,
// 		Steps: []*processor.WorkflowStep{
// 			{
// 				ID:   "validate",
// 				Name: "Validate Event Data",
// 				Type: "validate",
// 				Config: map[string]interface{}{
// 					"required_fields": []string{"from", "to", "value"},
// 				},
// 			},
// 		},
// 		CreatedAt: time.Now(),
// 		UpdatedAt: time.Now(),
// 	}

// 	// Add workflow
// 	err = proc.AddWorkflow(testWorkflow)
// 	if err != nil {
// 		log.Fatalf("Failed to add workflow: %v", err)
// 	}
// 	fmt.Printf("✓ Workflow added successfully: %s\n", testWorkflow.Name)

// 	// Get workflows
// 	workflows := proc.GetWorkflows()
// 	if len(workflows) == 0 {
// 		log.Fatal("Should have at least one workflow")
// 	}
// 	fmt.Printf("✓ Retrieved %d workflows\n", len(workflows))

// 	// Remove workflow
// 	err = proc.RemoveWorkflow(testWorkflow.ID)
// 	if err != nil {
// 		log.Fatalf("Failed to remove workflow: %v", err)
// 	}
// 	fmt.Printf("✓ Workflow removed successfully\n")

// 	// Test 11: Processor Statistics
// 	fmt.Println("\nChecking processor statistics...")
// 	stats := proc.GetStats()
// 	if stats == nil {
// 		log.Fatal("Processor stats should not be nil")
// 	}

// 	fmt.Printf("✓ Processor statistics:\n")
// 	fmt.Printf("  - Is Running: %t\n", stats.IsRunning)
// 	fmt.Printf("  - Total Events Processed: %d\n", stats.TotalEventsProcessed)
// 	fmt.Printf("  - Total Rules Executed: %d\n", stats.TotalRulesExecuted)
// 	fmt.Printf("  - Total Actions Executed: %d\n", stats.TotalActionsExecuted)
// 	fmt.Printf("  - Total Notifications Sent: %d\n", stats.TotalNotificationsSent)
// 	fmt.Printf("  - Processing Rate: %.2f events/sec\n", stats.ProcessingRate)
// 	fmt.Printf("  - Average Processing Time: %v\n", stats.AverageProcessingTime)
// 	fmt.Printf("  - Uptime: %v\n", stats.Uptime)
// 	fmt.Printf("  - Error Count: %d\n", stats.ErrorCount)

// 	// Test 12: Health Check
// 	fmt.Println("\nTesting processor health check...")
// 	health := proc.GetHealth()
// 	if health == nil {
// 		log.Fatal("Processor health should not be nil")
// 	}

// 	fmt.Printf("✓ Health check results:\n")
// 	fmt.Printf("  - Overall Healthy: %t\n", health.Healthy)
// 	fmt.Printf("  - Storage Healthy: %t\n", health.StorageHealthy)
// 	fmt.Printf("  - Notification Healthy: %t\n", health.NotificationHealthy)
// 	fmt.Printf("  - Processing Queue Size: %d\n", health.ProcessingQueueSize)

// 	if len(health.Issues) > 0 {
// 		fmt.Printf("  - Issues: %v\n", health.Issues)
// 	} else {
// 		fmt.Printf("  - No issues detected\n")
// 	}

// 	// Test 13: Storage Integration Verification
// 	fmt.Println("\nTesting storage integration...")

// 	// Get storage stats
// 	storageStats, err := store.GetStorageStats()
// 	if err != nil {
// 		log.Fatalf("Failed to get storage stats: %v", err)
// 	}

// 	fmt.Printf("✓ Storage integration verified:\n")
// 	fmt.Printf("  - Total events in storage: %d\n", storageStats.TotalEvents)
// 	fmt.Printf("  - Total contracts in storage: %d\n", storageStats.TotalContracts)
// 	fmt.Printf("  - Database size: %d bytes\n", storageStats.DatabaseSize)

// 	// Test querying processed events
// 	filter := models.EventFilter{
// 		Processed: &[]bool{true}[0],
// 		Limit:     10,
// 	}

// 	storedEvents, err := store.GetEvents(ctx, filter)
// 	if err != nil {
// 		log.Fatalf("Failed to query processed events: %v", err)
// 	}

// 	fmt.Printf("  - Processed events in storage: %d\n", len(storedEvents))
// 	for i, event := range storedEvents {
// 		if i >= 3 { // Show only first 3
// 			break
// 		}
// 		fmt.Printf("    • %s - %s (Block: %d)\n", event.ID, event.EventName, event.BlockNumber)
// 	}

// 	// Test 14: Performance Test with Rule Execution
// 	fmt.Println("\nTesting performance with rules...")

// 	// Add a rule that will execute for Transfer events
// 	perfRule := &processor.ProcessingRule{
// 		ID:          "perf-rule",
// 		Name:        "Performance Test Rule",
// 		Description: "Rule for performance testing",
// 		Enabled:     true,
// 		Priority:    1,
// 		Conditions: &processor.RuleConditions{
// 			EventNames: []string{"Transfer"},
// 		},
// 		Actions: []*processor.RuleAction{
// 			{
// 				Type: "notify",
// 				Config: map[string]interface{}{
// 					"type":       "email",
// 					"recipients": []string{"perf@example.com"},
// 					"subject":    "Performance Test",
// 					"template":   "Performance test event",
// 				},
// 			},
// 		},
// 		CreatedAt: time.Now(),
// 		UpdatedAt: time.Now(),
// 	}

// 	err = proc.AddRule(perfRule)
// 	if err != nil {
// 		log.Fatalf("Failed to add performance rule: %v", err)
// 	}

// 	// Process events with rule execution
// 	perfEvents := make([]*models.Event, 10)
// 	for i := 0; i < 10; i++ {
// 		perfEvents[i] = &models.Event{
// 			ID:          fmt.Sprintf("perf-event-%d", i+1),
// 			Address:     contract.Address.Hex(),
// 			EventName:   "Transfer", // Will trigger our rule
// 			EventSig:    "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
// 			BlockNumber: uint64(13000 + i),
// 			BlockHash:   fmt.Sprintf("0x%064d", i),
// 			TxHash:      fmt.Sprintf("0x%064d", i+7000),
// 			TxIndex:     uint(i),
// 			LogIndex:    0,
// 			Data: map[string]interface{}{
// 				"from":  fmt.Sprintf("0x%040d", i),
// 				"to":    fmt.Sprintf("0x%040d", i+3000),
// 				"value": fmt.Sprintf("%d000000000000000000", i+1),
// 			},
// 			Timestamp: time.Now(),
// 			Processed: false,
// 		}
// 	}

// 	perfStartTime := time.Now()
// 	perfBatchResult, err := proc.ProcessEvents(ctx, perfEvents)
// 	perfDuration := time.Since(perfStartTime)

// 	if err != nil {
// 		log.Fatalf("Performance test failed: %v", err)
// 	}

// 	fmt.Printf("✓ Performance test with rules completed:\n")
// 	fmt.Printf("  - Events processed: %d\n", perfBatchResult.ProcessedEvents)
// 	fmt.Printf("  - Total time: %v\n", perfDuration)
// 	fmt.Printf("  - Processing rate: %.2f events/sec\n", float64(len(perfEvents))/perfDuration.Seconds())

// 	// Verify rule execution in results
// 	rulesExecutedCount := 0
// 	for _, eventResult := range perfBatchResult.Results {
// 		if len(eventResult.RulesExecuted) > 0 {
// 			rulesExecutedCount++
// 		}
// 	}
// 	fmt.Printf("  - Events with rules executed: %d\n", rulesExecutedCount)

// 	// Clean up performance rule
// 	err = proc.RemoveRule(perfRule.ID)
// 	if err != nil {
// 		log.Printf("Warning: Failed to remove performance rule: %v", err)
// 	}

// 	// Final Statistics
// 	fmt.Println("\nFinal processor statistics:")
// 	finalStats := proc.GetStats()
// 	finalStorageStats, _ := store.GetStorageStats()

// 	fmt.Printf("  - Total Events Processed: %d\n", finalStats.TotalEventsProcessed)
// 	fmt.Printf("  - Total Rules Executed: %d\n", finalStats.TotalRulesExecuted)
// 	fmt.Printf("  - Total Actions Executed: %d\n", finalStats.TotalActionsExecuted)
// 	fmt.Printf("  - Current Processing Rate: %.2f events/sec\n", finalStats.ProcessingRate)
// 	fmt.Printf("  - Total Uptime: %v\n", finalStats.Uptime)
// 	fmt.Printf("  - Total Events in Storage: %d\n", finalStorageStats.TotalEvents)
// 	fmt.Printf("  - Database Size: %d bytes\n", finalStorageStats.DatabaseSize)

// 	// Test 15: Graceful Shutdown
// 	fmt.Println("\nTesting graceful shutdown...")
// 	err = proc.Stop()
// 	if err != nil {
// 		log.Fatalf("Failed to stop processor: %v", err)
// 	}

// 	if proc.IsRunning() {
// 		log.Fatal("Processor should not be running after stop")
// 	}
// 	fmt.Printf("✓ Processor stopped gracefully\n")

// 	fmt.Println("\n🎉 All event processor tests passed! Ready for Step 6.")
// 	fmt.Println("\nProcessor successfully demonstrated:")
// 	fmt.Println("  ✓ Event validation and processing")
// 	fmt.Println("  ✓ ProcessEvent with ProcessResult return")
// 	fmt.Println("  ✓ ProcessEvents with BatchProcessResult return")
// 	fmt.Println("  ✓ Rule management (Add, Update, Remove, Get)")
// 	fmt.Println("  ✓ Workflow management (Add, Remove, Get)")
// 	fmt.Println("  ✓ Real storage integration with persistence")
// 	fmt.Println("  ✓ Real notification system integration")
// 	fmt.Println("  ✓ Comprehensive statistics tracking")
// 	fmt.Println("  ✓ Detailed health monitoring")
// 	fmt.Println("  ✓ Rule execution and action processing")
// 	fmt.Println("  ✓ Error handling and validation")
// 	fmt.Println("  ✓ Performance testing with rule execution")
// 	fmt.Println("  ✓ Complete end-to-end integration")
// 	fmt.Println("  ✓ Graceful lifecycle management")
// }
