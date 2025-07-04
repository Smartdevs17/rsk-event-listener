// File: internal/notification/webhook.go
package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
)

// WebhookSender handles webhook notifications
type WebhookSender struct {
	config     *NotificationManagerConfig
	logger     *NotificationLogger
	httpClient *http.Client
}

// WebhookPayload defines the webhook payload structure
type WebhookPayload struct {
	Event     interface{} `json:"event"`
	Timestamp time.Time   `json:"timestamp"`
	Source    string      `json:"source"`
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Version   string      `json:"version"`
}

// WebhookRetryConfig defines retry configuration for webhooks
type WebhookRetryConfig struct {
	MaxAttempts int           `json:"max_attempts"`
	BaseDelay   time.Duration `json:"base_delay"`
	MaxDelay    time.Duration `json:"max_delay"`
	Backoff     string        `json:"backoff"` // linear, exponential
}

// WebhookResponse represents a webhook response
type WebhookResponse struct {
	StatusCode   int           `json:"status_code"`
	ResponseTime time.Duration `json:"response_time"`
	Success      bool          `json:"success"`
	Error        error         `json:"error,omitempty"`
	Headers      http.Header   `json:"headers,omitempty"`
	Body         string        `json:"body,omitempty"`
}

// NewWebhookSender creates a new webhook sender
func NewWebhookSender(config *NotificationManagerConfig, logger *NotificationLogger) *WebhookSender {
	return &WebhookSender{
		config: config,
		logger: logger.WithField("component", "webhook_sender"),
		httpClient: &http.Client{
			Timeout: config.NotificationTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     30 * time.Second,
			},
		},
	}
}

// Start starts the webhook sender
func (ws *WebhookSender) Start(ctx context.Context) error {
	ws.logger.Info("Webhook sender started")
	return nil
}

// Stop stops the webhook sender
func (ws *WebhookSender) Stop() error {
	ws.logger.Info("Webhook sender stopped")
	return nil
}

// SendWebhook sends a webhook notification
func (ws *WebhookSender) SendWebhook(ctx context.Context, config *WebhookConfig, data map[string]interface{}) error {
	startTime := time.Now()

	// Log webhook attempt
	ws.logger.LogWebhookAttempt(config.URL, config.Method, config.Headers)

	// Build webhook payload
	payload := ws.buildWebhookPayload(data)

	// Send webhook with retry logic
	response := ws.sendWebhookWithRetry(ctx, config, payload)

	// Log webhook response
	ws.logger.LogWebhookResponse(config.URL, response.StatusCode, response.ResponseTime, response.Error)

	// Update statistics
	duration := time.Since(startTime)
	if response.Success {
		ws.logger.Debug("Webhook sent successfully", map[string]interface{}{
			"url":           config.URL,
			"status_code":   response.StatusCode,
			"response_time": duration,
		})
	} else {
		ws.logger.Error("Webhook failed", map[string]interface{}{
			"url":           config.URL,
			"status_code":   response.StatusCode,
			"error":         response.Error,
			"response_time": duration,
		})
	}

	return response.Error
}

// sendWebhookWithRetry sends a webhook with retry logic
func (ws *WebhookSender) sendWebhookWithRetry(ctx context.Context, config *WebhookConfig, payload *WebhookPayload) *WebhookResponse {
	retryConfig := &WebhookRetryConfig{
		MaxAttempts: ws.config.RetryAttempts,
		BaseDelay:   ws.config.RetryDelay,
		MaxDelay:    30 * time.Second,
		Backoff:     "exponential",
	}

	var lastResponse *WebhookResponse

	for attempt := 1; attempt <= retryConfig.MaxAttempts; attempt++ {
		// Add delay for retry attempts
		if attempt > 1 {
			delay := ws.calculateRetryDelay(retryConfig, attempt)
			ws.logger.LogRetryAttempt("webhook", attempt, retryConfig.MaxAttempts, delay)

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return &WebhookResponse{
					Success: false,
					Error:   ctx.Err(),
				}
			}
		}

		// Send webhook
		response := ws.sendSingleWebhook(ctx, config, payload)
		lastResponse = response

		// Check if successful
		if response.Success {
			return response
		}

		// Log retry if not the last attempt
		if attempt < retryConfig.MaxAttempts {
			ws.logger.Warn("Webhook attempt failed, retrying", map[string]interface{}{
				"url":         config.URL,
				"attempt":     attempt,
				"status_code": response.StatusCode,
				"error":       response.Error,
			})
		}
	}

	return lastResponse
}

// sendSingleWebhook sends a single webhook request
func (ws *WebhookSender) sendSingleWebhook(ctx context.Context, config *WebhookConfig, payload *WebhookPayload) *WebhookResponse {
	startTime := time.Now()

	response := &WebhookResponse{
		Success: false,
	}

	// Marshal payload
	jsonData, err := json.Marshal(payload)
	if err != nil {
		response.Error = utils.NewAppError(utils.ErrCodeInternal, "Failed to marshal webhook payload", err.Error())
		response.ResponseTime = time.Since(startTime)
		return response
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, config.Method, config.URL, bytes.NewReader(jsonData))
	if err != nil {
		response.Error = utils.NewAppError(utils.ErrCodeInternal, "Failed to create webhook request", err.Error())
		response.ResponseTime = time.Since(startTime)
		return response
	}

	// Set headers
	ws.setRequestHeaders(req, config.Headers)

	// Send request
	resp, err := ws.httpClient.Do(req)
	response.ResponseTime = time.Since(startTime)

	if err != nil {
		response.Error = utils.NewAppError(utils.ErrCodeExternal, "Failed to send webhook", err.Error())
		return response
	}
	defer resp.Body.Close()

	// Set response data
	response.StatusCode = resp.StatusCode
	response.Headers = resp.Header

	// Read response body (limited to prevent memory issues)
	bodyBuffer := make([]byte, 1024)
	n, _ := resp.Body.Read(bodyBuffer)
	response.Body = string(bodyBuffer[:n])

	// Check if successful (2xx status codes)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		response.Success = true
	} else {
		response.Error = utils.NewAppError(utils.ErrCodeExternal,
			"Webhook returned non-success status",
			fmt.Sprintf("status: %d, body: %s", resp.StatusCode, response.Body))
	}

	return response
}

// buildWebhookPayload builds the webhook payload
func (ws *WebhookSender) buildWebhookPayload(data map[string]interface{}) *WebhookPayload {
	return &WebhookPayload{
		Event:     data["event"],
		Timestamp: time.Now(),
		Source:    "rsk-event-listener",
		Type:      "event_notification",
		Data:      data,
		Version:   "1.0",
	}
}

// setRequestHeaders sets HTTP request headers
func (ws *WebhookSender) setRequestHeaders(req *http.Request, headers map[string]string) {
	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Set default headers if not provided
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "RSK-Event-Listener/1.0")
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}

	// Add request timestamp
	req.Header.Set("X-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))

	// Add request ID for tracing
	if requestID, err := utils.GenerateID(); err == nil {
		req.Header.Set("X-Request-ID", requestID)
	}
}

// calculateRetryDelay calculates the delay for retry attempts
func (ws *WebhookSender) calculateRetryDelay(config *WebhookRetryConfig, attempt int) time.Duration {
	var delay time.Duration

	switch config.Backoff {
	case "exponential":
		// Exponential backoff: base_delay * 2^(attempt-1)
		delay = time.Duration(int64(config.BaseDelay) << uint(attempt-2))
	case "linear":
		// Linear backoff: base_delay * attempt
		delay = time.Duration(int64(config.BaseDelay) * int64(attempt))
	default:
		// Fixed delay
		delay = config.BaseDelay
	}

	// Cap at max delay
	if delay > config.MaxDelay {
		delay = config.MaxDelay
	}

	return delay
}

// ValidateWebhookConfig validates webhook configuration
func (ws *WebhookSender) ValidateWebhookConfig(config *WebhookConfig) error {
	if config.URL == "" {
		return utils.NewAppError(utils.ErrCodeValidation, "Webhook URL is required", "")
	}

	if config.Method == "" {
		config.Method = "POST"
	}

	if config.Headers == nil {
		config.Headers = make(map[string]string)
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return nil
}

// GetWebhookStats returns webhook statistics
func (ws *WebhookSender) GetWebhookStats() map[string]interface{} {
	return map[string]interface{}{
		"timeout":        ws.httpClient.Timeout,
		"retry_attempts": ws.config.RetryAttempts,
		"retry_delay":    ws.config.RetryDelay,
	}
}
