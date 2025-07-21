package config

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Config represents the application configuration structure
type Config struct {
	Server struct {
		EnableMetrics bool   `mapstructure:"enable_metrics"`
		EnableHealth  bool   `mapstructure:"enable_health"`
		Port          int    `mapstructure:"port"`
		Host          string `mapstructure:"host"`
	} `mapstructure:"server"`
}

// Load loads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	// Set up viper to read from config file
	viper.SetConfigFile(configPath)

	// Enable automatic environment variable support
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Explicitly bind environment variables that might override config
	viper.BindEnv("server.enable_metrics", "ENABLE_METRICS", "RSK_SERVER_ENABLE_METRICS")
	viper.BindEnv("server.enable_health", "ENABLE_HEALTH", "RSK_SERVER_ENABLE_HEALTH")
	viper.BindEnv("server.port", "SERVER_PORT", "RSK_SERVER_PORT")
	viper.BindEnv("server.host", "SERVER_HOST", "RSK_SERVER_HOST")

	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Log the effective metrics setting for debugging
	logrus.WithFields(logrus.Fields{
		"config_file_metrics":   viper.GetBool("server.enable_metrics"),
		"env_enable_metrics":    viper.GetString("ENABLE_METRICS"),
		"final_metrics_enabled": config.Server.EnableMetrics,
	}).Debug("Metrics configuration loaded")

	return &config, nil
}
