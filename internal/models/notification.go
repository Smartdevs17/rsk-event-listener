package models

import (
	"time"
)

// NotificationType defines the type of notification
type NotificationType string

const (
	NotificationTypeWebhook NotificationType = "webhook"
	NotificationTypeLog     NotificationType = "log"
	NotificationTypeSlack   NotificationType = "slack"
	NotificationTypeEmail   NotificationType = "email"
)

// Notification represents a notification to be sent
type Notification struct {
	ID        string                 `json:"id"`
	Type      NotificationType       `json:"type"`
	EventID   string                 `json:"event_id"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Target    string                 `json:"target"` // webhook URL, email address, etc.
	Status    string                 `json:"status"` // pending, sent, failed
	Attempts  int                    `json:"attempts"`
	CreatedAt time.Time              `json:"created_at"`
	SentAt    *time.Time             `json:"sent_at,omitempty"`
	Error     *string                `json:"error,omitempty"`
}

// NotificationChannel defines a notification channel configuration
type NotificationChannel struct {
	ID       string                 `json:"id"`
	Type     NotificationType       `json:"type"`
	Name     string                 `json:"name"`
	Config   map[string]interface{} `json:"config"`
	Active   bool                   `json:"active"`
	Priority int                    `json:"priority"`
}

// AlertRule defines when to send notifications
type AlertRule struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	EventFilter EventFilter   `json:"event_filter"`
	Channels    []string      `json:"channels"` // Channel IDs
	Template    string        `json:"template"`
	Cooldown    time.Duration `json:"cooldown"`
	Active      bool          `json:"active"`
}
