// File: cmd/listener/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/smartdevs17/rsk-event-listener/internal/config"
	"github.com/smartdevs17/rsk-event-listener/internal/connection"
	"github.com/smartdevs17/rsk-event-listener/internal/metrics"
	"github.com/smartdevs17/rsk-event-listener/internal/monitor"
	"github.com/smartdevs17/rsk-event-listener/internal/notification"
	"github.com/smartdevs17/rsk-event-listener/internal/processor"
	"github.com/smartdevs17/rsk-event-listener/internal/server"
	"github.com/smartdevs17/rsk-event-listener/internal/storage"
	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
)

// AppVersion contains the application version
const AppVersion = "1.0.0"

// Application represents the main application
type Application struct {
	config       *config.Config
	logger       *logrus.Logger
	connection   *connection.ConnectionManager
	storage      storage.Storage
	monitor      *monitor.EventMonitor
	processor    *processor.EventProcessor
	notification *notification.NotificationManager
	server       *server.HTTPServer
	ctx          context.Context
	cancel       context.CancelFunc
	startTime    time.Time

	metricsManager *metrics.Manager
}

// NewApplication creates a new application instance
func NewApplication(cfg *config.Config) (*Application, error) {
	ctx, cancel := context.WithCancel(context.Background())

	app := &Application{
		config:    cfg,
		ctx:       ctx,
		cancel:    cancel,
		startTime: time.Now(),
	}

	// Initialize logger
	if err := app.initializeLogger(); err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Initialize components
	if err := app.initializeComponents(); err != nil {
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	return app, nil
}

// initializeLogger initializes the application logger
func (app *Application) initializeLogger() error {
	logCfg := app.config.Logging

	if err := utils.InitLogger(logCfg.Level, logCfg.Format, logCfg.Output, logCfg.File); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	app.logger = utils.GetLogger()
	app.logger.WithFields(logrus.Fields{
		"level":  logCfg.Level,
		"format": logCfg.Format,
		"output": logCfg.Output,
	}).Info("Logger initialized")

	return nil
}

// initializeComponents initializes all application components
func (app *Application) initializeComponents() error {
	app.logger.Info("Initializing application components")

	// Initialize metrics manager
	if err := app.initializeMetrics(); err != nil {
		return fmt.Errorf("failed to initialize metrics: %w", err)
	}

	// Initialize connection manager
	if err := app.initializeConnection(); err != nil {
		return fmt.Errorf("failed to initialize connection: %w", err)
	}

	// Initialize storage
	if err := app.initializeStorage(); err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Initialize notification manager
	if err := app.initializeNotification(); err != nil {
		return fmt.Errorf("failed to initialize notification: %w", err)
	}

	// Initialize event processor
	if err := app.initializeProcessor(); err != nil {
		return fmt.Errorf("failed to initialize processor: %w", err)
	}

	// Initialize event monitor
	if err := app.initializeMonitor(); err != nil {
		return fmt.Errorf("failed to initialize monitor: %w", err)
	}

	// Initialize HTTP server
	if err := app.initializeServer(); err != nil {
		return fmt.Errorf("failed to initialize server: %w", err)
	}

	app.logger.Info("All components initialized successfully")
	return nil
}

// initializeConnection initializes the connection manager
func (app *Application) initializeConnection() error {
	app.logger.Info("Initializing connection manager")

	// Use the actual RSKConfig structure
	app.connection = connection.NewConnectionManager(&app.config.RSK)

	// Test connection with health check
	if err := app.connection.HealthCheck(); err != nil {
		return fmt.Errorf("failed to connect to RSK node: %w", err)
	}

	app.logger.Info("Connection manager initialized successfully")
	return nil
}

// initializeStorage initializes the storage layer
func (app *Application) initializeStorage() error {
	app.logger.Info("Initializing storage layer")

	var err error
	app.storage, err = storage.NewStorage(&app.config.Storage)
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}

	// Connect to storage
	if err := app.storage.Connect(); err != nil {
		return fmt.Errorf("failed to connect to storage: %w", err)
	}

	// Run migrations
	if err := app.storage.Migrate(); err != nil {
		return fmt.Errorf("failed to run storage migrations: %w", err)
	}

	app.logger.Info("Storage layer initialized successfully")
	return nil
}

// initializeNotification initializes the notification manager
func (app *Application) initializeNotification() error {
	app.logger.Info("Initializing notification manager")

	notificationCfg := &notification.NotificationManagerConfig{
		MaxConcurrentNotifications: app.config.Notifications.MaxConcurrentNotifications,
		NotificationTimeout:        app.config.Notifications.NotificationTimeout,
		RetryAttempts:              app.config.Notifications.RetryAttempts,
		RetryDelay:                 app.config.Notifications.RetryDelay,
		EnableEmailNotifications:   app.config.Notifications.EnableEmailNotifications,
		EnableWebhookNotifications: app.config.Notifications.EnableWebhookNotifications,
		QueueSize:                  app.config.Notifications.QueueSize,
		LogLevel:                   app.config.Logging.Level,
	}

	app.notification = notification.NewNotificationManager(notificationCfg)

	// Start notification manager
	if err := app.notification.Start(app.ctx); err != nil {
		return fmt.Errorf("failed to start notification manager: %w", err)
	}

	app.logger.Info("Notification manager initialized successfully")
	return nil
}

// initializeProcessor initializes the event processor
func (app *Application) initializeProcessor() error {
	app.logger.Info("Initializing event processor")

	processorCfg := &processor.ProcessorConfig{
		MaxConcurrentProcessing: app.config.Processor.MaxConcurrentProcessing,
		ProcessingTimeout:       app.config.Processor.ProcessingTimeout,
		RetryAttempts:           app.config.Processor.RetryAttempts,
		RetryDelay:              app.config.Processor.RetryDelay,
		EnableAggregation:       app.config.Processor.EnableAggregation,
		AggregationWindow:       app.config.Processor.AggregationWindow,
		EnableValidation:        app.config.Processor.EnableValidation,
		BufferSize:              app.config.Processor.BufferSize,
	}

	var err error
	app.processor = processor.NewEventProcessor(app.storage, app.notification, processorCfg)
	if err != nil {
		return fmt.Errorf("failed to create event processor: %w", err)
	}

	// Start processor
	if err := app.processor.Start(app.ctx); err != nil {
		return fmt.Errorf("failed to start event processor: %w", err)
	}

	app.logger.Info("Event processor initialized successfully")
	return nil
}

// initializeMonitor initializes the event monitor
func (app *Application) initializeMonitor() error {
	app.logger.Info("Initializing event monitor")

	// Use actual monitor configuration structure
	monitorCfg := &monitor.MonitorConfig{
		PollInterval:       app.config.Monitor.PollInterval,
		BatchSize:          app.config.Monitor.BatchSize,
		ConfirmationBlocks: app.config.Monitor.ConfirmationBlocks,
		StartBlock:         app.config.Monitor.StartBlock,
		EnableWebSocket:    app.config.Monitor.EnableWebSocket,
		MaxReorgDepth:      app.config.Monitor.MaxReorgDepth,
	}

	// Use actual constructor signature
	app.monitor = monitor.NewEventMonitor(app.connection, app.storage, monitorCfg)

	app.logger.Info("Event monitor initialized successfully")
	return nil
}

// initializeMetrics initializes the metrics manager if enabled
func (app *Application) initializeMetrics() error {
	if !app.config.Server.EnableMetrics {
		app.logger.Info("Metrics disabled in configuration")
		app.metricsManager = nil
		return nil
	}

	app.logger.Info("Initializing metrics manager")
	app.metricsManager = metrics.NewManager()
	app.logger.Info("Metrics manager initialized successfully")
	return nil
}

// initializeServer initializes the HTTP server
func (app *Application) initializeServer() error {
	app.logger.Info("Initializing HTTP server")

	serverCfg := &server.ServerConfig{
		Port:          app.config.Server.Port,
		Host:          app.config.Server.Host,
		ReadTimeout:   app.config.Server.ReadTimeout,
		WriteTimeout:  app.config.Server.WriteTimeout,
		EnableMetrics: app.config.Server.EnableMetrics,
		EnableHealth:  app.config.Server.EnableHealth,
	}

	var err error
	app.server, err = server.NewHTTPServer(serverCfg, app.storage, app.monitor, app.processor, app.notification, app.metricsManager)
	if err != nil {
		return fmt.Errorf("failed to create HTTP server: %w", err)
	}

	app.logger.Info("HTTP server initialized successfully")
	return nil
}

// Start starts the application
func (app *Application) Start() error {
	app.logger.WithFields(logrus.Fields{
		"version":     AppVersion,
		"environment": app.config.App.Environment,
	}).Info("Starting RSK Event Listener")

	// Start HTTP server
	if err := app.server.Start(); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	// Start event monitor
	if err := app.monitor.Start(app.ctx); err != nil {
		return fmt.Errorf("failed to start event monitor: %w", err)
	}

	app.logger.WithFields(logrus.Fields{
		"server_address": fmt.Sprintf("%s:%d", app.config.Server.Host, app.config.Server.Port),
		"rsk_endpoint":   app.config.RSK.NodeURL,
		"storage_type":   app.config.Storage.Type,
	}).Info("RSK Event Listener started successfully")

	return nil
}

// Stop stops the application gracefully
func (app *Application) Stop() error {
	app.logger.Info("Stopping RSK Event Listener")

	// Create a shutdown timeout context (separate from main context)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop components in reverse order with controlled shutdown
	if app.server != nil {
		if err := app.server.Stop(); err != nil {
			app.logger.WithError(err).Error("Failed to stop HTTP server")
		}
	}

	if app.monitor != nil {
		// Stop monitor gracefully with timeout
		done := make(chan error, 1)
		go func() {
			done <- app.monitor.Stop()
		}()

		select {
		case err := <-done:
			if err != nil {
				app.logger.WithError(err).Error("Failed to stop event monitor")
			}
		case <-shutdownCtx.Done():
			app.logger.Warn("Monitor shutdown timed out")
		}
	}

	// Stop other components...
	if app.processor != nil {
		if err := app.processor.Stop(); err != nil {
			app.logger.WithError(err).Error("Failed to stop event processor")
		}
	}

	if app.notification != nil {
		if err := app.notification.Stop(); err != nil {
			app.logger.WithError(err).Error("Failed to stop notification manager")
		}
	}

	if app.storage != nil {
		if err := app.storage.Close(); err != nil {
			app.logger.WithError(err).Error("Failed to close storage")
		}
	}

	if app.connection != nil {
		if err := app.connection.Close(); err != nil {
			app.logger.WithError(err).Error("Failed to close connection")
		}
	}

	// Finally cancel the main context
	app.cancel()

	app.logger.Info("RSK Event Listener stopped successfully")
	return nil
}

// GetStats returns application statistics
func (app *Application) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"version":   AppVersion,
		"uptime":    time.Since(app.startTime).String(),
		"timestamp": time.Now(),
	}

	if app.connection != nil {
		stats["connection"] = app.connection.Stats()
	}

	if app.storage != nil {
		if storageStats, err := app.storage.GetStats(); err == nil {
			stats["storage"] = storageStats
		}
	}

	if app.monitor != nil {
		stats["monitor"] = app.monitor.GetStats()
	}

	if app.processor != nil {
		stats["processor"] = app.processor.GetStats()
	}

	if app.notification != nil {
		stats["notification"] = app.notification.GetStats()
	}

	return stats
}

// GetHealth returns application health status
func (app *Application) GetHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   AppVersion,
		"uptime":    time.Since(app.startTime).String(),
	}

	components := make(map[string]bool)
	var allHealthy = true

	if app.connection != nil {
		components["connection"] = app.connection.IsConnected()
		if !components["connection"] {
			allHealthy = false
		}
	}

	if app.storage != nil {
		storageHealth := app.storage.GetHealth()
		components["storage"] = storageHealth.Healthy
		if !components["storage"] {
			allHealthy = false
		}
	}

	if app.monitor != nil {
		monitorHealth := app.monitor.GetHealth()
		components["monitor"] = monitorHealth.Healthy
		if !components["monitor"] {
			allHealthy = false
		}
	}

	if app.processor != nil {
		processorHealth := app.processor.GetHealth()
		components["processor"] = processorHealth.Healthy
		if !components["processor"] {
			allHealthy = false
		}
	}

	if app.notification != nil {
		notificationHealth := app.notification.GetHealth()
		components["notification"] = notificationHealth.Healthy
		if !components["notification"] {
			allHealthy = false
		}
	}

	health["components"] = components

	if !allHealthy {
		health["status"] = "unhealthy"
	}

	return health
}

// CLI Commands

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "rsk-event-listener",
	Short:   "RSK Smart Contract Event Listener",
	Long:    `A high-performance, production-ready event monitoring system for Rootstock (RSK) blockchain smart contracts.`,
	Version: AppVersion,
	RunE:    runListener,
}

// runListener is the main command to run the event listener
func runListener(cmd *cobra.Command, args []string) error {
	// Load configuration
	configPath := viper.GetString("config")
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create application
	app, err := NewApplication(cfg)
	if err != nil {
		return fmt.Errorf("failed to create application: %w", err)
	}

	// Set up signal handling for graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Start application
	if err := app.Start(); err != nil {
		return fmt.Errorf("failed to start application: %w", err)
	}

	// Wait for shutdown signal
	<-signalChan
	fmt.Println("\nReceived shutdown signal, stopping application...")

	// Stop application
	if err := app.Stop(); err != nil {
		return fmt.Errorf("failed to stop application: %w", err)
	}

	return nil
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("RSK Event Listener %s\n", AppVersion)
	},
}

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
}

// validateConfigCmd validates the configuration
var validateConfigCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := viper.GetString("config")
		cfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}

		fmt.Printf("Configuration is valid!\n")
		fmt.Printf("Environment: %s\n", cfg.App.Environment)
		fmt.Printf("RSK Node: %s\n", cfg.RSK.NodeURL)
		fmt.Printf("Database: %s\n", cfg.Storage.Type)

		return nil
	},
}

// testCmd represents the test command
var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test connectivity and configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := viper.GetString("config")
		cfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		fmt.Println("Testing RSK Event Listener connectivity...")

		// Initialize logger for testing
		if err := utils.InitLogger("info", "text", "stdout", ""); err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}

		// Test RSK connection
		fmt.Printf("Testing RSK connection to %s...\n", cfg.RSK.NodeURL)
		conn := connection.NewConnectionManager(&cfg.RSK)
		defer conn.Close()

		if err := conn.HealthCheck(); err != nil {
			return fmt.Errorf("failed to connect to RSK node: %w", err)
		}
		fmt.Println("✓ RSK connection successful")

		// Test storage
		fmt.Printf("Testing storage connection (%s)...\n", cfg.Storage.Type)
		store, err := storage.NewStorage(&cfg.Storage)
		if err != nil {
			return fmt.Errorf("failed to create storage: %w", err)
		}
		defer store.Close()

		if err := store.Connect(); err != nil {
			return fmt.Errorf("failed to connect to storage: %w", err)
		}
		fmt.Println("✓ Storage connection successful")

		// Test email (if configured)
		if cfg.Notifications.EnableEmailNotifications {
			fmt.Println("✓ Email notifications enabled")
		}

		fmt.Println("\nAll connectivity tests passed! ✓")
		return nil
	},
}

// init initializes the CLI commands
func init() {
	// Add persistent flags
	rootCmd.PersistentFlags().StringP("config", "c", "", "config file path")
	rootCmd.PersistentFlags().StringP("log-level", "l", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().Bool("debug", false, "enable debug mode")

	// Bind flags to viper
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(testCmd)
	configCmd.AddCommand(validateConfigCmd)
}

// main is the entry point
func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
