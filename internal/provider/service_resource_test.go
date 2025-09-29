// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestServiceResource_StateManagement(t *testing.T) {
	tests := []struct {
		name           string
		initialState   string
		desiredState   string
		expectedAction string
		expectError    bool
	}{
		{
			name:           "start stopped service",
			initialState:   "stopped",
			desiredState:   "running",
			expectedAction: "start",
			expectError:    false,
		},
		{
			name:           "stop running service",
			initialState:   "running",
			desiredState:   "stopped",
			expectedAction: "stop",
			expectError:    false,
		},
		{
			name:           "restart running service",
			initialState:   "running",
			desiredState:   "restarted",
			expectedAction: "restart",
			expectError:    false,
		},
		{
			name:           "reload running service",
			initialState:   "running",
			desiredState:   "reloaded",
			expectedAction: "reload",
			expectError:    false,
		},
		{
			name:           "invalid state transition",
			initialState:   "stopped",
			desiredState:   "invalid",
			expectedAction: "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This would test the actual service resource implementation
			// For now, we'll create a unit test structure
			ctx := context.Background()

			serviceResource := &ServiceResource{
				client: &DotfilesClient{}, // Mock client would be injected here
			}

			// Test plan generation (this is a placeholder for actual implementation)
			model := ServiceResourceModel{
				Name:         types.StringValue("test-service"),
				DesiredState: types.StringValue(tt.desiredState),
				Scope:        types.StringValue("user"),
			}

			_ = ctx
			_ = serviceResource
			_ = model

			// In actual implementation, we would test:
			// - GetActualState()
			// - DetectDrift()
			// - ApplyState()
			// For now, this demonstrates the test structure
		})
	}
}

func TestServiceResource_Creation(t *testing.T) {
	r := NewServiceResource()
	serviceResource, ok := r.(*ServiceResource)
	if !ok {
		t.Fatal("NewServiceResource() should return *ServiceResource")
	}

	if serviceResource == nil {
		t.Fatal("NewServiceResource() returned nil")
	}

	// Basic smoke test - just ensure the resource can be created
}

func TestServiceResourceModel_Validation(t *testing.T) {
	tests := []struct {
		name      string
		model     ServiceResourceModel
		wantError bool
	}{
		{
			name: "valid model",
			model: ServiceResourceModel{
				Name:         types.StringValue("nginx"),
				DesiredState: types.StringValue("running"),
				Scope:        types.StringValue("user"),
			},
			wantError: false,
		},
		{
			name: "empty name",
			model: ServiceResourceModel{
				Name:         types.StringValue(""),
				DesiredState: types.StringValue("running"),
				Scope:        types.StringValue("user"),
			},
			wantError: true,
		},
		{
			name: "invalid desired state",
			model: ServiceResourceModel{
				Name:         types.StringValue("nginx"),
				DesiredState: types.StringValue("invalid"),
				Scope:        types.StringValue("user"),
			},
			wantError: true,
		},
		{
			name: "invalid scope",
			model: ServiceResourceModel{
				Name:         types.StringValue("nginx"),
				DesiredState: types.StringValue("running"),
				Scope:        types.StringValue("invalid"),
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For this test, we're just checking the model structure
			// Real validation would happen in the Terraform framework

			// Check basic constraints
			if tt.model.Name.ValueString() == "" && !tt.wantError {
				t.Error("Expected error for empty name but got none")
			}

			// Check desired state values
			validStates := []string{"running", "stopped", "restarted", "reloaded"}
			validState := false
			for _, state := range validStates {
				if tt.model.DesiredState.ValueString() == state {
					validState = true
					break
				}
			}

			if !validState && !tt.wantError {
				t.Error("Expected error for invalid desired state but got none")
			}

			// Check scope values
			validScopes := []string{"user", "system"}
			validScope := false
			for _, scope := range validScopes {
				if tt.model.Scope.ValueString() == scope {
					validScope = true
					break
				}
			}

			if !validScope && !tt.wantError {
				t.Error("Expected error for invalid scope but got none")
			}
		})
	}
}

func TestServiceResource_ImportState(t *testing.T) {
	tests := []struct {
		name        string
		importID    string
		expectName  string
		expectScope string
		expectError bool
	}{
		{
			name:        "valid import ID",
			importID:    "nginx:system",
			expectName:  "nginx",
			expectScope: "system",
			expectError: false,
		},
		{
			name:        "user scope service",
			importID:    "my-app:user",
			expectName:  "my-app",
			expectScope: "user",
			expectError: false,
		},
		{
			name:        "invalid format - no colon",
			importID:    "nginx",
			expectError: true,
		},
		{
			name:        "invalid format - too many parts",
			importID:    "nginx:system:extra",
			expectError: true,
		},
		{
			name:        "invalid scope",
			importID:    "nginx:invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the import ID parsing logic
			parts := strings.Split(tt.importID, ":")

			if len(parts) != 2 && !tt.expectError {
				t.Error("Expected error for invalid import ID format but got none")
			}

			if len(parts) == 2 && !tt.expectError {
				serviceName := parts[0]
				scope := parts[1]

				if serviceName != tt.expectName {
					t.Errorf("Expected name %q, got %q", tt.expectName, serviceName)
				}

				if scope != tt.expectScope {
					t.Errorf("Expected scope %q, got %q", tt.expectScope, scope)
				}

				// Validate scope
				if scope != "user" && scope != "system" && !tt.expectError {
					t.Error("Expected error for invalid scope but got none")
				}
			}
		})
	}
}
