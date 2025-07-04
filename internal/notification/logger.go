// File: internal/notification/logger.go
package notification

import (
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
)

// NotificationLogger handles logging for notification operations
type NotificationLogger struct {
	logger   *logrus.Logger
	logLevel logrus.Level
	context  map[string]interface{}
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Component string                 `json:"component"`
}

// NewNotificationLogger creates a new notification logger
func NewNotificationLogger(logLevel string) *NotificationLogger {
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}

	logger := utils.GetLogger()
	logger.SetLevel(level)

	return &NotificationLogger{
		logger:   logger,
		logLevel: level,
		context:  make(map[string]interface{}),
	}
}

// WithContext adds context to the logger
func (nl *NotificationLogger) WithContext(context map[string]interface{}) *NotificationLogger {
	newLogger := &NotificationLogger{
		logger:   nl.logger,
		logLevel: nl.logLevel,
		context:  make(map[string]interface{}),
	}

	// Copy existing context
	for k, v := range nl.context {
		newLogger.context[k] = v
	}

	// Add new context
	for k, v := range context {
		newLogger.context[k] = v
	}

	return newLogger
}

// WithField adds a single field to the logger context
func (nl *NotificationLogger) WithField(key string, value interface{}) *NotificationLogger {
	return nl.WithContext(map[string]interface{}{key: value})
}

// Debug logs a debug message
func (nl *NotificationLogger) Debug(message string, context ...map[string]interface{}) {
	nl.log(logrus.DebugLevel, message, context...)
}

// Info logs an info message
func (nl *NotificationLogger) Info(message string, context ...map[string]interface{}) {
	nl.log(logrus.InfoLevel, message, context...)
}

// Warn logs a warning message
func (nl *NotificationLogger) Warn(message string, context ...map[string]interface{}) {
	nl.log(logrus.WarnLevel, message, context...)
}

// Error logs an error message
func (nl *NotificationLogger) Error(message string, context ...map[string]interface{}) {
	nl.log(logrus.ErrorLevel, message, context...)
}

// Fatal logs a fatal message and exits
func (nl *NotificationLogger) Fatal(message string, context ...map[string]interface{}) {
	nl.log(logrus.FatalLevel, message, context...)
}

// log is the internal logging method
func (nl *NotificationLogger) log(level logrus.Level, message string, context ...map[string]interface{}) {
	if level < nl.logLevel {
		return
	}

	// Merge context
	mergedContext := make(map[string]interface{})

	// Add base context
	for k, v := range nl.context {
		mergedContext[k] = v
	}

	// Add method context
	for _, ctx := range context {
		for k, v := range ctx {
			mergedContext[k] = v
		}
	}

	// Add component identifier
	mergedContext["component"] = "notification"

	// Create log entry
	entry := nl.logger.WithFields(logrus.Fields(mergedContext))

	// Log based on level
	switch level {
	case logrus.DebugLevel:
		entry.Debug(message)
	case logrus.InfoLevel:
		entry.Info(message)
	case logrus.WarnLevel:
		entry.Warn(message)
	case logrus.ErrorLevel:
		entry.Error(message)
	case logrus.FatalLevel:
		entry.Fatal(message)
	}
}

// LogNotificationAttempt logs a notification attempt
func (nl *NotificationLogger) LogNotificationAttempt(notificationID, notificationType string, recipients []string) {
	nl.Info("Notification attempt started", map[string]interface{}{
		"notification_id":   notificationID,
		"notification_type": notificationType,
		"recipient_count":   len(recipients),
		"recipients":        strings.Join(recipients, ", "),
	})
}

// LogNotificationSuccess logs a successful notification
func (nl *NotificationLogger) LogNotificationSuccess(notificationID, notificationType string, duration time.Duration) {
	nl.Info("Notification sent successfully", map[string]interface{}{
		"notification_id":   notificationID,
		"notification_type": notificationType,
		"duration_ms":       duration.Milliseconds(),
	})
}

// LogNotificationFailure logs a failed notification
func (nl *NotificationLogger) LogNotificationFailure(notificationID, notificationType string, err error, duration time.Duration) {
	nl.Error("Notification failed", map[string]interface{}{
		"notification_id":   notificationID,
		"notification_type": notificationType,
		"error":             err.Error(),
		"duration_ms":       duration.Milliseconds(),
	})
}

// LogWebhookAttempt logs a webhook attempt
func (nl *NotificationLogger) LogWebhookAttempt(url, method string, headers map[string]string) {
	nl.Debug("Webhook attempt started", map[string]interface{}{
		"url":     url,
		"method":  method,
		"headers": headers,
	})
}

// LogWebhookResponse logs a webhook response
func (nl *NotificationLogger) LogWebhookResponse(url string, statusCode int, duration time.Duration, err error) {
	context := map[string]interface{}{
		"url":         url,
		"status_code": statusCode,
		"duration_ms": duration.Milliseconds(),
	}

	if err != nil {
		context["error"] = err.Error()
		nl.Error("Webhook failed", context)
	} else {
		nl.Info("Webhook completed", context)
	}
}

// LogEmailAttempt logs an email attempt
func (nl *NotificationLogger) LogEmailAttempt(to []string, subject string) {
	nl.Debug("Email attempt started", map[string]interface{}{
		"to":      strings.Join(to, ", "),
		"subject": subject,
	})
}

// LogEmailResult logs an email result
func (nl *NotificationLogger) LogEmailResult(to []string, subject string, success bool, duration time.Duration, err error) {
	context := map[string]interface{}{
		"to":          strings.Join(to, ", "),
		"subject":     subject,
		"success":     success,
		"duration_ms": duration.Milliseconds(),
	}

	if err != nil {
		context["error"] = err.Error()
		nl.Error("Email failed", context)
	} else {
		nl.Info("Email sent successfully", context)
	}
}

// LogRetryAttempt logs a retry attempt
func (nl *NotificationLogger) LogRetryAttempt(operation string, attempt int, maxAttempts int, delay time.Duration) {
	nl.Warn("Retrying operation", map[string]interface{}{
		"operation":    operation,
		"attempt":      attempt,
		"max_attempts": maxAttempts,
		"retry_delay":  delay.String(),
	})
}

// LogChannelOperation logs channel management operations
func (nl *NotificationLogger) LogChannelOperation(operation, channelID, channelType string) {
	nl.Info("Channel operation", map[string]interface{}{
		"operation":    operation,
		"channel_id":   channelID,
		"channel_type": channelType,
	})
}

// LogConfigurationChange logs configuration changes
func (nl *NotificationLogger) LogConfigurationChange(component string, changes map[string]interface{}) {
	nl.Info("Configuration changed", map[string]interface{}{
		"component": component,
		"changes":   changes,
	})
}

// LogHealthCheck logs health check results
func (nl *NotificationLogger) LogHealthCheck(component string, healthy bool, issues []string) {
	context := map[string]interface{}{
		"component": component,
		"healthy":   healthy,
	}

	if len(issues) > 0 {
		context["issues"] = strings.Join(issues, ", ")
	}

	if healthy {
		nl.Debug("Health check passed", context)
	} else {
		nl.Warn("Health check failed", context)
	}
}

// GetLogLevel returns the current log level
func (nl *NotificationLogger) GetLogLevel() logrus.Level {
	return nl.logLevel
}

// SetLogLevel sets the log level
func (nl *NotificationLogger) SetLogLevel(level logrus.Level) {
	nl.logLevel = level
	nl.logger.SetLevel(level)
}

// CreateStructuredLog creates a structured log entry
func (nl *NotificationLogger) CreateStructuredLog(level, message string, context map[string]interface{}) *LogEntry {
	return &LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Context:   context,
		Component: "notification",
	}
}
