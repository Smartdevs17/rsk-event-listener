// File: internal/processor/processor.go
package processor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/smartdevs17/rsk-event-listener/internal/models"
	"github.com/smartdevs17/rsk-event-listener/internal/notification"
	"github.com/smartdevs17/rsk-event-listener/internal/storage"
	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
)

// Processor defines the event processor interface
type Processor interface {
	// Lifecycle management
	Start(ctx context.Context) error
	Stop() error
	IsRunning() bool

	// Event processing
	ProcessEvent(ctx context.Context, event *models.Event) (*ProcessResult, error)
	ProcessEvents(ctx context.Context, events []*models.Event) (*BatchProcessResult, error)

	// Rule management
	AddRule(rule *ProcessingRule) error
	RemoveRule(ruleID string) error
	GetRules() []*ProcessingRule
	UpdateRule(rule *ProcessingRule) error

	// Workflow management
	AddWorkflow(workflow *Workflow) error
	RemoveWorkflow(workflowID string) error
	GetWorkflows() []*Workflow

	// Statistics and monitoring
	GetStats() *ProcessorStats
	GetHealth() *ProcessorHealth
}

// EventProcessor implements the Processor interface
type EventProcessor struct {
	// Dependencies
	storage  storage.Storage
	notifier notification.Notifier
	logger   *logrus.Logger

	// Configuration
	config *ProcessorConfig

	// State management
	mu        sync.RWMutex
	running   bool
	rules     map[string]*ProcessingRule
	workflows map[string]*Workflow
	stopOnce  sync.Once
	stopChan  chan struct{}
	wg        sync.WaitGroup

	// Processing components
	router      *EventRouter
	transformer *EventTransformer
	aggregator  *EventAggregator
	validator   *EventValidator

	// Statistics
	stats *ProcessorStats
}

// ProcessorConfig holds processor configuration
type ProcessorConfig struct {
	MaxConcurrentProcessing int           `json:"max_concurrent_processing"`
	ProcessingTimeout       time.Duration `json:"processing_timeout"`
	RetryAttempts           int           `json:"retry_attempts"`
	RetryDelay              time.Duration `json:"retry_delay"`
	EnableAggregation       bool          `json:"enable_aggregation"`
	AggregationWindow       time.Duration `json:"aggregation_window"`
	EnableValidation        bool          `json:"enable_validation"`
	BufferSize              int           `json:"buffer_size"`
}

// ProcessingRule defines a rule for processing events
type ProcessingRule struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Enabled     bool            `json:"enabled"`
	Priority    int             `json:"priority"`
	Conditions  *RuleConditions `json:"conditions"`
	Actions     []*RuleAction   `json:"actions"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// RuleConditions defines conditions for rule execution
type RuleConditions struct {
	ContractAddresses []string               `json:"contract_addresses,omitempty"`
	EventNames        []string               `json:"event_names,omitempty"`
	EventSignatures   []string               `json:"event_signatures,omitempty"`
	BlockRange        *BlockRange            `json:"block_range,omitempty"`
	DataConditions    map[string]interface{} `json:"data_conditions,omitempty"`
	CustomScript      string                 `json:"custom_script,omitempty"`
}

// BlockRange defines a block range for conditions
type BlockRange struct {
	FromBlock *uint64 `json:"from_block,omitempty"`
	ToBlock   *uint64 `json:"to_block,omitempty"`
}

// RuleAction defines an action to execute when rule conditions are met
type RuleAction struct {
	Type       string                 `json:"type"`
	Config     map[string]interface{} `json:"config"`
	RetryLogic *RetryConfig           `json:"retry_logic,omitempty"`
}

// RetryConfig defines retry logic for actions
type RetryConfig struct {
	MaxAttempts int           `json:"max_attempts"`
	Delay       time.Duration `json:"delay"`
	Backoff     string        `json:"backoff"` // linear, exponential
}

// Workflow defines a sequence of processing steps
type Workflow struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Enabled     bool            `json:"enabled"`
	Steps       []*WorkflowStep `json:"steps"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// WorkflowStep defines a step in a workflow
type WorkflowStep struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Config     map[string]interface{} `json:"config"`
	Conditions *StepConditions        `json:"conditions,omitempty"`
	OnSuccess  string                 `json:"on_success,omitempty"`
	OnFailure  string                 `json:"on_failure,omitempty"`
}

// StepConditions defines conditions for step execution
type StepConditions struct {
	Expression string                 `json:"expression"`
	Variables  map[string]interface{} `json:"variables,omitempty"`
}

// ProcessResult contains the result of processing a single event
type ProcessResult struct {
	EventID           string                 `json:"event_id"`
	ProcessedAt       time.Time              `json:"processed_at"`
	ProcessingTime    time.Duration          `json:"processing_time"`
	RulesExecuted     []string               `json:"rules_executed"`
	ActionsExecuted   []string               `json:"actions_executed"`
	WorkflowsExecuted []string               `json:"workflows_executed"`
	Transformations   []string               `json:"transformations"`
	Notifications     []string               `json:"notifications"`
	Success           bool                   `json:"success"`
	Error             error                  `json:"error,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// BatchProcessResult contains the result of processing multiple events
type BatchProcessResult struct {
	TotalEvents     int                       `json:"total_events"`
	ProcessedEvents int                       `json:"processed_events"`
	FailedEvents    int                       `json:"failed_events"`
	ProcessingTime  time.Duration             `json:"processing_time"`
	Results         map[string]*ProcessResult `json:"results"`
	Errors          []error                   `json:"errors,omitempty"`
}

// ProcessorStats provides processor statistics
type ProcessorStats struct {
	StartTime              time.Time     `json:"start_time"`
	Uptime                 time.Duration `json:"uptime"`
	IsRunning              bool          `json:"is_running"`
	TotalEventsProcessed   uint64        `json:"total_events_processed"`
	TotalRulesExecuted     uint64        `json:"total_rules_executed"`
	TotalActionsExecuted   uint64        `json:"total_actions_executed"`
	TotalNotificationsSent uint64        `json:"total_notifications_sent"`
	ActiveRules            int           `json:"active_rules"`
	ActiveWorkflows        int           `json:"active_workflows"`
	AverageProcessingTime  time.Duration `json:"average_processing_time"`
	ProcessingRate         float64       `json:"processing_rate"` // events per second
	ErrorCount             uint64        `json:"error_count"`
	LastError              *string       `json:"last_error,omitempty"`
	LastErrorTime          *time.Time    `json:"last_error_time,omitempty"`
}

// ProcessorHealth provides processor health information
type ProcessorHealth struct {
	Healthy             bool     `json:"healthy"`
	StorageHealthy      bool     `json:"storage_healthy"`
	NotificationHealthy bool     `json:"notification_healthy"`
	ProcessingQueueSize int      `json:"processing_queue_size"`
	Issues              []string `json:"issues,omitempty"`
}

// NewEventProcessor creates a new event processor
func NewEventProcessor(
	storage storage.Storage,
	notifier notification.Notifier,
	config *ProcessorConfig,
) *EventProcessor {
	processor := &EventProcessor{
		storage:   storage,
		notifier:  notifier,
		config:    config,
		logger:    utils.GetLogger(),
		rules:     make(map[string]*ProcessingRule),
		workflows: make(map[string]*Workflow),
		stopChan:  make(chan struct{}),
		stats: &ProcessorStats{
			StartTime: time.Now(),
		},
	}

	// Initialize components
	processor.router = NewEventRouter(config)
	processor.transformer = NewEventTransformer(config)
	processor.aggregator = NewEventAggregator(config)
	processor.validator = NewEventValidator(config)

	return processor
}

// Start starts the event processor
func (ep *EventProcessor) Start(ctx context.Context) error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	if ep.running {
		return utils.NewAppError(utils.ErrCodeInternal, "Processor already running", "")
	}

	ep.logger.Info("Starting event processor")

	// Load rules and workflows from storage
	if err := ep.loadRulesAndWorkflows(ctx); err != nil {
		return utils.NewAppError(utils.ErrCodeInternal, "Failed to load rules and workflows", err.Error())
	}

	// Start processing
	ep.running = true
	ep.stats.StartTime = time.Now()
	ep.stats.IsRunning = true

	ep.logger.Info("Event processor started",
		"rules", len(ep.rules),
		"workflows", len(ep.workflows))

	return nil
}

// Stop stops the event processor
func (ep *EventProcessor) Stop() error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	if !ep.running {
		return nil
	}

	ep.logger.Info("Stopping event processor")

	ep.running = false
	ep.stats.IsRunning = false

	ep.stopOnce.Do(func() {
		close(ep.stopChan)
	})

	// Wait for goroutines to finish
	ep.wg.Wait()

	ep.logger.Info("Event processor stopped")
	return nil
}

// IsRunning returns whether the processor is running
func (ep *EventProcessor) IsRunning() bool {
	ep.mu.RLock()
	defer ep.mu.RUnlock()
	return ep.running
}

// ProcessEvent processes a single event
func (ep *EventProcessor) ProcessEvent(ctx context.Context, event *models.Event) (*ProcessResult, error) {
	startTime := time.Now()

	result := &ProcessResult{
		EventID:     event.ID,
		ProcessedAt: startTime,
		Success:     true,
		Metadata:    make(map[string]interface{}),
	}

	// Validate event if validation is enabled
	if ep.config.EnableValidation {
		if err := ep.validator.ValidateEvent(event); err != nil {
			result.Success = false
			result.Error = err
			return result, err
		}
	}

	// Route event to determine applicable rules and workflows
	routingResult := ep.router.RouteEvent(event, ep.rules, ep.workflows)
	result.RulesExecuted = routingResult.ApplicableRules
	result.WorkflowsExecuted = routingResult.ApplicableWorkflows

	// Execute applicable rules
	for _, ruleID := range routingResult.ApplicableRules {
		rule := ep.rules[ruleID]

		// Check if rule exists
		if rule == nil {
			ep.logger.Warn("Rule not found", "rule_id", ruleID)
			continue
		}
		if err := ep.executeRule(ctx, event, rule, result); err != nil {
			ep.logger.Error("Failed to execute rule", "rule_id", ruleID, "error", err)
			result.Success = false
			result.Error = err
		}
	}

	// Execute applicable workflows
	for _, workflowID := range routingResult.ApplicableWorkflows {
		workflow := ep.workflows[workflowID]
		if workflow == nil {
			ep.logger.Warn("Workflow not found", "workflow_id", workflowID)
			continue
		}
		if err := ep.executeWorkflow(ctx, event, workflow, result); err != nil {
			ep.logger.Error("Failed to execute workflow", "workflow_id", workflowID, "error", err)
			result.Success = false
			result.Error = err
		}
	}

	// Transform event if needed
	if len(routingResult.Transformations) > 0 {
		transformedEvent, err := ep.transformer.TransformEvent(event, routingResult.Transformations)
		if err != nil {
			ep.logger.Warn("Failed to transform event", "error", err)
		} else {
			result.Transformations = routingResult.Transformations
			// Update event in storage with transformed data
			if err := ep.storage.UpdateEvent(ctx, transformedEvent); err != nil {
				ep.logger.Warn("Failed to update transformed event", "error", err)
			}
		}
	}

	// Mark event as processed
	event.Processed = true
	if err := ep.storage.UpdateEvent(ctx, event); err != nil {
		ep.logger.Warn("Failed to mark event as processed", "error", err)
	}

	result.ProcessingTime = time.Since(startTime)

	// Update statistics
	ep.updateStats(result)

	ep.logger.Debug("Event processed",
		"event_id", event.ID,
		"rules_executed", len(result.RulesExecuted),
		"workflows_executed", len(result.WorkflowsExecuted),
		"processing_time", result.ProcessingTime)

	return result, nil
}

// ProcessEvents processes multiple events
func (ep *EventProcessor) ProcessEvents(ctx context.Context, events []*models.Event) (*BatchProcessResult, error) {
	startTime := time.Now()

	result := &BatchProcessResult{
		TotalEvents: len(events),
		Results:     make(map[string]*ProcessResult),
		Errors:      []error{},
	}

	// Process events concurrently with rate limiting
	semaphore := make(chan struct{}, ep.config.MaxConcurrentProcessing)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, event := range events {
		wg.Add(1)
		go func(e *models.Event) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					mu.Lock()
					result.FailedEvents++
					result.Errors = append(result.Errors, fmt.Errorf("panic: %v", r))
					mu.Unlock()
				}
			}()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			processResult, err := ep.ProcessEvent(ctx, e)

			mu.Lock()
			result.Results[e.ID] = processResult
			if err != nil {
				result.FailedEvents++
				result.Errors = append(result.Errors, err)
			} else {
				result.ProcessedEvents++
			}
			mu.Unlock()
		}(event)
	}

	wg.Wait()
	result.ProcessingTime = time.Since(startTime)

	ep.logger.Info("Batch processing completed",
		"total_events", result.TotalEvents,
		"processed_events", result.ProcessedEvents,
		"failed_events", result.FailedEvents,
		"processing_time", result.ProcessingTime)

	return result, nil
}

// executeRule executes a processing rule
func (ep *EventProcessor) executeRule(ctx context.Context, event *models.Event, rule *ProcessingRule, result *ProcessResult) error {
	if !rule.Enabled {
		return nil
	}

	ep.logger.Debug("Executing rule", "rule_id", rule.ID, "rule_name", rule.Name)

	// Execute rule actions
	for _, action := range rule.Actions {
		if err := ep.executeAction(ctx, event, action, result); err != nil {
			if action.RetryLogic != nil {
				if err := ep.retryAction(ctx, event, action, result); err != nil {
					return fmt.Errorf("failed to execute action %s after retries: %w", action.Type, err)
				}
			} else {
				return fmt.Errorf("failed to execute action %s: %w", action.Type, err)
			}
		}
		result.ActionsExecuted = append(result.ActionsExecuted, action.Type)
	}

	// Update rule execution statistics
	ep.mu.Lock()
	ep.stats.TotalRulesExecuted++
	ep.mu.Unlock()

	return nil
}

// executeAction executes a rule action
func (ep *EventProcessor) executeAction(ctx context.Context, event *models.Event, action *RuleAction, result *ProcessResult) error {
	switch action.Type {
	case "notify":
		return ep.executeNotifyAction(ctx, event, action, result)
	case "webhook":
		return ep.executeWebhookAction(ctx, event, action, result)
	case "store":
		return ep.executeStoreAction(ctx, event, action, result)
	case "transform":
		return ep.executeTransformAction(ctx, event, action, result)
	case "aggregate":
		return ep.executeAggregateAction(ctx, event, action, result)
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

// executeNotifyAction executes a notification action
func (ep *EventProcessor) executeNotifyAction(ctx context.Context, event *models.Event, action *RuleAction, result *ProcessResult) error {
	notificationConfig := &notification.NotificationConfig{
		Type:       action.Config["type"].(string),
		Recipients: action.Config["recipients"].([]string),
		Subject:    action.Config["subject"].(string),
		Template:   action.Config["template"].(string),
	}

	notificationData := map[string]interface{}{
		"event":     event,
		"timestamp": time.Now(),
		"metadata":  result.Metadata,
	}

	notificationID, err := ep.notifier.SendNotification(ctx, notificationConfig, notificationData)
	if err != nil {
		return err
	}

	result.Notifications = append(result.Notifications, notificationID)

	// Update notification statistics
	ep.mu.Lock()
	ep.stats.TotalNotificationsSent++
	ep.mu.Unlock()

	return nil
}

// executeWebhookAction executes a webhook action
func (ep *EventProcessor) executeWebhookAction(ctx context.Context, event *models.Event, action *RuleAction, result *ProcessResult) error {
	webhookConfig := &notification.WebhookConfig{
		URL:     action.Config["url"].(string),
		Method:  action.Config["method"].(string),
		Headers: action.Config["headers"].(map[string]string),
		Timeout: time.Duration(action.Config["timeout"].(int)) * time.Second,
	}

	webhookData := map[string]interface{}{
		"event":     event,
		"timestamp": time.Now(),
		"metadata":  result.Metadata,
	}

	return ep.notifier.SendWebhook(ctx, webhookConfig, webhookData)
}

// executeStoreAction executes a store action
func (ep *EventProcessor) executeStoreAction(ctx context.Context, event *models.Event, action *RuleAction, result *ProcessResult) error {
	// Store additional data or create derived records
	storageData := action.Config["data"].(map[string]interface{})

	// Add event context to storage data
	storageData["source_event_id"] = event.ID
	storageData["processed_at"] = time.Now()

	return ep.storage.StoreProcessingResult(ctx, storageData)
}

// executeTransformAction executes a transform action
func (ep *EventProcessor) executeTransformAction(ctx context.Context, event *models.Event, action *RuleAction, result *ProcessResult) error {
	transformationRules := action.Config["rules"].([]string)

	transformedEvent, err := ep.transformer.TransformEvent(event, transformationRules)
	if err != nil {
		return err
	}

	// Update the event
	*event = *transformedEvent
	result.Transformations = append(result.Transformations, transformationRules...)

	return nil
}

// executeAggregateAction executes an aggregate action
func (ep *EventProcessor) executeAggregateAction(ctx context.Context, event *models.Event, action *RuleAction, result *ProcessResult) error {
	aggregationConfig := &AggregationConfig{
		GroupBy:   action.Config["group_by"].([]string),
		Functions: action.Config["functions"].([]string),
		Window:    time.Duration(action.Config["window"].(int)) * time.Second,
	}

	return ep.aggregator.AddEventToAggregation(event, aggregationConfig)
}

// retryAction retries an action with configured retry logic
func (ep *EventProcessor) retryAction(ctx context.Context, event *models.Event, action *RuleAction, result *ProcessResult) error {
	retryConfig := action.RetryLogic
	var lastErr error

	for attempt := 1; attempt <= retryConfig.MaxAttempts; attempt++ {
		if attempt > 1 {
			// Calculate delay based on backoff strategy
			delay := retryConfig.Delay
			if retryConfig.Backoff == "exponential" {
				delay = time.Duration(int64(delay) << uint(attempt-2))
			}

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		err := ep.executeAction(ctx, event, action, result)
		if err == nil {
			return nil
		}

		lastErr = err
		ep.logger.Warn("Action retry failed",
			"action_type", action.Type,
			"attempt", attempt,
			"error", err)
	}

	return fmt.Errorf("action failed after %d attempts: %w", retryConfig.MaxAttempts, lastErr)
}

// executeWorkflow executes a workflow
func (ep *EventProcessor) executeWorkflow(ctx context.Context, event *models.Event, workflow *Workflow, result *ProcessResult) error {
	if !workflow.Enabled {
		return nil
	}

	ep.logger.Debug("Executing workflow", "workflow_id", workflow.ID, "workflow_name", workflow.Name)

	workflowContext := &WorkflowContext{
		Event:     event,
		Variables: make(map[string]interface{}),
		Result:    result,
	}

	for _, step := range workflow.Steps {
		if err := ep.executeWorkflowStep(ctx, step, workflowContext); err != nil {
			return fmt.Errorf("workflow step %s failed: %w", step.Name, err)
		}
	}

	return nil
}
