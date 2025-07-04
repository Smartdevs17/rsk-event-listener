// File: internal/processor/aggregator.go
package processor

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/smartdevs17/rsk-event-listener/internal/models"
)

// EventAggregator handles event aggregation
type EventAggregator struct {
	config       *ProcessorConfig
	mu           sync.RWMutex
	aggregations map[string]*AggregationGroup
}

// AggregationConfig defines aggregation configuration
type AggregationConfig struct {
	GroupBy   []string      `json:"group_by"`
	Functions []string      `json:"functions"`
	Window    time.Duration `json:"window"`
}

// AggregationGroup represents a group of aggregated events
type AggregationGroup struct {
	Key       string                 `json:"key"`
	Events    []*models.Event        `json:"events"`
	Count     int                    `json:"count"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
	Results   map[string]interface{} `json:"results"`
}

// NewEventAggregator creates a new event aggregator
func NewEventAggregator(config *ProcessorConfig) *EventAggregator {
	return &EventAggregator{
		config:       config,
		aggregations: make(map[string]*AggregationGroup),
	}
}

// AddEventToAggregation adds an event to aggregation
func (ea *EventAggregator) AddEventToAggregation(event *models.Event, config *AggregationConfig) error {
	ea.mu.Lock()
	defer ea.mu.Unlock()

	key := ea.generateGroupKey(event, config.GroupBy)

	group, exists := ea.aggregations[key]
	if !exists {
		group = &AggregationGroup{
			Key:       key,
			Events:    []*models.Event{},
			Count:     0,
			StartTime: time.Now(),
			Results:   make(map[string]interface{}),
		}
		ea.aggregations[key] = group
	}

	group.Events = append(group.Events, event)
	group.Count++
	group.EndTime = time.Now()

	// Apply aggregation functions
	ea.applyAggregationFunctions(group, config.Functions)

	return nil
}

// generateGroupKey generates a key for grouping events
func (ea *EventAggregator) generateGroupKey(event *models.Event, groupBy []string) string {
	var keyParts []string

	for _, field := range groupBy {
		switch field {
		case "contract_address":
			keyParts = append(keyParts, event.Address)
		case "event_name":
			keyParts = append(keyParts, event.EventName)
		case "block_number":
			keyParts = append(keyParts, string(rune(event.BlockNumber)))
		default:
			if value, exists := event.Data[field]; exists {
				keyParts = append(keyParts, fmt.Sprintf("%v", value))
			}
		}
	}

	return strings.Join(keyParts, "|")
}

// applyAggregationFunctions applies aggregation functions to a group
func (ea *EventAggregator) applyAggregationFunctions(group *AggregationGroup, functions []string) {
	for _, function := range functions {
		switch function {
		case "count":
			group.Results["count"] = group.Count
		case "sum":
			ea.applySumFunction(group)
		case "avg":
			ea.applyAvgFunction(group)
		case "min":
			ea.applyMinFunction(group)
		case "max":
			ea.applyMaxFunction(group)
		}
	}
}

// applySumFunction sums the "value" field across all events in the group
func (ea *EventAggregator) applySumFunction(group *AggregationGroup) {
	var sum float64
	for _, event := range group.Events {
		if event.Data != nil {
			if v, ok := event.Data["value"]; ok {
				switch val := v.(type) {
				case float64:
					sum += val
				case int:
					sum += float64(val)
				case int64:
					sum += float64(val)
				case uint64:
					sum += float64(val)
				case string:
					// Try to parse string as float
					if f, err := strconv.ParseFloat(val, 64); err == nil {
						sum += f
					}
				}
			}
		}
	}
	group.Results["sum"] = sum
}

// applyAvgFunction calculates the average of the "value" field across all events in the group
func (ea *EventAggregator) applyAvgFunction(group *AggregationGroup) {
	var sum float64
	var count int
	for _, event := range group.Events {
		if event.Data != nil {
			if v, ok := event.Data["value"]; ok {
				switch val := v.(type) {
				case float64:
					sum += val
					count++
				case int:
					sum += float64(val)
					count++
				case int64:
					sum += float64(val)
					count++
				case uint64:
					sum += float64(val)
					count++
				case string:
					if f, err := strconv.ParseFloat(val, 64); err == nil {
						sum += f
						count++
					}
				}
			}
		}
	}
	if count > 0 {
		group.Results["avg"] = sum / float64(count)
	} else {
		group.Results["avg"] = 0.0
	}
}

// applyMinFunction finds the minimum "value" field across all events in the group
func (ea *EventAggregator) applyMinFunction(group *AggregationGroup) {
	var min *float64
	for _, event := range group.Events {
		if event.Data != nil {
			if v, ok := event.Data["value"]; ok {
				var f float64
				switch val := v.(type) {
				case float64:
					f = val
				case int:
					f = float64(val)
				case int64:
					f = float64(val)
				case uint64:
					f = float64(val)
				case string:
					if parsed, err := strconv.ParseFloat(val, 64); err == nil {
						f = parsed
					} else {
						continue
					}
				default:
					continue
				}
				if min == nil || f < *min {
					min = &f
				}
			}
		}
	}
	if min != nil {
		group.Results["min"] = *min
	}
}

// applyMaxFunction finds the maximum "value" field across all events in the group
func (ea *EventAggregator) applyMaxFunction(group *AggregationGroup) {
	var max *float64
	for _, event := range group.Events {
		if event.Data != nil {
			if v, ok := event.Data["value"]; ok {
				var f float64
				switch val := v.(type) {
				case float64:
					f = val
				case int:
					f = float64(val)
				case int64:
					f = float64(val)
				case uint64:
					f = float64(val)
				case string:
					if parsed, err := strconv.ParseFloat(val, 64); err == nil {
						f = parsed
					} else {
						continue
					}
				default:
					continue
				}
				if max == nil || f > *max {
					max = &f
				}
			}
		}
	}
	if max != nil {
		group.Results["max"] = *max
	}
}

// // File: internal/processor/aggregator.go - AggregationConfig definition
// package processor

// import (
// 	"strings"
// 	"sync"
// 	"time"

// 	"github.com/smartdevs17/rsk-event-listener/internal/models"
// )

// // EventAggregator handles event aggregation
// type EventAggregator struct {
// 	config       *ProcessorConfig
// 	mu           sync.RWMutex
// 	aggregations map[string]*AggregationGroup
// }

// // AggregationConfig defines aggregation configuration
// type AggregationConfig struct {
// 	GroupBy   []string      `json:"group_by"`
// 	Functions []string      `json:"functions"`
// 	Window    time.Duration `json:"window"`
// }

// // AggregationGroup represents a group of aggregated events
// type AggregationGroup struct {
// 	Key       string                 `json:"key"`
// 	Events    []*models.Event        `json:"events"`
// 	Count     int                    `json:"count"`
// 	StartTime time.Time              `json:"start_time"`
// 	EndTime   time.Time              `json:"end_time"`
// 	Results   map[string]interface{} `json:"results"`
// }

// // NewEventAggregator creates a new event aggregator
// func NewEventAggregator(config *ProcessorConfig) *EventAggregator {
// 	return &EventAggregator{
// 		config:       config,
// 		aggregations: make(map[string]*AggregationGroup),
// 	}
// }

// // AddEventToAggregation adds an event to aggregation
// func (ea *EventAggregator) AddEventToAggregation(event *models.Event, config *AggregationConfig) error {
// 	ea.mu.Lock()
// 	defer ea.mu.Unlock()

// 	key := ea.generateGroupKey(event, config.GroupBy)

// 	group, exists := ea.aggregations[key]
// 	if !exists {
// 		group = &AggregationGroup{
// 			Key:       key,
// 			Events:    []*models.Event{},
// 			Count:     0,
// 			StartTime: time.Now(),
// 			Results:   make(map[string]interface{}),
// 		}
// 		ea.aggregations[key] = group
// 	}

// 	group.Events = append(group.Events, event)
// 	group.Count++
// 	group.EndTime = time.Now()

// 	// Apply aggregation functions
// 	ea.applyAggregationFunctions(group, config.Functions)

// 	return nil
// }

// // generateGroupKey generates a key for grouping events
// func (ea *EventAggregator) generateGroupKey(event *models.Event, groupBy []string) string {
// 	var keyParts []string

// 	for _, field := range groupBy {
// 		switch field {
// 		case "contract_address":
// 			keyParts = append(keyParts, event.Address)
// 		case "event_name":
// 			keyParts = append(keyParts, event.EventName)
// 		case "block_number":
// 			keyParts = append(keyParts, string(rune(event.BlockNumber)))
// 		default:
// 			if value, exists := event.Data[field]; exists {
// 				keyParts = append(keyParts, fmt.Sprintf("%v", value))
// 			}
// 		}
// 	}

// 	return strings.Join(keyParts, "|")
// }

// // applyAggregationFunctions applies aggregation functions to a group
// func (ea *EventAggregator) applyAggregationFunctions(group *AggregationGroup, functions []string) {
// 	for _, function := range functions {
// 		switch function {
// 		case "count":
// 			group.Results["count"] = group.Count
// 		case "sum":
// 			ea.applySumFunction(group)
// 		case "avg":
// 			ea.applyAvgFunction(group)
// 		case "min":
// 			ea.applyMinFunction(group)
// 		case "max":
// 			ea.applyMaxFunction(group)
// 		}
// 	}
// }

// // Placeholder aggregation function implementations
// func (ea *EventAggregator) applySumFunction(group *AggregationGroup) {
// 	// Implementation for sum function
// 	group.Results["sum"] = 0
// }

// func (ea *EventAggregator) applyAvgFunction(group *AggregationGroup) {
// 	// Implementation for average function
// 	group.Results["avg"] = 0
// }

// func (ea *EventAggregator) applyMinFunction(group *AggregationGroup) {
// 	// Implementation for min function
// 	group.Results["min"] = 0
// }

// func (ea *EventAggregator) applyMaxFunction(group *AggregationGroup) {
// 	// Implementation for max function
// 	group.Results["max"] = 0
// }
