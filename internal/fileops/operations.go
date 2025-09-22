// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fileops

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/platform"
	"github.com/jamesainslie/terraform-provider-dotfiles/internal/utils"
)

// FileManager handles file operations for dotfiles management.
type FileManager struct {
	platform platform.PlatformProvider
	dryRun   bool
}

// ConflictResolution represents the result of conflict resolution.
type ConflictResolution struct {
	Action        string // "backup", "overwrite", "skip"
	BackupPath    string // Path where backup was created (if any)
	ShouldProceed bool   // Whether to proceed with the operation
}

// NewFileManager creates a new file manager.
func NewFileManager(platformProvider platform.PlatformProvider, dryRun bool) *FileManager {
	return &FileManager{
		platform: platformProvider,
		dryRun:   dryRun,
	}
}

// CopyFile copies a file from source to target with specified permissions.
func (fm *FileManager) CopyFile(sourcePath, targetPath, fileMode string) error {
	if fm.dryRun {
		fmt.Printf("DRY RUN: Would copy %s to %s with mode %s\n", sourcePath, targetPath, fileMode)
		return nil
	}
	
	// Parse file mode
	mode, err := utils.ParseFileMode(fileMode)
	if err != nil {
		return fmt.Errorf("invalid file mode %s: %w", fileMode, err)
	}
	
	// Use platform provider to copy file
	err = fm.platform.CopyFile(sourcePath, targetPath)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}
	
	// Set permissions
	err = fm.platform.SetPermissions(targetPath, mode)
	if err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}
	
	return nil
}

// CopyFileWithBackup copies a file and creates a backup if target exists.
func (fm *FileManager) CopyFileWithBackup(sourcePath, targetPath, fileMode, backupDir string) error {
	// Check if target exists
	if utils.PathExists(targetPath) {
		// Create backup first
		_, err := fm.CreateBackup(targetPath, backupDir)
		if err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}
	
	// Copy the file
	return fm.CopyFile(sourcePath, targetPath, fileMode)
}

// CreateSymlink creates a symbolic link from source to target.
func (fm *FileManager) CreateSymlink(sourcePath, targetPath string) error {
	if fm.dryRun {
		fmt.Printf("DRY RUN: Would create symlink %s -> %s\n", targetPath, sourcePath)
		return nil
	}
	
	return fm.platform.CreateSymlink(sourcePath, targetPath)
}

// CreateSymlinkWithParents creates a symbolic link and creates parent directories.
func (fm *FileManager) CreateSymlinkWithParents(sourcePath, targetPath string) error {
	if fm.dryRun {
		fmt.Printf("DRY RUN: Would create symlink %s -> %s (with parents)\n", targetPath, sourcePath)
		return nil
	}
	
	// Create parent directories
	parentDir := filepath.Dir(targetPath)
	err := os.MkdirAll(parentDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create parent directories: %w", err)
	}
	
	return fm.platform.CreateSymlink(sourcePath, targetPath)
}

// CreateBackup creates a backup of an existing file.
func (fm *FileManager) CreateBackup(filePath, backupDir string) (string, error) {
	if fm.dryRun {
		return fmt.Sprintf("%s/backup-dry-run", backupDir), nil
	}
	
	// Create backup directory
	err := os.MkdirAll(backupDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}
	
	// Generate backup filename with timestamp
	fileName := filepath.Base(filePath)
	timestamp := time.Now().Format("2006-01-02-150405")
	backupFileName := fmt.Sprintf("%s.backup.%s", fileName, timestamp)
	backupPath := filepath.Join(backupDir, backupFileName)
	
	// Copy file to backup location
	err = fm.platform.CopyFile(filePath, backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to copy file to backup: %w", err)
	}
	
	return backupPath, nil
}

// ResolveConflict resolves conflicts based on the specified strategy.
func (fm *FileManager) ResolveConflict(existingPath, backupDir, strategy string) (*ConflictResolution, error) {
	result := &ConflictResolution{
		Action: strategy,
	}
	
	switch strategy {
	case "backup":
		if utils.PathExists(existingPath) {
			backupPath, err := fm.CreateBackup(existingPath, backupDir)
			if err != nil {
				return nil, fmt.Errorf("failed to create backup: %w", err)
			}
			result.BackupPath = backupPath
		}
		result.ShouldProceed = true
		
	case "overwrite":
		result.ShouldProceed = true
		
	case "skip":
		result.ShouldProceed = false
		
	default:
		return nil, fmt.Errorf("unknown conflict resolution strategy: %s", strategy)
	}
	
	return result, nil
}

// ProcessTemplate processes a template file and writes the result to target.
func (fm *FileManager) ProcessTemplate(templatePath, targetPath string, variables map[string]interface{}, fileMode string) error {
	if fm.dryRun {
		fmt.Printf("DRY RUN: Would process template %s to %s\n", templatePath, targetPath)
		return nil
	}
	
	// For now, return an error since template engine is not implemented
	// This will make the tests fail as expected in TDD
	return fmt.Errorf("template processing not implemented yet")
}
