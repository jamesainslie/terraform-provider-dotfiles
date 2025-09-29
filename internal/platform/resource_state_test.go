// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package platform

import (
	"context"
	"fmt"
	"testing"
)

// TestDeclarativeResource tests the base declarative resource interface
func TestDeclarativeResource(t *testing.T) {
	tests := []struct {
		name        string
		resource    DeclarativeResource
		expectError bool
		expectDrift bool
	}{
		{
			name:        "mock resource - no drift",
			resource:    &MockResource{hasDrift: false},
			expectError: false,
			expectDrift: false,
		},
		{
			name:        "mock resource - has drift",
			resource:    &MockResource{hasDrift: true},
			expectError: false,
			expectDrift: true,
		},
		{
			name:        "mock resource - validation error",
			resource:    &MockResource{validationError: true},
			expectError: true,
			expectDrift: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Test validation
			err := tt.resource.ValidateState()
			if tt.expectError && err == nil {
				t.Error("expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}

			if tt.expectError {
				return // Skip further tests if validation failed
			}

			// Test desired state
			desiredState := tt.resource.GetDesiredState()
			if desiredState == nil {
				t.Error("GetDesiredState() returned nil")
			}

			// Test actual state
			actualState, err := tt.resource.GetActualState(ctx)
			if err != nil {
				t.Errorf("GetActualState() failed: %v", err)
			}
			if actualState == nil {
				t.Error("GetActualState() returned nil")
			}

			// Test drift detection
			hasDrift, err := tt.resource.DetectDrift()
			if err != nil {
				t.Errorf("DetectDrift() failed: %v", err)
			}
			if hasDrift != tt.expectDrift {
				t.Errorf("expected drift=%v, got drift=%v", tt.expectDrift, hasDrift)
			}

			// Test apply state (only if no drift expected or if we expect to fix drift)
			err = tt.resource.ApplyState(ctx)
			if err != nil {
				t.Errorf("ApplyState() failed: %v", err)
			}

			// Test description
			desc := tt.resource.Description()
			if desc == "" {
				t.Error("Description() returned empty string")
			}
		})
	}
}

// MockResource implements DeclarativeResource for testing
type MockResource struct {
	hasDrift        bool
	validationError bool
	description     string
}

func (m *MockResource) GetDesiredState() interface{} {
	return map[string]interface{}{
		"state": "desired",
	}
}

func (m *MockResource) GetActualState(ctx context.Context) (interface{}, error) {
	if m.hasDrift {
		return map[string]interface{}{
			"state": "actual_different",
		}, nil
	}
	return map[string]interface{}{
		"state": "desired",
	}, nil
}

func (m *MockResource) ApplyState(ctx context.Context) error {
	// Simulate applying state (in real impl, this would make actual == desired)
	m.hasDrift = false
	return nil
}

func (m *MockResource) ValidateState() error {
	if m.validationError {
		return fmt.Errorf("mock validation error")
	}
	return nil
}

func (m *MockResource) DetectDrift() (bool, error) {
	return m.hasDrift, nil
}

func (m *MockResource) Description() string {
	if m.description != "" {
		return m.description
	}
	return "Mock resource for testing"
}
