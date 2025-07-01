package connection

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/smartdevs17/rsk-event-listener/internal/config"
	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
)

// ConnectionPool manages multiple connection managers for load balancing
type ConnectionPool struct {
	managers     []Manager
	currentIndex int
	mu           sync.RWMutex
	config       *config.RSKConfig
	logger       *logrus.Logger
	closed       bool
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(cfg *config.RSKConfig) *ConnectionPool {
	pool := &ConnectionPool{
		config: cfg,
		logger: utils.GetLogger(),
	}

	// Create multiple connection managers for load balancing
	maxConnections := cfg.MaxConnections
	if maxConnections <= 0 {
		maxConnections = 1
	}

	for i := 0; i < maxConnections; i++ {
		manager := NewConnectionManager(cfg)
		pool.managers = append(pool.managers, manager)
	}

	pool.logger.Info("Connection pool created", "size", len(pool.managers))
	return pool
}

// GetManager returns the next available connection manager (round-robin)
func (cp *ConnectionPool) GetManager() Manager {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.closed {
		return nil
	}

	if len(cp.managers) == 0 {
		return nil
	}

	manager := cp.managers[cp.currentIndex]
	cp.currentIndex = (cp.currentIndex + 1) % len(cp.managers)
	return manager
}

// GetHealthyManager returns a healthy connection manager
func (cp *ConnectionPool) GetHealthyManager(ctx context.Context) (Manager, error) {
	// Try to find a healthy manager
	for i := 0; i < len(cp.managers); i++ {
		manager := cp.GetManager()
		if manager == nil {
			continue
		}

		if manager.IsConnected() {
			return manager, nil
		}

		// Try to perform health check with timeout
		checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := manager.HealthCheckWithContext(checkCtx)
		cancel()

		if err == nil {
			return manager, nil
		}

		cp.logger.Warn("Manager failed health check", "error", err)
	}

	return nil, utils.NewAppError(utils.ErrCodeConnection, "No healthy connection managers available", "")
}

// HealthCheck checks all managers in the pool
func (cp *ConnectionPool) HealthCheck() map[int]error {
	cp.mu.RLock()
	managers := make([]Manager, len(cp.managers))
	copy(managers, cp.managers)
	cp.mu.RUnlock()

	results := make(map[int]error)
	var wg sync.WaitGroup

	for i, manager := range managers {
		wg.Add(1)
		go func(index int, mgr Manager) {
			defer wg.Done()
			results[index] = mgr.HealthCheck()
		}(i, manager)
	}

	wg.Wait()
	return results
}

// GetStats returns statistics for all managers
func (cp *ConnectionPool) GetStats() map[int]ConnectionStats {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	stats := make(map[int]ConnectionStats)
	for i, manager := range cp.managers {
		stats[i] = manager.Stats()
	}
	return stats
}

// Close closes all connection managers in the pool
func (cp *ConnectionPool) Close() error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.closed = true
	var lastErr error

	for i, manager := range cp.managers {
		if err := manager.Close(); err != nil {
			cp.logger.Error("Failed to close connection manager", "index", i, "error", err)
			lastErr = err
		}
	}

	cp.managers = nil
	cp.logger.Info("Connection pool closed")
	return lastErr
}

// Size returns the number of managers in the pool
func (cp *ConnectionPool) Size() int {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return len(cp.managers)
}

// ActiveConnections returns the number of active connections
func (cp *ConnectionPool) ActiveConnections() int {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	active := 0
	for _, manager := range cp.managers {
		if manager.IsConnected() {
			active++
		}
	}
	return active
}
