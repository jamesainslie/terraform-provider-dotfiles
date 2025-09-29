// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package platform

import (
	"testing"
)

func TestProcessManager(t *testing.T) {
	tests := []struct {
		name            string
		manager         ProcessManager
		processName     string
		expectError     bool
		expectProcesses int
	}{
		{
			name:            "find existing processes",
			manager:         &MockProcessManager{processes: map[int]*Process{123: {PID: 123, Name: "test-app"}}},
			processName:     "test-app",
			expectError:     false,
			expectProcesses: 1,
		},
		{
			name:            "find no processes",
			manager:         &MockProcessManager{processes: make(map[int]*Process)},
			processName:     "non-existent",
			expectError:     false,
			expectProcesses: 0,
		},
		{
			name:            "error finding processes",
			manager:         &MockProcessManager{shouldError: true},
			processName:     "test-app",
			expectError:     true,
			expectProcesses: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processes, err := tt.manager.FindProcessesByName(tt.processName)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(processes) != tt.expectProcesses {
				t.Errorf("expected %d processes, got %d", tt.expectProcesses, len(processes))
			}
		})
	}
}

func TestFileManager(t *testing.T) {
	tests := []struct {
		name        string
		manager     FileManager
		path        string
		mode        FilePermission
		expectError bool
	}{
		{
			name:        "set permissions successfully",
			manager:     &MockFileManager{permissions: make(map[string]FilePermission)},
			path:        "/test/file",
			mode:        FilePermission{Mode: 0644},
			expectError: false,
		},
		{
			name:        "error setting permissions",
			manager:     &MockFileManager{shouldError: true},
			path:        "/test/file",
			mode:        FilePermission{Mode: 0644},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.manager.SetPermissions(tt.path, tt.mode)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestNotificationManager(t *testing.T) {
	tests := []struct {
		name        string
		manager     NotificationManager
		title       string
		message     string
		level       NotificationLevel
		expectError bool
	}{
		{
			name:        "send notification successfully",
			manager:     &MockNotificationManager{},
			title:       "Test",
			message:     "Test message",
			level:       NotificationInfo,
			expectError: false,
		},
		{
			name:        "error sending notification",
			manager:     &MockNotificationManager{shouldError: true},
			title:       "Test",
			message:     "Test message",
			level:       NotificationInfo,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.manager.SendDesktopNotification(tt.title, tt.message, tt.level)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestMockPlatformProvider(t *testing.T) {
	// Create a mock base platform provider
	basePlatform := &LinuxProvider{BasePlatform{platform: "linux", architecture: "amd64"}}
	mockProvider := NewMockPlatformProvider(basePlatform)

	// Test that all interfaces are implemented
	if mockProvider.ProcessManager() == nil {
		t.Error("ProcessManager() returned nil")
	}
	if mockProvider.FileManager() == nil {
		t.Error("FileManager() returned nil")
	}
	if mockProvider.NotificationManager() == nil {
		t.Error("NotificationManager() returned nil")
	}

	// Test base platform functionality is preserved
	if mockProvider.GetPlatform() != "linux" {
		t.Errorf("Expected platform 'linux', got '%s'", mockProvider.GetPlatform())
	}
}

// Mock implementations are in dotfiles_platform_provider.go
