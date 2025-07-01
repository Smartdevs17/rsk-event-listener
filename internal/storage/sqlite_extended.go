package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/smartdevs17/rsk-event-listener/internal/models"
	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
)

// GetBlockProcessingStatus retrieves block processing status
func (s *SQLiteStorage) GetBlockProcessingStatus(blockNumber uint64) (*BlockProcessingStatus, error) {
	query := `
		SELECT block_number, status, started_at, completed_at, events_found, 
		       events_saved, error, processing_time
		FROM block_processing_status WHERE block_number = ?
	`

	row := s.db.QueryRow(query, blockNumber)

	var status BlockProcessingStatus
	var completedAt sql.NullTime
	var errorStr sql.NullString
	var processingTimeMs sql.NullInt64

	err := row.Scan(&status.BlockNumber, &status.Status, &status.StartedAt,
		&completedAt, &status.EventsFound, &status.EventsSaved, &errorStr, &processingTimeMs)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to get block processing status", err.Error())
	}

	if completedAt.Valid {
		status.CompletedAt = &completedAt.Time
	}

	if errorStr.Valid {
		status.Error = &errorStr.String
	}

	if processingTimeMs.Valid {
		duration := time.Duration(processingTimeMs.Int64) * time.Millisecond
		status.ProcessingTime = &duration
	}

	return &status, nil
}

// SetBlockProcessingStatus sets block processing status
func (s *SQLiteStorage) SetBlockProcessingStatus(status *BlockProcessingStatus) error {
	var processingTimeMs *int64
	if status.ProcessingTime != nil {
		ms := int64(status.ProcessingTime.Nanoseconds() / 1000000)
		processingTimeMs = &ms
	}

	query := `
		INSERT OR REPLACE INTO block_processing_status 
		(block_number, status, started_at, completed_at, events_found, events_saved, error, processing_time)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query, status.BlockNumber, status.Status, status.StartedAt,
		status.CompletedAt, status.EventsFound, status.EventsSaved, status.Error, processingTimeMs)

	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to set block processing status", err.Error())
	}

	return nil
}

// SaveNotification saves a notification
func (s *SQLiteStorage) SaveNotification(ctx context.Context, notification *models.Notification) error {
	dataJSON, err := json.Marshal(notification.Data)
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to marshal notification data", err.Error())
	}

	query := `
		INSERT OR REPLACE INTO notifications 
		(id, type, event_id, title, message, data, target, status, attempts, created_at, sent_at, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, query,
		notification.ID, notification.Type, notification.EventID, notification.Title,
		notification.Message, string(dataJSON), notification.Target, notification.Status,
		notification.Attempts, notification.CreatedAt, notification.SentAt, notification.Error)

	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to save notification", err.Error())
	}

	return nil
}

// GetPendingNotifications retrieves pending notifications
func (s *SQLiteStorage) GetPendingNotifications(ctx context.Context, limit int) ([]*models.Notification, error) {
	query := `
		SELECT id, type, event_id, title, message, data, target, status, 
		       attempts, created_at, sent_at, error
		FROM notifications 
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to query pending notifications", err.Error())
	}
	defer rows.Close()

	var notifications []*models.Notification
	for rows.Next() {
		var notification models.Notification
		var dataJSON string
		var sentAt sql.NullTime
		var errorStr sql.NullString

		err := rows.Scan(&notification.ID, &notification.Type, &notification.EventID,
			&notification.Title, &notification.Message, &dataJSON, &notification.Target,
			&notification.Status, &notification.Attempts, &notification.CreatedAt,
			&sentAt, &errorStr)

		if err != nil {
			return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to scan notification", err.Error())
		}

		if err := json.Unmarshal([]byte(dataJSON), &notification.Data); err != nil {
			return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to unmarshal notification data", err.Error())
		}

		if sentAt.Valid {
			notification.SentAt = &sentAt.Time
		}

		if errorStr.Valid {
			notification.Error = &errorStr.String
		}

		notifications = append(notifications, &notification)
	}

	return notifications, nil
}

// UpdateNotificationStatus updates notification status
func (s *SQLiteStorage) UpdateNotificationStatus(ctx context.Context, id string, status string, errorMsg *string) error {
	var sentAt *time.Time
	if status == "sent" {
		now := time.Now()
		sentAt = &now
	}

	query := `
		UPDATE notifications 
		SET status = ?, sent_at = ?, error = ?, attempts = attempts + 1
		WHERE id = ?
	`

	result, err := s.db.ExecContext(ctx, query, status, sentAt, errorMsg, id)
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to update notification status", err.Error())
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to get rows affected", err.Error())
	}

	if rowsAffected == 0 {
		return utils.NewAppError(utils.ErrCodeNotFound, "Notification not found", id)
	}

	return nil
}

// GetStorageStats returns storage statistics
func (s *SQLiteStorage) GetStorageStats() (*StorageStats, error) {
	stats := &StorageStats{}

	// Get event count
	err := s.db.QueryRow("SELECT COUNT(*) FROM events").Scan(&stats.TotalEvents)
	if err != nil {
		return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to get event count", err.Error())
	}

	// Get contract count
	err = s.db.QueryRow("SELECT COUNT(*) FROM contracts").Scan(&stats.TotalContracts)
	if err != nil {
		return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to get contract count", err.Error())
	}

	// Get notification count
	err = s.db.QueryRow("SELECT COUNT(*) FROM notifications").Scan(&stats.TotalNotifications)
	if err != nil {
		return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to get notification count", err.Error())
	}

	// Get oldest event timestamp
	var oldestEvent sql.NullTime
	err = s.db.QueryRow("SELECT MIN(timestamp) FROM events").Scan(&oldestEvent)
	if err == nil && oldestEvent.Valid {
		stats.OldestEvent = &oldestEvent.Time
	}

	// Get latest event timestamp
	var latestEvent sql.NullTime
	err = s.db.QueryRow("SELECT MAX(timestamp) FROM events").Scan(&latestEvent)
	if err == nil && latestEvent.Valid {
		stats.LatestEvent = &latestEvent.Time
	}

	// Get latest processed block
	stats.LatestBlock, _ = s.GetLatestProcessedBlock()

	// Get database size (SQLite specific)
	err = s.db.QueryRow("SELECT page_count * page_size FROM pragma_page_count(), pragma_page_size()").Scan(&stats.DatabaseSize)
	if err != nil {
		// Fallback: try to get file size
		stats.DatabaseSize = 0
	}

	return stats, nil
}

// GetEventStats returns event statistics for a time period
func (s *SQLiteStorage) GetEventStats(ctx context.Context, fromTime, toTime time.Time) (*EventStats, error) {
	stats := &EventStats{
		TimeRange:        TimeRange{From: fromTime, To: toTime},
		EventsByType:     make(map[string]int64),
		EventsByContract: make(map[string]int64),
		EventsByHour:     make(map[string]int64),
	}

	// Get total events in time range
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM events WHERE timestamp BETWEEN ? AND ?",
		fromTime, toTime).Scan(&stats.TotalEvents)
	if err != nil {
		return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to get total events", err.Error())
	}

	// Get events by type
	rows, err := s.db.QueryContext(ctx, `
		SELECT event_name, COUNT(*) 
		FROM events 
		WHERE timestamp BETWEEN ? AND ? 
		GROUP BY event_name
	`, fromTime, toTime)
	if err != nil {
		return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to get events by type", err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var eventName string
		var count int64
		if err := rows.Scan(&eventName, &count); err != nil {
			continue
		}
		stats.EventsByType[eventName] = count
	}

	// Get events by contract
	rows, err = s.db.QueryContext(ctx, `
		SELECT address, COUNT(*) 
		FROM events 
		WHERE timestamp BETWEEN ? AND ? 
		GROUP BY address
	`, fromTime, toTime)
	if err != nil {
		return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to get events by contract", err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var address string
		var count int64
		if err := rows.Scan(&address, &count); err != nil {
			continue
		}
		stats.EventsByContract[address] = count
	}

	// Get events by hour
	rows, err = s.db.QueryContext(ctx, `
		SELECT strftime('%Y-%m-%d %H:00:00', timestamp) as hour, COUNT(*) 
		FROM events 
		WHERE timestamp BETWEEN ? AND ? 
		GROUP BY hour
		ORDER BY hour
	`, fromTime, toTime)
	if err != nil {
		return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to get events by hour", err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var hour string
		var count int64
		if err := rows.Scan(&hour, &count); err != nil {
			continue
		}
		stats.EventsByHour[hour] = count
	}

	// Get processing stats
	var totalProcessed, totalFailed int64
	s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM events WHERE processed = true AND timestamp BETWEEN ? AND ?",
		fromTime, toTime).Scan(&totalProcessed)
	s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM events WHERE processed = false AND timestamp BETWEEN ? AND ?",
		fromTime, toTime).Scan(&totalFailed)

	stats.ProcessingStats = ProcessingStats{
		TotalProcessed: totalProcessed,
		TotalFailed:    totalFailed,
	}

	// Get last processed timestamp
	var lastProcessed sql.NullTime
	s.db.QueryRowContext(ctx,
		"SELECT MAX(processed_at) FROM events WHERE processed = true AND timestamp BETWEEN ? AND ?",
		fromTime, toTime).Scan(&lastProcessed)
	if lastProcessed.Valid {
		stats.ProcessingStats.LastProcessedAt = &lastProcessed.Time
	}

	return stats, nil
}

// Cleanup removes old data based on retention policy
func (s *SQLiteStorage) Cleanup(ctx context.Context, retentionDays int) error {
	if retentionDays <= 0 {
		return nil
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to begin cleanup transaction", err.Error())
	}
	defer tx.Rollback()

	// Delete old events
	result, err := tx.ExecContext(ctx, "DELETE FROM events WHERE timestamp < ?", cutoffTime)
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to cleanup old events", err.Error())
	}

	eventsDeleted, _ := result.RowsAffected()

	// Delete old notifications
	result, err = tx.ExecContext(ctx, "DELETE FROM notifications WHERE created_at < ?", cutoffTime)
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to cleanup old notifications", err.Error())
	}

	notificationsDeleted, _ := result.RowsAffected()

	// Delete old block processing status
	result, err = tx.ExecContext(ctx, "DELETE FROM block_processing_status WHERE started_at < ?", cutoffTime)
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to cleanup old block processing status", err.Error())
	}

	blockStatusDeleted, _ := result.RowsAffected()

	// Update last cleanup time
	_, err = tx.ExecContext(ctx,
		"INSERT OR REPLACE INTO system_state (key, value, updated_at) VALUES ('last_cleanup', ?, ?)",
		time.Now().Format(time.RFC3339), time.Now())
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to update last cleanup time", err.Error())
	}

	if err := tx.Commit(); err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to commit cleanup transaction", err.Error())
	}

	s.logger.Info("Database cleanup completed",
		"events_deleted", eventsDeleted,
		"notifications_deleted", notificationsDeleted,
		"block_status_deleted", blockStatusDeleted,
		"retention_days", retentionDays)

	return nil
}

// Vacuum optimizes the database
func (s *SQLiteStorage) Vacuum() error {
	s.logger.Info("Starting database vacuum")

	_, err := s.db.Exec("VACUUM")
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to vacuum database", err.Error())
	}

	s.logger.Info("Database vacuum completed")
	return nil
}

// Additional helper methods
func (s *SQLiteStorage) GetEventsByBlock(ctx context.Context, blockNumber uint64) ([]*models.Event, error) {
	filter := models.EventFilter{
		FromBlock: &blockNumber,
		ToBlock:   &blockNumber,
	}
	return s.GetEvents(ctx, filter)
}

func (s *SQLiteStorage) GetUnprocessedEvents(ctx context.Context, limit int) ([]*models.Event, error) {
	processed := false
	filter := models.EventFilter{
		Processed: &processed,
		Limit:     limit,
	}
	return s.GetEvents(ctx, filter)
}

func (s *SQLiteStorage) MarkEventProcessed(ctx context.Context, eventID string) error {
	now := time.Now()
	query := "UPDATE events SET processed = true, processed_at = ? WHERE id = ?"

	result, err := s.db.ExecContext(ctx, query, now, eventID)
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to mark event as processed", err.Error())
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to get rows affected", err.Error())
	}

	if rowsAffected == 0 {
		return utils.NewAppError(utils.ErrCodeNotFound, "Event not found", eventID)
	}

	return nil
}

func (s *SQLiteStorage) GetDatabaseInfo() (map[string]interface{}, error) {
	info := make(map[string]interface{})

	// SQLite version
	var version string
	s.db.QueryRow("SELECT sqlite_version()").Scan(&version)
	info["sqlite_version"] = version

	// Database file size
	var pageCount, pageSize int64
	s.db.QueryRow("PRAGMA page_count").Scan(&pageCount)
	s.db.QueryRow("PRAGMA page_size").Scan(&pageSize)
	info["file_size_bytes"] = pageCount * pageSize
	info["page_count"] = pageCount
	info["page_size"] = pageSize

	// Journal mode
	var journalMode string
	s.db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	info["journal_mode"] = journalMode

	// Foreign keys enabled
	var foreignKeys bool
	s.db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys)
	info["foreign_keys_enabled"] = foreignKeys

	return info, nil
}
