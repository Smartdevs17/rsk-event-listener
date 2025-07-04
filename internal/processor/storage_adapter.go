// File: internal/processor/storage_adapter.go
package processor

import (
	"context"
	"encoding/json"
	"time"
)

// This file provides adapter methods for storage operations that aren't yet in the main storage interface
// These methods handle the conversion between processor types and storage operations

// StoreProcessingResult stores processing result data
func (ep *EventProcessor) StoreProcessingResult(ctx context.Context, data map[string]interface{}) error {
	// Convert processing result data to a format that can be stored
	// For now, we can store it as metadata or in a generic storage method

	// Convert to JSON for storage
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Use the LogEvent method if available, or extend storage interface
	// This is a placeholder implementation
	ep.logger.Debug("Storing processing result", "data_size", len(jsonData))

	// If storage interface has a LogEvent method, use it:
	// return ep.storage.LogEvent(ctx, "processing_result", data)

	// For now, just log the operation
	return nil
}

// GetProcessingResults retrieves processing results
func (ep *EventProcessor) GetProcessingResults(ctx context.Context, eventID string) (map[string]interface{}, error) {
	// This would query processing results from storage
	// Placeholder implementation
	return map[string]interface{}{}, nil
}

// CleanupOldProcessingResults removes old processing results
func (ep *EventProcessor) CleanupOldProcessingResults(ctx context.Context, olderThan time.Duration) error {
	// This would cleanup old processing results
	// Placeholder implementation
	ep.logger.Debug("Cleaning up old processing results", "older_than", olderThan)
	return nil
}
