// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package platform

import (
	"context"
	"fmt"
	"time"
)

// MockPlatformProvider extends the existing platform providers with service management capabilities
// This is used for testing and as a base for platform-specific implementations
type MockPlatformProvider struct {
	PlatformProvider
	mockServiceManager      *ExtendedMockServiceManager
	mockProcessManager      *MockProcessManager
	mockFileManager         *MockFileManager
	mockNotificationManager *MockNotificationManager
}

// NewMockPlatformProvider creates a new mock platform provider with service management capabilities
func NewMockPlatformProvider(base PlatformProvider) *MockPlatformProvider {
	return &MockPlatformProvider{
		PlatformProvider:        base,
		mockServiceManager:      NewExtendedMockServiceManager(),
		mockProcessManager:      &MockProcessManager{processes: make(map[int]*Process)},
		mockFileManager:         &MockFileManager{permissions: make(map[string]FilePermission)},
		mockNotificationManager: &MockNotificationManager{},
	}
}

func (m *MockPlatformProvider) ServiceManager() ServiceManager {
	return m.mockServiceManager
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

// ExtendedMockServiceManager extends MockServiceManager with additional test helpers
type ExtendedMockServiceManager struct {
	services       map[string]ServiceStatus
	shouldError    bool
	operationDelay time.Duration
}

func NewExtendedMockServiceManager() *ExtendedMockServiceManager {
	return &ExtendedMockServiceManager{
		services: make(map[string]ServiceStatus),
	}
}

func (m *ExtendedMockServiceManager) StartService(name string, userLevel bool) error {
	if m.shouldError {
		return fmt.Errorf("mock error starting service")
	}
	key := fmt.Sprintf("%s:%t", name, userLevel)
	status := m.services[key]
	status.Name = name
	status.State = "running"
	status.UserLevel = userLevel
	status.StartTime = time.Now()
	m.services[key] = status
	time.Sleep(m.operationDelay)
	return nil
}

func (m *ExtendedMockServiceManager) StopService(name string, userLevel bool) error {
	if m.shouldError {
		return fmt.Errorf("mock error stopping service")
	}
	key := fmt.Sprintf("%s:%t", name, userLevel)
	status := m.services[key]
	status.Name = name
	status.State = "stopped"
	status.UserLevel = userLevel
	m.services[key] = status
	time.Sleep(m.operationDelay)
	return nil
}

func (m *ExtendedMockServiceManager) RestartService(name string, userLevel bool) error {
	if m.shouldError {
		return fmt.Errorf("mock error restarting service")
	}
	key := fmt.Sprintf("%s:%t", name, userLevel)
	status := m.services[key]
	status.Name = name
	status.State = "running"
	status.UserLevel = userLevel
	status.StartTime = time.Now()
	m.services[key] = status
	time.Sleep(m.operationDelay)
	return nil
}

func (m *ExtendedMockServiceManager) ReloadService(name string, userLevel bool) error {
	if m.shouldError {
		return fmt.Errorf("mock error reloading service")
	}
	key := fmt.Sprintf("%s:%t", name, userLevel)
	status := m.services[key]
	if !status.SupportsReload {
		return fmt.Errorf("service does not support reload")
	}
	status.Name = name
	status.UserLevel = userLevel
	now := time.Now()
	status.LastReload = &now
	m.services[key] = status
	time.Sleep(m.operationDelay)
	return nil
}

func (m *ExtendedMockServiceManager) GetServiceStatus(name string, userLevel bool) (ServiceStatus, error) {
	if m.shouldError {
		return ServiceStatus{}, fmt.Errorf("mock error getting service status")
	}
	key := fmt.Sprintf("%s:%t", name, userLevel)
	status, exists := m.services[key]
	if !exists {
		return ServiceStatus{}, fmt.Errorf("service not found")
	}
	return status, nil
}

func (m *ExtendedMockServiceManager) ServiceExists(name string, userLevel bool) bool {
	key := fmt.Sprintf("%s:%t", name, userLevel)
	_, exists := m.services[key]
	return exists
}

// SetServiceExists is a test helper to configure service existence
func (m *ExtendedMockServiceManager) SetServiceExists(name string, userLevel bool, exists bool) {
	key := fmt.Sprintf("%s:%t", name, userLevel)
	if exists {
		if _, found := m.services[key]; !found {
			m.services[key] = ServiceStatus{
				Name:           name,
				State:          "stopped",
				UserLevel:      userLevel,
				SupportsReload: true,
			}
		}
	} else {
		delete(m.services, key)
	}
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
