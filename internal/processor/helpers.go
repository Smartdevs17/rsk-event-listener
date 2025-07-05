// File: internal/processor/helpers.go
package processor

import (
	"context"
	"time"
)

// loadRulesAndWorkflows loads rules and workflows from storage
func (ep *EventProcessor) loadRulesAndWorkflows(ctx context.Context) error {
	// Load rules - using a placeholder method for now
	rules, err := ep.loadProcessingRules(ctx)
	if err != nil {
		return err
	}

	for _, rule := range rules {
		ep.rules[rule.ID] = rule
	}

	// Load workflows - using a placeholder method for now
	workflows, err := ep.loadWorkflows(ctx)
	if err != nil {
		return err
	}

	for _, workflow := range workflows {
		ep.workflows[workflow.ID] = workflow
	}

	ep.logger.Info("Loaded processing configuration",
		"rules", len(ep.rules),
		"workflows", len(ep.workflows))

	return nil
}

// loadProcessingRules loads processing rules from storage
func (ep *EventProcessor) loadProcessingRules(ctx context.Context) ([]*ProcessingRule, error) {
	// Since we don't have GetProcessingRules in storage interface yet,
	// return empty slice for now - this can be implemented when storage interface is extended
	return []*ProcessingRule{}, nil
}

// loadWorkflows loads workflows from storage
func (ep *EventProcessor) loadWorkflows(ctx context.Context) ([]*Workflow, error) {
	// Since we don't have GetWorkflows in storage interface yet,
	// return empty slice for now - this can be implemented when storage interface is extended
	return []*Workflow{}, nil
}

// updateStats updates processor statistics
func (ep *EventProcessor) updateStats(result *ProcessResult) {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	ep.stats.TotalEventsProcessed++
	ep.stats.TotalActionsExecuted += uint64(len(result.ActionsExecuted))

	if result.Error != nil {
		ep.stats.ErrorCount++
		errorStr := result.Error.Error()
		ep.stats.LastError = &errorStr
		now := time.Now()
		ep.stats.LastErrorTime = &now
	}

	// Calculate processing rate
	if ep.stats.TotalEventsProcessed > 0 {
		ep.stats.ProcessingRate = float64(ep.stats.TotalEventsProcessed) /
			time.Since(ep.stats.StartTime).Seconds()
	}

	// Update average processing time
	if ep.stats.TotalEventsProcessed == 1 {
		ep.stats.AverageProcessingTime = result.ProcessingTime
	} else {
		ep.stats.AverageProcessingTime = (ep.stats.AverageProcessingTime + result.ProcessingTime) / 2
	}
}

// Rule and Workflow management methods

// AddRule adds a processing rule
func (ep *EventProcessor) AddRule(rule *ProcessingRule) error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	// Save to storage (placeholder implementation)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := ep.saveProcessingRule(ctx, rule); err != nil {
		return err
	}

	// Add to memory
	ep.rules[rule.ID] = rule

	ep.logger.Info("Processing rule added", "rule_id", rule.ID, "rule_name", rule.Name)
	return nil
}

// RemoveRule removes a processing rule
func (ep *EventProcessor) RemoveRule(ruleID string) error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	// Remove from storage (placeholder implementation)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := ep.deleteProcessingRule(ctx, ruleID); err != nil {
		return err
	}

	// Remove from memory
	delete(ep.rules, ruleID)

	ep.logger.Info("Processing rule removed", "rule_id", ruleID)
	return nil
}

// GetRules returns all processing rules
func (ep *EventProcessor) GetRules() []*ProcessingRule {
	ep.mu.RLock()
	defer ep.mu.RUnlock()

	rules := make([]*ProcessingRule, 0, len(ep.rules))
	for _, rule := range ep.rules {
		rules = append(rules, rule)
	}
	return rules
}

// UpdateRule updates a processing rule
func (ep *EventProcessor) UpdateRule(rule *ProcessingRule) error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	// Update in storage (placeholder implementation)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rule.UpdatedAt = time.Now()
	if err := ep.updateProcessingRule(ctx, rule); err != nil {
		return err
	}

	// Update in memory
	ep.rules[rule.ID] = rule

	ep.logger.Info("Processing rule updated", "rule_id", rule.ID, "rule_name", rule.Name)
	return nil
}

// AddWorkflow adds a workflow
func (ep *EventProcessor) AddWorkflow(workflow *Workflow) error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	// Save to storage (placeholder implementation)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := ep.saveWorkflow(ctx, workflow); err != nil {
		return err
	}

	// Add to memory
	ep.workflows[workflow.ID] = workflow

	ep.logger.Info("Workflow added", "workflow_id", workflow.ID, "workflow_name", workflow.Name)
	return nil
}

// RemoveWorkflow removes a workflow
func (ep *EventProcessor) RemoveWorkflow(workflowID string) error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	// Remove from storage (placeholder implementation)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := ep.deleteWorkflow(ctx, workflowID); err != nil {
		return err
	}

	// Remove from memory
	delete(ep.workflows, workflowID)

	ep.logger.Info("Workflow removed", "workflow_id", workflowID)
	return nil
}

// GetWorkflows returns all workflows
func (ep *EventProcessor) GetWorkflows() []*Workflow {
	ep.mu.RLock()
	defer ep.mu.RUnlock()

	workflows := make([]*Workflow, 0, len(ep.workflows))
	for _, workflow := range ep.workflows {
		workflows = append(workflows, workflow)
	}
	return workflows
}

// GetStats returns processor statistics
func (ep *EventProcessor) GetStats() *ProcessorStats {
	ep.mu.RLock()
	defer ep.mu.RUnlock()

	ep.stats.Uptime = time.Since(ep.stats.StartTime)
	ep.stats.ActiveRules = len(ep.rules)
	ep.stats.ActiveWorkflows = len(ep.workflows)

	return ep.stats
}

// GetHealth returns processor health
func (ep *EventProcessor) GetHealth() *ProcessorHealth {
	health := &ProcessorHealth{
		Healthy: true,
		Issues:  []string{},
	}

	// Check storage health
	_, err := ep.storage.GetLatestProcessedBlock()
	health.StorageHealthy = err == nil
	if err != nil {
		health.Healthy = false
		health.Issues = append(health.Issues, "Storage unhealthy: "+err.Error())
	}

	// Check notification health
	health.NotificationHealthy = ep.notifier.IsHealthy()
	if !health.NotificationHealthy {
		health.Healthy = false
		health.Issues = append(health.Issues, "Notification system unhealthy")
	}

	return health
}

// Placeholder storage methods - these would be implemented when storage interface is extended

// saveProcessingRule saves a processing rule to storage
func (ep *EventProcessor) saveProcessingRule(ctx context.Context, rule *ProcessingRule) error {
	// This would call storage.SaveProcessingRule when implemented
	// For now, just log the operation
	ep.logger.Debug("Saving processing rule", "rule_id", rule.ID)
	return nil
}

// deleteProcessingRule deletes a processing rule from storage
func (ep *EventProcessor) deleteProcessingRule(ctx context.Context, ruleID string) error {
	// This would call storage.DeleteProcessingRule when implemented
	// For now, just log the operation
	ep.logger.Debug("Deleting processing rule", "rule_id", ruleID)
	return nil
}

// updateProcessingRule updates a processing rule in storage
func (ep *EventProcessor) updateProcessingRule(ctx context.Context, rule *ProcessingRule) error {
	// This would call storage.UpdateProcessingRule when implemented
	// For now, just log the operation
	ep.logger.Debug("Updating processing rule", "rule_id", rule.ID)
	return nil
}

// saveWorkflow saves a workflow to storage
func (ep *EventProcessor) saveWorkflow(ctx context.Context, workflow *Workflow) error {
	// This would call storage.SaveWorkflow when implemented
	// For now, just log the operation
	ep.logger.Debug("Saving workflow", "workflow_id", workflow.ID)
	return nil
}

// deleteWorkflow deletes a workflow from storage
func (ep *EventProcessor) deleteWorkflow(ctx context.Context, workflowID string) error {
	// This would call storage.DeleteWorkflow when implemented
	// For now, just log the operation
	ep.logger.Debug("Deleting workflow", "workflow_id", workflowID)
	return nil
}
