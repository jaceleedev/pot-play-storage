package errors

import (
	"errors"
	"fmt"
	"strings"
)

// SecurityError represents a security-related error
type SecurityError struct {
	UserMessage    string // Safe message for users
	InternalError  error  // Internal error with sensitive details
	ErrorCode      string // Error code for logging
}

func (e *SecurityError) Error() string {
	return e.UserMessage
}

func (e *SecurityError) Unwrap() error {
	return e.InternalError
}

// NewSecurityError creates a new security error with sanitized user message
func NewSecurityError(code, userMsg string, internalErr error) *SecurityError {
	return &SecurityError{
		UserMessage:   userMsg,
		InternalError: internalErr,
		ErrorCode:     code,
	}
}

// SanitizeError removes sensitive information from error messages
func SanitizeError(err error) string {
	if err == nil {
		return ""
	}

	errMsg := err.Error()
	
	// Remove file paths that might contain sensitive information
	if strings.Contains(errMsg, "/") {
		return "file operation failed"
	}
	
	// Remove database connection strings
	if strings.Contains(errMsg, "postgres://") || strings.Contains(errMsg, "password") {
		return "database connection failed"
	}
	
	// Remove system paths
	sensitiveKeywords := []string{
		"/etc/", "/var/", "/usr/", "/home/", "/root/",
		"no such file or directory",
		"permission denied",
		"access denied",
	}
	
	lowerErrMsg := strings.ToLower(errMsg)
	for _, keyword := range sensitiveKeywords {
		if strings.Contains(lowerErrMsg, keyword) {
			return "operation failed"
		}
	}
	
	// Generic error messages for common issues
	if strings.Contains(lowerErrMsg, "connection refused") {
		return "service unavailable"
	}
	
	if strings.Contains(lowerErrMsg, "timeout") {
		return "request timeout"
	}
	
	// For other errors, return a generic message
	return "internal server error"
}

// Common security errors
var (
	ErrInvalidPath       = NewSecurityError("INVALID_PATH", "invalid file path", errors.New("path validation failed"))
	ErrUnauthorized      = NewSecurityError("UNAUTHORIZED", "unauthorized access", errors.New("authentication failed"))
	ErrForbidden         = NewSecurityError("FORBIDDEN", "access denied", errors.New("authorization failed"))
	ErrInvalidFileType   = NewSecurityError("INVALID_FILE", "unsupported file type", errors.New("file type validation failed"))
	ErrFileTooLarge      = NewSecurityError("FILE_TOO_LARGE", "file size exceeds limit", errors.New("file size validation failed"))
	ErrInvalidInput      = NewSecurityError("INVALID_INPUT", "invalid input provided", errors.New("input validation failed"))
)

// FileOperationError creates appropriate error for file operations
func FileOperationError(operation string, err error) *SecurityError {
	return NewSecurityError(
		fmt.Sprintf("FILE_%s_FAILED", strings.ToUpper(operation)),
		fmt.Sprintf("file %s operation failed", operation),
		err,
	)
}