// File: internal/processor/router.go
package processor

import (
	"strings"

	"github.com/smartdevs17/rsk-event-listener/internal/models"
)

// EventRouter handles routing events to appropriate rules and workflows
type EventRouter struct {
	config *ProcessorConfig
}

// RoutingResult contains the result of event routing
type RoutingResult struct {
	ApplicableRules     []string `json:"applicable_rules"`
	ApplicableWorkflows []string `json:"applicable_workflows"`
	Transformations     []string `json:"transformations"`
}

// NewEventRouter creates a new event router
func NewEventRouter(config *ProcessorConfig) *EventRouter {
	return &EventRouter{
		config: config,
	}
}

// RouteEvent routes an event to applicable rules and workflows
func (er *EventRouter) RouteEvent(event *models.Event, rules map[string]*ProcessingRule, workflows map[string]*Workflow) *RoutingResult {
	result := &RoutingResult{
		ApplicableRules:     []string{},
		ApplicableWorkflows: []string{},
		Transformations:     []string{},
	}

	// Check rules
	for _, rule := range rules {
		if er.matchesRuleConditions(event, rule.Conditions) {
			result.ApplicableRules = append(result.ApplicableRules, rule.ID)
		}
	}

	// Check workflows
	for _, workflow := range workflows {
		if er.matchesWorkflowConditions(event, workflow) {
			result.ApplicableWorkflows = append(result.ApplicableWorkflows, workflow.ID)
		}
	}

	return result
}

// matchesRuleConditions checks if an event matches rule conditions
func (er *EventRouter) matchesRuleConditions(event *models.Event, conditions *RuleConditions) bool {
	if conditions == nil {
		return true
	}

	// Check contract addresses
	if len(conditions.ContractAddresses) > 0 {
		found := false
		for _, addr := range conditions.ContractAddresses {
			if strings.EqualFold(event.Address, addr) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check event names
	if len(conditions.EventNames) > 0 {
		found := false
		for _, name := range conditions.EventNames {
			if strings.EqualFold(event.EventName, name) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check event signatures
	if len(conditions.EventSignatures) > 0 {
		found := false
		for _, sig := range conditions.EventSignatures {
			if event.EventSig == sig {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check block range
	if conditions.BlockRange != nil {
		if conditions.BlockRange.FromBlock != nil && event.BlockNumber < *conditions.BlockRange.FromBlock {
			return false
		}
		if conditions.BlockRange.ToBlock != nil && event.BlockNumber > *conditions.BlockRange.ToBlock {
			return false
		}
	}

	// Check data conditions
	if len(conditions.DataConditions) > 0 {
		for field, expectedValue := range conditions.DataConditions {
			if !er.checkDataCondition(event.Data, field, expectedValue) {
				return false
			}
		}
	}

	return true
}

// matchesWorkflowConditions checks if an event matches workflow conditions
func (er *EventRouter) matchesWorkflowConditions(event *models.Event, workflow *Workflow) bool {
	// Simple workflow matching - can be extended with more complex logic
	return workflow.Enabled
}

// checkDataCondition checks if event data matches a specific condition
func (er *EventRouter) checkDataCondition(eventData map[string]interface{}, field string, expectedValue interface{}) bool {
	actualValue, exists := eventData[field]
	if !exists {
		return false
	}

	// Simple equality check - can be extended with operators (>, <, !=, etc.)
	return actualValue == expectedValue
}
