// package main

// import (
// 	"fmt"
// 	"log"
// 	"time"

// 	"github.com/ethereum/go-ethereum/common"
// 	"github.com/smartdevs17/rsk-event-listener/internal/config"
// 	"github.com/smartdevs17/rsk-event-listener/internal/models"
// 	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
// )

// func main() {
// 	fmt.Println("RSK Event Listener - Foundation Test")

// 	// Test 1: Load configuration
// 	fmt.Println("\n=== Testing Configuration ===")
// 	cfg, err := config.Load("")
// 	if err != nil {
// 		log.Fatalf("Failed to load config: %v", err)
// 	}
// 	fmt.Printf("✓ Config loaded successfully")
// 	fmt.Printf("  - App: %s v%s\n", cfg.App.Name, cfg.App.Version)
// 	fmt.Printf("  - RSK Node: %s\n", cfg.RSK.NodeURL)
// 	fmt.Printf("  - Storage: %s\n", cfg.Storage.Type)

// 	// Test 2: Initialize logger
// 	fmt.Println("\n=== Testing Logger ===")
// 	err = utils.InitLogger(cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.Output, cfg.Logging.File)
// 	if err != nil {
// 		log.Fatalf("Failed to initialize logger: %v", err)
// 	}
// 	logger := utils.GetLogger()
// 	logger.Info("✓ Logger initialized successfully")

// 	// Test 3: Test models
// 	fmt.Println("\n=== Testing Models ===")

// 	// Create a test event
// 	event := &models.Event{
// 		ID:          "test-event-1",
// 		BlockNumber: 12345,
// 		BlockHash:   "0x1234567890abcdef",
// 		TxHash:      "0xabcdef1234567890",
// 		TxIndex:     0,
// 		LogIndex:    0,
// 		Address:     "0x1234567890123456789012345678901234567890",
// 		EventName:   "Transfer",
// 		EventSig:    "Transfer(address,address,uint256)",
// 		Data:        map[string]interface{}{"from": "0x123", "to": "0x456", "value": "1000"},
// 		Timestamp:   time.Now(),
// 		Processed:   false,
// 	}
// 	fmt.Printf("✓ Event model created: %s\n", event.EventName)

// 	// Create a test contract
// 	contract := &models.Contract{
// 		Address:    common.HexToAddress("0x1234567890123456789012345678901234567890"),
// 		Name:       "TestToken",
// 		ABI:        `[{"type":"event","name":"Transfer"}]`,
// 		StartBlock: 12345,
// 		Active:     true,
// 		CreatedAt:  time.Now(),
// 		UpdatedAt:  time.Now(),
// 	}
// 	fmt.Printf("✓ Contract model created: %s\n", contract.Name)

// 	// Test 4: Test utilities
// 	fmt.Println("\n=== Testing Utilities ===")

// 	// Test ID generation
// 	id, err := utils.GenerateID()
// 	if err != nil {
// 		log.Fatalf("Failed to generate ID: %v", err)
// 	}
// 	fmt.Printf("✓ Generated ID: %s\n", id)

// 	// Test address validation
// 	validAddr := "0x1234567890123456789012345678901234567890"
// 	if utils.IsValidAddress(validAddr) {
// 		fmt.Printf("✓ Address validation works: %s\n", validAddr)
// 	}

// 	// Test event signature
// 	signature := utils.GetEventSignature("Transfer(address,address,uint256)")
// 	fmt.Printf("✓ Event signature: %s\n", signature)

// 	// Test 5: Test error handling
// 	fmt.Println("\n=== Testing Error Handling ===")
// 	appErr := utils.NewAppError(utils.ErrCodeValidation, "Test validation error", "This is a test")
// 	fmt.Printf("✓ App error created: %s\n", appErr.Error())

// 	fmt.Println("\n🎉 All foundation tests passed! Ready for Step 2.")
// }
