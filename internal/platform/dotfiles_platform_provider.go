// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package platform

import (
	"context"
	"fmt"
)

// MockPlatformProvider extends the existing platform providers with dotfiles-specific management capabilities
// This is used for testing dotfiles operations (file permissions, backups, notifications, process management)
type MockPlatformProvider struct {
	PlatformProvider
	mockProcessManager      *MockProcessManager
	mockFileManager         *MockFileManager
	mockNotificationManager *MockNotificationManager
}

// NewMockPlatformProvider creates a new mock platform provider for dotfiles operations
func NewMockPlatformProvider(base PlatformProvider) *MockPlatformProvider {
	return &MockPlatformProvider{
		PlatformProvider:        base,
		mockProcessManager:      &MockProcessManager{processes: make(map[int]*Process)},
		mockFileManager:         &MockFileManager{permissions: make(map[string]FilePermission)},
		mockNotificationManager: &MockNotificationManager{},
	}
}

func (m *MockPlatformProvider) ProcessManager() ProcessManager {
	return m.mockProcessManager
}

func (m *MockPlatformProvider) FileManager() FileManager {
	return m.mockFileManager
}

func (m *MockPlatformProvider) NotificationManager() NotificationManager {
	return m.mockNotificationManager
}

// MockProcessManager implements ProcessManager for testing
type MockProcessManager struct {
	processes   map[int]*Process
	shouldError bool
}

func (m *MockProcessManager) FindProcessesByName(name string) ([]Process, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error finding processes")
	}
	var results []Process
	for _, p := range m.processes {
		if p.Name == name {
			results = append(results, *p)
		}
	}
	return results, nil
}

func (m *MockProcessManager) FindProcessesByPattern(pattern string) ([]Process, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error finding processes by pattern")
	}
	// Simple mock implementation
	var results []Process
	for _, p := range m.processes {
		if p.Command == pattern {
			results = append(results, *p)
		}
	}
	return results, nil
}

func (m *MockProcessManager) SendSignalToProcess(pid int, signal ProcessSignal) error {
	if m.shouldError {
		return fmt.Errorf("mock error sending signal")
	}
	if _, exists := m.processes[pid]; !exists {
		return fmt.Errorf("process not found")
	}
	return nil
}

func (m *MockProcessManager) TerminateProcess(pid int, graceful bool) error {
	if m.shouldError {
		return fmt.Errorf("mock error terminating process")
	}
	delete(m.processes, pid)
	return nil
}

func (m *MockProcessManager) IsProcessRunning(pid int) bool {
	_, exists := m.processes[pid]
	return exists
}

func (m *MockProcessManager) GetProcessInfo(pid int) (*Process, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error getting process info")
	}
	p, exists := m.processes[pid]
	if !exists {
		return nil, fmt.Errorf("process not found")
	}
	return p, nil
}

// MockFileManager implements FileManager for testing
type MockFileManager struct {
	permissions map[string]FilePermission
	shouldError bool
}

func (m *MockFileManager) SetPermissions(path string, mode FilePermission) error {
	if m.shouldError {
		return fmt.Errorf("mock error setting permissions")
	}
	m.permissions[path] = mode
	return nil
}

func (m *MockFileManager) SetOwnership(path string, owner, group string) error {
	if m.shouldError {
		return fmt.Errorf("mock error setting ownership")
	}
	return nil
}

func (m *MockFileManager) CreateBackup(source, backup string, format BackupFormat) error {
	if m.shouldError {
		return fmt.Errorf("mock error creating backup")
	}
	return nil
}

func (m *MockFileManager) ValidateBackup(backup string) error {
	if m.shouldError {
		return fmt.Errorf("mock error validating backup")
	}
	return nil
}

func (m *MockFileManager) RestoreBackup(backup, target string) error {
	if m.shouldError {
		return fmt.Errorf("mock error restoring backup")
	}
	return nil
}

func (m *MockFileManager) GetFilePermissions(path string) (FilePermission, error) {
	if m.shouldError {
		return FilePermission{}, fmt.Errorf("mock error getting permissions")
	}
	perm, exists := m.permissions[path]
	if !exists {
		return FilePermission{Mode: 0644}, nil
	}
	return perm, nil
}

func (m *MockFileManager) GetFileOwnership(path string) (string, string, error) {
	if m.shouldError {
		return "", "", fmt.Errorf("mock error getting ownership")
	}
	return "testuser", "testgroup", nil
}

// MockNotificationManager implements NotificationManager for testing
type MockNotificationManager struct {
	notifications []string
	shouldError   bool
}

func (m *MockNotificationManager) SendDesktopNotification(title, message string, level NotificationLevel) error {
	if m.shouldError {
		return fmt.Errorf("mock error sending desktop notification")
	}
	m.notifications = append(m.notifications, fmt.Sprintf("%s: %s (%s)", title, message, level))
	return nil
}

func (m *MockNotificationManager) WriteLogNotification(message string, level LogLevel, fields map[string]interface{}) error {
	if m.shouldError {
		return fmt.Errorf("mock error writing log notification")
	}
	m.notifications = append(m.notifications, fmt.Sprintf("LOG %s: %s", level, message))
	return nil
}

func (m *MockNotificationManager) SendWebhookNotification(ctx context.Context, url string, payload interface{}) error {
	if m.shouldError {
		return fmt.Errorf("mock error sending webhook notification")
	}
	m.notifications = append(m.notifications, fmt.Sprintf("WEBHOOK %s: %+v", url, payload))
	return nil
}

func (m *MockNotificationManager) IsDesktopNotificationSupported() bool {
	return !m.shouldError
}
