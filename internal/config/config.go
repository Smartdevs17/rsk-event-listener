// File: internal/config/config.go
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	App           AppConfig          `mapstructure:"app"`
	RSK           RSKConfig          `mapstructure:"rsk"`
	Storage       StorageConfig      `mapstructure:"storage"`
	Monitor       MonitorConfig      `mapstructure:"monitor"`
	Processor     ProcessorConfig    `mapstructure:"processor"`
	Notifications NotificationConfig `mapstructure:"notifications"`
	Server        ServerConfig       `mapstructure:"server"`
	Logging       LoggingConfig      `mapstructure:"logging"`
}

// AppConfig contains application-level configuration
type AppConfig struct {
	Name        string `mapstructure:"name"`
	Version     string `mapstructure:"version"`
	Environment string `mapstructure:"environment"`
	Debug       bool   `mapstructure:"debug"`
}

// RSKConfig contains RSK blockchain connection configuration
type RSKConfig struct {
	NodeURL        string        `mapstructure:"node_url"`
	NetworkID      int           `mapstructure:"network_id"`
	BackupNodes    []string      `mapstructure:"backup_nodes"`
	RequestTimeout time.Duration `mapstructure:"request_timeout"`
	RetryAttempts  int           `mapstructure:"retry_attempts"`
	RetryDelay     time.Duration `mapstructure:"retry_delay"`
	MaxConnections int           `mapstructure:"max_connections"`
}

// StorageConfig contains database configuration
type StorageConfig struct {
	Type             string        `mapstructure:"type"` // sqlite, postgres
	ConnectionString string        `mapstructure:"connection_string"`
	MaxConnections   int           `mapstructure:"max_connections"`
	MaxIdleTime      time.Duration `mapstructure:"max_idle_time"`
	MigrationsPath   string        `mapstructure:"migrations_path"`
}

// MonitorConfig contains event monitoring configuration
type MonitorConfig struct {
	PollInterval       time.Duration `mapstructure:"poll_interval"`
	BatchSize          int           `mapstructure:"batch_size"`
	ConfirmationBlocks int           `mapstructure:"confirmation_blocks"`
	StartBlock         uint64        `mapstructure:"start_block"`
	EnableWebSocket    bool          `mapstructure:"enable_websocket"`
	MaxReorgDepth      int           `mapstructure:"max_reorg_depth"`
}

// ProcessorConfig contains event processing configuration
type ProcessorConfig struct {
	Workers                 int           `mapstructure:"workers"`
	QueueSize               int           `mapstructure:"queue_size"`
	ProcessTimeout          time.Duration `mapstructure:"process_timeout"`
	RetryAttempts           int           `mapstructure:"retry_attempts"`
	RetryDelay              time.Duration `mapstructure:"retry_delay"`
	EnableAsync             bool          `mapstructure:"enable_async"`
	MaxConcurrentProcessing int           `mapstructure:"max_concurrent_processing"`
	ProcessingTimeout       time.Duration `mapstructure:"processing_timeout"`
	EnableAggregation       bool          `mapstructure:"enable_aggregation"`
	AggregationWindow       time.Duration `mapstructure:"aggregation_window"`
	EnableValidation        bool          `mapstructure:"enable_validation"`
	BufferSize              int           `mapstructure:"buffer_size"`
}

// NotificationConfig contains notification system configuration
type NotificationConfig struct {
	Enabled                    bool          `mapstructure:"enabled"`
	QueueSize                  int           `mapstructure:"queue_size"`
	Workers                    int           `mapstructure:"workers"`
	RetryDelay                 time.Duration `mapstructure:"retry_delay"`
	MaxRetries                 int           `mapstructure:"max_retries"`
	DefaultChannel             string        `mapstructure:"default_channel"`
	MaxConcurrentNotifications int           `mapstructure:"max_concurrent_notifications"`
	NotificationTimeout        time.Duration `mapstructure:"notification_timeout"`
	RetryAttempts              int           `mapstructure:"retry_attempts"`
	EnableEmailNotifications   bool          `mapstructure:"enable_email_notifications"`
	EnableWebhookNotifications bool          `mapstructure:"enable_webhook_notifications"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Port          int           `mapstructure:"port"`
	Host          string        `mapstructure:"host"`
	ReadTimeout   time.Duration `mapstructure:"read_timeout"`
	WriteTimeout  time.Duration `mapstructure:"write_timeout"`
	EnableMetrics bool          `mapstructure:"enable_metrics"`
	EnableHealth  bool          `mapstructure:"enable_health"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"` // json, text
	Output     string `mapstructure:"output"` // stdout, file
	File       string `mapstructure:"file"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
}

// Load loads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	viper.SetConfigType("yaml")

	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.SetConfigName("config")
		viper.AddConfigPath(".")
		viper.AddConfigPath("./internal/config")
	}

	// Set environment variable prefix
	viper.SetEnvPrefix("RSK_LISTENER")
	viper.AutomaticEnv()

	// Set defaults
	setDefaults()

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("Config file not found, using defaults and environment variables")
		} else {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Override with environment variables if present
	if nodeURL := os.Getenv("RSK_NODE_URL"); nodeURL != "" {
		config.RSK.NodeURL = nodeURL
	}
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		config.Storage.ConnectionString = dbURL
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// App defaults
	viper.SetDefault("app.name", "rsk-event-listener")
	viper.SetDefault("app.version", "1.0.0")
	viper.SetDefault("app.environment", "development")
	viper.SetDefault("app.debug", false)

	// RSK defaults
	viper.SetDefault("rsk.node_url", "https://public-node.testnet.rsk.co")
	viper.SetDefault("rsk.network_id", 31) // RSK Testnet
	viper.SetDefault("rsk.request_timeout", "30s")
	viper.SetDefault("rsk.retry_attempts", 3)
	viper.SetDefault("rsk.retry_delay", "5s")
	viper.SetDefault("rsk.max_connections", 10)

	// Storage defaults
	viper.SetDefault("storage.type", "sqlite")
	viper.SetDefault("storage.connection_string", "./data/events.db")
	viper.SetDefault("storage.max_connections", 25)
	viper.SetDefault("storage.max_idle_time", "15m")
	viper.SetDefault("storage.migrations_path", "./internal/storage/migrations")

	// Monitor defaults (RSK block time is ~30 seconds)
	viper.SetDefault("monitor.poll_interval", "15s")
	viper.SetDefault("monitor.batch_size", 100)
	viper.SetDefault("monitor.confirmation_blocks", 12)
	viper.SetDefault("monitor.start_block", 0)
	viper.SetDefault("monitor.enable_websocket", false)
	viper.SetDefault("monitor.max_reorg_depth", 64)

	// Processor defaults
	viper.SetDefault("processor.workers", 4)
	viper.SetDefault("processor.queue_size", 1000)
	viper.SetDefault("processor.process_timeout", "30s")
	viper.SetDefault("processor.retry_attempts", 3)
	viper.SetDefault("processor.retry_delay", "5s")
	viper.SetDefault("processor.enable_async", true)

	// Notification defaults
	viper.SetDefault("notifications.enabled", true)
	viper.SetDefault("notifications.queue_size", 100)
	viper.SetDefault("notifications.workers", 2)
	viper.SetDefault("notifications.retry_delay", "10s")
	viper.SetDefault("notifications.max_retries", 3)
	viper.SetDefault("notifications.default_channel", "log")

	// Server defaults
	viper.SetDefault("server.port", 8081)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.read_timeout", "10s")
	viper.SetDefault("server.write_timeout", "10s")
	viper.SetDefault("server.enable_metrics", true)
	viper.SetDefault("server.enable_health", true)

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output", "stdout")
	viper.SetDefault("logging.max_size", 100)
	viper.SetDefault("logging.max_backups", 3)
	viper.SetDefault("logging.max_age", 28)
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.RSK.NodeURL == "" {
		return fmt.Errorf("RSK node URL is required")
	}
	if c.Storage.ConnectionString == "" {
		return fmt.Errorf("storage connection string is required")
	}
	if c.Monitor.PollInterval <= 0 {
		return fmt.Errorf("monitor poll interval must be positive")
	}
	if c.Processor.Workers <= 0 {
		return fmt.Errorf("processor workers must be positive")
	}
	return nil
}
