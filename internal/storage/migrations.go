package storage

import (
	"time"
)

// Migration represents a database migration
type Migration struct {
	ID          int       `db:"id"`
	Version     string    `db:"version"`
	Description string    `db:"description"`
	SQL         string    `db:"sql"`
	AppliedAt   time.Time `db:"applied_at"`
	Checksum    string    `db:"checksum"`
}

// MigrationManager handles database migrations
type MigrationManager interface {
	GetAppliedMigrations() ([]*Migration, error)
	ApplyMigration(migration *Migration) error
	CreateMigrationTable() error
	GetPendingMigrations() ([]*Migration, error)
	ApplyPendingMigrations() error
}

// GetSQLiteMigrations returns SQLite migration scripts
func GetSQLiteMigrations() []*Migration {
	return []*Migration{
		{
			Version:     "001",
			Description: "Create events table",
			SQL: `
				CREATE TABLE IF NOT EXISTS events (
					id TEXT PRIMARY KEY,
					block_number INTEGER NOT NULL,
					block_hash TEXT NOT NULL,
					tx_hash TEXT NOT NULL,
					tx_index INTEGER NOT NULL,
					log_index INTEGER NOT NULL,
					address TEXT NOT NULL,
					event_name TEXT NOT NULL,
					event_signature TEXT NOT NULL,
					data TEXT NOT NULL, -- JSON
					timestamp DATETIME NOT NULL,
					processed BOOLEAN DEFAULT FALSE,
					processed_at DATETIME,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);
				
				CREATE INDEX IF NOT EXISTS idx_events_block_number ON events(block_number);
				CREATE INDEX IF NOT EXISTS idx_events_address ON events(address);
				CREATE INDEX IF NOT EXISTS idx_events_event_name ON events(event_name);
				CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
				CREATE INDEX IF NOT EXISTS idx_events_processed ON events(processed);
				CREATE UNIQUE INDEX IF NOT EXISTS idx_events_unique ON events(block_hash, tx_hash, log_index);
			`,
		},
		{
			Version:     "002",
			Description: "Create contracts table",
			SQL: `
				CREATE TABLE IF NOT EXISTS contracts (
					address TEXT PRIMARY KEY,
					name TEXT NOT NULL,
					abi TEXT NOT NULL,
					start_block INTEGER NOT NULL DEFAULT 0,
					active BOOLEAN DEFAULT TRUE,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);
				
				CREATE INDEX IF NOT EXISTS idx_contracts_active ON contracts(active);
				CREATE INDEX IF NOT EXISTS idx_contracts_start_block ON contracts(start_block);
			`,
		},
		{
			Version:     "003",
			Description: "Create notifications table",
			SQL: `
				CREATE TABLE IF NOT EXISTS notifications (
					id TEXT PRIMARY KEY,
					type TEXT NOT NULL,
					event_id TEXT NOT NULL,
					title TEXT NOT NULL,
					message TEXT NOT NULL,
					data TEXT, -- JSON
					target TEXT NOT NULL,
					status TEXT DEFAULT 'pending',
					attempts INTEGER DEFAULT 0,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					sent_at DATETIME,
					error TEXT,
					FOREIGN KEY (event_id) REFERENCES events (id)
				);
				
				CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);
				CREATE INDEX IF NOT EXISTS idx_notifications_type ON notifications(type);
				CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at);
			`,
		},
		{
			Version:     "004",
			Description: "Create block_processing_status table",
			SQL: `
				CREATE TABLE IF NOT EXISTS block_processing_status (
					block_number INTEGER PRIMARY KEY,
					status TEXT NOT NULL DEFAULT 'pending',
					started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					completed_at DATETIME,
					events_found INTEGER DEFAULT 0,
					events_saved INTEGER DEFAULT 0,
					error TEXT,
					processing_time INTEGER -- duration in milliseconds
				);
				
				CREATE INDEX IF NOT EXISTS idx_block_status ON block_processing_status(status);
				CREATE INDEX IF NOT EXISTS idx_block_started_at ON block_processing_status(started_at);
			`,
		},
		{
			Version:     "005",
			Description: "Create system_state table",
			SQL: `
				CREATE TABLE IF NOT EXISTS system_state (
					key TEXT PRIMARY KEY,
					value TEXT NOT NULL,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);
				
				-- Insert default latest processed block
				INSERT OR IGNORE INTO system_state (key, value) VALUES ('latest_processed_block', '0');
			`,
		},
		{
			Version:     "006",
			Description: "Create migrations table",
			SQL: `
				CREATE TABLE IF NOT EXISTS migrations (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					version TEXT NOT NULL UNIQUE,
					description TEXT NOT NULL,
					checksum TEXT NOT NULL,
					applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);
			`,
		},
	}
}

// GetPostgresMigrations returns PostgreSQL migration scripts
func GetPostgresMigrations() []*Migration {
	return []*Migration{
		{
			Version:     "001",
			Description: "Create events table",
			SQL: `
				CREATE TABLE IF NOT EXISTS events (
					id TEXT PRIMARY KEY,
					block_number BIGINT NOT NULL,
					block_hash TEXT NOT NULL,
					tx_hash TEXT NOT NULL,
					tx_index INTEGER NOT NULL,
					log_index INTEGER NOT NULL,
					address TEXT NOT NULL,
					event_name TEXT NOT NULL,
					event_signature TEXT NOT NULL,
					data JSONB NOT NULL,
					timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
					processed BOOLEAN DEFAULT FALSE,
					processed_at TIMESTAMP WITH TIME ZONE,
					created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
				);
				
				CREATE INDEX IF NOT EXISTS idx_events_block_number ON events(block_number);
				CREATE INDEX IF NOT EXISTS idx_events_address ON events(address);
				CREATE INDEX IF NOT EXISTS idx_events_event_name ON events(event_name);
				CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
				CREATE INDEX IF NOT EXISTS idx_events_processed ON events(processed);
				CREATE UNIQUE INDEX IF NOT EXISTS idx_events_unique ON events(block_hash, tx_hash, log_index);
				CREATE INDEX IF NOT EXISTS idx_events_data_gin ON events USING GIN(data);
			`,
		},
		{
			Version:     "002",
			Description: "Create contracts table",
			SQL: `
				CREATE TABLE IF NOT EXISTS contracts (
					address TEXT PRIMARY KEY,
					name TEXT NOT NULL,
					abi TEXT NOT NULL,
					start_block BIGINT NOT NULL DEFAULT 0,
					active BOOLEAN DEFAULT TRUE,
					created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
					updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
				);
				
				CREATE INDEX IF NOT EXISTS idx_contracts_active ON contracts(active);
				CREATE INDEX IF NOT EXISTS idx_contracts_start_block ON contracts(start_block);
			`,
		},
		{
			Version:     "003",
			Description: "Create notifications table",
			SQL: `
				CREATE TABLE IF NOT EXISTS notifications (
					id TEXT PRIMARY KEY,
					type TEXT NOT NULL,
					event_id TEXT NOT NULL,
					title TEXT NOT NULL,
					message TEXT NOT NULL,
					data JSONB,
					target TEXT NOT NULL,
					status TEXT DEFAULT 'pending',
					attempts INTEGER DEFAULT 0,
					created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
					sent_at TIMESTAMP WITH TIME ZONE,
					error TEXT,
					CONSTRAINT fk_notifications_event FOREIGN KEY (event_id) REFERENCES events (id)
				);
				
				CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);
				CREATE INDEX IF NOT EXISTS idx_notifications_type ON notifications(type);
				CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at);
			`,
		},
		{
			Version:     "004",
			Description: "Create block_processing_status table",
			SQL: `
				CREATE TABLE IF NOT EXISTS block_processing_status (
					block_number BIGINT PRIMARY KEY,
					status TEXT NOT NULL DEFAULT 'pending',
					started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
					completed_at TIMESTAMP WITH TIME ZONE,
					events_found INTEGER DEFAULT 0,
					events_saved INTEGER DEFAULT 0,
					error TEXT,
					processing_time INTERVAL
				);
				
				CREATE INDEX IF NOT EXISTS idx_block_status ON block_processing_status(status);
				CREATE INDEX IF NOT EXISTS idx_block_started_at ON block_processing_status(started_at);
			`,
		},
		{
			Version:     "005",
			Description: "Create system_state table",
			SQL: `
				CREATE TABLE IF NOT EXISTS system_state (
					key TEXT PRIMARY KEY,
					value TEXT NOT NULL,
					updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
				);
				
				-- Insert default latest processed block
				INSERT INTO system_state (key, value) VALUES ('latest_processed_block', '0')
				ON CONFLICT (key) DO NOTHING;
			`,
		},
		{
			Version:     "006",
			Description: "Create migrations table",
			SQL: `
				CREATE TABLE IF NOT EXISTS migrations (
					id SERIAL PRIMARY KEY,
					version TEXT NOT NULL UNIQUE,
					description TEXT NOT NULL,
					checksum TEXT NOT NULL,
					applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
				);
			`,
		},
	}
}
