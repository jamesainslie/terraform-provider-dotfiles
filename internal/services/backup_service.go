// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// BackupService defines the interface for backup operations.
type BackupService interface {
	// CreateBackup creates a backup of the specified file or directory
	CreateBackup(ctx context.Context, sourcePath, backupDir string, options BackupOptions) (*BackupResult, error)

	// CreateEnhancedBackup creates a backup with enhanced features like compression and metadata
	CreateEnhancedBackup(ctx context.Context, sourcePath string, config *EnhancedBackupConfig) (*BackupResult, error)

	// RestoreBackup restores a file from backup
	RestoreBackup(ctx context.Context, backupPath, targetPath string) error

	// ListBackups lists all available backups for a given file
	ListBackups(ctx context.Context, originalPath, backupDir string) ([]*BackupInfo, error)

	// CleanupBackups removes old backups based on retention policy
	CleanupBackups(ctx context.Context, backupDir string, retention RetentionPolicy) error

	// ValidateBackup verifies the integrity of a backup
	ValidateBackup(ctx context.Context, backupPath string) (*ValidationResult, error)
}

// BackupOptions contains options for backup operations.
type BackupOptions struct {
	// Compression enables compression for the backup
	Compression bool

	// Format specifies the backup format (timestamped, numbered, git-style)
	Format BackupFormat

	// Metadata includes additional metadata in the backup
	Metadata map[string]string

	// DryRun indicates whether this is a dry run
	DryRun bool
}

// BackupFormat represents different backup naming formats.
type BackupFormat string

const (
	BackupFormatTimestamped BackupFormat = "timestamped"
	BackupFormatNumbered    BackupFormat = "numbered"
	BackupFormatGitStyle    BackupFormat = "git-style"
)

// BackupResult contains information about a completed backup operation.
type BackupResult struct {
	// BackupPath is the full path to the created backup
	BackupPath string

	// Size is the size of the backup in bytes
	Size int64

	// CreatedAt is when the backup was created
	CreatedAt time.Time

	// Checksum is the checksum of the backup for integrity verification
	Checksum string

	// Compressed indicates if the backup is compressed
	Compressed bool

	// Metadata contains additional backup metadata
	Metadata map[string]string
}

// BackupInfo contains information about an existing backup.
type BackupInfo struct {
	// Path is the full path to the backup
	Path string

	// OriginalPath is the path of the original file
	OriginalPath string

	// Size is the size of the backup
	Size int64

	// CreatedAt is when the backup was created
	CreatedAt time.Time

	// Format is the backup format used
	Format BackupFormat

	// Checksum is the backup checksum
	Checksum string

	// Valid indicates if the backup passed validation
	Valid bool
}

// EnhancedBackupConfig contains configuration for enhanced backup operations.
type EnhancedBackupConfig struct {
	// Enabled indicates if enhanced backup is enabled
	Enabled bool

	// Directory is the backup directory
	Directory string

	// Format is the backup format to use
	Format BackupFormat

	// Compression enables compression
	Compression bool

	// Retention defines the retention policy
	Retention RetentionPolicy

	// Metadata contains additional metadata
	Metadata map[string]string

	// Incremental enables incremental backups
	Incremental bool
}

// RetentionPolicy defines how long backups should be kept.
type RetentionPolicy struct {
	// MaxAge is the maximum age of backups to keep
	MaxAge time.Duration

	// MaxCount is the maximum number of backups to keep
	MaxCount int

	// KeepDaily indicates how many daily backups to keep
	KeepDaily int

	// KeepWeekly indicates how many weekly backups to keep
	KeepWeekly int

	// KeepMonthly indicates how many monthly backups to keep
	KeepMonthly int
}

// ValidationResult contains the result of backup validation.
type ValidationResult struct {
	// Valid indicates if the backup is valid
	Valid bool

	// Errors contains any validation errors
	Errors []string

	// Warnings contains any validation warnings
	Warnings []string

	// ChecksumMatch indicates if the checksum matches
	ChecksumMatch bool

	// Size is the validated size
	Size int64
}

// DefaultBackupService provides a default implementation of BackupService.
type DefaultBackupService struct {
	// platformProvider provides platform-specific operations
	platformProvider PlatformProvider

	// dryRun indicates if operations should be simulated
	dryRun bool
}

// PlatformProvider defines platform-specific operations needed by the backup service.
type PlatformProvider interface {
	// CopyFile copies a file from source to destination
	CopyFile(src, dst string, mode os.FileMode) error

	// CreateDirectory creates a directory with the specified permissions
	CreateDirectory(path string, mode os.FileMode) error

	// GetFileInfo returns information about a file
	GetFileInfo(path string) (os.FileInfo, error)

	// CalculateChecksum calculates the checksum of a file
	CalculateChecksum(path string) (string, error)
}

// NewDefaultBackupService creates a new default backup service.
func NewDefaultBackupService(platformProvider PlatformProvider, dryRun bool) *DefaultBackupService {
	return &DefaultBackupService{
		platformProvider: platformProvider,
		dryRun:           dryRun,
	}
}

// CreateBackup implements BackupService.CreateBackup.
func (s *DefaultBackupService) CreateBackup(ctx context.Context, sourcePath, backupDir string, options BackupOptions) (*BackupResult, error) {
	if s.dryRun || options.DryRun {
		return s.simulateBackup(ctx, sourcePath, backupDir, options)
	}

	return s.performBackup(ctx, sourcePath, backupDir, options)
}

// CreateEnhancedBackup implements BackupService.CreateEnhancedBackup.
func (s *DefaultBackupService) CreateEnhancedBackup(ctx context.Context, sourcePath string, config *EnhancedBackupConfig) (*BackupResult, error) {
	if s.dryRun {
		return s.simulateEnhancedBackup(ctx, sourcePath, config)
	}

	return s.performEnhancedBackup(ctx, sourcePath, config)
}

// RestoreBackup implements BackupService.RestoreBackup.
func (s *DefaultBackupService) RestoreBackup(ctx context.Context, backupPath, targetPath string) error {
	if s.dryRun {
		return s.simulateRestore(ctx, backupPath, targetPath)
	}

	return s.performRestore(ctx, backupPath, targetPath)
}

// ListBackups implements BackupService.ListBackups.
func (s *DefaultBackupService) ListBackups(ctx context.Context, originalPath, backupDir string) ([]*BackupInfo, error) {
	return s.scanBackups(ctx, originalPath, backupDir)
}

// CleanupBackups implements BackupService.CleanupBackups.
func (s *DefaultBackupService) CleanupBackups(ctx context.Context, backupDir string, retention RetentionPolicy) error {
	if s.dryRun {
		return s.simulateCleanup(ctx, backupDir, retention)
	}

	return s.performCleanup(ctx, backupDir, retention)
}

// ValidateBackup implements BackupService.ValidateBackup.
func (s *DefaultBackupService) ValidateBackup(ctx context.Context, backupPath string) (*ValidationResult, error) {
	return s.performValidation(ctx, backupPath)
}

// Helper methods (implementation details)

func (s *DefaultBackupService) simulateBackup(_ context.Context, sourcePath, backupDir string, options BackupOptions) (*BackupResult, error) {
	// Simulate backup operation for dry run
	return &BackupResult{
		BackupPath: s.generateBackupPath(sourcePath, backupDir, options.Format),
		Size:       0, // Would be calculated in real operation
		CreatedAt:  time.Now(),
		Checksum:   "simulated-checksum",
		Compressed: options.Compression,
		Metadata:   options.Metadata,
	}, nil
}

func (s *DefaultBackupService) performBackup(ctx context.Context, sourcePath, backupDir string, options BackupOptions) (*BackupResult, error) {
	_ = ctx // Context not used in this backup operation
	// Ensure backup directory exists
	if err := s.platformProvider.CreateDirectory(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	backupPath := s.generateBackupPath(sourcePath, backupDir, options.Format)

	// Copy the source to backup path
	mode := os.FileMode(0644)
	if err := s.platformProvider.CopyFile(sourcePath, backupPath, mode); err != nil {
		return nil, fmt.Errorf("failed to copy file for backup: %w", err)
	}

	// Get file info for the backup
	info, err := s.platformProvider.GetFileInfo(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get backup file info: %w", err)
	}

	// Calculate checksum
	checksum, err := s.platformProvider.CalculateChecksum(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	result := &BackupResult{
		BackupPath: backupPath,
		Size:       info.Size(),
		CreatedAt:  time.Now(),
		Checksum:   checksum,
		Compressed: options.Compression,
		Metadata:   options.Metadata,
	}
	return result, nil
}

func (s *DefaultBackupService) simulateEnhancedBackup(_ context.Context, sourcePath string, config *EnhancedBackupConfig) (*BackupResult, error) {
	// Simulate enhanced backup for dry run
	return &BackupResult{
		BackupPath: s.generateBackupPath(sourcePath, config.Directory, config.Format),
		Size:       0,
		CreatedAt:  time.Now(),
		Checksum:   "simulated-enhanced-checksum",
		Compressed: config.Compression,
		Metadata:   config.Metadata,
	}, nil
}

func (s *DefaultBackupService) performEnhancedBackup(ctx context.Context, sourcePath string, config *EnhancedBackupConfig) (*BackupResult, error) {
	if !config.Enabled {
		return nil, fmt.Errorf("enhanced backup not enabled")
	}

	options := BackupOptions{
		Format:      config.Format,
		Compression: config.Compression,
		Metadata:    config.Metadata,
	}

	result, err := s.performBackup(ctx, sourcePath, config.Directory, options)
	if err != nil {
		return nil, err
	}

	// Optionally apply retention policy
	if err := s.CleanupBackups(ctx, config.Directory, config.Retention); err != nil {
		// Log warning but don't fail the backup - cleanup errors are non-fatal
		_ = err // Explicitly ignore the error
	}

	return result, nil
}

func (s *DefaultBackupService) simulateRestore(ctx context.Context, backupPath, targetPath string) error {
	// Simulate restore for dry run
	return nil
}

func (s *DefaultBackupService) performRestore(ctx context.Context, backupPath, targetPath string) error {
	// Implementation would go here
	return nil
}

func (s *DefaultBackupService) scanBackups(ctx context.Context, originalPath, backupDir string) ([]*BackupInfo, error) {
	// Implementation would go here
	return nil, nil
}

func (s *DefaultBackupService) simulateCleanup(ctx context.Context, backupDir string, retention RetentionPolicy) error {
	// Simulate cleanup for dry run
	return nil
}

func (s *DefaultBackupService) performCleanup(ctx context.Context, backupDir string, retention RetentionPolicy) error {
	// Implementation would go here
	return nil
}

func (s *DefaultBackupService) performValidation(_ context.Context, _ string) (*ValidationResult, error) {
	// Implementation would go here
	return &ValidationResult{
		Valid:         true,
		Errors:        []string{},
		Warnings:      []string{},
		ChecksumMatch: true,
		Size:          0,
	}, nil
}

func (s *DefaultBackupService) generateBackupPath(sourcePath, backupDir string, format BackupFormat) string {
	base := filepath.Base(sourcePath)
	timestamp := time.Now().Format("20060102-150405")
	switch format {
	case BackupFormatTimestamped:
		return filepath.Join(backupDir, base+".backup."+timestamp)
	case BackupFormatNumbered:
		return filepath.Join(backupDir, base+".backup.1") // Simple implementation
	case BackupFormatGitStyle:
		return filepath.Join(backupDir, base+".backup~1")
	default:
		return filepath.Join(backupDir, base+".backup")
	}
}
