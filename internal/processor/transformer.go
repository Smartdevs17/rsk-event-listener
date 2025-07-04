// File: internal/processor/transformer.go
package processor

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/smartdevs17/rsk-event-listener/internal/models"
)

// EventTransformer handles event transformations
type EventTransformer struct {
	config *ProcessorConfig
}

// NewEventTransformer creates a new event transformer
func NewEventTransformer(config *ProcessorConfig) *EventTransformer {
	return &EventTransformer{
		config: config,
	}
}

// TransformEvent transforms an event based on transformation rules
func (et *EventTransformer) TransformEvent(event *models.Event, transformationRules []string) (*models.Event, error) {
	transformedEvent := *event // Copy event

	for _, rule := range transformationRules {
		if err := et.applyTransformationRule(&transformedEvent, rule); err != nil {
			return nil, fmt.Errorf("failed to apply transformation rule %s: %w", rule, err)
		}
	}

	return &transformedEvent, nil
}

// applyTransformationRule applies a single transformation rule
func (et *EventTransformer) applyTransformationRule(event *models.Event, rule string) error {
	parts := strings.Split(rule, ":")
	if len(parts) < 2 {
		return fmt.Errorf("invalid transformation rule format: %s", rule)
	}

	transformationType := parts[0]
	transformationConfig := parts[1]

	switch transformationType {
	case "convert_wei_to_eth":
		return et.convertWeiToEth(event, transformationConfig)
	case "add_timestamp_formatted":
		return et.addFormattedTimestamp(event, transformationConfig)
	case "normalize_address":
		return et.normalizeAddress(event, transformationConfig)
	case "add_computed_field":
		return et.addComputedField(event, transformationConfig)
	default:
		return fmt.Errorf("unknown transformation type: %s", transformationType)
	}
}

// convertWeiToEth converts wei values to ETH
func (et *EventTransformer) convertWeiToEth(event *models.Event, config string) error {
	fields := strings.Split(config, ",")

	for _, field := range fields {
		if value, exists := event.Data[field]; exists {
			if strValue, ok := value.(string); ok {
				// Convert string to big int, then to ETH
				// This is a simplified conversion - use proper big.Int arithmetic in production
				if weiValue, err := strconv.ParseFloat(strValue, 64); err == nil {
					ethValue := weiValue / 1e18
					event.Data[field+"_eth"] = fmt.Sprintf("%.6f", ethValue)
				}
			}
		}
	}

	return nil
}

// addFormattedTimestamp adds formatted timestamp fields
func (et *EventTransformer) addFormattedTimestamp(event *models.Event, config string) error {
	event.Data["timestamp_formatted"] = event.Timestamp.Format(config)
	return nil
}

// normalizeAddress normalizes address fields
func (et *EventTransformer) normalizeAddress(event *models.Event, config string) error {
	fields := strings.Split(config, ",")

	for _, field := range fields {
		if value, exists := event.Data[field]; exists {
			if addrValue, ok := value.(string); ok {
				// Normalize to lowercase
				event.Data[field] = strings.ToLower(addrValue)
			}
		}
	}

	return nil
}

// addComputedField adds a computed field based on other fields
func (et *EventTransformer) addComputedField(event *models.Event, config string) error {
	// Parse config: "field_name=expression"
	parts := strings.Split(config, "=")
	if len(parts) != 2 {
		return fmt.Errorf("invalid computed field config: %s", config)
	}

	fieldName := parts[0]
	expression := parts[1]

	// Simple expression evaluation - extend as needed
	result, err := et.evaluateExpression(event.Data, expression)
	if err != nil {
		return err
	}

	event.Data[fieldName] = result
	return nil
}

// evaluateExpression evaluates a simple expression
func (et *EventTransformer) evaluateExpression(data map[string]interface{}, expression string) (interface{}, error) {
	// This is a very basic expression evaluator
	// In production, consider using a proper expression library

	if strings.HasPrefix(expression, "$") {
		// Field reference
		fieldName := expression[1:]
		if value, exists := data[fieldName]; exists {
			return value, nil
		}
		return nil, fmt.Errorf("field not found: %s", fieldName)
	}

	// Literal value
	if strings.HasPrefix(expression, "'") && strings.HasSuffix(expression, "'") {
		return expression[1 : len(expression)-1], nil
	}

	// Try to parse as number
	if value, err := strconv.ParseFloat(expression, 64); err == nil {
		return value, nil
	}

	return expression, nil
}
