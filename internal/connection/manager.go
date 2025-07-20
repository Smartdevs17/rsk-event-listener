package connection

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
	"github.com/smartdevs17/rsk-event-listener/internal/config"
	"github.com/smartdevs17/rsk-event-listener/internal/metrics"
	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
)

// Manager defines the connection manager interface
type Manager interface {
	GetClient() (*ethclient.Client, error)
	GetClientWithContext(ctx context.Context) (*ethclient.Client, error)
	HealthCheck() error
	HealthCheckWithContext(ctx context.Context) error
	GetNetworkID() (uint64, error)
	GetLatestBlockNumber() (uint64, error)
	IsConnected() bool
	Close() error
	Stats() ConnectionStats
}

// ConnectionManager implements the Manager interface
type ConnectionManager struct {
	config          *config.RSKConfig
	primaryURL      string
	backupURLs      []string
	currentIndex    int
	client          *ethclient.Client
	mu              sync.RWMutex
	logger          *logrus.Logger
	stats           ConnectionStats
	lastHealthCheck time.Time
	isHealthy       bool
	metricsManager  *metrics.Manager
}

// ConnectionStats holds connection statistics
type ConnectionStats struct {
	TotalRequests   uint64    `json:"total_requests"`
	FailedRequests  uint64    `json:"failed_requests"`
	Reconnects      uint64    `json:"reconnects"`
	CurrentURL      string    `json:"current_url"`
	LastConnectedAt time.Time `json:"last_connected_at"`
	LastHealthCheck time.Time `json:"last_health_check"`
	IsHealthy       bool      `json:"is_healthy"`
	NetworkID       uint64    `json:"network_id"`
	LatestBlock     uint64    `json:"latest_block"`
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(cfg *config.RSKConfig) *ConnectionManager {
	urls := []string{cfg.NodeURL}
	urls = append(urls, cfg.BackupNodes...)

	return &ConnectionManager{
		config:       cfg,
		primaryURL:   cfg.NodeURL,
		backupURLs:   cfg.BackupNodes,
		currentIndex: 0,
		logger:       utils.GetLogger(),
		stats: ConnectionStats{
			CurrentURL: cfg.NodeURL,
		},
	}
}

// GetClient returns the current client connection
func (cm *ConnectionManager) GetClient() (*ethclient.Client, error) {
	return cm.GetClientWithContext(context.Background())
}

// GetClientWithContext returns the current client with context
func (cm *ConnectionManager) GetClientWithContext(ctx context.Context) (*ethclient.Client, error) {
	cm.mu.RLock()
	client := cm.client
	cm.mu.RUnlock()

	if client == nil {
		return cm.connect(ctx)
	}

	// Test the connection if it's been a while since last health check
	if time.Since(cm.lastHealthCheck) > time.Minute {
		if err := cm.quickHealthCheck(ctx, client); err != nil {
			cm.logger.Warn("Client health check failed, reconnecting", "error", err)
			return cm.reconnect(ctx)
		}
	}

	cm.stats.TotalRequests++
	return client, nil
}

// connect establishes a new connection
func (cm *ConnectionManager) connect(ctx context.Context) (*ethclient.Client, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	urls := cm.getAllURLs()

	for attempt := 0; attempt < cm.config.RetryAttempts; attempt++ {
		for i, url := range urls {
			cm.logger.Info("Attempting connection", "url", url, "attempt", attempt+1)

			client, err := cm.dialWithTimeout(ctx, url)
			if err != nil {
				cm.logger.Warn("Connection failed", "url", url, "error", err)
				cm.stats.FailedRequests++
				continue
			}

			// Verify the connection works
			if err := cm.quickHealthCheck(ctx, client); err != nil {
				client.Close()
				cm.logger.Warn("Health check failed after connection", "url", url, "error", err)
				continue
			}

			cm.client = client
			cm.currentIndex = i
			cm.stats.CurrentURL = url
			cm.stats.LastConnectedAt = time.Now()
			cm.isHealthy = true
			cm.lastHealthCheck = time.Now()

			cm.logger.Info("Successfully connected to RSK node", "url", url)
			return client, nil
		}

		if attempt < cm.config.RetryAttempts-1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(cm.config.RetryDelay):
				// Continue to next attempt
			}
		}
	}

	err := utils.NewAppError(utils.ErrCodeConnection, "Failed to connect to any RSK node",
		"All connection attempts exhausted")
	return nil, err
}

// reconnect tries to reconnect to RSK nodes
func (cm *ConnectionManager) reconnect(ctx context.Context) (*ethclient.Client, error) {
	cm.mu.Lock()
	if cm.client != nil {
		cm.client.Close()
		cm.client = nil
	}
	cm.stats.Reconnects++
	cm.mu.Unlock()

	return cm.connect(ctx)
}

// dialWithTimeout creates a connection with timeout
func (cm *ConnectionManager) dialWithTimeout(ctx context.Context, url string) (*ethclient.Client, error) {
	dialCtx, cancel := context.WithTimeout(ctx, cm.config.RequestTimeout)
	defer cancel()

	return ethclient.DialContext(dialCtx, url)
}

// quickHealthCheck performs a quick health check
func (cm *ConnectionManager) quickHealthCheck(ctx context.Context, client *ethclient.Client) error {
	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := client.NetworkID(checkCtx)
	return err
}

// HealthCheck performs a comprehensive health check
func (cm *ConnectionManager) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return cm.HealthCheckWithContext(ctx)
}

// HealthCheckWithContext performs a comprehensive health check with context
func (cm *ConnectionManager) HealthCheckWithContext(ctx context.Context) error {
	client, err := cm.GetClientWithContext(ctx)
	if err != nil {
		cm.isHealthy = false
		return err
	}

	// Check network ID
	networkID, err := client.NetworkID(ctx)
	if err != nil {
		cm.isHealthy = false
		return utils.NewAppError(utils.ErrCodeConnection, "Failed to get network ID", err.Error())
	}

	expectedNetworkID := uint64(cm.config.NetworkID)
	if networkID.Uint64() != expectedNetworkID {
		cm.isHealthy = false
		return utils.NewAppError(utils.ErrCodeConnection,
			"Network ID mismatch",
			fmt.Sprintf("expected %d, got %d", expectedNetworkID, networkID.Uint64()))
	}

	// Check latest block
	blockNumber, err := client.BlockNumber(ctx)
	if err != nil {
		cm.isHealthy = false
		return utils.NewAppError(utils.ErrCodeConnection, "Failed to get latest block", err.Error())
	}

	// Update stats
	cm.mu.Lock()
	cm.stats.NetworkID = networkID.Uint64()
	cm.stats.LatestBlock = blockNumber
	cm.stats.LastHealthCheck = time.Now()
	cm.stats.IsHealthy = true
	cm.lastHealthCheck = time.Now()
	cm.isHealthy = true
	cm.mu.Unlock()

	cm.logger.Info("Health check passed",
		"network_id", networkID.Uint64(),
		"latest_block", blockNumber,
		"url", cm.stats.CurrentURL)

	return nil
}

// GetNetworkID returns the network ID
func (cm *ConnectionManager) GetNetworkID() (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cm.GetClientWithContext(ctx)
	if err != nil {
		return 0, err
	}

	networkID, err := client.NetworkID(ctx)
	if err != nil {
		return 0, err
	}

	return networkID.Uint64(), nil
}

// GetLatestBlockNumber returns the latest block number
func (cm *ConnectionManager) GetLatestBlockNumber() (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := cm.GetClientWithContext(ctx)
	if err != nil {
		return 0, err
	}

	blockNumber, err := client.BlockNumber(ctx)
	if err != nil {
		return 0, err
	}

	cm.mu.Lock()
	cm.stats.LatestBlock = blockNumber
	cm.mu.Unlock()

	return blockNumber, nil
}

// IsConnected returns whether the manager is connected
func (cm *ConnectionManager) IsConnected() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.client != nil && cm.isHealthy
}

// Close closes the connection
func (cm *ConnectionManager) Close() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.client != nil {
		cm.client.Close()
		cm.client = nil
	}

	cm.isHealthy = false
	cm.logger.Info("Connection manager closed")
	return nil
}

// Stats returns connection statistics
func (cm *ConnectionManager) Stats() ConnectionStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.stats
}

// getAllURLs returns all available URLs starting from current index
func (cm *ConnectionManager) getAllURLs() []string {
	urls := []string{cm.primaryURL}
	urls = append(urls, cm.backupURLs...)

	// Start from current index for load balancing
	if cm.currentIndex > 0 && cm.currentIndex < len(urls) {
		rotated := make([]string, len(urls))
		copy(rotated, urls[cm.currentIndex:])
		copy(rotated[len(urls)-cm.currentIndex:], urls[:cm.currentIndex])
		return rotated
	}

	return urls
}

// Call invokes a contract method call
func (cm *ConnectionManager) Call(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	start := time.Now()
	client, err := cm.GetClientWithContext(ctx)
	endpoint := ""
	if client != nil {
		endpoint = cm.stats.CurrentURL
	}
	if err != nil {
		status := "error"
		if cm.metricsManager != nil {
			cm.metricsManager.GetPrometheusMetrics().RecordConnectionError(endpoint, "rpc_call_failed")
			cm.metricsManager.GetPrometheusMetrics().RecordRPCRequest(endpoint, method, status, time.Since(start))
		}
		return err
	}
	// Get the underlying RPC client
	rpcClient := client.Client() // ethclient.Client.Client() returns *rpc.Client
	if rpcClient == nil {
		status := "error"
		if cm.metricsManager != nil {
			cm.metricsManager.GetPrometheusMetrics().RecordConnectionError(endpoint, "rpc_client_nil")
			cm.metricsManager.GetPrometheusMetrics().RecordRPCRequest(endpoint, method, status, time.Since(start))
		}
		return fmt.Errorf("underlying RPC client is nil")
	}
	callErr := rpcClient.CallContext(ctx, result, method, args...)
	status := "success"
	if callErr != nil {
		status = "error"
		if cm.metricsManager != nil {
			cm.metricsManager.GetPrometheusMetrics().RecordConnectionError(endpoint, "rpc_call_failed")
		}
	}
	if cm.metricsManager != nil {
		cm.metricsManager.GetPrometheusMetrics().RecordRPCRequest(endpoint, method, status, time.Since(start))
	}
	return callErr
}
