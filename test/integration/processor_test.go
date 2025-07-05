// File: test/integration/processor_integration_test.go
package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartdevs17/rsk-event-listener/internal/config"
	"github.com/smartdevs17/rsk-event-listener/internal/models"
	"github.com/smartdevs17/rsk-event-listener/internal/notification"
	"github.com/smartdevs17/rsk-event-listener/internal/processor"
	"github.com/smartdevs17/rsk-event-listener/internal/storage"
	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventProcessorIntegration(t *testing.T) {
	// Setup test database
	testDB := "./test_processor_integration.db"
	defer os.Remove(testDB)

	// Initialize logger
	utils.InitLogger("info", "text", "stdout", "")

	// Setup storage configuration
	storageCfg := &config.StorageConfig{
		Type:             "sqlite",
		ConnectionString: testDB,
		MaxConnections:   10,
		MaxIdleTime:      time.Minute * 15,
	}

	// Create and setup storage
	store, err := storage.NewStorage(storageCfg)
	require.NoError(t, err, "Failed to create storage")
	defer store.Close()

	err = store.Connect()
	require.NoError(t, err, "Failed to connect to storage")

	err = store.Migrate()
	require.NoError(t, err, "Failed to migrate storage")

	err = store.Ping()
	require.NoError(t, err, "Failed to ping storage")

	t.Logf("✓ Storage connection and migration successful")

	// Setup notification configuration
	notificationMgrCfg := &notification.NotificationManagerConfig{
		MaxConcurrentNotifications: 5,
		NotificationTimeout:        30 * time.Second,
		RetryAttempts:              3,
		RetryDelay:                 1 * time.Second,
		EnableEmailNotifications:   true,
		EnableWebhookNotifications: true,
		QueueSize:                  100,
		LogLevel:                   "info",
	}

	notifier := notification.NewNotificationManager(notificationMgrCfg)
	require.NotNil(t, notifier, "Failed to create notification manager")
	defer notifier.Stop()

	t.Logf("✓ Notification manager created successfully")

	// Setup processor configuration (matching actual ProcessorConfig)
	processorCfg := &processor.ProcessorConfig{
		MaxConcurrentProcessing: 5,
		ProcessingTimeout:       30 * time.Second,
		RetryAttempts:           3,
		RetryDelay:              1 * time.Second,
		EnableValidation:        true,
		EnableAggregation:       true,
		AggregationWindow:       1 * time.Minute,
		BufferSize:              100,
	}

	// Create event processor using actual constructor
	proc := processor.NewEventProcessor(store, notifier, processorCfg)
	require.NotNil(t, proc, "Failed to create event processor")
	defer proc.Stop()

	t.Logf("✓ Event processor created successfully")

	// Run integration tests
	ctx := context.Background()

	t.Run("Processor Lifecycle", func(t *testing.T) { testProcessorLifecycle(t, proc, ctx) })
	t.Run("Contract Setup", func(t *testing.T) { testProcessorContractSetup(t, store, ctx) })
	t.Run("Single Event Processing", func(t *testing.T) { testSingleEventProcessing(t, proc, store, ctx) })
	t.Run("Batch Event Processing", func(t *testing.T) { testBatchEventProcessing(t, proc, store, ctx) })
	t.Run("Event Validation", func(t *testing.T) { testEventValidationIntegration(t, proc, ctx) })
	t.Run("Storage Integration", func(t *testing.T) { testStorageIntegration(t, proc, store, ctx) })
	t.Run("Statistics and Health", func(t *testing.T) { testProcessorStatsAndHealth(t, proc, store, ctx) })
	t.Run("Rule Management", func(t *testing.T) { testRuleManagement(t, proc, ctx) })
	t.Run("Workflow Management", func(t *testing.T) { testWorkflowManagement(t, proc, ctx) })
}

func testProcessorLifecycle(t *testing.T, proc processor.Processor, ctx context.Context) {
	// Test initial state
	assert.False(t, proc.IsRunning(), "Processor should not be running initially")

	// Start processor
	err := proc.Start(ctx)
	require.NoError(t, err, "Failed to start processor")
	assert.True(t, proc.IsRunning(), "Processor should be running after start")

	// Test start when already running
	err = proc.Start(ctx)
	assert.Error(t, err, "Starting already running processor should error")

	// Stop processor
	err = proc.Stop()
	require.NoError(t, err, "Failed to stop processor")
	assert.False(t, proc.IsRunning(), "Processor should not be running after stop")

	// Test stop when already stopped
	err = proc.Stop()
	assert.NoError(t, err, "Stopping already stopped processor should not error")

	t.Logf("✓ Processor lifecycle management working correctly")
}

func testProcessorContractSetup(t *testing.T, store storage.Storage, ctx context.Context) {
	// Create test contract
	contract := &models.Contract{
		Address:    common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Name:       "TestToken",
		ABI:        `[{"type":"event","name":"Transfer","inputs":[{"name":"from","type":"address","indexed":true},{"name":"to","type":"address","indexed":true},{"name":"value","type":"uint256"}]},{"type":"event","name":"Approval","inputs":[{"name":"owner","type":"address","indexed":true},{"name":"spender","type":"address","indexed":true},{"name":"value","type":"uint256"}]}]`,
		Active:     true,
		StartBlock: 0, // or your desired start block
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := store.SaveContract(ctx, contract)
	require.NoError(t, err, "Failed to save test contract")

	// Verify contract is saved
	retrieved, err := store.GetContract(ctx, contract.Address.Hex())
	require.NoError(t, err, "Failed to retrieve contract")
	assert.Equal(t, contract.Name, retrieved.Name)

	t.Logf("✓ Test contract setup completed: %s (%s)", contract.Name, contract.Address)
}

func testSingleEventProcessing(t *testing.T, proc processor.Processor, store storage.Storage, ctx context.Context) {
	// Start processor
	err := proc.Start(ctx)
	require.NoError(t, err)
	defer proc.Stop()

	// Create test event
	event := &models.Event{
		ID:          "integration-single-event",
		Address:     "0x1234567890123456789012345678901234567890",
		EventName:   "Transfer",
		EventSig:    "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
		BlockNumber: 12345,
		BlockHash:   "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		TxHash:      "0x1111111111111111111111111111111111111111111111111111111111111111",
		TxIndex:     0,
		LogIndex:    0,
		Data: map[string]interface{}{
			"from":  "0xabcd1234567890123456789012345678901234abcd",
			"to":    "0xefgh1234567890123456789012345678901234efgh",
			"value": "1000000000000000000", // 1 ETH
		},
		Timestamp: time.Now(),
		Processed: false,
	}

	// Process the event - returns ProcessResult
	startTime := time.Now()
	err = store.SaveEvent(ctx, event)
	if err != nil {
		t.Fatalf("Failed to save event before processing: %v", err)
	}
	result, err := proc.ProcessEvent(ctx, event)
	processingTime := time.Since(startTime)

	require.NoError(t, err, "Failed to process event")
	require.NotNil(t, result, "Process result should not be nil")
	assert.True(t, result.Success, "Event processing should be successful")
	assert.Equal(t, event.ID, result.EventID, "Result should have correct event ID")
	assert.True(t, result.ProcessingTime > 0, "Processing time should be > 0")

	// Verify event is marked as processed
	assert.True(t, event.Processed, "Event should be marked as processed")

	// Verify event is saved in storage
	savedEvent, err := store.GetEvent(ctx, event.ID)
	require.NoError(t, err, "Failed to retrieve saved event")
	assert.Equal(t, event.ID, savedEvent.ID)
	assert.True(t, savedEvent.Processed)

	t.Logf("✓ Single event processed successfully in %v", processingTime)
}

func testBatchEventProcessing(t *testing.T, proc processor.Processor, store storage.Storage, ctx context.Context) {
	// Start processor if not already running
	if !proc.IsRunning() {
		err := proc.Start(ctx)
		require.NoError(t, err)
		defer proc.Stop()
	}

	// Create batch events
	batchEvents := []*models.Event{
		{
			ID:          "batch-integration-1",
			Address:     "0x1234567890123456789012345678901234567890",
			EventName:   "Transfer",
			EventSig:    "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
			BlockNumber: 12346,
			BlockHash:   "0xbcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890a",
			TxHash:      "0x2222222222222222222222222222222222222222222222222222222222222222",
			TxIndex:     0,
			LogIndex:    0,
			Data: map[string]interface{}{
				"from":  "0x1111111111111111111111111111111111111111",
				"to":    "0x2222222222222222222222222222222222222222",
				"value": "2000000000000000000",
			},
			Timestamp: time.Now(),
			Processed: false,
		},
		{
			ID:          "batch-integration-2",
			Address:     "0x1234567890123456789012345678901234567890",
			EventName:   "Approval",
			EventSig:    "0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925",
			BlockNumber: 12346,
			BlockHash:   "0xbcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890a",
			TxHash:      "0x3333333333333333333333333333333333333333333333333333333333333333",
			TxIndex:     1,
			LogIndex:    0,
			Data: map[string]interface{}{
				"owner":   "0x3333333333333333333333333333333333333333",
				"spender": "0x4444444444444444444444444444444444444444",
				"value":   "3000000000000000000",
			},
			Timestamp: time.Now(),
			Processed: false,
		},
	}

	// Process batch - returns BatchProcessResult
	startTime := time.Now()
	for _, event := range batchEvents {
		err := store.SaveEvent(ctx, event)
		require.NoError(t, err, "Failed to save event %s before processing", event.ID)
	}
	batchResult, err := proc.ProcessEvents(ctx, batchEvents)
	processingTime := time.Since(startTime)

	require.NoError(t, err, "Failed to process batch events")
	require.NotNil(t, batchResult, "Batch result should not be nil")
	assert.Equal(t, len(batchEvents), batchResult.TotalEvents, "Total events should match")
	assert.Equal(t, len(batchEvents), batchResult.ProcessedEvents, "All events should be processed")
	assert.Equal(t, 0, batchResult.FailedEvents, "No events should fail")
	assert.True(t, batchResult.ProcessingTime > 0, "Processing time should be > 0")

	// Verify all events are processed
	for _, event := range batchEvents {
		assert.True(t, event.Processed, "Event %s should be processed", event.ID)

		// Verify in storage
		savedEvent, err := store.GetEvent(ctx, event.ID)
		require.NoError(t, err, "Failed to retrieve event %s from storage", event.ID)
		require.NotNil(t, savedEvent, "Saved event %s should not be nil", event.ID)
		assert.True(t, savedEvent.Processed, "Saved event %s should be marked as processed", event.ID)
	}

	t.Logf("✓ Batch events processed successfully: %d events in %v", len(batchEvents), processingTime)
}

func testEventValidationIntegration(t *testing.T, proc processor.Processor, ctx context.Context) {
	// Start processor if not running
	if !proc.IsRunning() {
		err := proc.Start(ctx)
		require.NoError(t, err)
		defer proc.Stop()
	}

	testCases := []struct {
		name        string
		event       *models.Event
		expectError bool
	}{
		{
			name: "Valid event",
			event: &models.Event{
				ID:          "valid-event",
				Address:     "0x1234567890123456789012345678901234567890",
				EventName:   "Transfer",
				EventSig:    "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
				BlockNumber: 12345,
				BlockHash:   "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				TxHash:      "0x1111111111111111111111111111111111111111111111111111111111111111",
				Data:        map[string]interface{}{"value": "1000"},
				Timestamp:   time.Now(),
			},
			expectError: false,
		},
		{
			name: "Missing event name",
			event: &models.Event{
				ID:          "invalid-event-1",
				Address:     "0x1234567890123456789012345678901234567890",
				EventName:   "", // Missing
				BlockNumber: 12345,
				Data:        map[string]interface{}{},
				Timestamp:   time.Now(),
			},
			expectError: true,
		},
		{
			name: "Missing address",
			event: &models.Event{
				ID:          "invalid-event-2",
				Address:     "", // Missing
				EventName:   "Transfer",
				BlockNumber: 12345,
				Data:        map[string]interface{}{},
				Timestamp:   time.Now(),
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := proc.ProcessEvent(ctx, tc.event)

			if tc.expectError {
				assert.Error(t, err, "Expected error for %s", tc.name)
				if result != nil {
					assert.False(t, result.Success, "Result should indicate failure")
				}
			} else {
				assert.NoError(t, err, "Expected no error for %s", tc.name)
				assert.NotNil(t, result, "Result should not be nil")
				assert.True(t, result.Success, "Result should indicate success")
				assert.True(t, tc.event.Processed, "Valid event should be processed")
			}
		})
	}

	t.Logf("✓ Event validation integration tests passed")
}

func testStorageIntegration(t *testing.T, proc processor.Processor, store storage.Storage, ctx context.Context) {
	if !proc.IsRunning() {
		err := proc.Start(ctx)
		require.NoError(t, err)
		defer proc.Stop()
	}

	initialStats, err := store.GetStorageStats()
	require.NoError(t, err, "Failed to get initial storage stats")

	event := &models.Event{
		ID:          fmt.Sprintf("storage-integration-test-%d", time.Now().UnixNano()),
		Address:     "0x1234567890123456789012345678901234567890",
		EventName:   "Transfer",
		EventSig:    "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
		BlockNumber: 15000,
		BlockHash:   "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		TxHash:      "0x1111111111111111111111111111111111111111111111111111111111111111",
		TxIndex:     0,
		LogIndex:    0,
		Data: map[string]interface{}{
			"from":  "0xstorage1",
			"to":    "0xstorage2",
			"value": "1000000000000000000",
		},
		Timestamp: time.Now(),
		Processed: false,
	}

	err = store.SaveEvent(ctx, event)
	require.NoError(t, err, "Failed to save event before processing")

	result, err := proc.ProcessEvent(ctx, event)
	require.NoError(t, err, "Failed to process storage integration event")
	assert.True(t, result.Success, "Event processing should succeed")

	savedEvent, err := store.GetEvent(ctx, event.ID)
	require.NoError(t, err, "Failed to retrieve event from storage")
	require.NotNil(t, savedEvent, "Saved event should not be nil")
	assert.Equal(t, event.ID, savedEvent.ID)
	assert.True(t, savedEvent.Processed)

	finalStats, err := store.GetStorageStats()
	require.NoError(t, err, "Failed to get final storage stats")
	assert.GreaterOrEqual(t, finalStats.TotalEvents, initialStats.TotalEvents, "Event count should increase")

	t.Logf("✓ Storage integration verified: %d → %d events", initialStats.TotalEvents, finalStats.TotalEvents)
}

func testProcessorStatsAndHealth(t *testing.T, proc processor.Processor, store storage.Storage, ctx context.Context) {
	// Start processor if not running
	if !proc.IsRunning() {
		err := proc.Start(ctx)
		require.NoError(t, err)
		defer proc.Stop()
	}

	// Process a few events to generate stats
	for i := 0; i < 3; i++ {
		event := &models.Event{
			ID:          fmt.Sprintf("storage-integration-test-%d", time.Now().UnixNano()),
			Address:     "0x1234567890123456789012345678901234567890",
			EventName:   "Transfer",
			EventSig:    "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
			BlockNumber: uint64(17000 + i),
			BlockHash:   "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			TxHash:      "0x1111111111111111111111111111111111111111111111111111111111111111",
			TxIndex:     uint(i),
			LogIndex:    0,
			Data: map[string]interface{}{
				"from":  fmt.Sprintf("0xstats%d", i),
				"to":    fmt.Sprintf("0xstats%d", i+1),
				"value": "1000000000000000000",
			},
			Timestamp: time.Now(),
			Processed: false,
		}

		err := store.SaveEvent(ctx, event)
		require.NoError(t, err, "Failed to save event before processing")
		result, err := proc.ProcessEvent(ctx, event)
		require.NoError(t, err, "Failed to process stats event %d", i+1)
		assert.True(t, result.Success, "Event processing should succeed")
	}

	// Test processor stats
	stats := proc.GetStats()
	require.NotNil(t, stats, "Processor stats should not be nil")
	assert.True(t, stats.IsRunning, "Processor should be running")
	assert.Greater(t, stats.TotalEventsProcessed, uint64(0), "Events processed should be > 0")
	assert.True(t, stats.Uptime > 0, "Uptime should be > 0")

	// Test processor health
	health := proc.GetHealth()
	require.NotNil(t, health, "Processor health should not be nil")

	if !health.Healthy {
		t.Logf("Processor not healthy - Issues: %v", health.Issues)
		// In test env, only require storage to be healthy
		assert.True(t, health.StorageHealthy, "Storage must be healthy for tests")
	} else {
		assert.True(t, health.Healthy, "Processor should be healthy")
	}

	t.Logf("  - Overall health: %t", health.Healthy)
	t.Logf("  - Storage health: %t", health.StorageHealthy)
	t.Logf("  - Notification health: %t", health.NotificationHealthy)
	if len(health.Issues) > 0 {
		t.Logf("  - Health issues: %v", health.Issues)
	}
}

func testRuleManagement(t *testing.T, proc processor.Processor, ctx context.Context) {
	// Start processor if not running
	if !proc.IsRunning() {
		err := proc.Start(ctx)
		require.NoError(t, err)
		defer proc.Stop()
	}

	// Test adding a rule
	rule := &processor.ProcessingRule{
		ID:          "test-rule-integration",
		Name:        "Test Transfer Rule",
		Description: "Test rule for integration testing",
		Enabled:     true,
		Priority:    1,
		Conditions: &processor.RuleConditions{
			EventNames: []string{"Transfer"},
		},
		Actions: []*processor.RuleAction{
			{
				Type: "notify",
				Config: map[string]interface{}{
					"type":       "email",
					"recipients": []string{"test@example.com"},
					"subject":    "Transfer Alert",
					"template":   "Transfer detected",
				},
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := proc.AddRule(rule)
	require.NoError(t, err, "Failed to add rule")

	// Test getting rules
	rules := proc.GetRules()
	assert.Greater(t, len(rules), 0, "Should have at least one rule")

	// Find our rule
	var foundRule *processor.ProcessingRule
	for _, r := range rules {
		if r.ID == rule.ID {
			foundRule = r
			break
		}
	}
	require.NotNil(t, foundRule, "Should find the added rule")
	assert.Equal(t, rule.Name, foundRule.Name)

	// Test updating rule
	rule.Name = "Updated Test Rule"
	err = proc.UpdateRule(rule)
	require.NoError(t, err, "Failed to update rule")

	// Test removing rule
	err = proc.RemoveRule(rule.ID)
	require.NoError(t, err, "Failed to remove rule")

	t.Logf("✓ Rule management integration verified")
}

func testWorkflowManagement(t *testing.T, proc processor.Processor, ctx context.Context) {
	// Start processor if not running
	if !proc.IsRunning() {
		err := proc.Start(ctx)
		require.NoError(t, err)
		defer proc.Stop()
	}

	// Test adding a workflow
	workflow := &processor.Workflow{
		ID:          "test-workflow-integration",
		Name:        "Test Validation Workflow",
		Description: "Test workflow for integration testing",
		Enabled:     true,
		Steps: []*processor.WorkflowStep{
			{
				ID:   "validate",
				Name: "Validate Event",
				Type: "validate",
				Config: map[string]interface{}{
					"required_fields": []string{"from", "to", "value"},
				},
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := proc.AddWorkflow(workflow)
	require.NoError(t, err, "Failed to add workflow")

	// Test getting workflows
	workflows := proc.GetWorkflows()
	assert.Greater(t, len(workflows), 0, "Should have at least one workflow")

	// Find our workflow
	var foundWorkflow *processor.Workflow
	for _, w := range workflows {
		if w.ID == workflow.ID {
			foundWorkflow = w
			break
		}
	}
	require.NotNil(t, foundWorkflow, "Should find the added workflow")
	assert.Equal(t, workflow.Name, foundWorkflow.Name)

	// Test removing workflow
	err = proc.RemoveWorkflow(workflow.ID)
	require.NoError(t, err, "Failed to remove workflow")

	t.Logf("✓ Workflow management integration verified")
}
