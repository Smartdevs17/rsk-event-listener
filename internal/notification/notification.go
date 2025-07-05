// File: internal/notification/notification.go
package notification

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
)

// Notifier defines the notification interface
type Notifier interface {
	// Lifecycle management
	Start(ctx context.Context) error
	Stop() error
	IsHealthy() bool

	// Notification methods
	SendNotification(ctx context.Context, config *NotificationConfig, data map[string]interface{}) (string, error)
	SendWebhook(ctx context.Context, config *WebhookConfig, data map[string]interface{}) error
	SendEmail(ctx context.Context, config *EmailConfig, data map[string]interface{}) error

	// Channel management
	AddChannel(channel *NotificationChannel) error
	RemoveChannel(channelID string) error
	GetChannels() []*NotificationChannel

	// Statistics
	GetStats() *NotificationStats
}

// NotificationManager implements the Notifier interface
type NotificationManager struct {
	config *NotificationManagerConfig
	logger *NotificationLogger

	mu       sync.RWMutex
	running  bool
	channels map[string]*NotificationChannel

	// Components
	emailSender   *EmailSender
	webhookSender *WebhookSender

	// Statistics
	stats *NotificationStats
}

// NotificationManagerConfig holds notification manager configuration
type NotificationManagerConfig struct {
	MaxConcurrentNotifications int           `json:"max_concurrent_notifications"`
	NotificationTimeout        time.Duration `json:"notification_timeout"`
	RetryAttempts              int           `json:"retry_attempts"`
	RetryDelay                 time.Duration `json:"retry_delay"`
	EnableEmailNotifications   bool          `json:"enable_email_notifications"`
	EnableWebhookNotifications bool          `json:"enable_webhook_notifications"`
	QueueSize                  int           `json:"queue_size"`
	LogLevel                   string        `json:"log_level"`
}

// NotificationConfig defines a notification configuration
type NotificationConfig struct {
	Type       string   `json:"type"`
	Recipients []string `json:"recipients"`
	Subject    string   `json:"subject"`
	Template   string   `json:"template"`
	Priority   string   `json:"priority"`
	ChannelID  string   `json:"channel_id,omitempty"`
}

// WebhookConfig defines webhook configuration
type WebhookConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Timeout time.Duration     `json:"timeout"`
}

// EmailConfig defines email configuration
type EmailConfig struct {
	To       []string `json:"to"`
	CC       []string `json:"cc,omitempty"`
	BCC      []string `json:"bcc,omitempty"`
	Subject  string   `json:"subject"`
	Template string   `json:"template"`
	Priority string   `json:"priority"`
}

// NotificationChannel defines a notification channel
type NotificationChannel struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Enabled   bool                   `json:"enabled"`
	Config    map[string]interface{} `json:"config"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// NotificationStats provides notification statistics
type NotificationStats struct {
	TotalNotificationsSent   uint64        `json:"total_notifications_sent"`
	TotalEmailsSent          uint64        `json:"total_emails_sent"`
	TotalWebhooksSent        uint64        `json:"total_webhooks_sent"`
	TotalNotificationsFailed uint64        `json:"total_notifications_failed"`
	AverageResponseTime      time.Duration `json:"average_response_time"`
	ActiveChannels           int           `json:"active_channels"`
	QueueLength              int           `json:"queue_length"`
	ErrorCount               uint64        `json:"error_count"`
	LastError                *string       `json:"last_error,omitempty"`
	LastErrorTime            *time.Time    `json:"last_error_time,omitempty"`
}

type NotificationHealth struct {
	Healthy bool   `json:"healthy"`
	Error   string `json:"error,omitempty"`
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager(config *NotificationManagerConfig) *NotificationManager {
	nm := &NotificationManager{
		config:   config,
		logger:   NewNotificationLogger(config.LogLevel),
		channels: make(map[string]*NotificationChannel),
		stats:    &NotificationStats{},
	}

	// Initialize components
	nm.emailSender = NewEmailSender(config, nm.logger)
	nm.webhookSender = NewWebhookSender(config, nm.logger)

	return nm
}

// Start starts the notification manager
func (nm *NotificationManager) Start(ctx context.Context) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if nm.running {
		return utils.NewAppError(utils.ErrCodeInternal, "Notification manager already running", "")
	}

	nm.logger.Info("Starting notification manager")
	nm.running = true

	// Start email sender if enabled
	if nm.config.EnableEmailNotifications {
		if err := nm.emailSender.Start(ctx); err != nil {
			nm.logger.Warn("Failed to start email sender", map[string]interface{}{"error": err})
		}
	}

	// Start webhook sender if enabled
	if nm.config.EnableWebhookNotifications {
		if err := nm.webhookSender.Start(ctx); err != nil {
			nm.logger.Warn("Failed to start webhook sender", map[string]interface{}{"error": err})
		}
	}

	nm.logger.Info("Notification manager started")
	return nil
}

// Stop stops the notification manager
func (nm *NotificationManager) Stop() error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if !nm.running {
		return nil
	}

	nm.logger.Info("Stopping notification manager")
	nm.running = false

	// Stop components
	if nm.emailSender != nil {
		nm.emailSender.Stop()
	}
	if nm.webhookSender != nil {
		nm.webhookSender.Stop()
	}

	nm.logger.Info("Notification manager stopped")
	return nil
}

// IsHealthy returns whether the notification manager is healthy
func (nm *NotificationManager) IsHealthy() bool {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	return nm.running
}

// SendNotification sends a notification
func (nm *NotificationManager) SendNotification(ctx context.Context, config *NotificationConfig, data map[string]interface{}) (string, error) {
	startTime := time.Now()
	notificationID, _ := utils.GenerateID()

	nm.logger.Debug("Sending notification", map[string]interface{}{
		"notification_id": notificationID,
		"type":            config.Type,
		"recipients":      len(config.Recipients),
	})

	// Render template if provided
	content, err := nm.renderTemplate(config.Template, data)
	if err != nil {
		nm.logger.Error("Failed to render template", map[string]interface{}{
			"notification_id": notificationID,
			"error":           err,
		})
		return "", utils.NewAppError(utils.ErrCodeInternal, "Failed to render template", err.Error())
	}

	var err2 error
	switch config.Type {
	case "email":
		emailConfig := &EmailConfig{
			To:       config.Recipients,
			Subject:  config.Subject,
			Template: content,
			Priority: config.Priority,
		}
		err2 = nm.SendEmail(ctx, emailConfig, data)
	case "webhook":
		// Extract webhook config from channel or use default
		webhookConfig := &WebhookConfig{
			URL:     config.Recipients[0], // First recipient as URL
			Method:  "POST",
			Headers: map[string]string{"Content-Type": "application/json"},
			Timeout: nm.config.NotificationTimeout,
		}
		err2 = nm.SendWebhook(ctx, webhookConfig, data)
	default:
		err2 = utils.NewAppError(utils.ErrCodeValidation, "Unsupported notification type", config.Type)
	}

	// Update statistics
	nm.updateNotificationStats(startTime, err2)

	if err2 != nil {
		nm.logger.Error("Failed to send notification", map[string]interface{}{
			"notification_id": notificationID,
			"type":            config.Type,
			"error":           err2,
		})
		return "", err2
	}

	nm.logger.Info("Notification sent successfully", map[string]interface{}{
		"notification_id": notificationID,
		"type":            config.Type,
		"recipients":      len(config.Recipients),
		"duration":        time.Since(startTime),
	})

	return notificationID, nil
}

// SendWebhook sends a webhook notification
func (nm *NotificationManager) SendWebhook(ctx context.Context, config *WebhookConfig, data map[string]interface{}) error {
	return nm.webhookSender.SendWebhook(ctx, config, data)
}

// SendEmail sends an email notification
func (nm *NotificationManager) SendEmail(ctx context.Context, config *EmailConfig, data map[string]interface{}) error {
	return nm.emailSender.SendEmail(ctx, config, data)
}

// renderTemplate renders a notification template
func (nm *NotificationManager) renderTemplate(templateStr string, data map[string]interface{}) (string, error) {
	if templateStr == "" {
		// Return JSON representation of data
		return nm.formatDataAsJSON(data), nil
	}

	// Simple template rendering - replace placeholders
	content := templateStr
	for key, value := range data {
		placeholder := "{{." + key + "}}"
		content = strings.Replace(content, placeholder, fmt.Sprintf("%v", value), -1)
	}

	return content, nil
}

// formatDataAsJSON formats data as JSON string
func (nm *NotificationManager) formatDataAsJSON(data map[string]interface{}) string {
	var builder strings.Builder
	builder.WriteString("Event Data:\n")

	for key, value := range data {
		builder.WriteString(fmt.Sprintf("%s: %v\n", key, value))
	}

	return builder.String()
}

// updateNotificationStats updates notification statistics
func (nm *NotificationManager) updateNotificationStats(startTime time.Time, err error) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	nm.stats.TotalNotificationsSent++

	if err != nil {
		nm.stats.TotalNotificationsFailed++
		nm.stats.ErrorCount++
		errorStr := err.Error()
		nm.stats.LastError = &errorStr
		now := time.Now()
		nm.stats.LastErrorTime = &now
	}

	// Update average response time
	responseTime := time.Since(startTime)
	if nm.stats.TotalNotificationsSent == 1 {
		nm.stats.AverageResponseTime = responseTime
	} else {
		nm.stats.AverageResponseTime = (nm.stats.AverageResponseTime + responseTime) / 2
	}
}

// Channel management methods

// AddChannel adds a notification channel
func (nm *NotificationManager) AddChannel(channel *NotificationChannel) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	nm.channels[channel.ID] = channel
	nm.logger.Info("Notification channel added", map[string]interface{}{
		"channel_id": channel.ID,
		"type":       channel.Type,
		"name":       channel.Name,
	})
	return nil
}

// RemoveChannel removes a notification channel
func (nm *NotificationManager) RemoveChannel(channelID string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	delete(nm.channels, channelID)
	nm.logger.Info("Notification channel removed", map[string]interface{}{
		"channel_id": channelID,
	})
	return nil
}

// GetChannels returns all notification channels
func (nm *NotificationManager) GetChannels() []*NotificationChannel {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	channels := make([]*NotificationChannel, 0, len(nm.channels))
	for _, channel := range nm.channels {
		channels = append(channels, channel)
	}
	return channels
}

// GetStats returns notification statistics
func (nm *NotificationManager) GetStats() *NotificationStats {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	nm.stats.ActiveChannels = len(nm.channels)
	return nm.stats
}

func (nm *NotificationManager) GetHealth() *NotificationHealth {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	health := &NotificationHealth{
		Healthy: nm.running,
	}
	// Optionally, add more checks here (e.g., last error)
	if nm.stats != nil && nm.stats.LastError != nil {
		health.Error = *nm.stats.LastError
	}
	return health
}
