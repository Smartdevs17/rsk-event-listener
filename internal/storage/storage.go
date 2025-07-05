// File: internal/storage/storage.go
package storage

import (
	"context"
	"time"

	"github.com/smartdevs17/rsk-event-listener/internal/models"
)

// Storage defines the interface for event storage operations
type Storage interface {
	// Connection management
	Connect() error
	Close() error
	Ping() error
	Migrate() error

	// Event operations
	SaveEvent(ctx context.Context, event *models.Event) error
	SaveEvents(ctx context.Context, events []*models.Event) error
	GetEvent(ctx context.Context, id string) (*models.Event, error)
	GetEvents(ctx context.Context, filter models.EventFilter) ([]*models.Event, error)
	GetEventCount(ctx context.Context, filter models.EventFilter) (int64, error)
	UpdateEvent(ctx context.Context, event *models.Event) error
	DeleteEvent(ctx context.Context, id string) error

	// Contract operations
	SaveContract(ctx context.Context, contract *models.Contract) error
	GetContract(ctx context.Context, address string) (*models.Contract, error)
	GetContracts(ctx context.Context, active *bool) ([]*models.Contract, error)
	UpdateContract(ctx context.Context, contract *models.Contract) error
	DeleteContract(ctx context.Context, address string) error

	// Block tracking operations
	GetLatestProcessedBlock() (uint64, error)
	SetLatestProcessedBlock(blockNumber uint64) error
	GetBlockProcessingStatus(blockNumber uint64) (*BlockProcessingStatus, error)
	SetBlockProcessingStatus(status *BlockProcessingStatus) error

	// Notification operations
	SaveNotification(ctx context.Context, notification *models.Notification) error
	GetPendingNotifications(ctx context.Context, limit int) ([]*models.Notification, error)
	UpdateNotificationStatus(ctx context.Context, id string, status string, error *string) error

	// Statistics and monitoring
	GetStorageStats() (*StorageStats, error)
	GetEventStats(ctx context.Context, fromTime, toTime time.Time) (*EventStats, error)

	// Maintenance operations
	Cleanup(ctx context.Context, retentionDays int) error
	Vacuum() error

	// --- Custom methods for reorg and logs ---
	GetBlockHash(blockNumber uint64) (string, error)
	DeleteEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) (int, error)
	LogEvent(ctx context.Context, eventType string, data map[string]interface{}) error
	GetLogsByType(ctx context.Context, eventType string, limit int) ([]*models.LogEntry, error)
	GetAllContracts(ctx context.Context) ([]*models.Contract, error)
	// Custom processing result storage
	StoreProcessingResult(ctx context.Context, data map[string]interface{}) error
}

// BlockProcessingStatus tracks block processing status
type BlockProcessingStatus struct {
	BlockNumber    uint64         `json:"block_number" db:"block_number"`
	Status         string         `json:"status" db:"status"` // pending, processing, completed, failed
	StartedAt      time.Time      `json:"started_at" db:"started_at"`
	CompletedAt    *time.Time     `json:"completed_at,omitempty" db:"completed_at"`
	EventsFound    int            `json:"events_found" db:"events_found"`
	EventsSaved    int            `json:"events_saved" db:"events_saved"`
	Error          *string        `json:"error,omitempty" db:"error"`
	ProcessingTime *time.Duration `json:"processing_time,omitempty" db:"processing_time"`
}

// StorageStats provides storage statistics
type StorageStats struct {
	TotalEvents        int64      `json:"total_events"`
	TotalContracts     int64      `json:"total_contracts"`
	TotalNotifications int64      `json:"total_notifications"`
	OldestEvent        *time.Time `json:"oldest_event,omitempty"`
	LatestEvent        *time.Time `json:"latest_event,omitempty"`
	DatabaseSize       int64      `json:"database_size_bytes"`
	LastCleanup        *time.Time `json:"last_cleanup,omitempty"`
	LatestBlock        uint64     `json:"latest_processed_block"`
}

// EventStats provides event statistics for a time period
type EventStats struct {
	TimeRange        TimeRange        `json:"time_range"`
	TotalEvents      int64            `json:"total_events"`
	EventsByType     map[string]int64 `json:"events_by_type"`
	EventsByContract map[string]int64 `json:"events_by_contract"`
	EventsByHour     map[string]int64 `json:"events_by_hour"`
	ProcessingStats  ProcessingStats  `json:"processing_stats"`
}

// TimeRange represents a time range
type TimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// ProcessingStats provides processing statistics
type ProcessingStats struct {
	TotalProcessed        int64         `json:"total_processed"`
	TotalFailed           int64         `json:"total_failed"`
	AverageProcessingTime time.Duration `json:"average_processing_time"`
	LastProcessedAt       *time.Time    `json:"last_processed_at,omitempty"`
}

// StorageConfig holds storage configuration
type StorageConfig struct {
	Type             string        `json:"type"`
	ConnectionString string        `json:"connection_string"`
	MaxConnections   int           `json:"max_connections"`
	MaxIdleTime      time.Duration `json:"max_idle_time"`
	MigrationsPath   string        `json:"migrations_path"`
	RetentionDays    int           `json:"retention_days"`
	BatchSize        int           `json:"batch_size"`
}
