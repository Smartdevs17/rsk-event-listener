package utils

import (
	"fmt"
	"runtime"
)

// AppError represents an application error with context
type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Details    string `json:"details,omitempty"`
	File       string `json:"file,omitempty"`
	Line       int    `json:"line,omitempty"`
	StackTrace string `json:"stack_trace,omitempty"`
}

func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewAppError creates a new application error
func NewAppError(code, message string, details ...string) *AppError {
	_, file, line, _ := runtime.Caller(1)

	err := &AppError{
		Code:    code,
		Message: message,
		File:    file,
		Line:    line,
	}

	if len(details) > 0 {
		err.Details = details[0]
	}

	return err
}

// WithStackTrace adds stack trace to the error
func (e *AppError) WithStackTrace() *AppError {
	buf := make([]byte, 1024)
	n := runtime.Stack(buf, false)
	e.StackTrace = string(buf[:n])
	return e
}

// Common error codes
const (
	ErrCodeConnection    = "CONNECTION_ERROR"
	ErrCodeDatabase      = "DATABASE_ERROR"
	ErrCodeValidation    = "VALIDATION_ERROR"
	ErrCodeNotFound      = "NOT_FOUND"
	ErrCodeInternal      = "INTERNAL_ERROR"
	ErrCodeBlockchain    = "BLOCKCHAIN_ERROR"
	ErrCodeConfiguration = "CONFIGURATION_ERROR"
	ErrCodeProcessing    = "PROCESSING_ERROR"
)
