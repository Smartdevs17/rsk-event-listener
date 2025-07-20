// File: internal/server/server.go
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"github.com/smartdevs17/rsk-event-listener/internal/metrics"
	"github.com/smartdevs17/rsk-event-listener/internal/models"
	"github.com/smartdevs17/rsk-event-listener/internal/monitor"
	"github.com/smartdevs17/rsk-event-listener/internal/notification"
	"github.com/smartdevs17/rsk-event-listener/internal/processor"
	"github.com/smartdevs17/rsk-event-listener/internal/storage"
	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
)

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port          int           `json:"port"`
	Host          string        `json:"host"`
	ReadTimeout   time.Duration `json:"read_timeout"`
	WriteTimeout  time.Duration `json:"write_timeout"`
	EnableMetrics bool          `json:"enable_metrics"`
	EnableHealth  bool          `json:"enable_health"`
}

// HTTPServer represents the HTTP server
type HTTPServer struct {
	config         *ServerConfig
	server         *http.Server
	router         *mux.Router
	storage        storage.Storage
	monitor        *monitor.EventMonitor
	processor      *processor.EventProcessor
	notification   *notification.NotificationManager
	metricsManager *metrics.Manager
	logger         *logrus.Logger
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(
	config *ServerConfig,
	storage storage.Storage,
	monitor *monitor.EventMonitor,
	processor *processor.EventProcessor,
	notification *notification.NotificationManager,
	metricsManager *metrics.Manager,
) (*HTTPServer, error) {

	server := &HTTPServer{
		config:         config,
		storage:        storage,
		monitor:        monitor,
		processor:      processor,
		notification:   notification,
		metricsManager: metricsManager,
		logger:         utils.GetLogger(),
	}

	// Setup router
	server.setupRouter()

	// Create HTTP server
	server.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Handler:      server.router,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	}

	return server, nil
}

// setupRouter sets up the HTTP routes
func (s *HTTPServer) setupRouter() {
	s.router = mux.NewRouter()

	// Middleware
	s.router.Use(s.loggingMiddleware)
	s.router.Use(s.corsMiddleware)
	if s.metricsManager != nil {
		s.router.Use(s.metricsMiddleware)
	}

	// API routes
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Health check endpoint
	if s.config.EnableHealth {
		api.HandleFunc("/health", s.healthHandler).Methods("GET")
		api.HandleFunc("/health/detailed", s.detailedHealthHandler).Methods("GET")
	}

	// Metrics endpoint
	if s.config.EnableMetrics {
		s.router.Handle("/metrics", promhttp.Handler())
		api.HandleFunc("/stats", s.statsHandler).Methods("GET")
	}

	// Event endpoints
	api.HandleFunc("/events", s.listEventsHandler).Methods("GET")
	api.HandleFunc("/events/{hash}", s.getEventHandler).Methods("GET")
	api.HandleFunc("/events/search", s.searchEventsHandler).Methods("GET")

	// Contract endpoints
	api.HandleFunc("/contracts", s.listContractsHandler).Methods("GET")
	api.HandleFunc("/contracts", s.addContractHandler).Methods("POST")
	api.HandleFunc("/contracts/{address}", s.getContractHandler).Methods("GET")
	api.HandleFunc("/contracts/{address}", s.removeContractHandler).Methods("DELETE")

	// Processor endpoints
	api.HandleFunc("/processor/rules", s.listRulesHandler).Methods("GET")
	api.HandleFunc("/processor/rules", s.addRuleHandler).Methods("POST")
	api.HandleFunc("/processor/rules/{id}", s.getRuleHandler).Methods("GET")
	api.HandleFunc("/processor/rules/{id}", s.updateRuleHandler).Methods("PUT")
	api.HandleFunc("/processor/rules/{id}", s.removeRuleHandler).Methods("DELETE")

	// Notification endpoints
	api.HandleFunc("/notifications/channels", s.listChannelsHandler).Methods("GET")
	api.HandleFunc("/notifications/channels", s.addChannelHandler).Methods("POST")
	api.HandleFunc("/notifications/channels/{id}", s.removeChannelHandler).Methods("DELETE")
	api.HandleFunc("/notifications/test", s.testNotificationHandler).Methods("POST")

	// Monitor endpoints
	api.HandleFunc("/monitor/status", s.monitorStatusHandler).Methods("GET")
	api.HandleFunc("/monitor/start", s.startMonitorHandler).Methods("POST")
	api.HandleFunc("/monitor/stop", s.stopMonitorHandler).Methods("POST")

	// Static files for simple web interface (optional)
	s.router.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/static/")))
}

// Start starts the HTTP server
func (s *HTTPServer) Start() error {
	s.logger.Info("Starting HTTP server", map[string]interface{}{
		"address":         s.server.Addr,
		"metrics_enabled": s.config.EnableMetrics,
	})

	// Immediately update system and component metrics so they appear on first scrape
	if s.metricsManager != nil {
		s.metricsManager.UpdateSystemMetrics()
		if s.storage != nil {
			health := s.storage.GetHealth()
			s.metricsManager.GetPrometheusMetrics().UpdateComponentHealth("storage", health.Healthy)
		}
		if s.monitor != nil {
			health := s.monitor.GetHealth()
			s.metricsManager.GetPrometheusMetrics().UpdateComponentHealth("monitor", health.Healthy)
		}
		if s.processor != nil {
			health := s.processor.GetHealth()
			s.metricsManager.GetPrometheusMetrics().UpdateComponentHealth("processor", health.Healthy)
		}
		if s.notification != nil {
			health := s.notification.GetHealth()
			s.metricsManager.GetPrometheusMetrics().UpdateComponentHealth("notification", health.Healthy)
		}
	}

	// Start periodic updater as before
	if s.metricsManager != nil {
		go s.systemMetricsUpdater()
	}

	// Create a channel to receive startup errors
	errChan := make(chan error, 1)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server error", map[string]interface{}{"error": err})
			errChan <- err
		}
	}()

	// Give the server a moment to start and check for immediate binding errors
	select {
	case err := <-errChan:
		return fmt.Errorf("failed to start HTTP server: %w", err)
	case <-time.After(100 * time.Millisecond):
		// Server started successfully
		return nil
	}
}

// systemMetricsUpdater updates system metrics periodically
func (s *HTTPServer) systemMetricsUpdater() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.metricsManager.UpdateSystemMetrics()

			// Update component health metrics
			if s.metricsManager != nil {
				s.metricsManager.UpdateSystemMetrics()
				// Update component health metrics immediately
				if s.storage != nil {
					health := s.storage.GetHealth()
					s.metricsManager.GetPrometheusMetrics().UpdateComponentHealth("storage", health.Healthy)
				}
				if s.monitor != nil {
					health := s.monitor.GetHealth()
					s.metricsManager.GetPrometheusMetrics().UpdateComponentHealth("monitor", health.Healthy)
				}
				if s.processor != nil {
					health := s.processor.GetHealth()
					s.metricsManager.GetPrometheusMetrics().UpdateComponentHealth("processor", health.Healthy)
				}
				if s.notification != nil {
					health := s.notification.GetHealth()
					s.metricsManager.GetPrometheusMetrics().UpdateComponentHealth("notification", health.Healthy)
				}
			}
		}
	}
}

// Stop stops the HTTP server
func (s *HTTPServer) Stop() error {
	s.logger.Info("Stopping HTTP server")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}

// Middleware

// loggingMiddleware logs HTTP requests
func (s *HTTPServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Call the next handler
		next.ServeHTTP(w, r)

		// Log the request
		s.logger.Info("HTTP request", map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"duration":   time.Since(start),
			"user_agent": r.UserAgent(),
			"remote_ip":  r.RemoteAddr,
		})
	})
}

// corsMiddleware handles CORS
func (s *HTTPServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Health Handlers

// healthHandler returns basic health status
func (s *HTTPServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"status":          "healthy",
		"timestamp":       time.Now().UTC().Format(time.RFC3339Nano),
		"version":         "1.0.0",
		"metrics_enabled": s.config.EnableMetrics,
	}
	s.writeJSON(w, http.StatusOK, resp)
}

// detailedHealthHandler returns detailed health status
func (s *HTTPServer) detailedHealthHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   "1.0.0",
		"components": map[string]interface{}{
			"storage":      s.storage.GetHealth(),
			"monitor":      s.monitor.GetHealth().Healthy,
			"processor":    s.processor.GetHealth().Healthy,
			"notification": s.notification.GetHealth().Healthy,
		},
	}

	s.writeJSON(w, http.StatusOK, health)
}

// statsHandler returns application statistics
func (s *HTTPServer) statsHandler(w http.ResponseWriter, r *http.Request) {
	storageStats, err := s.storage.GetStats()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to retrieve storage stats", err)
		return
	}

	stats := map[string]interface{}{
		"timestamp":       time.Now(),
		"storage":         storageStats,
		"monitor":         s.monitor.GetStats(),
		"processor":       s.processor.GetStats(),
		"notification":    s.notification.GetStats(),
		"metrics_enabled": s.config.EnableMetrics,
	}

	s.writeJSON(w, http.StatusOK, stats)
}

// Event Handlers

// listEventsHandler lists recent events
func (s *HTTPServer) listEventsHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50 // default
	offset := 0 // default

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	// Get events from storage
	filter := models.EventFilter{
		Limit:  limit,
		Offset: offset,
		// Add more filter fields if needed
	}

	events, err := s.storage.GetEvents(r.Context(), filter)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to retrieve events", err)
		return
	}

	response := map[string]interface{}{
		"events": events,
		"limit":  limit,
		"offset": offset,
		"total":  len(events),
	}

	s.writeJSON(w, http.StatusOK, response)
}

// getEventHandler gets a specific event by hash
func (s *HTTPServer) getEventHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hash := vars["hash"]

	event, err := s.storage.GetEventByHash(r.Context(), hash)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "Event not found", err)
		return
	}

	s.writeJSON(w, http.StatusOK, event)
}

// searchEventsHandler searches events
func (s *HTTPServer) searchEventsHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	contract := r.URL.Query().Get("contract")
	eventName := r.URL.Query().Get("event")

	if query == "" && contract == "" && eventName == "" {
		s.writeError(w, http.StatusBadRequest, "Search query is required", nil)
		return
	}

	filter := models.EventFilter{
		Limit:  50,
		Offset: 0,
	}

	if query != "" {
		filter.Query = &query
	}
	if contract != "" {
		addr := common.HexToAddress(contract)
		filter.ContractAddress = &addr
	}
	if eventName != "" {
		filter.EventName = &eventName
	}

	events, err := s.storage.SearchEvents(r.Context(), filter)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Search failed", err)
		return
	}

	response := map[string]interface{}{
		"events": events,
		"query":  query,
		"total":  len(events),
	}

	s.writeJSON(w, http.StatusOK, response)
}

// Contract Handlers

// listContractsHandler lists monitored contracts
func (s *HTTPServer) listContractsHandler(w http.ResponseWriter, r *http.Request) {
	contracts, err := s.storage.GetContracts(r.Context(), nil)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to retrieve contracts", err)
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"contracts": contracts,
		"total":     len(contracts),
	})
}

// addContractHandler adds a new contract to monitor
func (s *HTTPServer) addContractHandler(w http.ResponseWriter, r *http.Request) {
	var contract struct {
		Address string `json:"address"`
		Name    string `json:"name"`
		ABI     string `json:"abi"`
	}

	if err := json.NewDecoder(r.Body).Decode(&contract); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if contract.Address == "" {
		s.writeError(w, http.StatusBadRequest, "Contract address is required", nil)
		return
	}

	// Add contract to storage
	contractModel := &models.Contract{
		Address:    common.HexToAddress(contract.Address),
		Name:       contract.Name,
		ABI:        contract.ABI,
		StartBlock: 0, // or parse from request if provided
		Active:     true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	err := s.storage.SaveContract(r.Context(), contractModel)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to add contract", err)
		return
	}

	s.writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "Contract added successfully",
		"address": contract.Address,
	})
}

// getContractHandler gets a specific contract
func (s *HTTPServer) getContractHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]

	contract, err := s.storage.GetContract(r.Context(), address)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "Contract not found", err)
		return
	}

	s.writeJSON(w, http.StatusOK, contract)
}

// removeContractHandler removes a contract from monitoring
func (s *HTTPServer) removeContractHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]

	err := s.storage.DeleteContract(r.Context(), address)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to remove contract", err)
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Contract removed successfully",
		"address": address,
	})
}

// Processor Handlers

// listRulesHandler lists processing rules
func (s *HTTPServer) listRulesHandler(w http.ResponseWriter, r *http.Request) {
	rules := s.processor.GetRules()
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"rules": rules,
		"total": len(rules),
	})
}

// addRuleHandler adds a new processing rule
func (s *HTTPServer) addRuleHandler(w http.ResponseWriter, r *http.Request) {
	var rule processor.ProcessingRule

	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := s.processor.AddRule(&rule); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to add rule", err)
		return
	}

	s.writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "Rule added successfully",
		"rule_id": rule.ID,
	})
}

// getRuleHandler gets a specific processing rule
func (s *HTTPServer) getRuleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["id"]

	rules := s.processor.GetRules()
	for _, rule := range rules {
		if rule.ID == ruleID {
			s.writeJSON(w, http.StatusOK, rule)
			return
		}
	}

	s.writeError(w, http.StatusNotFound, "Rule not found", nil)
}

// updateRuleHandler updates a processing rule
func (s *HTTPServer) updateRuleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["id"]

	var rule processor.ProcessingRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	rule.ID = ruleID
	if err := s.processor.UpdateRule(&rule); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to update rule", err)
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Rule updated successfully",
		"rule_id": ruleID,
	})
}

// removeRuleHandler removes a processing rule
func (s *HTTPServer) removeRuleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["id"]

	if err := s.processor.RemoveRule(ruleID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to remove rule", err)
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Rule removed successfully",
		"rule_id": ruleID,
	})
}

// Notification Handlers

// listChannelsHandler lists notification channels
func (s *HTTPServer) listChannelsHandler(w http.ResponseWriter, r *http.Request) {
	channels := s.notification.GetChannels()
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"channels": channels,
		"total":    len(channels),
	})
}

// addChannelHandler adds a new notification channel
func (s *HTTPServer) addChannelHandler(w http.ResponseWriter, r *http.Request) {
	var channel notification.NotificationChannel

	if err := json.NewDecoder(r.Body).Decode(&channel); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := s.notification.AddChannel(&channel); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to add channel", err)
		return
	}

	s.writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message":    "Channel added successfully",
		"channel_id": channel.ID,
	})
}

// removeChannelHandler removes a notification channel
func (s *HTTPServer) removeChannelHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	channelID := vars["id"]

	if err := s.notification.RemoveChannel(channelID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to remove channel", err)
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":    "Channel removed successfully",
		"channel_id": channelID,
	})
}

// testNotificationHandler tests notification delivery
func (s *HTTPServer) testNotificationHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Type       string                 `json:"type"`
		Recipients []string               `json:"recipients"`
		Subject    string                 `json:"subject"`
		Message    string                 `json:"message"`
		Config     map[string]interface{} `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	notificationConfig := &notification.NotificationConfig{
		Type:       request.Type,
		Recipients: request.Recipients,
		Subject:    request.Subject,
		Template:   request.Message,
	}

	testData := map[string]interface{}{
		"test":      true,
		"timestamp": time.Now(),
		"message":   "Test notification from RSK Event Listener",
	}

	notificationID, err := s.notification.SendNotification(r.Context(), notificationConfig, testData)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to send test notification", err)
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":         "Test notification sent successfully",
		"notification_id": notificationID,
	})
}

// Monitor Handlers

// monitorStatusHandler gets monitor status
func (s *HTTPServer) monitorStatusHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"running":   s.monitor.IsRunning(),
		"healthy":   s.monitor.GetHealth(),
		"stats":     s.monitor.GetStats(),
		"timestamp": time.Now(),
	}

	s.writeJSON(w, http.StatusOK, status)
}

// startMonitorHandler starts the monitor
func (s *HTTPServer) startMonitorHandler(w http.ResponseWriter, r *http.Request) {
	if s.monitor.IsRunning() {
		s.writeError(w, http.StatusConflict, "Monitor is already running", nil)
		return
	}

	if err := s.monitor.Start(r.Context()); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to start monitor", err)
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Monitor started successfully",
	})
}

// stopMonitorHandler stops the monitor
func (s *HTTPServer) stopMonitorHandler(w http.ResponseWriter, r *http.Request) {
	if !s.monitor.IsRunning() {
		s.writeError(w, http.StatusConflict, "Monitor is not running", nil)
		return
	}

	if err := s.monitor.Stop(); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to stop monitor", err)
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Monitor stopped successfully",
	})
}

// Utility Methods

// writeJSON writes a JSON response
func (s *HTTPServer) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Error("Failed to encode JSON response", map[string]interface{}{"error": err})
	}
}

// writeError writes an error response
func (s *HTTPServer) writeError(w http.ResponseWriter, status int, message string, err error) {
	errorResponse := map[string]interface{}{
		"error":     message,
		"status":    status,
		"timestamp": time.Now(),
	}

	if err != nil {
		errorResponse["details"] = err.Error()
		s.logger.Error("HTTP error", map[string]interface{}{
			"status":  status,
			"message": message,
			"error":   err,
		})
	}

	s.writeJSON(w, status, errorResponse)
}
