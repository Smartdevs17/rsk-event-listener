package metrics

import (
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
)

// Manager handles all application metrics
type Manager struct {
	prometheus *PrometheusMetrics
	logger     *logrus.Entry
	startTime  time.Time
}

// NewManager creates a new metrics manager
func NewManager() *Manager {
	return &Manager{
		prometheus: NewPrometheusMetrics(),
		logger:     logrus.WithField("component", "metrics"),
		startTime:  time.Now(),
	}
}

// GetPrometheusMetrics returns the Prometheus metrics instance
func (m *Manager) GetPrometheusMetrics() *PrometheusMetrics {
	return m.prometheus
}

// UpdateSystemMetrics updates system-level metrics like memory and goroutines
func (m *Manager) UpdateSystemMetrics() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	m.prometheus.UpdateMemoryUsage(memStats.Alloc)
	m.prometheus.UpdateGoroutineCount(runtime.NumGoroutine())
	m.prometheus.UpdateApplicationUptime(m.startTime)
}
