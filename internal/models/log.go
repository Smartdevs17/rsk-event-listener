package models

import "time"

// LogEntry represents a log entry in the system (e.g., for reorg events)
type LogEntry struct {
	ID        string                 `json:"id" db:"id"`
	Type      string                 `json:"type" db:"type"`
	Data      map[string]interface{} `json:"data" db:"data"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
}
