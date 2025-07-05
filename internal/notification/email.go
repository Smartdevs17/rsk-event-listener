// File: internal/notification/email.go
package notification

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
)

// EmailSender handles email notifications
type EmailSender struct {
	config *EmailSenderConfig
	logger *NotificationLogger
	auth   smtp.Auth
	client *smtp.Client
}

// EmailSenderConfig holds email sender configuration
type EmailSenderConfig struct {
	SMTPHost    string        `json:"smtp_host"`
	SMTPPort    int           `json:"smtp_port"`
	Username    string        `json:"username"`
	Password    string        `json:"password"`
	FromEmail   string        `json:"from_email"`
	FromName    string        `json:"from_name"`
	UseTLS      bool          `json:"use_tls"`
	UseStartTLS bool          `json:"use_start_tls"`
	Timeout     time.Duration `json:"timeout"`
}

// NewEmailSender creates a new email sender
func NewEmailSender(config *NotificationManagerConfig, logger *NotificationLogger) *EmailSender {
	// Email config would be extracted from the main config or environment
	emailConfig := &EmailSenderConfig{
		SMTPHost:    "localhost",
		SMTPPort:    587,
		UseTLS:      true,
		UseStartTLS: true,
		Timeout:     30 * time.Second,
		FromEmail:   "noreply@rsk-event-listener.com",
		FromName:    "RSK Event Listener",
	}

	return &EmailSender{
		config: emailConfig,
		logger: logger.WithField("component", "email_sender"),
	}
}

// Start starts the email sender
func (es *EmailSender) Start(ctx context.Context) error {
	// Setup SMTP authentication
	if es.config.Username != "" && es.config.Password != "" {
		es.auth = smtp.PlainAuth("", es.config.Username, es.config.Password, es.config.SMTPHost)
	}

	es.logger.Info("Email sender started", map[string]interface{}{
		"smtp_host": es.config.SMTPHost,
		"smtp_port": es.config.SMTPPort,
		"use_tls":   es.config.UseTLS,
	})
	return nil
}

// Stop stops the email sender
func (es *EmailSender) Stop() error {
	if es.client != nil {
		es.client.Close()
	}
	es.logger.Info("Email sender stopped")
	return nil
}

// SendEmail sends an email
func (es *EmailSender) SendEmail(ctx context.Context, config *EmailConfig, data map[string]interface{}) error {
	startTime := time.Now()

	// Log email attempt
	es.logger.LogEmailAttempt(config.To, config.Subject)

	// Validate email config
	if err := es.validateEmailConfig(config); err != nil {
		es.logger.LogEmailResult(config.To, config.Subject, false, time.Since(startTime), err)
		return err
	}

	// Build email message
	message := es.buildEmailMessage(config, data)

	// Send email
	var err error
	if es.config.UseTLS {
		err = es.sendEmailTLS(config.To, message)
	} else {
		err = es.sendEmailPlain(config.To, message)
	}

	// Log result
	success := err == nil
	duration := time.Since(startTime)
	es.logger.LogEmailResult(config.To, config.Subject, success, duration, err)

	if err != nil {
		return utils.NewAppError(utils.ErrCodeExternal, "Failed to send email", err.Error())
	}

	es.logger.Debug("Email sent successfully", map[string]interface{}{
		"to":          config.To,
		"subject":     config.Subject,
		"duration_ms": duration.Milliseconds(),
	})

	return nil
}

// sendEmailTLS sends email with STARTTLS (FIXED for Gmail)
func (es *EmailSender) sendEmailTLS(to []string, message string) error {
	addr := fmt.Sprintf("%s:%d", es.config.SMTPHost, es.config.SMTPPort)

	// Open plain connection first (FIXED: not direct TLS)
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Close()

	// Upgrade to TLS using STARTTLS (FIXED: proper Gmail approach)
	tlsConfig := &tls.Config{
		ServerName:         es.config.SMTPHost,
		InsecureSkipVerify: false,
	}

	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	// Authenticate
	if es.auth != nil {
		if err := client.Auth(es.auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(es.config.FromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Send message
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to open data writer: %w", err)
	}
	defer writer.Close()

	_, err = writer.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// sendEmailPlain sends email without TLS
func (es *EmailSender) sendEmailPlain(to []string, message string) error {
	addr := fmt.Sprintf("%s:%d", es.config.SMTPHost, es.config.SMTPPort)
	return smtp.SendMail(addr, es.auth, es.config.FromEmail, to, []byte(message))
}

// buildEmailMessage builds the email message
func (es *EmailSender) buildEmailMessage(config *EmailConfig, data map[string]interface{}) string {
	var message strings.Builder

	// Headers
	message.WriteString(fmt.Sprintf("From: %s <%s>\r\n", es.config.FromName, es.config.FromEmail))
	message.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(config.To, ", ")))

	if len(config.CC) > 0 {
		message.WriteString(fmt.Sprintf("CC: %s\r\n", strings.Join(config.CC, ", ")))
	}

	if len(config.BCC) > 0 {
		message.WriteString(fmt.Sprintf("BCC: %s\r\n", strings.Join(config.BCC, ", ")))
	}

	message.WriteString(fmt.Sprintf("Subject: %s\r\n", config.Subject))
	message.WriteString("MIME-Version: 1.0\r\n")
	message.WriteString("Content-Type: text/html; charset=UTF-8\r\n")

	if config.Priority == "high" {
		message.WriteString("X-Priority: 1\r\n")
		message.WriteString("Importance: high\r\n")
	} else if config.Priority == "low" {
		message.WriteString("X-Priority: 5\r\n")
		message.WriteString("Importance: low\r\n")
	}

	// Add timestamp header
	message.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))

	message.WriteString("\r\n")

	// Body
	if config.Template != "" {
		// Simple template replacement
		body := config.Template
		for key, value := range data {
			placeholder := fmt.Sprintf("{{.%s}}", key)
			body = strings.ReplaceAll(body, placeholder, fmt.Sprintf("%v", value))
		}
		message.WriteString(body)
	} else {
		// Default format
		message.WriteString("<html><body>")
		message.WriteString("<h2>RSK Event Notification</h2>")
		message.WriteString("<table border='1' cellpadding='5' cellspacing='0'>")
		for key, value := range data {
			message.WriteString(fmt.Sprintf("<tr><td><strong>%s</strong></td><td>%v</td></tr>", key, value))
		}
		message.WriteString("</table>")
		message.WriteString(fmt.Sprintf("<p><small>Sent at: %s</small></p>", time.Now().Format(time.RFC3339)))
		message.WriteString("</body></html>")
	}

	return message.String()
}

// validateEmailConfig validates email configuration
func (es *EmailSender) validateEmailConfig(config *EmailConfig) error {
	if len(config.To) == 0 {
		return utils.NewAppError(utils.ErrCodeValidation, "Email recipients are required", "")
	}

	if config.Subject == "" {
		return utils.NewAppError(utils.ErrCodeValidation, "Email subject is required", "")
	}

	// Validate email addresses
	for _, email := range config.To {
		if !es.isValidEmail(email) {
			return utils.NewAppError(utils.ErrCodeValidation, "Invalid email address", email)
		}
	}

	for _, email := range config.CC {
		if !es.isValidEmail(email) {
			return utils.NewAppError(utils.ErrCodeValidation, "Invalid CC email address", email)
		}
	}

	for _, email := range config.BCC {
		if !es.isValidEmail(email) {
			return utils.NewAppError(utils.ErrCodeValidation, "Invalid BCC email address", email)
		}
	}

	return nil
}

// isValidEmail performs basic email validation
func (es *EmailSender) isValidEmail(email string) bool {
	// Basic email validation
	if len(email) < 3 || len(email) > 254 {
		return false
	}

	if !strings.Contains(email, "@") {
		return false
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	local, domain := parts[0], parts[1]
	if len(local) == 0 || len(domain) == 0 {
		return false
	}

	if len(local) > 64 || len(domain) > 253 {
		return false
	}

	return true
}

// TestEmailConfig tests email configuration
func (es *EmailSender) TestEmailConfig(ctx context.Context) error {
	testConfig := &EmailConfig{
		To:       []string{es.config.FromEmail}, // Send test to self
		Subject:  "RSK Event Listener - Email Configuration Test",
		Template: "This is a test email to verify email configuration. If you receive this, email notifications are working correctly.",
	}

	testData := map[string]interface{}{
		"test":      true,
		"timestamp": time.Now().Format(time.RFC3339),
		"message":   "Email configuration test successful",
	}

	return es.SendEmail(ctx, testConfig, testData)
}

// GetEmailStats returns email sender statistics
func (es *EmailSender) GetEmailStats() map[string]interface{} {
	return map[string]interface{}{
		"smtp_host":     es.config.SMTPHost,
		"smtp_port":     es.config.SMTPPort,
		"from_email":    es.config.FromEmail,
		"use_tls":       es.config.UseTLS,
		"use_start_tls": es.config.UseStartTLS,
		"timeout":       es.config.Timeout,
	}
}

// SetGmailConfig configures the email sender for Gmail
func (es *EmailSender) SetGmailConfig(username, password string) {
	es.config = &EmailSenderConfig{
		SMTPHost:    "smtp.gmail.com",
		SMTPPort:    587,
		Username:    username,
		Password:    password,
		FromEmail:   username,
		FromName:    "RSK Event Listener",
		UseTLS:      true,
		UseStartTLS: true,
		Timeout:     30 * time.Second,
	}

	// Setup SMTP authentication for Gmail
	es.auth = smtp.PlainAuth("", username, password, "smtp.gmail.com")
}

// SetCustomEmailConfig configures the email sender with custom settings
func (es *EmailSender) SetCustomEmailConfig(config *EmailSenderConfig) {
	es.config = config

	// Setup SMTP authentication if credentials are provided
	if config.Username != "" && config.Password != "" {
		es.auth = smtp.PlainAuth("", config.Username, config.Password, config.SMTPHost)
	}
}

// GetConfig returns the current email configuration (without password)
func (es *EmailSender) GetConfig() *EmailSenderConfig {
	if es.config == nil {
		return nil
	}

	// Return a copy without the password for security
	return &EmailSenderConfig{
		SMTPHost:    es.config.SMTPHost,
		SMTPPort:    es.config.SMTPPort,
		Username:    es.config.Username,
		Password:    "***hidden***",
		FromEmail:   es.config.FromEmail,
		FromName:    es.config.FromName,
		UseTLS:      es.config.UseTLS,
		UseStartTLS: es.config.UseStartTLS,
		Timeout:     es.config.Timeout,
	}
}

// TestGmailConnection tests the Gmail SMTP connection
func (es *EmailSender) TestGmailConnection() error {
	if es.config == nil {
		return utils.NewAppError(utils.ErrCodeValidation, "Email configuration not set", "")
	}

	addr := fmt.Sprintf("%s:%d", es.config.SMTPHost, es.config.SMTPPort)

	// Open plain connection
	client, err := smtp.Dial(addr)
	if err != nil {
		return utils.NewAppError(utils.ErrCodeExternal, "Failed to connect to Gmail SMTP", err.Error())
	}
	defer client.Close()

	// Upgrade to TLS (STARTTLS)
	tlsConfig := &tls.Config{
		ServerName: es.config.SMTPHost,
	}
	if err := client.StartTLS(tlsConfig); err != nil {
		return utils.NewAppError(utils.ErrCodeExternal, "Failed to start TLS with Gmail SMTP", err.Error())
	}

	// Test authentication
	if es.auth != nil {
		if err := client.Auth(es.auth); err != nil {
			return utils.NewAppError(utils.ErrCodeExternal, "Gmail authentication failed", err.Error())
		}
	}

	return nil
}

// IsGmailConfigured checks if the email sender is configured for Gmail
func (es *EmailSender) IsGmailConfigured() bool {
	return es.config != nil && es.config.SMTPHost == "smtp.gmail.com"
}

// ValidateGmailConfig validates Gmail-specific configuration
func (es *EmailSender) ValidateGmailConfig() error {
	if es.config == nil {
		return utils.NewAppError(utils.ErrCodeValidation, "Email configuration not set", "")
	}

	if es.config.SMTPHost != "smtp.gmail.com" {
		return utils.NewAppError(utils.ErrCodeValidation, "Not configured for Gmail", "")
	}

	if es.config.SMTPPort != 587 {
		return utils.NewAppError(utils.ErrCodeValidation, "Gmail requires port 587", "")
	}

	if es.config.Username == "" {
		return utils.NewAppError(utils.ErrCodeValidation, "Gmail username is required", "")
	}

	if es.config.Password == "" {
		return utils.NewAppError(utils.ErrCodeValidation, "Gmail password is required", "")
	}

	if !es.config.UseTLS {
		return utils.NewAppError(utils.ErrCodeValidation, "Gmail requires TLS", "")
	}

	return nil
}

// SendEmailDirect provides a direct email sending method for testing
func (es *EmailSender) SendEmailDirect(to []string, subject, body string) error {
	if es.config == nil {
		return utils.NewAppError(utils.ErrCodeValidation, "Email configuration not set", "")
	}

	// Build simple message
	message := fmt.Sprintf("From: %s <%s>\r\n", es.config.FromName, es.config.FromEmail)
	message += fmt.Sprintf("To: %s\r\n", strings.Join(to, ", "))
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += "MIME-Version: 1.0\r\n"
	message += "Content-Type: text/html; charset=UTF-8\r\n"
	message += fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	message += "\r\n"
	message += body

	return es.sendEmailTLS(to, message)
}
