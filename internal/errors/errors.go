// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package errors

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// ErrorType represents different categories of errors for better handling.
type ErrorType int

const (
	// ErrorTypeUnknown represents an unclassified error.
	ErrorTypeUnknown ErrorType = iota
	// ErrorTypeValidation represents validation errors.
	ErrorTypeValidation
	// ErrorTypePermission represents permission/access errors.
	ErrorTypePermission
	// ErrorTypeNetwork represents network-related errors.
	ErrorTypeNetwork
	// ErrorTypeIO represents I/O operation errors.
	ErrorTypeIO
	// ErrorTypeGit represents Git operation errors.
	ErrorTypeGit
	// ErrorTypeTemplate represents template processing errors.
	ErrorTypeTemplate
	// ErrorTypeConfiguration represents configuration errors.
	ErrorTypeConfiguration
)

// String returns a string representation of the error type.
func (et ErrorType) String() string {
	switch et {
	case ErrorTypeValidation:
		return "validation"
	case ErrorTypePermission:
		return "permission"
	case ErrorTypeNetwork:
		return "network"
	case ErrorTypeIO:
		return "io"
	case ErrorTypeGit:
		return "git"
	case ErrorTypeTemplate:
		return "template"
	case ErrorTypeConfiguration:
		return "configuration"
	default:
		return "unknown"
	}
}

// ProviderError represents an enhanced error with context and type information.
type ProviderError struct {
	Type      ErrorType
	Operation string
	Resource  string
	Path      string
	Message   string
	Cause     error
	Context   map[string]interface{}
	Retryable bool
}

// Error implements the error interface.
func (pe *ProviderError) Error() string {
	var parts []string

	if pe.Operation != "" {
		parts = append(parts, fmt.Sprintf("operation=%s", pe.Operation))
	}
	if pe.Resource != "" {
		parts = append(parts, fmt.Sprintf("resource=%s", pe.Resource))
	}
	if pe.Path != "" {
		parts = append(parts, fmt.Sprintf("path=%s", pe.Path))
	}

	contextStr := ""
	if len(parts) > 0 {
		contextStr = fmt.Sprintf(" (%s)", strings.Join(parts, ", "))
	}

	if pe.Cause != nil {
		return fmt.Sprintf("%s%s: %v", pe.Message, contextStr, pe.Cause)
	}
	return fmt.Sprintf("%s%s", pe.Message, contextStr)
}

// Unwrap returns the underlying cause error.
func (pe *ProviderError) Unwrap() error {
	return pe.Cause
}

// IsRetryable returns whether this error might succeed on retry.
func (pe *ProviderError) IsRetryable() bool {
	return pe.Retryable
}

// NewProviderError creates a new provider error with context.
func NewProviderError(errType ErrorType, operation, resource, message string, cause error) *ProviderError {
	return &ProviderError{
		Type:      errType,
		Operation: operation,
		Resource:  resource,
		Message:   message,
		Cause:     cause,
		Context:   make(map[string]interface{}),
		Retryable: isRetryableError(errType, cause),
	}
}

// WithPath adds path context to the error.
func (pe *ProviderError) WithPath(path string) *ProviderError {
	pe.Path = path
	return pe
}

// WithContext adds additional context to the error.
func (pe *ProviderError) WithContext(key string, value interface{}) *ProviderError {
	pe.Context[key] = value
	return pe
}

// WithRetryable sets whether this error is retryable.
func (pe *ProviderError) WithRetryable(retryable bool) *ProviderError {
	pe.Retryable = retryable
	return pe
}

// isRetryableError determines if an error type/cause combination might succeed on retry.
func isRetryableError(errType ErrorType, cause error) bool {
	switch errType {
	case ErrorTypeNetwork, ErrorTypeGit:
		return true // Network and Git operations often benefit from retry
	case ErrorTypeIO:
		// Some I/O errors are retryable (temporary file locks, etc.)
		if cause != nil {
			errStr := cause.Error()
			return strings.Contains(errStr, "resource temporarily unavailable") ||
				strings.Contains(errStr, "device or resource busy") ||
				strings.Contains(errStr, "interrupted system call")
		}
		return false
	default:
		return false
	}
}

// AddErrorToDiagnostics adds a ProviderError to Terraform diagnostics with enhanced context.
func AddErrorToDiagnostics(ctx context.Context, diags *diag.Diagnostics, err error, summary string) {
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		// Enhanced error with context
		detail := providerErr.Error()

		// Add context information to the detail
		if len(providerErr.Context) > 0 {
			var contextParts []string
			for k, v := range providerErr.Context {
				contextParts = append(contextParts, fmt.Sprintf("%s: %v", k, v))
			}
			detail += fmt.Sprintf("\nContext: %s", strings.Join(contextParts, ", "))
		}

		// Log the error with structured context
		tflog.Error(ctx, "Provider error occurred", map[string]interface{}{
			"error_type": providerErr.Type.String(),
			"operation":  providerErr.Operation,
			"resource":   providerErr.Resource,
			"path":       providerErr.Path,
			"retryable":  providerErr.Retryable,
			"message":    providerErr.Message,
			"context":    providerErr.Context,
		})

		diags.AddError(summary, detail)
	} else {
		// Regular error
		tflog.Error(ctx, "Untyped error occurred", map[string]interface{}{
			"error": err.Error(),
		})
		diags.AddError(summary, err.Error())
	}
}

// AddWarningToDiagnostics adds a warning to Terraform diagnostics with logging.
func AddWarningToDiagnostics(ctx context.Context, diags *diag.Diagnostics, message, detail string) {
	tflog.Warn(ctx, "Provider warning", map[string]interface{}{
		"message": message,
		"detail":  detail,
	})
	diags.AddWarning(message, detail)
}

// RetryConfig defines configuration for retry operations.
type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Multiplier  float64
}

// DefaultRetryConfig returns a sensible default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   time.Second,
		MaxDelay:    30 * time.Second,
		Multiplier:  2.0,
	}
}

// RetryableOperation represents an operation that can be retried.
type RetryableOperation func() error

// Retry executes a retryable operation with exponential backoff.
func Retry(ctx context.Context, config RetryConfig, operation RetryableOperation) error {
	var lastErr error

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := operation()
		if err == nil {
			if attempt > 1 {
				tflog.Info(ctx, "Retry operation succeeded", map[string]interface{}{
					"attempt": attempt,
				})
			}
			return nil
		}

		lastErr = err

		// Check if error is retryable
		var providerErr *ProviderError
		if errors.As(err, &providerErr) && !providerErr.IsRetryable() {
			tflog.Debug(ctx, "Error is not retryable, stopping", map[string]interface{}{
				"error_type": providerErr.Type.String(),
				"attempt":    attempt,
			})
			return err
		}

		if attempt < config.MaxAttempts {
			// Calculate delay with exponential backoff
			delay := time.Duration(float64(config.BaseDelay) * (config.Multiplier * float64(attempt-1)))
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}

			tflog.Info(ctx, "Retrying operation after delay", map[string]interface{}{
				"attempt":      attempt,
				"max_attempts": config.MaxAttempts,
				"delay_ms":     delay.Milliseconds(),
				"error":        err.Error(),
			})

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	tflog.Error(ctx, "All retry attempts exhausted", map[string]interface{}{
		"max_attempts": config.MaxAttempts,
		"final_error":  lastErr.Error(),
	})

	return fmt.Errorf("operation failed after %d attempts: %w", config.MaxAttempts, lastErr)
}

// ValidationError creates a validation error.
func ValidationError(operation, resource, message string, cause error) *ProviderError {
	return NewProviderError(ErrorTypeValidation, operation, resource, message, cause)
}

// PermissionError creates a permission error.
func PermissionError(operation, resource, message string, cause error) *ProviderError {
	return NewProviderError(ErrorTypePermission, operation, resource, message, cause)
}

// NetworkError creates a network error (retryable by default).
func NetworkError(operation, resource, message string, cause error) *ProviderError {
	return NewProviderError(ErrorTypeNetwork, operation, resource, message, cause)
}

// IOError creates an I/O error.
func IOError(operation, resource, message string, cause error) *ProviderError {
	return NewProviderError(ErrorTypeIO, operation, resource, message, cause)
}

// GitError creates a Git operation error (retryable by default).
func GitError(operation, resource, message string, cause error) *ProviderError {
	return NewProviderError(ErrorTypeGit, operation, resource, message, cause)
}

// TemplateError creates a template processing error.
func TemplateError(operation, resource, message string, cause error) *ProviderError {
	return NewProviderError(ErrorTypeTemplate, operation, resource, message, cause)
}

// ConfigurationError creates a configuration error.
func ConfigurationError(operation, resource, message string, cause error) *ProviderError {
	return NewProviderError(ErrorTypeConfiguration, operation, resource, message, cause)
}
