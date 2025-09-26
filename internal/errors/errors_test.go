// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package errors

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func TestErrorType(t *testing.T) {
	types := []ErrorType{
		ErrorTypeValidation,
		ErrorTypeConfiguration,
		ErrorTypePermission,
		ErrorTypeIO,
		ErrorTypeTemplate,
		ErrorTypeGit,
		ErrorTypeNetwork,
	}

	for _, errorType := range types {
		t.Run(errorType.String(), func(t *testing.T) {
			if errorType.String() == "" {
				t.Error("Error type should not be empty")
			}
		})
	}
}

func TestNewProviderError(t *testing.T) {
	err := NewProviderError(ErrorTypeValidation, "test-operation", "test-resource", "test message", nil)

	if err == nil {
		t.Fatal("Expected error to be created")
	}

	if err.Type != ErrorTypeValidation {
		t.Errorf("Expected type %v, got %v", ErrorTypeValidation, err.Type)
	}

	if err.Message != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", err.Message)
	}

	if err.Resource != "test-resource" {
		t.Errorf("Expected resource 'test-resource', got '%s'", err.Resource)
	}

	if err.Operation != "test-operation" {
		t.Errorf("Expected operation 'test-operation', got '%s'", err.Operation)
	}

	if err.Retryable {
		t.Error("Expected error to not be retryable by default")
	}
}

func TestProviderError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ProviderError
		contains []string
	}{
		{
			name: "basic error",
			err: &ProviderError{
				Type:    ErrorTypeValidation,
				Message: "validation failed",
			},
			contains: []string{"validation failed"},
		},
		{
			name: "error with context",
			err: &ProviderError{
				Type:      ErrorTypeIO,
				Message:   "file not found",
				Resource:  "dotfiles_file",
				Path:      "/path/to/file",
				Operation: "read",
			},
			contains: []string{"file not found", "dotfiles_file", "/path/to/file", "read"},
		},
		{
			name: "error with cause",
			err: &ProviderError{
				Type:    ErrorTypeConfiguration,
				Message: "unexpected error",
				Cause:   errors.New("underlying cause"),
			},
			contains: []string{"unexpected error", "underlying cause"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected error message to contain '%s', got: %s", expected, result)
				}
			}
		})
	}
}

func TestProviderError_Unwrap(t *testing.T) {
	cause := errors.New("underlying cause")
	err := &ProviderError{
		Type:    ErrorTypeConfiguration,
		Message: "wrapper error",
		Cause:   cause,
	}

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Expected unwrapped error to be %v, got %v", cause, unwrapped)
	}

	// Test with no cause
	errNoCause := &ProviderError{
		Type:    ErrorTypeValidation,
		Message: "no cause",
	}

	unwrappedNoCause := errNoCause.Unwrap()
	if unwrappedNoCause != nil {
		t.Errorf("Expected unwrapped error to be nil, got %v", unwrappedNoCause)
	}
}

func TestSpecificErrorConstructors(t *testing.T) {
	tests := []struct {
		name         string
		constructor  func(string, string, string, error) *ProviderError
		expectedType ErrorType
	}{
		{"ValidationError", ValidationError, ErrorTypeValidation},
		{"ConfigurationError", ConfigurationError, ErrorTypeConfiguration},
		{"PermissionError", PermissionError, ErrorTypePermission},
		{"IOError", IOError, ErrorTypeIO},
		{"TemplateError", TemplateError, ErrorTypeTemplate},
		{"GitError", GitError, ErrorTypeGit},
		{"NetworkError", NetworkError, ErrorTypeNetwork},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constructor("test-operation", "test-resource", "test message", nil)

			if err.Type != tt.expectedType {
				t.Errorf("Expected type %v, got %v", tt.expectedType, err.Type)
			}

			if err.Message != "test message" {
				t.Errorf("Expected message 'test message', got '%s'", err.Message)
			}
		})
	}
}

func TestAddErrorToDiagnostics(t *testing.T) {
	var diags diag.Diagnostics
	ctx := context.Background()

	err := ValidationError("create", "dotfiles_file", "validation failed", nil).WithPath("/path/to/file")

	AddErrorToDiagnostics(ctx, &diags, err, "Test Error")

	if len(diags) != 1 {
		t.Fatalf("Expected 1 diagnostic, got %d", len(diags))
	}

	diagnostic := diags[0]
	if diagnostic.Severity() != diag.SeverityError {
		t.Errorf("Expected error severity, got %s", diagnostic.Severity())
	}

	if diagnostic.Summary() != "Test Error" {
		t.Errorf("Expected summary 'Test Error', got '%s'", diagnostic.Summary())
	}
}

func TestAddWarningToDiagnostics(t *testing.T) {
	var diags diag.Diagnostics
	ctx := context.Background()

	AddWarningToDiagnostics(ctx, &diags, "Test Warning", "This is a warning message")

	if len(diags) != 1 {
		t.Fatalf("Expected 1 diagnostic, got %d", len(diags))
	}

	diagnostic := diags[0]
	if diagnostic.Severity() != diag.SeverityWarning {
		t.Errorf("Expected warning severity, got %s", diagnostic.Severity())
	}

	if diagnostic.Summary() != "Test Warning" {
		t.Errorf("Expected summary 'Test Warning', got '%s'", diagnostic.Summary())
	}
}

func TestRetryConfig(t *testing.T) {
	config := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
		Multiplier:  2.0,
	}

	if config.MaxAttempts != 3 {
		t.Errorf("Expected max attempts 3, got %d", config.MaxAttempts)
	}

	if config.BaseDelay != 100*time.Millisecond {
		t.Errorf("Expected base delay 100ms, got %v", config.BaseDelay)
	}

	if config.MaxDelay != 1*time.Second {
		t.Errorf("Expected max delay 1s, got %v", config.MaxDelay)
	}

	if config.Multiplier != 2.0 {
		t.Errorf("Expected multiplier 2.0, got %f", config.Multiplier)
	}
}

func TestRetrySuccess(t *testing.T) {
	attempts := 0
	operation := func() error {
		attempts++
		if attempts < 2 {
			return IOError("test", "test", "temporary failure", nil).WithRetryable(true)
		}
		return nil
	}

	config := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Multiplier:  2.0,
	}

	ctx := context.Background()
	err := Retry(ctx, config, operation)

	if err != nil {
		t.Errorf("Expected retry to succeed, got error: %v", err)
	}

	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

func TestRetryFailure(t *testing.T) {
	attempts := 0
	operation := func() error {
		attempts++
		return IOError("test", "test", "persistent failure", nil).WithRetryable(true)
	}

	config := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Multiplier:  2.0,
	}

	ctx := context.Background()
	err := Retry(ctx, config, operation)

	if err == nil {
		t.Error("Expected retry to fail")
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	// Should be a ProviderError
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) {
		t.Error("Expected ProviderError")
	}

	if providerErr.Message != "persistent failure" {
		t.Errorf("Expected original error message, got '%s'", providerErr.Message)
	}
}

func TestRetryNonRetryableError(t *testing.T) {
	attempts := 0
	operation := func() error {
		attempts++
		return ValidationError("test", "test", "non-retryable error", nil)
	}

	config := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Multiplier:  2.0,
	}

	ctx := context.Background()
	err := Retry(ctx, config, operation)

	if err == nil {
		t.Error("Expected error")
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt for non-retryable error, got %d", attempts)
	}
}

func TestRetryContextCancellation(t *testing.T) {
	attempts := 0
	operation := func() error {
		attempts++
		time.Sleep(50 * time.Millisecond) // Simulate slow operation
		return IOError("test", "test", "slow operation", nil).WithRetryable(true)
	}

	config := RetryConfig{
		MaxAttempts: 5,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
		Multiplier:  2.0,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Millisecond)
	defer cancel()

	err := Retry(ctx, config, operation)

	if err == nil {
		t.Error("Expected error due to context cancellation")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context deadline exceeded error, got: %v", err)
	}

	// Should have attempted at least once
	if attempts < 1 {
		t.Errorf("Expected at least 1 attempt, got %d", attempts)
	}
}

func TestRetryWithStandardError(t *testing.T) {
	attempts := 0
	standardErr := fmt.Errorf("standard go error")

	operation := func() error {
		attempts++
		if attempts < 2 {
			return standardErr
		}
		return nil
	}

	config := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Multiplier:  2.0,
	}

	ctx := context.Background()
	err := Retry(ctx, config, operation)

	// Standard errors should not be retried
	if err == nil {
		t.Error("Expected error")
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt for standard error, got %d", attempts)
	}

	if err != standardErr {
		t.Errorf("Expected original standard error, got %v", err)
	}
}

func TestProviderErrorChaining(t *testing.T) {
	rootCause := errors.New("root cause")

	wrappedErr := IOError("test", "test", "io operation failed", rootCause)

	finalErr := ConfigurationError("test", "test", "internal processing failed", wrappedErr)

	// Test error chain
	if !errors.Is(finalErr, wrappedErr) {
		t.Error("Expected finalErr to wrap wrappedErr")
	}

	if !errors.Is(finalErr, rootCause) {
		t.Error("Expected finalErr to ultimately wrap rootCause")
	}

	// Test error message includes chain
	errorMsg := finalErr.Error()
	if !strings.Contains(errorMsg, "internal processing failed") {
		t.Error("Expected final error message in chain")
	}

	if !strings.Contains(errorMsg, "io operation failed") {
		t.Error("Expected wrapped error message in chain")
	}
}
