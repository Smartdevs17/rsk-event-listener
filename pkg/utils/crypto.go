package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// GenerateID generates a random hex ID
func GenerateID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// IsValidAddress checks if a string is a valid Ethereum address
func IsValidAddress(address string) bool {
	return common.IsHexAddress(address)
}

// NormalizeAddress normalizes an address to lowercase with 0x prefix
func NormalizeAddress(address string) string {
	if !strings.HasPrefix(address, "0x") {
		address = "0x" + address
	}
	return strings.ToLower(address)
}

// GetEventSignature returns the keccak256 hash of an event signature
func GetEventSignature(signature string) string {
	hash := crypto.Keccak256Hash([]byte(signature))
	return hash.Hex()
}

// FormatBlockNumber formats a block number for display
func FormatBlockNumber(blockNumber uint64) string {
	return fmt.Sprintf("0x%x", blockNumber)
}

// ParseBlockNumber parses a hex block number string
func ParseBlockNumber(blockNumberHex string) (uint64, error) {
	if strings.HasPrefix(blockNumberHex, "0x") {
		blockNumberHex = blockNumberHex[2:]
	}

	var blockNumber uint64
	_, err := fmt.Sscanf(blockNumberHex, "%x", &blockNumber)
	if err != nil {
		return 0, err
	}

	return blockNumber, nil
}

// CreateEventID creates a unique ID for an event
func CreateEventID(blockHash, txHash string, logIndex uint) string {
	data := fmt.Sprintf("%s-%s-%d", blockHash, txHash, logIndex)
	hash := crypto.Keccak256Hash([]byte(data))
	return hash.Hex()
}
