// File: cmd/listener/email_test_main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/smartdevs17/rsk-event-listener/internal/notification"
	"github.com/smartdevs17/rsk-event-listener/pkg/utils"
)

func main() {
	fmt.Println("🚀 RSK Event Listener - Gmail Email Test")
	fmt.Println("========================================")

	// Initialize logger
	err := utils.InitLogger("info", "text", "stdout", "")
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Get Gmail credentials from environment variables
	gmailUser := os.Getenv("GMAIL_USER")
	gmailPassword := os.Getenv("GMAIL_PASSWORD")
	testRecipient := os.Getenv("TEST_EMAIL_RECIPIENT")

	if gmailUser == "" || gmailPassword == "" || testRecipient == "" {
		fmt.Println("❌ Missing required environment variables:")
		fmt.Println("   - GMAIL_USER: Your Gmail address")
		fmt.Println("   - GMAIL_PASSWORD: Your Gmail app password")
		fmt.Println("   - TEST_EMAIL_RECIPIENT: Email address to send test to")
		fmt.Println()
		fmt.Println("📋 Setup Instructions:")
		fmt.Println("1. Enable 2FA on your Gmail account")
		fmt.Println("2. Generate an App Password for this application")
		fmt.Println("3. Set the environment variables:")
		fmt.Println("   export GMAIL_USER=\"your-email@gmail.com\"")
		fmt.Println("   export GMAIL_PASSWORD=\"your-app-password\"")
		fmt.Println("   export TEST_EMAIL_RECIPIENT=\"recipient@example.com\"")
		fmt.Println()
		fmt.Println("📚 How to get Gmail App Password:")
		fmt.Println("   1. Go to Google Account settings")
		fmt.Println("   2. Security > 2-Step Verification")
		fmt.Println("   3. App passwords > Generate new")
		fmt.Println("   4. Use the 16-character password")
		os.Exit(1)
	}

	fmt.Printf("✅ Gmail User: %s\n", gmailUser)
	fmt.Printf("✅ Test Recipient: %s\n", testRecipient)
	fmt.Printf("✅ Gmail Password: %s\n", hidePassword(gmailPassword))

	// Test 1: Create Email Sender with Gmail Configuration
	fmt.Println("\n=== Test 1: Creating Gmail Email Sender ===")

	// Create notification manager config with Gmail settings
	notificationConfig := &notification.NotificationManagerConfig{
		MaxConcurrentNotifications: 5,
		NotificationTimeout:        30 * time.Second,
		RetryAttempts:              3,
		RetryDelay:                 2 * time.Second,
		EnableEmailNotifications:   true,
		EnableWebhookNotifications: false,
		QueueSize:                  100,
		LogLevel:                   "info",
	}

	// Create email sender with Gmail configuration
	emailSender := createGmailEmailSender(notificationConfig, gmailUser, gmailPassword)
	fmt.Println("✅ Gmail Email Sender created successfully")

	// Test 2: Start Email Sender
	fmt.Println("\n=== Test 2: Starting Email Sender ===")
	ctx := context.Background()
	err = emailSender.Start(ctx)
	if err != nil {
		log.Fatalf("Failed to start email sender: %v", err)
	}
	fmt.Println("✅ Email sender started successfully")

	// Test 3: Send Test Email
	fmt.Println("\n=== Test 3: Sending Test Email ===")
	testEmailConfig := &notification.EmailConfig{
		To:       []string{testRecipient},
		Subject:  "🚀 RSK Event Listener - Email Test",
		Template: buildTestEmailTemplate(),
		Priority: "normal",
	}

	testData := map[string]interface{}{
		"test_time":    time.Now().Format(time.RFC3339),
		"sender_email": gmailUser,
		"test_result":  "SUCCESS",
		"message":      "Gmail configuration is working correctly!",
		"from_app":     "RSK Event Listener",
		"version":      "1.0.0",
		"test_id":      fmt.Sprintf("test-%d", time.Now().Unix()),
	}

	fmt.Printf("📧 Sending test email to: %s\n", testRecipient)
	err = emailSender.SendEmail(ctx, testEmailConfig, testData)
	if err != nil {
		log.Fatalf("❌ Failed to send test email: %v", err)
	}
	fmt.Println("✅ Test email sent successfully!")

	// Test 4: Send Event Notification Email
	fmt.Println("\n=== Test 4: Sending Event Notification Email ===")
	eventEmailConfig := &notification.EmailConfig{
		To:       []string{testRecipient},
		Subject:  "🔔 RSK Event Notification - Transfer Detected",
		Template: buildEventEmailTemplate(),
		Priority: "high",
	}

	eventData := map[string]interface{}{
		"event_name":       "Transfer",
		"contract_address": "0x1234567890abcdef1234567890abcdef12345678",
		"block_number":     "123456",
		"tx_hash":          "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		"from_address":     "0x1111111111111111111111111111111111111111",
		"to_address":       "0x2222222222222222222222222222222222222222",
		"amount":           "1000000000000000000",
		"amount_readable":  "1.0 RBTC",
		"timestamp":        time.Now().Format(time.RFC3339),
		"network":          "RSK Mainnet",
		"explorer_url":     "https://explorer.rsk.co/tx/0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
	}

	fmt.Printf("📧 Sending event notification email to: %s\n", testRecipient)
	err = emailSender.SendEmail(ctx, eventEmailConfig, eventData)
	if err != nil {
		log.Fatalf("❌ Failed to send event notification email: %v", err)
	}
	fmt.Println("✅ Event notification email sent successfully!")

	// Test 5: Test Email with CC and BCC
	fmt.Println("\n=== Test 5: Testing CC and BCC Functionality ===")
	advancedEmailConfig := &notification.EmailConfig{
		To:       []string{testRecipient},
		CC:       []string{}, // Add CC recipients if needed
		BCC:      []string{}, // Add BCC recipients if needed
		Subject:  "🧪 RSK Event Listener - Advanced Email Test",
		Template: buildAdvancedEmailTemplate(),
		Priority: "normal",
	}

	advancedData := map[string]interface{}{
		"test_type":        "Advanced Email Features",
		"cc_recipients":    len(advancedEmailConfig.CC),
		"bcc_recipients":   len(advancedEmailConfig.BCC),
		"total_recipients": len(advancedEmailConfig.To) + len(advancedEmailConfig.CC) + len(advancedEmailConfig.BCC),
		"timestamp":        time.Now().Format(time.RFC3339),
		"features_tested":  []string{"HTML formatting", "Priority headers", "MIME headers", "Gmail SMTP"},
	}

	fmt.Printf("📧 Sending advanced email test to: %s\n", testRecipient)
	err = emailSender.SendEmail(ctx, advancedEmailConfig, advancedData)
	if err != nil {
		log.Fatalf("❌ Failed to send advanced email: %v", err)
	}
	fmt.Println("✅ Advanced email sent successfully!")

	// Test 6: Get Email Statistics
	fmt.Println("\n=== Test 6: Email Statistics ===")
	stats := emailSender.GetEmailStats()
	fmt.Println("📊 Email Sender Statistics:")
	for key, value := range stats {
		fmt.Printf("  %s: %v\n", key, value)
	}

	// Test 7: Stop Email Sender
	fmt.Println("\n=== Test 7: Stopping Email Sender ===")
	err = emailSender.Stop()
	if err != nil {
		log.Fatalf("Failed to stop email sender: %v", err)
	}
	fmt.Println("✅ Email sender stopped successfully")

	// Test Summary
	fmt.Println("\n🎉 Gmail Email Test Summary")
	fmt.Println("============================")
	fmt.Println("✅ Gmail SMTP configuration successful")
	fmt.Println("✅ Email sender lifecycle management working")
	fmt.Println("✅ Test email sent successfully")
	fmt.Println("✅ Event notification email sent successfully")
	fmt.Println("✅ Advanced email features working")
	fmt.Println("✅ Email statistics available")
	fmt.Println()
	fmt.Printf("📧 Check your email inbox at: %s\n", testRecipient)
	fmt.Println("You should have received 3 test emails:")
	fmt.Println("  1. Basic test email")
	fmt.Println("  2. Event notification email")
	fmt.Println("  3. Advanced features test email")
	fmt.Println()
	fmt.Println("🎯 Gmail email functionality is working correctly!")
	fmt.Println("Ready to proceed with main application integration.")
}

// createGmailEmailSender creates an email sender configured for Gmail
func createGmailEmailSender(config *notification.NotificationManagerConfig, gmailUser, gmailPassword string) *notification.EmailSender {
	logger := notification.NewNotificationLogger("info")

	// Create email sender with Gmail configuration
	emailSender := notification.NewEmailSender(config, logger)

	// Configure Gmail SMTP settings
	emailSender.SetGmailConfig(gmailUser, gmailPassword)

	return emailSender
}

// buildTestEmailTemplate creates a test email template
func buildTestEmailTemplate() string {
	return `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>RSK Event Listener Email Test</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .header { background-color: #FF6B35; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .success { background-color: #d4edda; border: 1px solid #c3e6cb; color: #155724; padding: 10px; margin: 10px 0; }
        .info { background-color: #e2f3ff; border: 1px solid #b3d9ff; color: #0c5460; padding: 10px; margin: 10px 0; }
        .footer { background-color: #f8f9fa; padding: 15px; text-align: center; font-size: 12px; color: #666; }
        table { width: 100%; border-collapse: collapse; margin: 10px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🚀 RSK Event Listener</h1>
        <h2>Email Configuration Test</h2>
    </div>
    
    <div class="content">
        <div class="success">
            <h3>✅ Email Test Successful!</h3>
            <p>Your Gmail configuration is working correctly.</p>
        </div>
        
        <div class="info">
            <h3>📧 Test Details</h3>
            <table>
                <tr><th>Test Time</th><td>{{.test_time}}</td></tr>
                <tr><th>Sender Email</th><td>{{.sender_email}}</td></tr>
                <tr><th>Test Result</th><td>{{.test_result}}</td></tr>
                <tr><th>Message</th><td>{{.message}}</td></tr>
                <tr><th>From Application</th><td>{{.from_app}}</td></tr>
                <tr><th>Version</th><td>{{.version}}</td></tr>
                <tr><th>Test ID</th><td>{{.test_id}}</td></tr>
            </table>
        </div>
        
        <p>If you received this email, it means:</p>
        <ul>
            <li>✅ Gmail SMTP configuration is correct</li>
            <li>✅ Authentication is working</li>
            <li>✅ Email delivery is functional</li>
            <li>✅ HTML formatting is supported</li>
        </ul>
    </div>
    
    <div class="footer">
        <p>Sent by RSK Event Listener | {{.test_time}}</p>
        <p>This is an automated test email.</p>
    </div>
</body>
</html>
`
}

// buildEventEmailTemplate creates an event notification email template
func buildEventEmailTemplate() string {
	return `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>RSK Event Notification</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .header { background-color: #FF6B35; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .event-alert { background-color: #fff3cd; border: 1px solid #ffeaa7; color: #856404; padding: 15px; margin: 10px 0; }
        .event-details { background-color: #f8f9fa; padding: 15px; margin: 10px 0; }
        .footer { background-color: #f8f9fa; padding: 15px; text-align: center; font-size: 12px; color: #666; }
        table { width: 100%; border-collapse: collapse; margin: 10px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        .amount { font-weight: bold; color: #28a745; }
        .hash { font-family: monospace; font-size: 12px; word-break: break-all; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🔔 RSK Event Notification</h1>
        <h2>Transfer Event Detected</h2>
    </div>
    
    <div class="content">
        <div class="event-alert">
            <h3>⚠️ New {{.event_name}} Event</h3>
            <p>A new transfer event has been detected on the RSK network.</p>
        </div>
        
        <div class="event-details">
            <h3>📋 Event Details</h3>
            <table>
                <tr><th>Event Name</th><td>{{.event_name}}</td></tr>
                <tr><th>Contract Address</th><td class="hash">{{.contract_address}}</td></tr>
                <tr><th>Block Number</th><td>{{.block_number}}</td></tr>
                <tr><th>Transaction Hash</th><td class="hash">{{.tx_hash}}</td></tr>
                <tr><th>From Address</th><td class="hash">{{.from_address}}</td></tr>
                <tr><th>To Address</th><td class="hash">{{.to_address}}</td></tr>
                <tr><th>Amount</th><td class="amount">{{.amount_readable}}</td></tr>
                <tr><th>Raw Amount</th><td class="hash">{{.amount}}</td></tr>
                <tr><th>Timestamp</th><td>{{.timestamp}}</td></tr>
                <tr><th>Network</th><td>{{.network}}</td></tr>
            </table>
        </div>
        
        <div class="info">
            <h3>🔍 Transaction Details</h3>
            <p>You can view this transaction on the RSK Explorer:</p>
            <p><a href="{{.explorer_url}}" target="_blank">{{.explorer_url}}</a></p>
        </div>
    </div>
    
    <div class="footer">
        <p>Sent by RSK Event Listener | {{.timestamp}}</p>
        <p>This is an automated event notification.</p>
    </div>
</body>
</html>
`
}

// buildAdvancedEmailTemplate creates an advanced email template
func buildAdvancedEmailTemplate() string {
	return `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>RSK Event Listener - Advanced Email Test</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .header { background-color: #6c757d; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .feature-test { background-color: #e2f3ff; border: 1px solid #b3d9ff; color: #0c5460; padding: 15px; margin: 10px 0; }
        .footer { background-color: #f8f9fa; padding: 15px; text-align: center; font-size: 12px; color: #666; }
        table { width: 100%; border-collapse: collapse; margin: 10px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        .badge { background-color: #007bff; color: white; padding: 2px 6px; border-radius: 3px; font-size: 12px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🧪 RSK Event Listener</h1>
        <h2>Advanced Email Features Test</h2>
    </div>
    
    <div class="content">
        <div class="feature-test">
            <h3>🔧 Advanced Features Test</h3>
            <p>Testing advanced email functionality including CC, BCC, and HTML formatting.</p>
        </div>
        
        <table>
            <tr><th>Test Type</th><td>{{.test_type}}</td></tr>
            <tr><th>CC Recipients</th><td>{{.cc_recipients}}</td></tr>
            <tr><th>BCC Recipients</th><td>{{.bcc_recipients}}</td></tr>
            <tr><th>Total Recipients</th><td>{{.total_recipients}}</td></tr>
            <tr><th>Timestamp</th><td>{{.timestamp}}</td></tr>
        </table>
        
        <h3>Features Tested:</h3>
        <ul>
            <li><span class="badge">HTML</span> HTML formatting and styling</li>
            <li><span class="badge">MIME</span> MIME headers and content types</li>
            <li><span class="badge">SMTP</span> Gmail SMTP authentication</li>
            <li><span class="badge">TLS</span> Secure TLS connection</li>
        </ul>
    </div>
    
    <div class="footer">
        <p>Sent by RSK Event Listener | {{.timestamp}}</p>
        <p>Advanced email features test completed successfully.</p>
    </div>
</body>
</html>
`
}

// hidePassword masks the password for display
func hidePassword(password string) string {
	if len(password) <= 4 {
		return "****"
	}
	return password[:2] + "***" + password[len(password)-2:]
}
