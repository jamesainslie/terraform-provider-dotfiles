// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package platform

import (
	"fmt"
	"testing"
	"time"
)

func TestServiceManager(t *testing.T) {
	tests := []struct {
		name               string
		manager            ServiceManager
		serviceName        string
		userLevel          bool
		expectError        bool
		initialState       string
		expectedFinalState string
	}{
		{
			name:               "start stopped service",
			manager:            &MockServiceManager{currentState: "stopped", serviceExists: true},
			serviceName:        "test-service",
			userLevel:          true,
			expectError:        false,
			initialState:       "stopped",
			expectedFinalState: "running",
		},
		{
			name:               "stop running service",
			manager:            &MockServiceManager{currentState: "running", serviceExists: true},
			serviceName:        "test-service",
			userLevel:          true,
			expectError:        false,
			initialState:       "running",
			expectedFinalState: "stopped",
		},
		{
			name:               "restart service",
			manager:            &MockServiceManager{currentState: "running", serviceExists: true},
			serviceName:        "test-service",
			userLevel:          false,
			expectError:        false,
			initialState:       "running",
			expectedFinalState: "running",
		},
		{
			name:               "service does not exist",
			manager:            &MockServiceManager{serviceExists: false},
			serviceName:        "non-existent-service",
			userLevel:          true,
			expectError:        true,
			initialState:       "unknown",
			expectedFinalState: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test service existence check
			exists := tt.manager.ServiceExists(tt.serviceName, tt.userLevel)
			expectedExists := !tt.expectError
			if exists != expectedExists {
				t.Errorf("ServiceExists() = %v, expected %v", exists, expectedExists)
			}

			if !exists {
				return // Skip further tests if service doesn't exist
			}

			// Test getting initial status
			status, err := tt.manager.GetServiceStatus(tt.serviceName, tt.userLevel)
			if err != nil && !tt.expectError {
				t.Errorf("GetServiceStatus() failed: %v", err)
				return
			}
			if err == nil && tt.expectError {
				t.Error("expected GetServiceStatus() to fail but it succeeded")
				return
			}

			if !tt.expectError {
				if status.State != tt.initialState {
					t.Errorf("initial state = %s, expected %s", status.State, tt.initialState)
				}

				// Test operations based on desired final state
				switch tt.expectedFinalState {
				case "running":
					err = tt.manager.StartService(tt.serviceName, tt.userLevel)
				case "stopped":
					err = tt.manager.StopService(tt.serviceName, tt.userLevel)
				}

				if err != nil {
					t.Errorf("service operation failed: %v", err)
					return
				}

				// Verify final state
				finalStatus, err := tt.manager.GetServiceStatus(tt.serviceName, tt.userLevel)
				if err != nil {
					t.Errorf("GetServiceStatus() after operation failed: %v", err)
					return
				}

				if finalStatus.State != tt.expectedFinalState {
					t.Errorf("final state = %s, expected %s", finalStatus.State, tt.expectedFinalState)
				}
			}
		})
	}
}

func TestServiceManagerRestart(t *testing.T) {
	manager := &MockServiceManager{currentState: "running", serviceExists: true}
	serviceName := "test-service"

	// Test restart operation
	err := manager.RestartService(serviceName, true)
	if err != nil {
		t.Errorf("RestartService() failed: %v", err)
	}

	// Service should still be running after restart
	status, err := manager.GetServiceStatus(serviceName, true)
	if err != nil {
		t.Errorf("GetServiceStatus() after restart failed: %v", err)
	}
	if status.State != "running" {
		t.Errorf("service state after restart = %s, expected running", status.State)
	}
}

func TestServiceManagerReload(t *testing.T) {
	manager := &MockServiceManager{currentState: "running", serviceExists: true, supportsReload: true}
	serviceName := "test-service"

	// Test reload operation
	err := manager.ReloadService(serviceName, true)
	if err != nil {
		t.Errorf("ReloadService() failed: %v", err)
	}

	// Service should still be running after reload
	status, err := manager.GetServiceStatus(serviceName, true)
	if err != nil {
		t.Errorf("GetServiceStatus() after reload failed: %v", err)
	}
	if status.State != "running" {
		t.Errorf("service state after reload = %s, expected running", status.State)
	}
}

// MockServiceManager implements ServiceManager for testing
type MockServiceManager struct {
	currentState   string
	serviceExists  bool
	supportsReload bool
	operationDelay time.Duration
	shouldError    bool
}

func (m *MockServiceManager) StartService(name string, userLevel bool) error {
	if m.shouldError {
		return fmt.Errorf("mock error starting service")
	}
	if !m.serviceExists {
		return fmt.Errorf("service does not exist")
	}
	time.Sleep(m.operationDelay)
	m.currentState = "running"
	return nil
}

func (m *MockServiceManager) StopService(name string, userLevel bool) error {
	if m.shouldError {
		return fmt.Errorf("mock error stopping service")
	}
	if !m.serviceExists {
		return fmt.Errorf("service does not exist")
	}
	time.Sleep(m.operationDelay)
	m.currentState = "stopped"
	return nil
}

func (m *MockServiceManager) RestartService(name string, userLevel bool) error {
	if m.shouldError {
		return fmt.Errorf("mock error restarting service")
	}
	if !m.serviceExists {
		return fmt.Errorf("service does not exist")
	}
	time.Sleep(m.operationDelay)
	// Simulate restart: briefly stopped, then running
	m.currentState = "running"
	return nil
}

func (m *MockServiceManager) ReloadService(name string, userLevel bool) error {
	if !m.supportsReload {
		return fmt.Errorf("service does not support reload")
	}
	if m.shouldError {
		return fmt.Errorf("mock error reloading service")
	}
	if !m.serviceExists {
		return fmt.Errorf("service does not exist")
	}
	time.Sleep(m.operationDelay)
	// Reload keeps service running but refreshes config
	return nil
}

func (m *MockServiceManager) GetServiceStatus(name string, userLevel bool) (ServiceStatus, error) {
	if m.shouldError {
		return ServiceStatus{}, fmt.Errorf("mock error getting service status")
	}
	if !m.serviceExists {
		return ServiceStatus{}, fmt.Errorf("service does not exist")
	}

	return ServiceStatus{
		Name:           name,
		State:          m.currentState,
		UserLevel:      userLevel,
		ProcessID:      12345,
		StartTime:      time.Now().Add(-1 * time.Hour),
		SupportsReload: m.supportsReload,
	}, nil
}

func (m *MockServiceManager) ServiceExists(name string, userLevel bool) bool {
	return m.serviceExists
}
