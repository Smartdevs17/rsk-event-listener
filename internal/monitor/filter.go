// File: internal/monitor/filter.go
package monitor

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sirupsen/logrus"
	"github.com/smartdevs17/rsk-event-listener/internal/models"
	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
)

// EventFilter handles event filtering operations
type EventFilter struct {
	config *MonitorConfig
	logger *logrus.Logger
}

// FilterCriteria defines filtering criteria for events
type FilterCriteria struct {
	ContractAddresses []common.Address `json:"contract_addresses,omitempty"`
	EventNames        []string         `json:"event_names,omitempty"`
	EventSignatures   []string         `json:"event_signatures,omitempty"`
	BlockRange        *BlockRange      `json:"block_range,omitempty"`
	Topics            [][]string       `json:"topics,omitempty"`
	RequiredFields    []string         `json:"required_fields,omitempty"`
}

// BlockRange defines a block range for filtering
type BlockRange struct {
	FromBlock *uint64 `json:"from_block,omitempty"`
	ToBlock   *uint64 `json:"to_block,omitempty"`
}

// NewEventFilter creates a new event filter
func NewEventFilter(config *MonitorConfig) *EventFilter {
	return &EventFilter{
		config: config,
		logger: utils.GetLogger(),
	}
}

// FilterEvents filters events based on criteria
func (ef *EventFilter) FilterEvents(events []*models.Event, criteria *FilterCriteria) []*models.Event {
	if criteria == nil {
		return events
	}

	filtered := make([]*models.Event, 0)

	for _, event := range events {
		if ef.matchesCriteria(event, criteria) {
			filtered = append(filtered, event)
		}
	}

	return filtered
}

// matchesCriteria checks if an event matches the filter criteria
func (ef *EventFilter) matchesCriteria(event *models.Event, criteria *FilterCriteria) bool {
	// Filter by contract addresses
	if len(criteria.ContractAddresses) > 0 {
		found := false
		for _, addr := range criteria.ContractAddresses {
			if strings.EqualFold(event.Address, addr.Hex()) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by event names
	if len(criteria.EventNames) > 0 {
		found := false
		for _, name := range criteria.EventNames {
			if strings.EqualFold(event.EventName, name) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by event signatures
	if len(criteria.EventSignatures) > 0 {
		found := false
		for _, sig := range criteria.EventSignatures {
			if event.EventSig == sig {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by block range
	if criteria.BlockRange != nil {
		if criteria.BlockRange.FromBlock != nil && event.BlockNumber < *criteria.BlockRange.FromBlock {
			return false
		}
		if criteria.BlockRange.ToBlock != nil && event.BlockNumber > *criteria.BlockRange.ToBlock {
			return false
		}
	}

	// Filter by required fields
	if len(criteria.RequiredFields) > 0 {
		for _, field := range criteria.RequiredFields {
			if !ef.hasRequiredField(event, field) {
				return false
			}
		}
	}

	return true
}

// hasRequiredField checks if event has a required field
func (ef *EventFilter) hasRequiredField(event *models.Event, field string) bool {
	if event.Data == nil {
		return false
	}

	_, exists := event.Data[field]
	return exists
}

// CreateEventFilters creates event filter specifications from contract ABI
func (ef *EventFilter) CreateEventFilters(contract *models.Contract) ([]EventFilterSpec, error) {
	if contract.ABI == "" {
		return []EventFilterSpec{}, nil
	}

	parsedABI, err := abi.JSON(strings.NewReader(contract.ABI))
	if err != nil {
		return nil, utils.NewAppError(utils.ErrCodeValidation, "Failed to parse ABI", err.Error())
	}

	filters := make([]EventFilterSpec, 0, len(parsedABI.Events))

	for eventName, event := range parsedABI.Events {
		filter := EventFilterSpec{
			EventName: eventName,
			EventSig:  event.ID.Hex(),
			Topics:    ef.extractTopics(event),
		}

		// Extract required data fields
		for _, input := range event.Inputs {
			if !input.Indexed {
				filter.RequiredData = append(filter.RequiredData, input.Name)
			}
		}

		filters = append(filters, filter)
	}

	return filters, nil
}

// extractTopics extracts topic information from ABI event
func (ef *EventFilter) extractTopics(event abi.Event) []string {
	topics := []string{event.ID.Hex()} // First topic is always the event signature

	for _, input := range event.Inputs {
		if input.Indexed {
			topics = append(topics, fmt.Sprintf("%s:%s", input.Name, input.Type.String()))
		}
	}

	return topics
}

// ValidateFilter validates filter criteria
func (ef *EventFilter) ValidateFilter(criteria *FilterCriteria) error {
	if criteria == nil {
		return utils.NewAppError(utils.ErrCodeValidation, "Filter criteria is nil", "")
	}

	// Validate block range
	if criteria.BlockRange != nil {
		if criteria.BlockRange.FromBlock != nil && criteria.BlockRange.ToBlock != nil {
			if *criteria.BlockRange.FromBlock > *criteria.BlockRange.ToBlock {
				return utils.NewAppError(utils.ErrCodeValidation, "Invalid block range", "fromBlock > toBlock")
			}
		}
	}

	// Validate contract addresses
	for _, addr := range criteria.ContractAddresses {
		if addr == (common.Address{}) {
			return utils.NewAppError(utils.ErrCodeValidation, "Invalid contract address", addr.Hex())
		}
	}

	return nil
}
