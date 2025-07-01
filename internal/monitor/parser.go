// File: internal/monitor/parser.go
package monitor

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sirupsen/logrus"
	"github.com/smartdevs17/rsk-event-listener/internal/models"
	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
)

// EventParser handles parsing of blockchain events
type EventParser struct {
	config *MonitorConfig
	logger *logrus.Logger

	// Cached ABIs
	abiCache map[common.Address]*abi.ABI
}

// ParsedEvent contains parsed event data
type ParsedEvent struct {
	Event     *models.Event
	EventName string
	Arguments map[string]interface{}
}

// NewEventParser creates a new event parser
func NewEventParser(config *MonitorConfig) *EventParser {
	return &EventParser{
		config:   config,
		logger:   utils.GetLogger(),
		abiCache: make(map[common.Address]*abi.ABI),
	}
}

// ParseLog parses a blockchain log into an event
func (ep *EventParser) ParseLog(log types.Log, contract *models.Contract) (*models.Event, error) {
	if contract == nil {
		return nil, utils.NewAppError(utils.ErrCodeValidation, "Contract is nil", "")
	}

	// Get or parse ABI
	contractABI, err := ep.getContractABI(contract)
	if err != nil {
		return nil, err
	}

	// Find matching event
	eventName, event, err := ep.findEventByTopic(contractABI, log.Topics[0])
	if err != nil {
		// Unknown event, create raw event
		return ep.createRawEvent(log, contract), nil
	}

	// Parse event data
	parsedData, err := ep.parseEventData(event, log)
	if err != nil {
		ep.logger.Warn("Failed to parse event data", "error", err, "event", eventName)
		return ep.createRawEvent(log, contract), nil
	}

	// Create event model
	id, _ := utils.GenerateID()
	modelEvent := &models.Event{
		ID:          id,
		BlockNumber: log.BlockNumber,
		BlockHash:   log.BlockHash.Hex(),
		TxHash:      log.TxHash.Hex(),
		TxIndex:     uint(log.TxIndex),
		LogIndex:    uint(log.Index),
		Address:     log.Address.Hex(),
		EventName:   eventName,
		EventSig:    event.Sig,
		Data:        parsedData,
		Timestamp:   time.Now(),
		Processed:   false,
	}

	return modelEvent, nil
}

// ParseEventData parses event data from log
func (ep *EventParser) parseEventData(event *abi.Event, log types.Log) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Parse indexed topics
	topicIndex := 1 // Skip first topic (event signature)
	for _, input := range event.Inputs {
		if input.Indexed {
			if topicIndex >= len(log.Topics) {
				return nil, fmt.Errorf("insufficient topics for indexed parameter %s", input.Name)
			}

			value, err := ep.parseTopicValue(input.Type, log.Topics[topicIndex])
			if err != nil {
				ep.logger.Warn("Failed to parse topic value",
					"parameter", input.Name,
					"type", input.Type.String(),
					"error", err)
				value = log.Topics[topicIndex].Hex()
			}

			result[input.Name] = value
			topicIndex++
		}
	}

	// Parse non-indexed data
	if len(log.Data) > 0 {
		nonIndexedInputs := make(abi.Arguments, 0)
		for _, input := range event.Inputs {
			if !input.Indexed {
				nonIndexedInputs = append(nonIndexedInputs, input)
			}
		}

		if len(nonIndexedInputs) > 0 {
			values, err := nonIndexedInputs.Unpack(log.Data)
			if err != nil {
				return nil, fmt.Errorf("failed to unpack event data: %w", err)
			}

			for i, input := range nonIndexedInputs {
				if i < len(values) {
					result[input.Name] = ep.convertValue(values[i])
				}
			}
		}
	}

	return result, nil
}

// getContractABI gets or parses contract ABI
func (ep *EventParser) getContractABI(contract *models.Contract) (*abi.ABI, error) {
	// Check cache
	if cachedABI, exists := ep.abiCache[contract.Address]; exists {
		return cachedABI, nil
	}

	// Parse ABI
	if contract.ABI == "" {
		return nil, utils.NewAppError(utils.ErrCodeValidation, "Contract ABI is empty", "")
	}

	parsedABI, err := abi.JSON(strings.NewReader(contract.ABI))
	if err != nil {
		return nil, utils.NewAppError(utils.ErrCodeValidation, "Failed to parse ABI", err.Error())
	}

	// Cache ABI
	ep.abiCache[contract.Address] = &parsedABI

	return &parsedABI, nil
}

// findEventByTopic finds event by topic hash
func (ep *EventParser) findEventByTopic(contractABI *abi.ABI, topic common.Hash) (string, *abi.Event, error) {
	for name, event := range contractABI.Events {
		if event.ID == topic {
			return name, &event, nil
		}
	}
	return "", nil, utils.NewAppError(utils.ErrCodeNotFound, "Event not found", topic.Hex())
}

// parseTopicValue parses a topic value based on type
func (ep *EventParser) parseTopicValue(typ abi.Type, topic common.Hash) (interface{}, error) {
	switch typ.T {
	case abi.AddressTy:
		return common.BytesToAddress(topic.Bytes()), nil
	case abi.IntTy, abi.UintTy:
		return new(big.Int).SetBytes(topic.Bytes()), nil
	case abi.BoolTy:
		return topic.Big().Cmp(big.NewInt(0)) != 0, nil
	case abi.BytesTy, abi.FixedBytesTy:
		return topic.Bytes(), nil
	case abi.StringTy:
		// For indexed strings, we only get the hash
		return topic.Hex(), nil
	default:
		return topic.Hex(), nil
	}
}

// convertValue converts ABI values to JSON-serializable types
func (ep *EventParser) convertValue(value interface{}) interface{} {
	switch v := value.(type) {
	case *big.Int:
		return v.String()
	case common.Address:
		return v.Hex()
	case []byte:
		return hex.EncodeToString(v)
	case bool:
		return v
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// createRawEvent creates a raw event for unparseable logs
func (ep *EventParser) createRawEvent(log types.Log, contract *models.Contract) *models.Event {
	topics := make([]string, len(log.Topics))
	for i, topic := range log.Topics {
		topics[i] = topic.Hex()
	}

	rawData := map[string]interface{}{
		"topics": topics,
		"data":   hex.EncodeToString(log.Data),
		"raw":    true,
	}

	id, _ := utils.GenerateID()
	return &models.Event{
		ID:          id,
		BlockNumber: log.BlockNumber,
		BlockHash:   log.BlockHash.Hex(),
		TxHash:      log.TxHash.Hex(),
		TxIndex:     uint(log.TxIndex),
		LogIndex:    uint(log.Index),
		Address:     log.Address.Hex(),
		EventName:   "Unknown",
		EventSig:    log.Topics[0].Hex(),
		Data:        rawData,
		Timestamp:   time.Now(),
		Processed:   false,
	}
}

// logToRawData converts log to raw data
func (ep *EventParser) logToRawData(log types.Log) map[string]interface{} {
	topics := make([]string, len(log.Topics))
	for i, topic := range log.Topics {
		topics[i] = topic.Hex()
	}

	return map[string]interface{}{
		"address":           log.Address.Hex(),
		"topics":            topics,
		"data":              hex.EncodeToString(log.Data),
		"block_number":      log.BlockNumber,
		"transaction_hash":  log.TxHash.Hex(),
		"transaction_index": log.TxIndex,
		"block_hash":        log.BlockHash.Hex(),
		"log_index":         log.Index,
		"removed":           log.Removed,
	}
}
