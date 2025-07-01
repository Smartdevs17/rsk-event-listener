package models

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// Event represents a blockchain event
type Event struct {
	ID          string                 `json:"id" db:"id"`
	BlockNumber uint64                 `json:"block_number" db:"block_number"`
	BlockHash   string                 `json:"block_hash" db:"block_hash"`
	TxHash      string                 `json:"tx_hash" db:"tx_hash"`
	TxIndex     uint                   `json:"tx_index" db:"tx_index"`
	LogIndex    uint                   `json:"log_index" db:"log_index"`
	Address     string                 `json:"address" db:"address"`
	EventName   string                 `json:"event_name" db:"event_name"`
	EventSig    string                 `json:"event_signature" db:"event_signature"`
	Data        map[string]interface{} `json:"data" db:"data"`
	Timestamp   time.Time              `json:"timestamp" db:"timestamp"`
	Processed   bool                   `json:"processed" db:"processed"`
	ProcessedAt *time.Time             `json:"processed_at,omitempty" db:"processed_at"`
}

// EventFilter for querying events
type EventFilter struct {
	ContractAddress *common.Address `json:"contract_address,omitempty"`
	EventName       *string         `json:"event_name,omitempty"`
	FromBlock       *uint64         `json:"from_block,omitempty"`
	ToBlock         *uint64         `json:"to_block,omitempty"`
	Processed       *bool           `json:"processed,omitempty"`
	Limit           int             `json:"limit,omitempty"`
	Offset          int             `json:"offset,omitempty"`
}
