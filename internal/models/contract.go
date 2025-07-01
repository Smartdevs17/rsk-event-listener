package models

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// Contract represents a smart contract to monitor
type Contract struct {
	Address    common.Address `json:"address" db:"address"`
	Name       string         `json:"name" db:"name"`
	ABI        string         `json:"abi" db:"abi"`
	StartBlock uint64         `json:"start_block" db:"start_block"`
	Active     bool           `json:"active" db:"active"`
	CreatedAt  time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at" db:"updated_at"`
}

// EventSignature represents an event signature we're monitoring
type EventSignature struct {
	Signature  string `json:"signature" db:"signature"`
	EventName  string `json:"event_name" db:"event_name"`
	ContractID string `json:"contract_id" db:"contract_id"`
}
