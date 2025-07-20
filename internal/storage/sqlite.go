// File: internal/storage/sqlite.go
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/sirupsen/logrus"
	"github.com/smartdevs17/rsk-event-listener/internal/metrics"
	"github.com/smartdevs17/rsk-event-listener/internal/models"
	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
	_ "modernc.org/sqlite"
)

// SQLiteStorage implements Storage interface using SQLite
type SQLiteStorage struct {
	db         *sql.DB
	config     *StorageConfig
	logger     *logrus.Logger
	migrations []*Migration

	metricsManager *metrics.Manager
}

// NewSQLiteStorage creates a new SQLite storage instance
func NewSQLiteStorage(config *StorageConfig) *SQLiteStorage {
	return &SQLiteStorage{
		config:     config,
		logger:     utils.GetLogger(),
		migrations: GetSQLiteMigrations(),
	}
}

// Connect establishes database connection
func (s *SQLiteStorage) Connect() error {
	// Ensure directory exists
	dir := filepath.Dir(s.config.ConnectionString)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return utils.NewAppError(utils.ErrCodeDatabase, "Failed to create database directory", err.Error())
		}
	}

	db, err := sql.Open("sqlite", s.config.ConnectionString)
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to open SQLite database", err.Error())
	}

	// Configure connection pool
	db.SetMaxOpenConns(s.config.MaxConnections)
	db.SetMaxIdleConns(s.config.MaxConnections / 2)
	db.SetConnMaxLifetime(s.config.MaxIdleTime)

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to enable WAL mode", err.Error())
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to enable foreign keys", err.Error())
	}

	s.db = db
	s.logger.Info("SQLite database connected", "path", s.config.ConnectionString)

	return nil
}

// Close closes the database connection
func (s *SQLiteStorage) Close() error {
	if s.db != nil {
		err := s.db.Close()
		s.db = nil
		s.logger.Info("SQLite database connection closed")
		return err
	}
	return nil
}

// Ping checks database connectivity
func (s *SQLiteStorage) Ping() error {
	if s.db == nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Database not connected", "")
	}
	return s.db.Ping()
}

// Migrate runs database migrations
func (s *SQLiteStorage) Migrate() error {
	if s.db == nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Database not connected", "")
	}

	s.logger.Info("Starting database migrations")

	for _, migration := range s.migrations {
		s.logger.Info("Applying migration", "version", migration.Version, "description", migration.Description)

		if _, err := s.db.Exec(migration.SQL); err != nil {
			return utils.NewAppError(utils.ErrCodeDatabase,
				fmt.Sprintf("Migration %s failed", migration.Version),
				err.Error())
		}
	}

	s.logger.Info("Database migrations completed")
	return nil
}

// SaveEvent saves a single event
func (s *SQLiteStorage) SaveEvent(ctx context.Context, event *models.Event) error {
	start := time.Now()

	dataJSON, err := json.Marshal(event.Data)
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to marshal event data", err.Error())
	}

	query := `
		INSERT OR REPLACE INTO events 
		(id, block_number, block_hash, tx_hash, tx_index, log_index, address, 
		 event_name, event_signature, data, timestamp, processed, processed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, query,
		event.ID, event.BlockNumber, event.BlockHash, event.TxHash,
		event.TxIndex, event.LogIndex, event.Address, event.EventName,
		event.EventSig, string(dataJSON), event.Timestamp, event.Processed, event.ProcessedAt)

	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to save event", err.Error())
	}

	if s.metricsManager != nil {
		status := "success"
		if err != nil {
			status = "error"
		}

		s.metricsManager.GetPrometheusMetrics().RecordDatabaseOperation(
			"upsert",
			"contracts",
			status,
			time.Since(start),
		)
	}

	return nil
}

// SaveEvents saves multiple events in a transaction
func (s *SQLiteStorage) SaveEvents(ctx context.Context, events []*models.Event) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to begin transaction", err.Error())
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO events 
		(id, block_number, block_hash, tx_hash, tx_index, log_index, address, 
		 event_name, event_signature, data, timestamp, processed, processed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to prepare statement", err.Error())
	}
	defer stmt.Close()

	for _, event := range events {
		dataJSON, err := json.Marshal(event.Data)
		if err != nil {
			return utils.NewAppError(utils.ErrCodeDatabase, "Failed to marshal event data", err.Error())
		}

		_, err = stmt.ExecContext(ctx,
			event.ID, event.BlockNumber, event.BlockHash, event.TxHash,
			event.TxIndex, event.LogIndex, event.Address, event.EventName,
			event.EventSig, string(dataJSON), event.Timestamp, event.Processed, event.ProcessedAt)

		if err != nil {
			return utils.NewAppError(utils.ErrCodeDatabase, "Failed to save event in batch", err.Error())
		}
	}

	if err := tx.Commit(); err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to commit transaction", err.Error())
	}

	s.logger.Debug("Saved events batch", "count", len(events))
	return nil
}

// GetEvent retrieves a single event by ID
func (s *SQLiteStorage) GetEvent(ctx context.Context, id string) (*models.Event, error) {
	query := `
		SELECT id, block_number, block_hash, tx_hash, tx_index, log_index, address,
		       event_name, event_signature, data, timestamp, processed, processed_at
		FROM events WHERE id = ?
	`

	row := s.db.QueryRowContext(ctx, query, id)

	var event models.Event
	var dataJSON string
	var processedAt sql.NullTime

	err := row.Scan(&event.ID, &event.BlockNumber, &event.BlockHash, &event.TxHash,
		&event.TxIndex, &event.LogIndex, &event.Address, &event.EventName,
		&event.EventSig, &dataJSON, &event.Timestamp, &event.Processed, &processedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to get event", err.Error())
	}

	if err := json.Unmarshal([]byte(dataJSON), &event.Data); err != nil {
		return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to unmarshal event data", err.Error())
	}

	if processedAt.Valid {
		event.ProcessedAt = &processedAt.Time
	}

	return &event, nil
}

// GetEvents retrieves events based on filter
func (s *SQLiteStorage) GetEvents(ctx context.Context, filter models.EventFilter) ([]*models.Event, error) {
	query := `
		SELECT id, block_number, block_hash, tx_hash, tx_index, log_index, address,
		       event_name, event_signature, data, timestamp, processed, processed_at
		FROM events WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	// Apply filters
	if filter.ContractAddress != nil {
		query += fmt.Sprintf(" AND address = $%d", argIndex)
		args = append(args, strings.ToLower(filter.ContractAddress.Hex()))
		argIndex++
	}

	if filter.EventName != nil {
		query += fmt.Sprintf(" AND event_name = $%d", argIndex)
		args = append(args, *filter.EventName)
		argIndex++
	}

	if filter.FromBlock != nil {
		query += fmt.Sprintf(" AND block_number >= $%d", argIndex)
		args = append(args, *filter.FromBlock)
		argIndex++
	}

	if filter.ToBlock != nil {
		query += fmt.Sprintf(" AND block_number <= $%d", argIndex)
		args = append(args, *filter.ToBlock)
		argIndex++
	}

	if filter.Processed != nil {
		query += fmt.Sprintf(" AND processed = $%d", argIndex)
		args = append(args, *filter.Processed)
		argIndex++
	}

	// Add ordering and pagination
	query += " ORDER BY block_number DESC, log_index ASC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
		argIndex++
	}

	// Convert numbered parameters to ? for SQLite
	for i := argIndex - 1; i >= 1; i-- {
		query = strings.Replace(query, fmt.Sprintf("$%d", i), "?", 1)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to query events", err.Error())
	}
	defer rows.Close()

	var events []*models.Event
	for rows.Next() {
		var event models.Event
		var dataJSON string
		var processedAt sql.NullTime

		err := rows.Scan(&event.ID, &event.BlockNumber, &event.BlockHash, &event.TxHash,
			&event.TxIndex, &event.LogIndex, &event.Address, &event.EventName,
			&event.EventSig, &dataJSON, &event.Timestamp, &event.Processed, &processedAt)

		if err != nil {
			return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to scan event", err.Error())
		}

		if err := json.Unmarshal([]byte(dataJSON), &event.Data); err != nil {
			return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to unmarshal event data", err.Error())
		}

		if processedAt.Valid {
			event.ProcessedAt = &processedAt.Time
		}

		events = append(events, &event)
	}

	return events, nil
}

// GetEventCount returns the count of events matching filter
func (s *SQLiteStorage) GetEventCount(ctx context.Context, filter models.EventFilter) (int64, error) {
	query := "SELECT COUNT(*) FROM events WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	// Apply same filters as GetEvents
	if filter.ContractAddress != nil {
		query += fmt.Sprintf(" AND address = $%d", argIndex)
		args = append(args, strings.ToLower(filter.ContractAddress.Hex()))
		argIndex++
	}

	if filter.EventName != nil {
		query += fmt.Sprintf(" AND event_name = $%d", argIndex)
		args = append(args, *filter.EventName)
		argIndex++
	}

	if filter.FromBlock != nil {
		query += fmt.Sprintf(" AND block_number >= $%d", argIndex)
		args = append(args, *filter.FromBlock)
		argIndex++
	}

	if filter.ToBlock != nil {
		query += fmt.Sprintf(" AND block_number <= $%d", argIndex)
		args = append(args, *filter.ToBlock)
		argIndex++
	}

	if filter.Processed != nil {
		query += fmt.Sprintf(" AND processed = $%d", argIndex)
		args = append(args, *filter.Processed)
		argIndex++
	}

	// Convert numbered parameters to ? for SQLite
	for i := argIndex - 1; i >= 1; i-- {
		query = strings.Replace(query, fmt.Sprintf("$%d", i), "?", 1)
	}

	var count int64
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, utils.NewAppError(utils.ErrCodeDatabase, "Failed to count events", err.Error())
	}

	return count, nil
}

// UpdateEvent updates an existing event
func (s *SQLiteStorage) UpdateEvent(ctx context.Context, event *models.Event) error {
	dataJSON, err := json.Marshal(event.Data)
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to marshal event data", err.Error())
	}

	query := `
		UPDATE events SET 
			block_number = ?, block_hash = ?, tx_hash = ?, tx_index = ?, 
			log_index = ?, address = ?, event_name = ?, event_signature = ?, 
			data = ?, timestamp = ?, processed = ?, processed_at = ?
		WHERE id = ?
	`

	result, err := s.db.ExecContext(ctx, query,
		event.BlockNumber, event.BlockHash, event.TxHash, event.TxIndex,
		event.LogIndex, event.Address, event.EventName, event.EventSig,
		string(dataJSON), event.Timestamp, event.Processed, event.ProcessedAt, event.ID)

	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to update event", err.Error())
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to get rows affected", err.Error())
	}

	if rowsAffected == 0 {
		return utils.NewAppError(utils.ErrCodeNotFound, "Event not found", event.ID)
	}

	return nil
}

// DeleteEvent deletes an event by ID
func (s *SQLiteStorage) DeleteEvent(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM events WHERE id = ?", id)
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to delete event", err.Error())
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to get rows affected", err.Error())
	}

	if rowsAffected == 0 {
		return utils.NewAppError(utils.ErrCodeNotFound, "Event not found", id)
	}

	return nil
}

// SaveContract saves a contract
func (s *SQLiteStorage) SaveContract(ctx context.Context, contract *models.Contract) error {
	start := time.Now()

	query := `
		INSERT OR REPLACE INTO contracts (address, name, abi, start_block, active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		strings.ToLower(contract.Address.Hex()), contract.Name, contract.ABI,
		contract.StartBlock, contract.Active, contract.CreatedAt, contract.UpdatedAt)

	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to save contract", err.Error())
	}

	if s.metricsManager != nil {
		status := "success"
		if err != nil {
			status = "error"
		}

		s.metricsManager.GetPrometheusMetrics().RecordDatabaseOperation(
			"upsert",
			"contracts",
			status,
			time.Since(start),
		)
	}

	return nil
}

// GetContract retrieves a contract by address
func (s *SQLiteStorage) GetContract(ctx context.Context, address string) (*models.Contract, error) {
	query := `
		SELECT address, name, abi, start_block, active, created_at, updated_at
		FROM contracts WHERE address = ?
	`

	row := s.db.QueryRowContext(ctx, query, strings.ToLower(address))

	var contract models.Contract
	var addressStr string

	err := row.Scan(&addressStr, &contract.Name, &contract.ABI,
		&contract.StartBlock, &contract.Active, &contract.CreatedAt, &contract.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to get contract", err.Error())
	}

	contract.Address = common.HexToAddress(addressStr)
	return &contract, nil
}

// GetContracts retrieves contracts, optionally filtered by active status
func (s *SQLiteStorage) GetContracts(ctx context.Context, active *bool) ([]*models.Contract, error) {
	query := "SELECT address, name, abi, start_block, active, created_at, updated_at FROM contracts"
	args := []interface{}{}

	if active != nil {
		query += " WHERE active = ?"
		args = append(args, *active)
	}

	query += " ORDER BY created_at ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to query contracts", err.Error())
	}
	defer rows.Close()

	var contracts []*models.Contract
	for rows.Next() {
		var contract models.Contract
		var addressStr string

		err := rows.Scan(&addressStr, &contract.Name, &contract.ABI,
			&contract.StartBlock, &contract.Active, &contract.CreatedAt, &contract.UpdatedAt)

		if err != nil {
			return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to scan contract", err.Error())
		}

		contract.Address = common.HexToAddress(addressStr)
		contracts = append(contracts, &contract)
	}

	return contracts, nil
}

// UpdateContract updates an existing contract
func (s *SQLiteStorage) UpdateContract(ctx context.Context, contract *models.Contract) error {
	query := `
		UPDATE contracts SET name = ?, abi = ?, start_block = ?, active = ?, updated_at = ?
		WHERE address = ?
	`

	result, err := s.db.ExecContext(ctx, query,
		contract.Name, contract.ABI, contract.StartBlock, contract.Active,
		contract.UpdatedAt, strings.ToLower(contract.Address.Hex()))

	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to update contract", err.Error())
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to get rows affected", err.Error())
	}

	if rowsAffected == 0 {
		return utils.NewAppError(utils.ErrCodeNotFound, "Contract not found", contract.Address.Hex())
	}

	return nil
}

// DeleteContract deletes a contract by address
func (s *SQLiteStorage) DeleteContract(ctx context.Context, address string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM contracts WHERE address = ?", strings.ToLower(address))
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to delete contract", err.Error())
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to get rows affected", err.Error())
	}

	if rowsAffected == 0 {
		return utils.NewAppError(utils.ErrCodeNotFound, "Contract not found", address)
	}

	return nil
}

// GetLatestProcessedBlock returns the latest processed block number
func (s *SQLiteStorage) GetLatestProcessedBlock() (uint64, error) {
	var blockNumber uint64
	err := s.db.QueryRow("SELECT value FROM system_state WHERE key = 'latest_processed_block'").Scan(&blockNumber)
	if err != nil {
		return 0, utils.NewAppError(utils.ErrCodeDatabase, "Failed to get latest processed block", err.Error())
	}
	return blockNumber, nil
}

// SetLatestProcessedBlock sets the latest processed block number
func (s *SQLiteStorage) SetLatestProcessedBlock(blockNumber uint64) error {
	_, err := s.db.Exec("UPDATE system_state SET value = ?, updated_at = ? WHERE key = 'latest_processed_block'",
		fmt.Sprintf("%d", blockNumber), time.Now())
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to set latest processed block", err.Error())
	}
	return nil
}

// --- Stub methods for Storage interface ---
func (s *SQLiteStorage) GetBlockHash(blockNumber uint64) (string, error) {
	var blockHash string
	err := s.db.QueryRow("SELECT block_hash FROM events WHERE block_number = ? LIMIT 1", blockNumber).Scan(&blockHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", utils.NewAppError(utils.ErrCodeDatabase, "Failed to get block hash", err.Error())
	}
	return blockHash, nil
}

func (s *SQLiteStorage) DeleteEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) (int, error) {
	res, err := s.db.ExecContext(ctx, "DELETE FROM events WHERE block_number >= ? AND block_number <= ?", fromBlock, toBlock)
	if err != nil {
		return 0, utils.NewAppError(utils.ErrCodeDatabase, "Failed to delete events by block range", err.Error())
	}
	count, err := res.RowsAffected()
	if err != nil {
		return 0, utils.NewAppError(utils.ErrCodeDatabase, "Failed to get rows affected", err.Error())
	}
	return int(count), nil
}

func (s *SQLiteStorage) LogEvent(ctx context.Context, eventType string, data map[string]interface{}) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to marshal log data", err.Error())
	}
	_, err = s.db.ExecContext(ctx, "INSERT INTO logs (type, data, created_at) VALUES (?, ?, ?)", eventType, string(dataJSON), time.Now())
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to insert log entry", err.Error())
	}
	return nil
}

func (s *SQLiteStorage) GetLogsByType(ctx context.Context, eventType string, limit int) ([]*models.LogEntry, error) {
	query := "SELECT id, type, data, created_at FROM logs WHERE type = ? ORDER BY created_at DESC"
	if limit > 0 {
		query += " LIMIT ?"
	}
	var rows *sql.Rows
	var err error
	if limit > 0 {
		rows, err = s.db.QueryContext(ctx, query, eventType, limit)
	} else {
		rows, err = s.db.QueryContext(ctx, query, eventType)
	}
	if err != nil {
		return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to query logs", err.Error())
	}
	defer rows.Close()

	logs := []*models.LogEntry{}
	for rows.Next() {
		var log models.LogEntry
		var dataJSON string
		err := rows.Scan(&log.ID, &log.Type, &dataJSON, &log.CreatedAt)
		if err != nil {
			return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to scan log entry", err.Error())
		}
		if err := json.Unmarshal([]byte(dataJSON), &log.Data); err != nil {
			return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to unmarshal log data", err.Error())
		}
		logs = append(logs, &log)
	}
	return logs, nil
}

func (s *SQLiteStorage) GetAllContracts(ctx context.Context) ([]*models.Contract, error) {
	return s.GetContracts(ctx, nil)
}

// StoreProcessingResult stores the result of processing in the database
func (s *SQLiteStorage) StoreProcessingResult(ctx context.Context, data map[string]interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to marshal processing result", err.Error())
	}
	_, err = s.db.ExecContext(ctx, "INSERT INTO processing_results (data, created_at) VALUES (?, ?)", string(jsonData), time.Now())
	if err != nil {
		return utils.NewAppError(utils.ErrCodeDatabase, "Failed to store processing result", err.Error())
	}
	return nil
}

// Health checks the health of the database connection
func (s *SQLiteStorage) Health() error {
	return s.Ping()
}

func (s *SQLiteStorage) GetHealth() *StorageHealth {
	return &StorageHealth{
		StorageType: "SQLite",
		Healthy:     s.Ping() == nil,
		Details:     map[string]string{"connection_string": s.config.ConnectionString},
		LastPing:    time.Now(),
	}
}

func (s *SQLiteStorage) GetStats() (*StorageStats, error) {
	return s.GetStorageStats()
}

func (s *SQLiteStorage) GetEventByHash(ctx context.Context, hash string) (*models.Event, error) {
	query := `
        SELECT id, block_number, block_hash, tx_hash, tx_index, log_index, address,
               event_name, event_signature, data, timestamp, processed, processed_at
        FROM events WHERE tx_hash = ?
    `
	row := s.db.QueryRowContext(ctx, query, hash)

	var event models.Event
	var dataJSON string
	var processedAt sql.NullTime

	err := row.Scan(&event.ID, &event.BlockNumber, &event.BlockHash, &event.TxHash,
		&event.TxIndex, &event.LogIndex, &event.Address, &event.EventName,
		&event.EventSig, &dataJSON, &event.Timestamp, &event.Processed, &processedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to get event by hash", err.Error())
	}

	if err := json.Unmarshal([]byte(dataJSON), &event.Data); err != nil {
		return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to unmarshal event data", err.Error())
	}

	if processedAt.Valid {
		event.ProcessedAt = &processedAt.Time
	}

	return &event, nil
}

func (s *SQLiteStorage) SearchEvents(ctx context.Context, filter models.EventFilter) ([]*models.Event, error) {
	query := `
        SELECT id, block_number, block_hash, tx_hash, tx_index, log_index, address,
               event_name, event_signature, data, timestamp, processed, processed_at
        FROM events WHERE 1=1
    `
	args := []interface{}{}

	if filter.Query != nil && *filter.Query != "" {
		query += " AND (event_name LIKE ? OR address LIKE ? OR tx_hash LIKE ?)"
		q := "%" + *filter.Query + "%"
		args = append(args, q, q, q)
	}

	// Add ordering and pagination
	query += " ORDER BY block_number DESC, log_index ASC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to search events", err.Error())
	}
	defer rows.Close()

	var events []*models.Event
	for rows.Next() {
		var event models.Event
		var dataJSON string
		var processedAt sql.NullTime

		err := rows.Scan(&event.ID, &event.BlockNumber, &event.BlockHash, &event.TxHash,
			&event.TxIndex, &event.LogIndex, &event.Address, &event.EventName,
			&event.EventSig, &dataJSON, &event.Timestamp, &event.Processed, &processedAt)
		if err != nil {
			return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to scan event", err.Error())
		}
		if err := json.Unmarshal([]byte(dataJSON), &event.Data); err != nil {
			return nil, utils.NewAppError(utils.ErrCodeDatabase, "Failed to unmarshal event data", err.Error())
		}
		if processedAt.Valid {
			event.ProcessedAt = &processedAt.Time
		}
		events = append(events, &event)
	}
	return events, nil
}

// Additional methods implementation continues...
