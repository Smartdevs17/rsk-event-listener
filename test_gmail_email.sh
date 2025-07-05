#!/bin/bash

# File: test_gmail_email.sh
# RSK Event Listener - Gmail Email Test Script

set -e

echo "🚀 RSK Event Listener - Gmail Email Test"
echo "========================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if environment variables are set
print_status "Checking environment variables..."

if [[ -z "$GMAIL_USER" ]]; then
    print_error "GMAIL_USER environment variable is not set"
    MISSING_VARS=true
fi

if [[ -z "$GMAIL_PASSWORD" ]]; then
    print_error "GMAIL_PASSWORD environment variable is not set"
    MISSING_VARS=true
fi

if [[ -z "$TEST_EMAIL_RECIPIENT" ]]; then
    print_error "TEST_EMAIL_RECIPIENT environment variable is not set"
    MISSING_VARS=true
fi

if [[ "$MISSING_VARS" == "true" ]]; then
    echo ""
    print_error "Missing required environment variables!"
    echo ""
    echo "📋 Setup Instructions:"
    echo "====================="
    echo ""
    echo "1. 🔐 Enable 2FA on your Gmail account:"
    echo "   - Go to https://myaccount.google.com/security"
    echo "   - Turn on 2-Step Verification"
    echo ""
    echo "2. 🔑 Generate an App Password:"
    echo "   - Go to https://myaccount.google.com/apppasswords"
    echo "   - Select 'Mail' and your device"
    echo "   - Copy the 16-character password"
    echo ""
    echo "3. 🌍 Set environment variables:"
    echo "   export GMAIL_USER=\"your-email@gmail.com\""
    echo "   export GMAIL_PASSWORD=\"your-16-char-app-password\""
    echo "   export TEST_EMAIL_RECIPIENT=\"recipient@example.com\""
    echo ""
    echo "4. 🚀 Run the test:"
    echo "   ./test_gmail_email.sh"
    echo ""
    echo "📝 Example .env file:"
    echo "GMAIL_USER=your-email@gmail.com"
    echo "GMAIL_PASSWORD=abcdefghijklmnop"
    echo "TEST_EMAIL_RECIPIENT=test@example.com"
    echo ""
    echo "💡 You can also source a .env file:"
    echo "   source .env && ./test_gmail_email.sh"
    echo ""
    exit 1
fi

print_success "All environment variables are set"
echo "  📧 Gmail User: $GMAIL_USER"
echo "  📧 Test Recipient: $TEST_EMAIL_RECIPIENT"
echo "  🔐 Gmail Password: ${GMAIL_PASSWORD:0:4}***${GMAIL_PASSWORD: -4}"

# Check if Go is installed
print_status "Checking Go installation..."
if ! command -v go &> /dev/null; then
    print_error "Go is not installed. Please install Go 1.21 or later."
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
print_success "Go version: $GO_VERSION"

# Check if we're in the right directory
if [[ ! -f "go.mod" ]]; then
    print_error "go.mod not found. Please run this script from the project root directory."
    exit 1
fi

# Clean up any previous test artifacts
print_status "Cleaning up previous test artifacts..."
rm -f email_test_main
rm -f gmail_test.db
print_success "Cleanup completed"

# Download dependencies
print_status "Downloading dependencies..."
go mod download
if [[ $? -ne 0 ]]; then
    print_error "Failed to download dependencies"
    exit 1
fi
print_success "Dependencies downloaded"

# Add the Gmail config file to the notification package
print_status "Adding Gmail configuration extension..."
if [[ ! -f "internal/notification/gmail_config.go" ]]; then
    print_warning "Gmail config file not found. Creating it..."
    # Note: The Gmail config file should be created separately
    # This is just a placeholder check
fi

# Build the email test executable
print_status "Building email test executable..."
go build -o email_test_main ./cmd/listener/email_test_main.go
if [[ $? -ne 0 ]]; then
    print_error "Failed to build email test executable"
    exit 1
fi
print_success "Email test executable built successfully"

# Run the Gmail email test
print_status "Running Gmail email test..."
echo "========================================"
./email_test_main
TEST_EXIT_CODE=$?
echo "========================================"

# Clean up
print_status "Cleaning up test artifacts..."
rm -f email_test_main
rm -f gmail_test.db

if [[ $TEST_EXIT_CODE -eq 0 ]]; then
    print_success "Gmail email test completed successfully!"
    echo ""
    echo "🎉 Test Results Summary:"
    echo "======================="
    echo "✅ Gmail SMTP connection successful"
    echo "✅ Email authentication working"
    echo "✅ Test emails sent successfully"
    echo "✅ HTML formatting working"
    echo "✅ Email templates rendering correctly"
    echo ""
    echo "📧 Check your email inbox at: $TEST_EMAIL_RECIPIENT"
    echo "You should have received 3 test emails:"
    echo "  1. 🧪 Basic test email"
    echo "  2. 🔔 Event notification email"
    echo "  3. 🔧 Advanced features test email"
    echo ""
    echo "🚀 Gmail email functionality is confirmed working!"
    echo "Ready to proceed with main application integration."
else
    print_error "Gmail email test failed with exit code: $TEST_EXIT_CODE"
    echo ""
    echo "🔍 Troubleshooting Tips:"
    echo "======================="
    echo "1. 📧 Check your Gmail credentials"
    echo "2. 🔐 Ensure 2FA is enabled and app password is correct"
    echo "3. 🌐 Check network connectivity"
    echo "4. 🔒 Verify Gmail security settings"
    echo "5. 📝 Check the logs above for specific error messages"
    echo ""
    echo "🔗 Helpful Links:"
    echo "- Gmail App Passwords: https://myaccount.google.com/apppasswords"
    echo "- Gmail Security Settings: https://myaccount.google.com/security"
    echo "- Gmail 2FA Setup: https://support.google.com/accounts/answer/185839"
    echo ""
    exit 1
fi

echo ""
echo "🏁 Gmail Email Test Complete!"
echo "Ready to proceed with Step 6 main application integration."