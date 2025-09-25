// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package fileops

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/platform"
	"github.com/jamesainslie/terraform-provider-dotfiles/internal/template"
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

// CreateBackup creates a backup of an existing file or directory.
func (fm *FileManager) CreateBackup(filePath, backupDir string) (string, error) {
	fmt.Printf("[DEBUG] CreateBackup: Starting backup operation for %s\n", filePath)

	if fm.dryRun {
		fmt.Printf("[DEBUG] CreateBackup: Dry run mode, skipping actual backup\n")
		return fmt.Sprintf("%s/backup-dry-run", backupDir), nil
	}

	// Create backup directory
	fmt.Printf("[DEBUG] CreateBackup: Creating backup directory %s\n", backupDir)
	err := os.MkdirAll(backupDir, 0755)
	if err != nil {
		fmt.Printf("[ERROR] CreateBackup: Failed to create backup directory: %v\n", err)
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate backup filename with timestamp
	fileName := filepath.Base(filePath)
	timestamp := time.Now().Format("2006-01-02-150405")
	backupFileName := fmt.Sprintf("%s.backup.%s", fileName, timestamp)
	backupPath := filepath.Join(backupDir, backupFileName)
	fmt.Printf("[DEBUG] CreateBackup: Generated backup path %s\n", backupPath)

	// Check if source is a directory
	fmt.Printf("[DEBUG] CreateBackup: Checking if source is directory\n")
	info, err := os.Stat(filePath)
	if err != nil {
		fmt.Printf("[ERROR] CreateBackup: Failed to stat source: %v\n", err)
		return "", fmt.Errorf("failed to stat source: %w", err)
	}

	isDir := info.IsDir()
	fmt.Printf("[DEBUG] CreateBackup: Source is directory: %v, mode: %s, size: %d\n", isDir, info.Mode().String(), info.Size())

	if isDir {
		fmt.Printf("[DEBUG] CreateBackup: Copying directory recursively\n")
		// Copy directory recursively
		err = fm.copyDirectoryRecursive(filePath, backupPath)
		if err != nil {
			fmt.Printf("[ERROR] CreateBackup: Failed to copy directory recursively: %v\n", err)
			return "", fmt.Errorf("failed to copy directory to backup: %w", err)
		}
		fmt.Printf("[DEBUG] CreateBackup: Directory copied successfully\n")
	} else {
		fmt.Printf("[DEBUG] CreateBackup: Copying file using platform provider\n")
		// Copy file to backup location
		err = fm.platform.CopyFile(filePath, backupPath)
		if err != nil {
			fmt.Printf("[ERROR] CreateBackup: Failed to copy file: %v\n", err)
			return "", fmt.Errorf("failed to copy file to backup: %w", err)
		}
		fmt.Printf("[DEBUG] CreateBackup: File copied successfully\n")
	}

	fmt.Printf("[DEBUG] CreateBackup: Backup completed successfully at %s\n", backupPath)
	return backupPath, nil
}

// copyDirectoryRecursive copies a directory and all its contents recursively.
func (fm *FileManager) copyDirectoryRecursive(src, dst string) error {
	fmt.Printf("[DEBUG] copyDirectoryRecursive: Starting recursive copy from %s to %s\n", src, dst)

	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		fmt.Printf("[ERROR] copyDirectoryRecursive: Failed to stat source directory: %v\n", err)
		return fmt.Errorf("failed to stat source directory: %w", err)
	}
	fmt.Printf("[DEBUG] copyDirectoryRecursive: Source dir mode: %s\n", srcInfo.Mode().String())

	// Create destination directory with same permissions
	fmt.Printf("[DEBUG] copyDirectoryRecursive: Creating destination directory with mode %s\n", srcInfo.Mode().String())
	err = os.MkdirAll(dst, srcInfo.Mode())
	if err != nil {
		fmt.Printf("[ERROR] copyDirectoryRecursive: Failed to create destination directory: %v\n", err)
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Read source directory entries
	fmt.Printf("[DEBUG] copyDirectoryRecursive: Reading source directory entries\n")
	entries, err := os.ReadDir(src)
	if err != nil {
		fmt.Printf("[ERROR] copyDirectoryRecursive: Failed to read source directory: %v\n", err)
		return fmt.Errorf("failed to read source directory: %w", err)
	}
	fmt.Printf("[DEBUG] copyDirectoryRecursive: Found %d entries to copy\n", len(entries))

	// Copy each entry
	for i, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		fmt.Printf("[DEBUG] copyDirectoryRecursive: Processing entry %d/%d: %s (isDir: %v)\n", i+1, len(entries), entry.Name(), entry.IsDir())

		if entry.IsDir() {
			// Recursively copy subdirectory
			fmt.Printf("[DEBUG] copyDirectoryRecursive: Recursively copying subdirectory %s\n", entry.Name())
			err = fm.copyDirectoryRecursive(srcPath, dstPath)
			if err != nil {
				fmt.Printf("[ERROR] copyDirectoryRecursive: Failed to copy subdirectory %s: %v\n", entry.Name(), err)
				return fmt.Errorf("failed to copy subdirectory %s: %w", entry.Name(), err)
			}
		} else {
			// Copy file using platform provider
			fmt.Printf("[DEBUG] copyDirectoryRecursive: Copying file %s\n", entry.Name())
			err = fm.platform.CopyFile(srcPath, dstPath)
			if err != nil {
				fmt.Printf("[ERROR] copyDirectoryRecursive: Failed to copy file %s: %v\n", entry.Name(), err)
				return fmt.Errorf("failed to copy file %s: %w", entry.Name(), err)
			}
		}
	}

	fmt.Printf("[DEBUG] copyDirectoryRecursive: Completed recursive copy successfully\n")
	return nil
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

	// Create template engine
	engine, err := template.NewGoTemplateEngine()
	if err != nil {
		return fmt.Errorf("failed to create template engine: %w", err)
	}

	// Process template file
	err = engine.ProcessTemplateFile(templatePath, targetPath, variables, fileMode)
	if err != nil {
		return fmt.Errorf("failed to process template: %w", err)
	}

	return nil
}
