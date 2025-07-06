// package main

// import (
// 	"context"
// 	"fmt"
// 	"log"
// 	"time"

// 	"github.com/smartdevs17/rsk-event-listener/internal/config"
// 	"github.com/smartdevs17/rsk-event-listener/internal/connection"
// 	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
// )

// func main() {
// 	fmt.Println("RSK Event Listener - Connection Manager Test")

// 	// Initialize logger
// 	err := utils.InitLogger("info", "text", "stdout", "")
// 	if err != nil {
// 		log.Fatalf("Failed to initialize logger: %v", err)
// 	}

// 	// Load configuration (empty string means default config path)
// 	cfg, err := config.Load("")
// 	if err != nil {
// 		log.Fatalf("Failed to load config: %v", err)
// 	}

// 	fmt.Println("\n=== Testing Connection Manager ===")

// 	// Test 1: Basic Connection
// 	fmt.Println("Testing basic connection...")
// 	manager := connection.NewConnectionManager(&cfg.RSK)
// 	defer manager.Close()

// 	_, err = manager.GetClient()
// 	if err != nil {
// 		log.Fatalf("Failed to get client: %v", err)
// 	}
// 	fmt.Printf("✓ Connected to RSK node: %s\n", cfg.RSK.NodeURL)

// 	// Test 2: Health Check
// 	fmt.Println("\nTesting health check...")
// 	err = manager.HealthCheck()
// 	if err != nil {
// 		log.Fatalf("Health check failed: %v", err)
// 	}

// 	stats := manager.Stats()
// 	fmt.Printf("✓ Health check passed:\n")
// 	fmt.Printf("  - Network ID: %d\n", stats.NetworkID)
// 	fmt.Printf("  - Latest Block: %d\n", stats.LatestBlock)
// 	fmt.Printf("  - Connection URL: %s\n", stats.CurrentURL)
// 	fmt.Printf("  - Is Healthy: %v\n", stats.IsHealthy)

// 	// Test 3: Network Operations
// 	fmt.Println("\nTesting network operations...")
// 	networkID, err := manager.GetNetworkID()
// 	if err != nil {
// 		log.Fatalf("Failed to get network ID: %v", err)
// 	}
// 	fmt.Printf("✓ Network ID: %d\n", networkID)

// 	latestBlock, err := manager.GetLatestBlockNumber()
// 	if err != nil {
// 		log.Fatalf("Failed to get latest block: %v", err)
// 	}
// 	fmt.Printf("✓ Latest block: %d\n", latestBlock)

// 	// Test 4: RSK Client
// 	fmt.Println("\nTesting RSK client wrapper...")
// 	rskClient := connection.NewRSKClient(manager)

// 	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
// 	defer cancel()

// 	block, err := rskClient.GetLatestBlock(ctx)
// 	if err != nil {
// 		log.Fatalf("Failed to get latest block via RSK client: %v", err)
// 	}
// 	fmt.Printf("✓ RSK Client - Latest block: %d (hash: %s)\n",
// 		block.Number().Uint64(), block.Hash().Hex()[:10]+"...")

// 	// Test 5: Connection Pool
// 	fmt.Println("\nTesting connection pool...")
// 	pool := connection.NewConnectionPool(&cfg.RSK)
// 	defer pool.Close()

// 	fmt.Printf("✓ Connection pool created with %d managers\n", pool.Size())

// 	// Test pool health
// 	healthResults := pool.HealthCheck()
// 	healthyCount := 0
// 	for _, err := range healthResults {
// 		if err == nil {
// 			healthyCount++
// 		}
// 	}
// 	fmt.Printf("✓ Pool health check: %d/%d managers healthy\n", healthyCount, pool.Size())

// 	activeConnections := pool.ActiveConnections()
// 	fmt.Printf("✓ Active connections: %d/%d\n", activeConnections, pool.Size())

// 	// Test 6: Connection Statistics
// 	fmt.Println("\nConnection statistics:")
// 	finalStats := manager.Stats()
// 	fmt.Printf("  - Total Requests: %d\n", finalStats.TotalRequests)
// 	fmt.Printf("  - Failed Requests: %d\n", finalStats.FailedRequests)
// 	fmt.Printf("  - Reconnects: %d\n", finalStats.Reconnects)
// 	fmt.Printf("  - Last Health Check: %s\n", finalStats.LastHealthCheck.Format(time.RFC3339))
// 	fmt.Printf("  - Connection Status: %v\n", manager.IsConnected())

// 	fmt.Println("\n🎉 All connection manager tests passed! Ready for Step 3.")
// }
