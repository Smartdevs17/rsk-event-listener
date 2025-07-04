// File: internal/processor/validator.go
package processor

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartdevs17/rsk-event-listener/internal/models"
	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
)

// EventValidator handles event validation
type EventValidator struct {
	config          *ProcessorConfig
	validationRules map[string][]*ValidationRule
	addressRegex    *regexp.Regexp
	txHashRegex     *regexp.Regexp
	blockHashRegex  *regexp.Regexp
}

// ValidationRule defines a validation rule
type ValidationRule struct {
	Field      string      `json:"field"`
	Type       string      `json:"type"`  // required, format, range, custom, regex
	Value      interface{} `json:"value"` // Changed from string to interface{}
	Message    string      `json:"message"`
	Required   bool        `json:"required"`
	MinValue   *float64    `json:"min_value,omitempty"`
	MaxValue   *float64    `json:"max_value,omitempty"`
	Pattern    string      `json:"pattern,omitempty"`
	CustomFunc string      `json:"custom_func,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string      `json:"field"`
	Type    string      `json:"type"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// ValidationResult contains validation results
type ValidationResult struct {
	Valid  bool               `json:"valid"`
	Errors []*ValidationError `json:"errors,omitempty"`
}

// NewEventValidator creates a new event validator
func NewEventValidator(config *ProcessorConfig) *EventValidator {
	// Compile regex patterns once for efficiency
	addressRegex := regexp.MustCompile(`^0x[a-fA-F0-9]{40}$`)
	txHashRegex := regexp.MustCompile(`^0x[a-fA-F0-9]{64}$`)
	blockHashRegex := regexp.MustCompile(`^0x[a-fA-F0-9]{64}$`)

	return &EventValidator{
		config:          config,
		validationRules: make(map[string][]*ValidationRule),
		addressRegex:    addressRegex,
		txHashRegex:     txHashRegex,
		blockHashRegex:  blockHashRegex,
	}
}

// ValidateEvent validates an event with comprehensive checks
func (ev *EventValidator) ValidateEvent(event *models.Event) error {
	result := ev.ValidateEventDetailed(event)
	if !result.Valid {
		// Combine all error messages
		var errorMessages []string
		for _, err := range result.Errors {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", err.Field, err.Message))
		}
		return utils.NewAppError(utils.ErrCodeValidation,
			"Event validation failed",
			strings.Join(errorMessages, "; "))
	}
	return nil
}

// ValidateEventDetailed validates an event and returns detailed results
func (ev *EventValidator) ValidateEventDetailed(event *models.Event) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Errors: []*ValidationError{},
	}

	// Basic required field validation
	ev.validateRequiredFields(event, result)

	// Format validation
	ev.validateFormats(event, result)

	// Business logic validation
	ev.validateBusinessLogic(event, result)

	// Custom validation rules (if any are configured)
	ev.validateCustomRules(event, result)

	// Set overall validity
	result.Valid = len(result.Errors) == 0

	return result
}

// validateRequiredFields validates required fields
func (ev *EventValidator) validateRequiredFields(event *models.Event, result *ValidationResult) {
	// Event ID is required
	if event.ID == "" {
		result.Errors = append(result.Errors, &ValidationError{
			Field:   "id",
			Type:    "required",
			Message: "Event ID is required",
		})
	}

	// Contract address is required
	if event.Address == "" {
		result.Errors = append(result.Errors, &ValidationError{
			Field:   "address",
			Type:    "required",
			Message: "Contract address is required",
		})
	}

	// Event name is required
	if event.EventName == "" {
		result.Errors = append(result.Errors, &ValidationError{
			Field:   "event_name",
			Type:    "required",
			Message: "Event name is required",
		})
	}

	// Transaction hash is required
	if event.TxHash == "" {
		result.Errors = append(result.Errors, &ValidationError{
			Field:   "tx_hash",
			Type:    "required",
			Message: "Transaction hash is required",
		})
	}

	// Block hash is required
	if event.BlockHash == "" {
		result.Errors = append(result.Errors, &ValidationError{
			Field:   "block_hash",
			Type:    "required",
			Message: "Block hash is required",
		})
	}
}

// validateFormats validates field formats
func (ev *EventValidator) validateFormats(event *models.Event, result *ValidationResult) {
	// Validate address format
	if event.Address != "" {
		if err := ev.validateAddressFormat(event.Address); err != nil {
			result.Errors = append(result.Errors, &ValidationError{
				Field:   "address",
				Type:    "format",
				Message: err.Error(),
				Value:   event.Address,
			})
		}
	}

	// Validate transaction hash format
	if event.TxHash != "" {
		if err := ev.validateTxHashFormat(event.TxHash); err != nil {
			result.Errors = append(result.Errors, &ValidationError{
				Field:   "tx_hash",
				Type:    "format",
				Message: err.Error(),
				Value:   event.TxHash,
			})
		}
	}

	// Validate block hash format
	if event.BlockHash != "" {
		if err := ev.validateBlockHashFormat(event.BlockHash); err != nil {
			result.Errors = append(result.Errors, &ValidationError{
				Field:   "block_hash",
				Type:    "format",
				Message: err.Error(),
				Value:   event.BlockHash,
			})
		}
	}

	// Validate event signature format (if present)
	if event.EventSig != "" {
		if err := ev.validateEventSignatureFormat(event.EventSig); err != nil {
			result.Errors = append(result.Errors, &ValidationError{
				Field:   "event_sig",
				Type:    "format",
				Message: err.Error(),
				Value:   event.EventSig,
			})
		}
	}
}

// validateBusinessLogic validates business logic rules
func (ev *EventValidator) validateBusinessLogic(event *models.Event, result *ValidationResult) {
	// Validate block number
	if event.BlockNumber == 0 {
		result.Errors = append(result.Errors, &ValidationError{
			Field:   "block_number",
			Type:    "range",
			Message: "Block number must be greater than 0",
			Value:   event.BlockNumber,
		})
	}

	// Validate reasonable block number (not too far in the future)
	currentTime := time.Now()
	// Assuming 15 second block time, calculate reasonable max block
	estimatedCurrentBlock := uint64(currentTime.Unix() / 15)
	if event.BlockNumber > estimatedCurrentBlock+1000 { // Allow some buffer
		result.Errors = append(result.Errors, &ValidationError{
			Field:   "block_number",
			Type:    "range",
			Message: "Block number appears to be too far in the future",
			Value:   event.BlockNumber,
		})
	}

	// Validate timestamp is not too far in the future
	if event.Timestamp.After(currentTime.Add(1 * time.Hour)) {
		result.Errors = append(result.Errors, &ValidationError{
			Field:   "timestamp",
			Type:    "range",
			Message: "Event timestamp is too far in the future",
			Value:   event.Timestamp,
		})
	}

	// Validate transaction and log indices are reasonable
	if event.TxIndex > 10000 { // Reasonable limit for transactions per block
		result.Errors = append(result.Errors, &ValidationError{
			Field:   "tx_index",
			Type:    "range",
			Message: "Transaction index is unusually high",
			Value:   event.TxIndex,
		})
	}

	if event.LogIndex > 100000 { // Reasonable limit for logs per block
		result.Errors = append(result.Errors, &ValidationError{
			Field:   "log_index",
			Type:    "range",
			Message: "Log index is unusually high",
			Value:   event.LogIndex,
		})
	}
}

// validateCustomRules validates custom validation rules
func (ev *EventValidator) validateCustomRules(event *models.Event, result *ValidationResult) {
	// Check if there are custom rules for this event type
	eventKey := fmt.Sprintf("%s:%s", event.Address, event.EventName)
	if rules, exists := ev.validationRules[eventKey]; exists {
		for _, rule := range rules {
			if err := ev.applyValidationRule(event, rule); err != nil {
				result.Errors = append(result.Errors, &ValidationError{
					Field:   rule.Field,
					Type:    rule.Type,
					Message: err.Error(),
				})
			}
		}
	}

	// Check for global rules (apply to all events)
	if rules, exists := ev.validationRules["*"]; exists {
		for _, rule := range rules {
			if err := ev.applyValidationRule(event, rule); err != nil {
				result.Errors = append(result.Errors, &ValidationError{
					Field:   rule.Field,
					Type:    rule.Type,
					Message: err.Error(),
				})
			}
		}
	}
}

// applyValidationRule applies a single validation rule
func (ev *EventValidator) applyValidationRule(event *models.Event, rule *ValidationRule) error {
	var fieldValue interface{}

	// Get field value based on field name
	switch rule.Field {
	case "id":
		fieldValue = event.ID
	case "address":
		fieldValue = event.Address
	case "event_name":
		fieldValue = event.EventName
	case "event_sig":
		fieldValue = event.EventSig
	case "block_number":
		fieldValue = event.BlockNumber
	case "block_hash":
		fieldValue = event.BlockHash
	case "tx_hash":
		fieldValue = event.TxHash
	case "tx_index":
		fieldValue = event.TxIndex
	case "log_index":
		fieldValue = event.LogIndex
	case "timestamp":
		fieldValue = event.Timestamp
	case "processed":
		fieldValue = event.Processed
	default:
		// Check in event data
		if event.Data != nil {
			if value, exists := event.Data[rule.Field]; exists {
				fieldValue = value
			}
		}
	}

	// Apply validation based on rule type
	switch rule.Type {
	case "required":
		if fieldValue == nil || fieldValue == "" {
			return fmt.Errorf(rule.Message)
		}
	case "format":
		return ev.validateFieldFormat(fieldValue, rule)
	case "range":
		return ev.validateFieldRange(fieldValue, rule)
	case "regex":
		return ev.validateFieldRegex(fieldValue, rule)
	case "custom":
		return ev.validateFieldCustom(fieldValue, rule)
	}

	return nil
}

// validateAddressFormat validates Ethereum address format
func (ev *EventValidator) validateAddressFormat(address string) error {
	if !ev.addressRegex.MatchString(address) {
		return fmt.Errorf("invalid address format: %s", address)
	}

	// Additional validation using go-ethereum's common package
	if !common.IsHexAddress(address) {
		return fmt.Errorf("invalid hex address format: %s", address)
	}

	return nil
}

// validateTxHashFormat validates transaction hash format
func (ev *EventValidator) validateTxHashFormat(txHash string) error {
	if !ev.txHashRegex.MatchString(txHash) {
		return fmt.Errorf("invalid transaction hash format: %s", txHash)
	}
	return nil
}

// validateBlockHashFormat validates block hash format
func (ev *EventValidator) validateBlockHashFormat(blockHash string) error {
	if !ev.blockHashRegex.MatchString(blockHash) {
		return fmt.Errorf("invalid block hash format: %s", blockHash)
	}
	return nil
}

// validateEventSignatureFormat validates event signature format
func (ev *EventValidator) validateEventSignatureFormat(eventSig string) error {
	// Event signatures are 32-byte hashes (64 hex characters with 0x prefix)
	eventSigRegex := regexp.MustCompile(`^0x[a-fA-F0-9]{64}$`)
	if !eventSigRegex.MatchString(eventSig) {
		return fmt.Errorf("invalid event signature format: %s", eventSig)
	}
	return nil
}

// validateFieldFormat validates field format based on rule
func (ev *EventValidator) validateFieldFormat(value interface{}, rule *ValidationRule) error {
	strValue := fmt.Sprintf("%v", value)

	if rule.Pattern != "" {
		matched, err := regexp.MatchString(rule.Pattern, strValue)
		if err != nil {
			return fmt.Errorf("invalid regex pattern: %s", rule.Pattern)
		}
		if !matched {
			return fmt.Errorf(rule.Message)
		}
	}

	return nil
}

// validateFieldRange validates field value range
func (ev *EventValidator) validateFieldRange(value interface{}, rule *ValidationRule) error {
	// Try to convert value to float64 for range checking
	var numValue float64
	var err error

	switch v := value.(type) {
	case float64:
		numValue = v
	case int:
		numValue = float64(v)
	case uint:
		numValue = float64(v)
	case uint64:
		numValue = float64(v)
	case string:
		numValue, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("cannot convert value to number for range validation: %v", value)
		}
	default:
		return fmt.Errorf("unsupported type for range validation: %T", value)
	}

	if rule.MinValue != nil && numValue < *rule.MinValue {
		return fmt.Errorf("value %v is below minimum %v", numValue, *rule.MinValue)
	}

	if rule.MaxValue != nil && numValue > *rule.MaxValue {
		return fmt.Errorf("value %v is above maximum %v", numValue, *rule.MaxValue)
	}

	return nil
}

// validateFieldRegex validates field using regex pattern
func (ev *EventValidator) validateFieldRegex(value interface{}, rule *ValidationRule) error {
	strValue := fmt.Sprintf("%v", value)

	if rule.Pattern == "" {
		return fmt.Errorf("regex pattern not specified")
	}

	matched, err := regexp.MatchString(rule.Pattern, strValue)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %s", rule.Pattern)
	}

	if !matched {
		return fmt.Errorf(rule.Message)
	}

	return nil
}

// validateFieldCustom validates field using custom function
func (ev *EventValidator) validateFieldCustom(value interface{}, rule *ValidationRule) error {
	// This is a placeholder for custom validation functions
	// In a real implementation, you might have a registry of custom validation functions
	switch rule.CustomFunc {
	case "is_positive":
		if numValue, ok := value.(float64); ok {
			if numValue <= 0 {
				return fmt.Errorf("value must be positive")
			}
		}
	case "is_valid_token_address":
		// Custom token address validation
		if addrValue, ok := value.(string); ok {
			return ev.validateAddressFormat(addrValue)
		}
	default:
		return fmt.Errorf("unknown custom validation function: %s", rule.CustomFunc)
	}

	return nil
}

// AddValidationRule adds a custom validation rule
func (ev *EventValidator) AddValidationRule(eventKey string, rule *ValidationRule) {
	if ev.validationRules[eventKey] == nil {
		ev.validationRules[eventKey] = []*ValidationRule{}
	}
	ev.validationRules[eventKey] = append(ev.validationRules[eventKey], rule)
}

// RemoveValidationRule removes a validation rule
func (ev *EventValidator) RemoveValidationRule(eventKey string, field string) {
	if rules, exists := ev.validationRules[eventKey]; exists {
		newRules := []*ValidationRule{}
		for _, rule := range rules {
			if rule.Field != field {
				newRules = append(newRules, rule)
			}
		}
		ev.validationRules[eventKey] = newRules
	}
}

// GetValidationRules returns validation rules for an event
func (ev *EventValidator) GetValidationRules(eventKey string) []*ValidationRule {
	if rules, exists := ev.validationRules[eventKey]; exists {
		return rules
	}
	return []*ValidationRule{}
}

// ValidateEventData validates specific event data fields
func (ev *EventValidator) ValidateEventData(eventData map[string]interface{}, rules []*ValidationRule) error {
	for _, rule := range rules {
		if _, exists := eventData[rule.Field]; exists {
			if err := ev.applyValidationRule(&models.Event{Data: eventData}, rule); err != nil {
				return fmt.Errorf("validation failed for field %s: %w", rule.Field, err)
			}
		} else if rule.Required {
			return fmt.Errorf("required field %s is missing", rule.Field)
		}
	}
	return nil
}

// IsValid performs a quick validation check
func (ev *EventValidator) IsValid(event *models.Event) bool {
	result := ev.ValidateEventDetailed(event)
	return result.Valid
}

// GetValidationSummary returns a summary of validation rules
func (ev *EventValidator) GetValidationSummary() map[string]int {
	summary := make(map[string]int)

	for eventKey, rules := range ev.validationRules {
		summary[eventKey] = len(rules)
	}

	return summary
}
