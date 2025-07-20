// File: internal/notification/notification_wrapper.go
package notification

import (
	"context"
	"time"

	"github.com/smartdevs17/rsk-event-listener/internal/metrics"
)

// NotificationManagerWithMetrics wraps NotificationManager with metrics
type NotificationManagerWithMetrics struct {
	*NotificationManager
	metricsManager *metrics.Manager
}

// NewNotificationManagerWithMetrics creates a notification manager wrapper with metrics
func NewNotificationManagerWithMetrics(manager *NotificationManager, metricsManager *metrics.Manager) *NotificationManagerWithMetrics {
	wrapper := &NotificationManagerWithMetrics{
		NotificationManager: manager,
		metricsManager:      metricsManager,
	}
	return wrapper
}

// SendNotification sends a notification and records metrics
func (nm *NotificationManagerWithMetrics) SendNotification(channel, notificationType string, content interface{}) error {
	start := time.Now()

	ctx := context.Background() // or pass the actual context
	config := &NotificationConfig{ /* fill fields */ }
	contentMap, ok := content.(map[string]interface{})
	if !ok {
		// handle error: content is not the right type
	}

	result, err := nm.NotificationManager.SendNotification(ctx, config, contentMap)
	println("Notification result:", result)
	prometheus := nm.metricsManager.GetPrometheusMetrics()

	if err != nil {
		prometheus.RecordNotificationFailure(channel, notificationType, "send_error")
	} else {
		prometheus.RecordNotificationSent(channel, notificationType, time.Since(start))
	}

	return err
}
