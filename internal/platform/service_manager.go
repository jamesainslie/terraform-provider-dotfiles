// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package platform

import (
	"context"
	"time"
)

// ServiceManager defines the interface for cross-platform service management operations.
// This replaces shell commands like "systemctl restart nginx" with native Go implementations.
type ServiceManager interface {
	// StartService starts a service
	StartService(name string, userLevel bool) error

	// StopService stops a service
	StopService(name string, userLevel bool) error

	// RestartService restarts a service (stop + start)
	RestartService(name string, userLevel bool) error

	// ReloadService reloads a service configuration without stopping
	ReloadService(name string, userLevel bool) error

	// GetServiceStatus retrieves the current status of a service
	GetServiceStatus(name string, userLevel bool) (ServiceStatus, error)

	// ServiceExists checks if a service exists on the system
	ServiceExists(name string, userLevel bool) bool
}

// ServiceStatus represents the current status of a system service
type ServiceStatus struct {
	Name           string     // Service name
	State          string     // Current state (running, stopped, failed, etc.)
	UserLevel      bool       // Whether this is a user-level service
	ProcessID      int        // Process ID if running
	StartTime      time.Time  // When the service was started
	SupportsReload bool       // Whether the service supports configuration reload
	LastReload     *time.Time // When the service was last reloaded
	Description    string     // Service description
	ExitCode       *int       // Exit code if stopped
}

// ProcessManager defines the interface for cross-platform process management operations.
// This replaces shell commands like "killall app" or "pkill app" with native Go implementations.
type ProcessManager interface {
	// FindProcessesByName finds all processes with the given name
	FindProcessesByName(name string) ([]Process, error)

	// FindProcessesByPattern finds processes matching a pattern
	FindProcessesByPattern(pattern string) ([]Process, error)

	// SendSignalToProcess sends a signal to a specific process
	SendSignalToProcess(pid int, signal ProcessSignal) error

	// TerminateProcess terminates a process gracefully or forcefully
	TerminateProcess(pid int, graceful bool) error

	// IsProcessRunning checks if a process with the given PID is running
	IsProcessRunning(pid int) bool

	// GetProcessInfo retrieves detailed information about a process
	GetProcessInfo(pid int) (*Process, error)
}

// Process represents information about a running process
type Process struct {
	PID         int       // Process ID
	PPID        int       // Parent process ID
	Name        string    // Process name
	Command     string    // Full command line
	StartTime   time.Time // When the process started
	CPUPercent  float64   // CPU usage percentage
	MemoryBytes int64     // Memory usage in bytes
	State       string    // Process state
	User        string    // User running the process
}

// ProcessSignal represents a signal that can be sent to a process
type ProcessSignal string

const (
	// Common signals across platforms
	SignalTerminate ProcessSignal = "SIGTERM" // Graceful termination
	SignalKill      ProcessSignal = "SIGKILL" // Force kill
	SignalHangup    ProcessSignal = "SIGHUP"  // Hangup (often used for reload)
	SignalInterrupt ProcessSignal = "SIGINT"  // Interrupt (Ctrl+C)
	SignalQuit      ProcessSignal = "SIGQUIT" // Quit
	SignalStop      ProcessSignal = "SIGSTOP" // Stop process
	SignalContinue  ProcessSignal = "SIGCONT" // Continue stopped process
)

// FileManager defines the interface for enhanced file management operations.
// This extends basic file operations with permission management and backup capabilities.
type FileManager interface {
	// SetPermissions sets file permissions natively
	SetPermissions(path string, mode FilePermission) error

	// SetOwnership sets file ownership (user and group)
	SetOwnership(path string, owner, group string) error

	// CreateBackup creates a backup of a file
	CreateBackup(source, backup string, format BackupFormat) error

	// ValidateBackup validates that a backup is correct
	ValidateBackup(backup string) error

	// RestoreBackup restores a file from backup
	RestoreBackup(backup, target string) error

	// GetFilePermissions retrieves current file permissions
	GetFilePermissions(path string) (FilePermission, error)

	// GetFileOwnership retrieves current file ownership
	GetFileOwnership(path string) (string, string, error) // returns user, group, error
}

// FilePermission represents file permissions in a cross-platform way
type FilePermission struct {
	Mode      uint32 // Numeric permission mode
	Owner     string // Owner permissions (rwx)
	Group     string // Group permissions (rwx)
	Other     string // Other permissions (rwx)
	Special   string // Special permissions (setuid, setgid, sticky)
	Recursive bool   // Whether to apply recursively
}

// BackupFormat defines the format for file backups
type BackupFormat string

const (
	BackupFormatCopy        BackupFormat = "copy"        // Simple file copy
	BackupFormatTimestamped BackupFormat = "timestamped" // Copy with timestamp
	BackupFormatArchive     BackupFormat = "archive"     // Compressed archive
	BackupFormatGit         BackupFormat = "git"         // Git commit
)

// NotificationManager defines the interface for cross-platform notification operations.
// This replaces shell commands like "notify-send" with native Go implementations.
type NotificationManager interface {
	// SendDesktopNotification sends a desktop notification
	SendDesktopNotification(title, message string, level NotificationLevel) error

	// WriteLogNotification writes a structured log notification
	WriteLogNotification(message string, level LogLevel, fields map[string]interface{}) error

	// SendWebhookNotification sends a webhook notification
	SendWebhookNotification(ctx context.Context, url string, payload interface{}) error

	// IsDesktopNotificationSupported checks if desktop notifications are available
	IsDesktopNotificationSupported() bool
}

// NotificationLevel defines the severity level of notifications
type NotificationLevel string

const (
	NotificationInfo    NotificationLevel = "info"
	NotificationWarning NotificationLevel = "warning"
	NotificationError   NotificationLevel = "error"
	NotificationSuccess NotificationLevel = "success"
)

// LogLevel defines the log level for structured logging
type LogLevel string

const (
	LogTrace LogLevel = "trace"
	LogDebug LogLevel = "debug"
	LogInfo  LogLevel = "info"
	LogWarn  LogLevel = "warn"
	LogError LogLevel = "error"
	LogFatal LogLevel = "fatal"
)

// ExtendedPlatformProvider extends PlatformProvider with the new management interfaces
type ExtendedPlatformProvider interface {
	PlatformProvider

	// Service management
	ServiceManager() ServiceManager

	// Process management
	ProcessManager() ProcessManager

	// Enhanced file management
	FileManager() FileManager

	// Notification management
	NotificationManager() NotificationManager
}
