// File: internal/storage/factory.go
package storage

import (
	"strings"

	"github.com/smartdevs17/rsk-event-listener/internal/config"
	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
)

// NewStorage creates a new storage instance based on configuration
func NewStorage(cfg *config.StorageConfig) (Storage, error) {
	storageConfig := &StorageConfig{
		Type:             cfg.Type,
		ConnectionString: cfg.ConnectionString,
		MaxConnections:   cfg.MaxConnections,
		MaxIdleTime:      cfg.MaxIdleTime,
		MigrationsPath:   cfg.MigrationsPath,
		RetentionDays:    30,  // Default retention
		BatchSize:        100, // Default batch size
	}

	switch strings.ToLower(cfg.Type) {
	case "sqlite":
		return NewSQLiteStorage(storageConfig), nil
	case "postgres", "postgresql":
		return NewPostgreSQLStorage(storageConfig), nil
	default:
		return nil, utils.NewAppError(utils.ErrCodeConfiguration,
			"Unsupported storage type", cfg.Type)
	}
}

// ValidateStorageConfig validates storage configuration
func ValidateStorageConfig(cfg *config.StorageConfig) error {
	if cfg.Type == "" {
		return utils.NewAppError(utils.ErrCodeConfiguration, "Storage type is required", "")
	}

	if cfg.ConnectionString == "" {
		return utils.NewAppError(utils.ErrCodeConfiguration, "Storage connection string is required", "")
	}

	if cfg.MaxConnections <= 0 {
		return utils.NewAppError(utils.ErrCodeConfiguration, "Max connections must be positive", "")
	}

	supportedTypes := []string{"sqlite", "postgres", "postgresql"}
	supported := false
	for _, t := range supportedTypes {
		if strings.ToLower(cfg.Type) == t {
			supported = true
			break
		}
	}

	if !supported {
		return utils.NewAppError(utils.ErrCodeConfiguration,
			"Unsupported storage type",
			"Supported types: "+strings.Join(supportedTypes, ", "))
	}

	return nil
}

// GetDefaultStorageConfig returns default storage configuration
func GetDefaultStorageConfig() *StorageConfig {
	return &StorageConfig{
		Type:             "sqlite",
		ConnectionString: "./data/events.db",
		MaxConnections:   25,
		RetentionDays:    30,
		BatchSize:        100,
	}
}
