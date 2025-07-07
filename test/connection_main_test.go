package integration

import (
	"context"
	"testing"
	"time"

	"github.com/smartdevs17/rsk-event-listener/internal/config"
	"github.com/smartdevs17/rsk-event-listener/internal/connection"
	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
	"github.com/stretchr/testify/require"
)

func TestConnectionManager(t *testing.T) {
	t.Log("RSK Event Listener - Connection Manager Test")

	// Initialize logger
	err := utils.InitLogger("info", "text", "stdout", "")
	require.NoError(t, err, "Failed to initialize logger")

	// Load configuration (empty string means default config path)
	cfg, err := config.Load("")
	require.NoError(t, err, "Failed to load config")

	t.Log("=== Testing Connection Manager ===")

	// Test 1: Basic Connection
	t.Log("Testing basic connection...")
	manager := connection.NewConnectionManager(&cfg.RSK)
	defer manager.Close()

	_, err = manager.GetClient()
	require.NoError(t, err, "Failed to get client")
	t.Logf("✓ Connected to RSK node: %s", cfg.RSK.NodeURL)

	// Test 2: Health Check
	t.Log("Testing health check...")
	err = manager.HealthCheck()
	require.NoError(t, err, "Health check failed")

	stats := manager.Stats()
	t.Logf("✓ Health check passed: Network ID: %d, Latest Block: %d, Connection URL: %s, Is Healthy: %v",
		stats.NetworkID, stats.LatestBlock, stats.CurrentURL, stats.IsHealthy)

	// Test 3: Network Operations
	t.Log("Testing network operations...")
	networkID, err := manager.GetNetworkID()
	require.NoError(t, err, "Failed to get network ID")
	t.Logf("✓ Network ID: %d", networkID)

	latestBlock, err := manager.GetLatestBlockNumber()
	require.NoError(t, err, "Failed to get latest block")
	t.Logf("✓ Latest block: %d", latestBlock)

	// Test 4: RSK Client
	t.Log("Testing RSK client wrapper...")
	rskClient := connection.NewRSKClient(manager)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	block, err := rskClient.GetLatestBlock(ctx)
	require.NoError(t, err, "Failed to get latest block via RSK client")
	t.Logf("✓ RSK Client - Latest block: %d (hash: %s)", block.Number().Uint64(), block.Hash().Hex()[:10]+"...")

	// Test 5: Connection Pool
	t.Log("Testing connection pool...")
	pool := connection.NewConnectionPool(&cfg.RSK)
	defer pool.Close()

	t.Logf("✓ Connection pool created with %d managers", pool.Size())

	// Test pool health
	healthResults := pool.HealthCheck()
	healthyCount := 0
	for _, err := range healthResults {
		if err == nil {
			healthyCount++
		}
	}
	t.Logf("✓ Pool health check: %d/%d managers healthy", healthyCount, pool.Size())

	activeConnections := pool.ActiveConnections()
	t.Logf("✓ Active connections: %d/%d", activeConnections, pool.Size())

	// Test 6: Connection Statistics
	t.Log("Connection statistics:")
	finalStats := manager.Stats()
	t.Logf("  - Total Requests: %d", finalStats.TotalRequests)
	t.Logf("  - Failed Requests: %d", finalStats.FailedRequests)
	t.Logf("  - Reconnects: %d", finalStats.Reconnects)
	t.Logf("  - Last Health Check: %s", finalStats.LastHealthCheck.Format(time.RFC3339))
	t.Logf("  - Connection Status: %v", manager.IsConnected())

	t.Log("🎉 All connection manager tests passed! Ready for Step 3.")
}
