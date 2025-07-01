package integration

import (
	"context"
	"testing"
	"time"

	"github.com/smartdevs17/rsk-event-listener/internal/config"
	"github.com/smartdevs17/rsk-event-listener/internal/connection"
	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
)

func TestConnectionManager(t *testing.T) {
	// Load test configuration
	cfg := &config.RSKConfig{
		NodeURL:        "https://public-node.testnet.rsk.co",
		NetworkID:      31, // RSK Testnet
		BackupNodes:    []string{"https://public-node.testnet.rsk.co"},
		RequestTimeout: 30 * time.Second,
		RetryAttempts:  3,
		RetryDelay:     5 * time.Second,
		MaxConnections: 2,
	}

	// Initialize logger
	utils.InitLogger("info", "text", "stdout", "")

	t.Run("Basic Connection", func(t *testing.T) {
		manager := connection.NewConnectionManager(cfg)
		defer manager.Close()

		// Test getting client
		client, err := manager.GetClient()
		if err != nil {
			t.Fatalf("Failed to get client: %v", err)
		}

		if client == nil {
			t.Fatal("Client is nil")
		}

		// Test connection status
		if !manager.IsConnected() {
			t.Fatal("Manager should be connected")
		}

		t.Logf("✓ Successfully connected to RSK node")
	})

	t.Run("Health Check", func(t *testing.T) {
		manager := connection.NewConnectionManager(cfg)
		defer manager.Close()

		err := manager.HealthCheck()
		if err != nil {
			t.Fatalf("Health check failed: %v", err)
		}

		stats := manager.Stats()
		if stats.NetworkID != 31 {
			t.Errorf("Expected network ID 31, got %d", stats.NetworkID)
		}

		if stats.LatestBlock == 0 {
			t.Error("Latest block should not be 0")
		}

		t.Logf("✓ Health check passed - Network ID: %d, Latest Block: %d",
			stats.NetworkID, stats.LatestBlock)
	})

	t.Run("Network ID", func(t *testing.T) {
		manager := connection.NewConnectionManager(cfg)
		defer manager.Close()

		networkID, err := manager.GetNetworkID()
		if err != nil {
			t.Fatalf("Failed to get network ID: %v", err)
		}

		expectedNetworkID := uint64(31) // RSK Testnet
		if networkID != expectedNetworkID {
			t.Errorf("Expected network ID %d, got %d", expectedNetworkID, networkID)
		}

		t.Logf("✓ Network ID verified: %d", networkID)
	})

	t.Run("Latest Block", func(t *testing.T) {
		manager := connection.NewConnectionManager(cfg)
		defer manager.Close()

		blockNumber, err := manager.GetLatestBlockNumber()
		if err != nil {
			t.Fatalf("Failed to get latest block number: %v", err)
		}

		if blockNumber == 0 {
			t.Error("Block number should not be 0")
		}

		t.Logf("✓ Latest block number: %d", blockNumber)
	})

	t.Run("Connection Stats", func(t *testing.T) {
		manager := connection.NewConnectionManager(cfg)
		defer manager.Close()

		// Make a few requests to generate stats
		_, err := manager.GetClient()
		if err != nil {
			t.Fatalf("Failed to get client: %v", err)
		}

		err = manager.HealthCheck()
		if err != nil {
			t.Fatalf("Health check failed: %v", err)
		}

		stats := manager.Stats()
		if stats.TotalRequests == 0 {
			t.Error("Total requests should be > 0")
		}

		if stats.CurrentURL == "" {
			t.Error("Current URL should not be empty")
		}

		if !stats.IsHealthy {
			t.Error("Connection should be healthy")
		}

		t.Logf("✓ Connection stats: Requests=%d, URL=%s, Healthy=%v",
			stats.TotalRequests, stats.CurrentURL, stats.IsHealthy)
	})
}

func TestRSKClient(t *testing.T) {
	cfg := &config.RSKConfig{
		NodeURL:        "https://public-node.testnet.rsk.co",
		NetworkID:      31,
		RequestTimeout: 30 * time.Second,
		RetryAttempts:  3,
		RetryDelay:     5 * time.Second,
	}

	utils.InitLogger("info", "text", "stdout", "")

	t.Run("RSK Client Operations", func(t *testing.T) {
		manager := connection.NewConnectionManager(cfg)
		defer manager.Close()

		rskClient := connection.NewRSKClient(manager)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Test getting latest block
		block, err := rskClient.GetLatestBlock(ctx)
		if err != nil {
			t.Fatalf("Failed to get latest block: %v", err)
		}

		if block == nil {
			t.Fatal("Block is nil")
		}

		if block.Number().Uint64() == 0 {
			t.Error("Block number should not be 0")
		}

		t.Logf("✓ Latest block: %d (hash: %s)",
			block.Number().Uint64(), block.Hash().Hex())

		// Test getting specific block
		specificBlock, err := rskClient.GetBlockByNumber(ctx, block.Number())
		if err != nil {
			t.Fatalf("Failed to get specific block: %v", err)
		}

		if specificBlock.Hash() != block.Hash() {
			t.Error("Block hashes should match")
		}

		t.Logf("✓ Specific block retrieved successfully")
	})
}

func TestConnectionPool(t *testing.T) {
	cfg := &config.RSKConfig{
		NodeURL:        "https://public-node.testnet.rsk.co",
		NetworkID:      31,
		RequestTimeout: 30 * time.Second,
		RetryAttempts:  3,
		RetryDelay:     5 * time.Second,
		MaxConnections: 3,
	}

	utils.InitLogger("info", "text", "stdout", "")

	t.Run("Pool Management", func(t *testing.T) {
		pool := connection.NewConnectionPool(cfg)
		defer pool.Close()

		if pool.Size() != 3 {
			t.Errorf("Expected pool size 3, got %d", pool.Size())
		}

		// Test getting managers
		for i := 0; i < 5; i++ {
			manager := pool.GetManager()
			if manager == nil {
				t.Fatal("Manager should not be nil")
			}
		}

		t.Logf("✓ Pool management working - Size: %d", pool.Size())
	})

	t.Run("Pool Health Check", func(t *testing.T) {
		pool := connection.NewConnectionPool(cfg)
		defer pool.Close()

		results := pool.HealthCheck()
		if len(results) != pool.Size() {
			t.Errorf("Expected %d health check results, got %d", pool.Size(), len(results))
		}

		healthyCount := 0
		for i, err := range results {
			if err == nil {
				healthyCount++
			} else {
				t.Logf("Manager %d health check failed: %v", i, err)
			}
		}

		t.Logf("✓ Pool health check completed - Healthy: %d/%d", healthyCount, pool.Size())
	})

	t.Run("Pool Stats", func(t *testing.T) {
		pool := connection.NewConnectionPool(cfg)
		defer pool.Close()

		stats := pool.GetStats()
		if len(stats) != pool.Size() {
			t.Errorf("Expected %d stat entries, got %d", pool.Size(), len(stats))
		}

		activeConnections := pool.ActiveConnections()
		t.Logf("✓ Pool stats retrieved - Active connections: %d/%d",
			activeConnections, pool.Size())
	})

	t.Run("Healthy Manager Selection", func(t *testing.T) {
		pool := connection.NewConnectionPool(cfg)
		defer pool.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		manager, err := pool.GetHealthyManager(ctx)
		if err != nil {
			t.Fatalf("Failed to get healthy manager: %v", err)
		}

		if manager == nil {
			t.Fatal("Healthy manager should not be nil")
		}

		// Test the healthy manager
		err = manager.HealthCheck()
		if err != nil {
			t.Errorf("Healthy manager failed health check: %v", err)
		}

		t.Logf("✓ Healthy manager selection working")
	})
}
