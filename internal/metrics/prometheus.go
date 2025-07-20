package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PrometheusMetrics contains all Prometheus metrics for the RSK Event Listener
type PrometheusMetrics struct {
	// Event processing metrics
	EventsProcessedTotal    *prometheus.CounterVec
	BlocksProcessedTotal    prometheus.Counter
	EventProcessingDuration *prometheus.HistogramVec
	BlockProcessingDuration prometheus.Histogram

	// Connection and error metrics
	ConnectionErrorsTotal *prometheus.CounterVec
	RPCRequestsTotal      *prometheus.CounterVec
	RPCRequestDuration    *prometheus.HistogramVec

	// Blockchain metrics
	LatestProcessedBlock prometheus.Gauge
	BlocksBehind         prometheus.Gauge
	ReorgsDetectedTotal  prometheus.Counter
	ReorgsHandledTotal   prometheus.Counter
	ReorgDepth           *prometheus.HistogramVec

	// Storage metrics
	DatabaseOperationsTotal   *prometheus.CounterVec
	DatabaseOperationDuration *prometheus.HistogramVec
	DatabaseConnections       prometheus.Gauge

	// Notification metrics
	NotificationsSentTotal    *prometheus.CounterVec
	NotificationFailuresTotal *prometheus.CounterVec
	NotificationDuration      *prometheus.HistogramVec

	// API metrics
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec

	// Application health metrics
	ApplicationUptime prometheus.Gauge
	ComponentHealth   *prometheus.GaugeVec
	MemoryUsage       prometheus.Gauge
	GoroutineCount    prometheus.Gauge

	// Contract monitoring metrics
	ContractsMonitored prometheus.Gauge
	EventFiltersActive prometheus.Gauge
}

// NewPrometheusMetrics creates and registers all Prometheus metrics
func NewPrometheusMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{
		// Event processing metrics
		EventsProcessedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rsk_events_processed_total",
				Help: "Total number of events processed",
			},
			[]string{"contract_address", "event_name", "status"},
		),

		BlocksProcessedTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "rsk_blocks_processed_total",
				Help: "Total number of blocks processed",
			},
		),

		EventProcessingDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "rsk_event_processing_duration_seconds",
				Help:    "Time spent processing individual events",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"contract_address", "event_name"},
		),

		BlockProcessingDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "rsk_block_processing_duration_seconds",
				Help:    "Time spent processing individual blocks",
				Buckets: prometheus.DefBuckets,
			},
		),

		// Connection and error metrics
		ConnectionErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rsk_connection_errors_total",
				Help: "Total number of connection errors to RSK nodes",
			},
			[]string{"endpoint", "error_type"},
		),

		RPCRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rsk_rpc_requests_total",
				Help: "Total number of RPC requests made to RSK nodes",
			},
			[]string{"endpoint", "method", "status"},
		),

		RPCRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "rsk_rpc_request_duration_seconds",
				Help:    "Duration of RPC requests to RSK nodes",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"endpoint", "method"},
		),

		// Blockchain metrics
		LatestProcessedBlock: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "rsk_latest_processed_block",
				Help: "Latest block number processed by the event listener",
			},
		),

		BlocksBehind: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "rsk_blocks_behind",
				Help: "Number of blocks behind the latest chain block",
			},
		),

		ReorgsDetectedTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "rsk_reorgs_detected_total",
				Help: "Total number of blockchain reorganizations detected",
			},
		),

		ReorgsHandledTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "rsk_reorgs_handled_total",
				Help: "Total number of blockchain reorganizations successfully handled",
			},
		),

		ReorgDepth: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "rsk_reorg_depth",
				Help:    "Depth of blockchain reorganizations",
				Buckets: []float64{1, 2, 5, 10, 20, 50, 100},
			},
			[]string{"outcome"},
		),

		// Storage metrics
		DatabaseOperationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rsk_database_operations_total",
				Help: "Total number of database operations",
			},
			[]string{"operation", "table", "status"},
		),

		DatabaseOperationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "rsk_database_operation_duration_seconds",
				Help:    "Duration of database operations",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation", "table"},
		),

		DatabaseConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "rsk_database_connections",
				Help: "Number of active database connections",
			},
		),

		// Notification metrics
		NotificationsSentTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rsk_notifications_sent_total",
				Help: "Total number of notifications sent",
			},
			[]string{"channel", "type"},
		),

		NotificationFailuresTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rsk_notification_failures_total",
				Help: "Total number of failed notifications",
			},
			[]string{"channel", "type", "error"},
		),

		NotificationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "rsk_notification_duration_seconds",
				Help:    "Duration of notification delivery",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"channel", "type"},
		),

		// API metrics
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rsk_http_requests_total",
				Help: "Total number of HTTP requests received",
			},
			[]string{"method", "path", "status"},
		),

		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "rsk_http_request_duration_seconds",
				Help:    "Duration of HTTP requests",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),

		// Application health metrics
		ApplicationUptime: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "rsk_application_uptime_seconds",
				Help: "Application uptime in seconds",
			},
		),

		ComponentHealth: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "rsk_component_health",
				Help: "Health status of application components (1=healthy, 0=unhealthy)",
			},
			[]string{"component"},
		),

		MemoryUsage: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "rsk_memory_usage_bytes",
				Help: "Current memory usage in bytes",
			},
		),

		GoroutineCount: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "rsk_goroutines",
				Help: "Number of running goroutines",
			},
		),

		// Contract monitoring metrics
		ContractsMonitored: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "rsk_contracts_monitored",
				Help: "Number of contracts currently being monitored",
			},
		),

		EventFiltersActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "rsk_event_filters_active",
				Help: "Number of active event filters",
			},
		),
	}
}

// RecordEventProcessed records a processed event
func (m *PrometheusMetrics) RecordEventProcessed(contractAddress, eventName, status string) {
	m.EventsProcessedTotal.WithLabelValues(contractAddress, eventName, status).Inc()
}

// RecordEventProcessingDuration records the time taken to process an event
func (m *PrometheusMetrics) RecordEventProcessingDuration(contractAddress, eventName string, duration time.Duration) {
	m.EventProcessingDuration.WithLabelValues(contractAddress, eventName).Observe(duration.Seconds())
}

// RecordBlockProcessed records a processed block
func (m *PrometheusMetrics) RecordBlockProcessed() {
	m.BlocksProcessedTotal.Inc()
}

// RecordBlockProcessingDuration records the time taken to process a block
func (m *PrometheusMetrics) RecordBlockProcessingDuration(duration time.Duration) {
	m.BlockProcessingDuration.Observe(duration.Seconds())
}

// RecordConnectionError records a connection error
func (m *PrometheusMetrics) RecordConnectionError(endpoint, errorType string) {
	m.ConnectionErrorsTotal.WithLabelValues(endpoint, errorType).Inc()
}

// RecordRPCRequest records an RPC request
func (m *PrometheusMetrics) RecordRPCRequest(endpoint, method, status string, duration time.Duration) {
	m.RPCRequestsTotal.WithLabelValues(endpoint, method, status).Inc()
	m.RPCRequestDuration.WithLabelValues(endpoint, method).Observe(duration.Seconds())
}

// UpdateLatestProcessedBlock updates the latest processed block metric
func (m *PrometheusMetrics) UpdateLatestProcessedBlock(blockNumber uint64) {
	m.LatestProcessedBlock.Set(float64(blockNumber))
}

// UpdateBlocksBehind updates the blocks behind metric
func (m *PrometheusMetrics) UpdateBlocksBehind(behind uint64) {
	m.BlocksBehind.Set(float64(behind))
}

// RecordReorgDetected records a detected reorganization
func (m *PrometheusMetrics) RecordReorgDetected() {
	m.ReorgsDetectedTotal.Inc()
}

// RecordReorgHandled records a successfully handled reorganization
func (m *PrometheusMetrics) RecordReorgHandled(depth int) {
	m.ReorgsHandledTotal.Inc()
	m.ReorgDepth.WithLabelValues("success").Observe(float64(depth))
}

// RecordReorgFailed records a failed reorganization handling
func (m *PrometheusMetrics) RecordReorgFailed(depth int) {
	m.ReorgDepth.WithLabelValues("failed").Observe(float64(depth))
}

// RecordDatabaseOperation records a database operation
func (m *PrometheusMetrics) RecordDatabaseOperation(operation, table, status string, duration time.Duration) {
	m.DatabaseOperationsTotal.WithLabelValues(operation, table, status).Inc()
	m.DatabaseOperationDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// UpdateDatabaseConnections updates the database connections metric
func (m *PrometheusMetrics) UpdateDatabaseConnections(count int) {
	m.DatabaseConnections.Set(float64(count))
}

// RecordNotificationSent records a sent notification
func (m *PrometheusMetrics) RecordNotificationSent(channel, notificationType string, duration time.Duration) {
	m.NotificationsSentTotal.WithLabelValues(channel, notificationType).Inc()
	m.NotificationDuration.WithLabelValues(channel, notificationType).Observe(duration.Seconds())
}

// RecordNotificationFailure records a failed notification
func (m *PrometheusMetrics) RecordNotificationFailure(channel, notificationType, errorType string) {
	m.NotificationFailuresTotal.WithLabelValues(channel, notificationType, errorType).Inc()
}

// RecordHTTPRequest records an HTTP request
func (m *PrometheusMetrics) RecordHTTPRequest(method, path, status string, duration time.Duration) {
	m.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

// UpdateApplicationUptime updates the application uptime metric
func (m *PrometheusMetrics) UpdateApplicationUptime(startTime time.Time) {
	m.ApplicationUptime.Set(time.Since(startTime).Seconds())
}

// UpdateComponentHealth updates the health status of a component
func (m *PrometheusMetrics) UpdateComponentHealth(component string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	m.ComponentHealth.WithLabelValues(component).Set(value)
}

// UpdateMemoryUsage updates the memory usage metric
func (m *PrometheusMetrics) UpdateMemoryUsage(bytes uint64) {
	m.MemoryUsage.Set(float64(bytes))
}

// UpdateGoroutineCount updates the goroutine count metric
func (m *PrometheusMetrics) UpdateGoroutineCount(count int) {
	m.GoroutineCount.Set(float64(count))
}

// UpdateContractsMonitored updates the number of monitored contracts
func (m *PrometheusMetrics) UpdateContractsMonitored(count int) {
	m.ContractsMonitored.Set(float64(count))
}

// UpdateEventFiltersActive updates the number of active event filters
func (m *PrometheusMetrics) UpdateEventFiltersActive(count int) {
	m.EventFiltersActive.Set(float64(count))
}
