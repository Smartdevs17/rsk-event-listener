package storage

import (
	"context"
	"time"

	"github.com/smartdevs17/rsk-event-listener/internal/metrics"
	"github.com/smartdevs17/rsk-event-listener/internal/models"
)

// StorageWithMetrics wraps a storage implementation with metrics
type StorageWithMetrics struct {
	Storage
	metricsManager *metrics.Manager
}

// NewStorageWithMetrics creates a storage wrapper with metrics
func NewStorageWithMetrics(storage Storage, metricsManager *metrics.Manager) *StorageWithMetrics {
	return &StorageWithMetrics{
		Storage:        storage,
		metricsManager: metricsManager,
	}
}

// SaveEvent saves an event and records metrics
func (s *StorageWithMetrics) SaveEvent(ctx context.Context, event *models.Event) error {
	start := time.Now()

	err := s.Storage.SaveEvent(ctx, event)

	if s.metricsManager != nil {
		status := "success"
		if err != nil {
			status = "error"
		}

		s.metricsManager.GetPrometheusMetrics().RecordDatabaseOperation(
			"insert",
			"events",
			status,
			time.Since(start),
		)
	}

	return err
}

// SaveContract saves a contract and records metrics
func (s *StorageWithMetrics) SaveContract(ctx context.Context, contract *models.Contract) error {
	start := time.Now()

	err := s.Storage.SaveContract(ctx, contract)

	if s.metricsManager != nil {
		status := "success"
		if err != nil {
			status = "error"
		}

		s.metricsManager.GetPrometheusMetrics().RecordDatabaseOperation(
			"upsert",
			"contracts",
			status,
			time.Since(start),
		)
	}

	return err
}
