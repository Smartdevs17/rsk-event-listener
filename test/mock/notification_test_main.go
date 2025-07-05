// // File: examples/notification_example.go
// package main

// import (
// 	"context"
// 	"fmt"
// 	"time"

// 	"github.com/smartdevs17/rsk-event-listener/internal/notification"
// )

// func main() {
// 	// Create notification manager configuration
// 	config := &notification.NotificationManagerConfig{
// 		MaxConcurrentNotifications: 5,
// 		NotificationTimeout:        10 * time.Second,
// 		RetryAttempts:              3,
// 		RetryDelay:                 2 * time.Second,
// 		EnableEmailNotifications:   true,
// 		EnableWebhookNotifications: true,
// 		QueueSize:                  500,
// 		LogLevel:                   "info",
// 	}

// 	// Create notification manager
// 	notifier := notification.NewNotificationManager(config)

// 	// Start notification manager
// 	ctx := context.Background()
// 	if err := notifier.Start(ctx); err != nil {
// 		panic(fmt.Sprintf("Failed to start notification manager: %v", err))
// 	}
// 	defer notifier.Stop()

// 	// Example 1: Send webhook notification
// 	webhookConfig := &notification.WebhookConfig{
// 		URL:    "https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK",
// 		Method: "POST",
// 		Headers: map[string]string{
// 			"Content-Type": "application/json",
// 		},
// 		Timeout: 10 * time.Second,
// 	}

// 	eventData := map[string]interface{}{
// 		"event": map[string]interface{}{
// 			"id":           "evt_123",
// 			"event_name":   "Transfer",
// 			"address":      "0x1234567890123456789012345678901234567890",
// 			"block_number": 12345,
// 			"data": map[string]interface{}{
// 				"from":  "0xabcd...",
// 				"to":    "0xefgh...",
// 				"value": "1000000000000000000", // 1 ETH in wei
// 			},
// 		},
// 		"timestamp": time.Now(),
// 		"metadata": map[string]interface{}{
// 			"source": "rsk-event-listener",
// 		},
// 	}

// 	if err := notifier.SendWebhook(ctx, webhookConfig, eventData); err != nil {
// 		fmt.Printf("Failed to send webhook: %v\n", err)
// 	} else {
// 		fmt.Println("Webhook sent successfully!")
// 	}

// 	// Example 2: Send email notification
// 	emailConfig := &notification.EmailConfig{
// 		To:      []string{"admin@example.com"},
// 		Subject: "High Value Transfer Alert",
// 		Template: `
// 			<h2>High Value Transfer Detected</h2>
// 			<p>A high value transfer has been detected:</p>
// 			<ul>
// 				<li><strong>From:</strong> {{.event.data.from}}</li>
// 				<li><strong>To:</strong> {{.event.data.to}}</li>
// 				<li><strong>Value:</strong> {{.event.data.value}} wei</li>
// 				<li><strong>Block:</strong> {{.event.block_number}}</li>
// 				<li><strong>Transaction:</strong> {{.event.tx_hash}}</li>
// 			</ul>
// 			<p>Please review this transaction for any suspicious activity.</p>
// 		`,
// 		Priority: "high",
// 	}

// 	if err := notifier.SendEmail(ctx, emailConfig, eventData); err != nil {
// 		fmt.Printf("Failed to send email: %v\n", err)
// 	} else {
// 		fmt.Println("Email sent successfully!")
// 	}

// 	// Example 3: Send notification using the unified interface
// 	notificationConfig := &notification.NotificationConfig{
// 		Type:       "webhook",
// 		Recipients: []string{"https://api.example.com/webhooks/events"},
// 		Subject:    "Event Notification",
// 		Template:   "Event {{.event.event_name}} detected at block {{.event.block_number}}",
// 		Priority:   "normal",
// 	}

// 	notificationID, err := notifier.SendNotification(ctx, notificationConfig, eventData)
// 	if err != nil {
// 		fmt.Printf("Failed to send notification: %v\n", err)
// 	} else {
// 		fmt.Printf("Notification sent successfully! ID: %s\n", notificationID)
// 	}

// 	// Example 4: Add and manage notification channels
// 	slackChannel := &notification.NotificationChannel{
// 		ID:      "slack_general",
// 		Name:    "Slack General Channel",
// 		Type:    "webhook",
// 		Enabled: true,
// 		Config: map[string]interface{}{
// 			"url":    "https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK",
// 			"format": "slack",
// 		},
// 		CreatedAt: time.Now(),
// 		UpdatedAt: time.Now(),
// 	}

// 	if err := notifier.AddChannel(slackChannel); err != nil {
// 		fmt.Printf("Failed to add channel: %v\n", err)
// 	} else {
// 		fmt.Println("Slack channel added successfully!")
// 	}

// 	// Example 5: Get notification statistics
// 	stats := notifier.GetStats()
// 	fmt.Printf("Notification Statistics:\n")
// 	fmt.Printf("  Total Sent: %d\n", stats.TotalNotificationsSent)
// 	fmt.Printf("  Total Failed: %d\n", stats.TotalNotificationsFailed)
// 	fmt.Printf("  Average Response Time: %v\n", stats.AverageResponseTime)
// 	fmt.Printf("  Active Channels: %d\n", stats.ActiveChannels)

// 	// Example 6: Check health
// 	if notifier.IsHealthy() {
// 		fmt.Println("Notification system is healthy!")
// 	} else {
// 		fmt.Println("Notification system is unhealthy!")
// 	}
// }
