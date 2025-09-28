// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package idempotency

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// OperationState represents the current state of an operation.
type OperationState int

const (
	// StateUnknown indicates the operation state cannot be determined.
	StateUnknown OperationState = iota
	// StateNotStarted indicates the operation has not been performed.
	StateNotStarted
	// StateInProgress indicates the operation is currently in progress.
	StateInProgress
	// StateCompleted indicates the operation completed successfully.
	StateCompleted
	// StateFailed indicates the operation failed.
	StateFailed
)

// String returns a string representation of the operation state.
func (os OperationState) String() string {
	switch os {
	case StateNotStarted:
		return "not_started"
	case StateInProgress:
		return "in_progress"
	case StateCompleted:
		return "completed"
	case StateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// FileState represents the current state of a file for idempotency checking.
type FileState struct {
	Path          string
	Exists        bool
	Size          int64
	ModTime       time.Time
	ContentHash   string
	IsSymlink     bool
	SymlinkTarget string
	Mode          os.FileMode
}

// GetFileState captures the current state of a file for idempotency checking.
func GetFileState(path string) (*FileState, error) {
	state := &FileState{
		Path: path,
	}

	// Check if file exists using Lstat (doesn't follow symlinks)
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		state.Exists = false
		return nil, fmt.Errorf("file does not exist: %s", path)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %s: %w", path, err)
	}

	state.Exists = true
	// For directories, set size to 0 for consistency
	if info.IsDir() {
		state.Size = 0
	} else {
		state.Size = info.Size()
	}
	state.ModTime = info.ModTime()
	state.Mode = info.Mode()

	// Check if it's a symlink
	if info.Mode()&os.ModeSymlink != 0 {
		state.IsSymlink = true
		target, err := os.Readlink(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read symlink target for %s: %w", path, err)
		}
		state.SymlinkTarget = target
	} else {
		// Calculate content hash for regular files
		if info.Mode().IsRegular() {
			hash, err := calculateFileHash(path)
			if err != nil {
				return nil, fmt.Errorf("failed to calculate hash for %s: %w", path, err)
			}
			state.ContentHash = hash
		}
	}

	return state, nil
}

// calculateFileHash calculates the SHA256 hash of a file's content.
func calculateFileHash(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(content)
	return fmt.Sprintf("%x", hash), nil
}

// CompareFileStates compares two file states to determine if they're equivalent.
func CompareFileStates(before, after *FileState) bool {
	if before == nil && after == nil {
		return true
	}
	if before == nil || after == nil {
		return false
	}

	// Basic existence check
	if before.Exists != after.Exists {
		return false
	}

	if !before.Exists && !after.Exists {
		return true // Both don't exist
	}

	// Compare file properties
	if before.IsSymlink != after.IsSymlink {
		return false
	}

	if before.IsSymlink {
		// For symlinks, compare the target
		return before.SymlinkTarget == after.SymlinkTarget
	}

	// For regular files, compare size and content hash
	if before.Size != after.Size {
		return false
	}

	if before.ContentHash != "" && after.ContentHash != "" {
		return before.ContentHash == after.ContentHash
	}

	// If we can't compare hashes, compare mod time and size
	return before.Size == after.Size && before.ModTime.Equal(after.ModTime)
}

// DirectoryState represents the current state of a directory for idempotency checking.
type DirectoryState struct {
	Path      string
	Exists    bool
	FileCount int64
	Files     map[string]*FileState
	ModTime   time.Time
}

// GetDirectoryState captures the current state of a directory and its contents.
func GetDirectoryState(ctx context.Context, path string, recursive bool) (*DirectoryState, error) {
	state := &DirectoryState{
		Path:  path,
		Files: make(map[string]*FileState),
	}

	// Check if directory exists
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		state.Exists = false
		return nil, fmt.Errorf("directory does not exist: %s", path)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to stat directory %s: %w", path, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("path %s is not a directory", path)
	}

	state.Exists = true
	state.ModTime = info.ModTime()

	// Walk the directory to get file states
	walkFunc := func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			tflog.Warn(ctx, "Error walking directory", map[string]interface{}{
				"path":  filePath,
				"error": err.Error(),
			})
			return err // Return the error to stop walking on serious issues
		}

		// Skip the root directory itself
		if filePath == path {
			return nil
		}

		// For non-recursive mode, skip subdirectories but don't include them in the state
		if !recursive && info.IsDir() {
			return filepath.SkipDir
		}

		// Get relative path for consistent keys
		relPath, err := filepath.Rel(path, filePath)
		if err != nil {
			tflog.Warn(ctx, "Failed to get relative path", map[string]interface{}{
				"base":  path,
				"file":  filePath,
				"error": err.Error(),
			})
			return err
		}

		// Get file state
		fileState, err := GetFileState(filePath)
		if err != nil {
			tflog.Warn(ctx, "Failed to get file state", map[string]interface{}{
				"path":  filePath,
				"error": err.Error(),
			})
			return err
		}

		// Only include files in the state, not directories
		if !info.IsDir() {
			state.Files[relPath] = fileState
			state.FileCount++
		}

		return nil
	}

	if recursive {
		err = filepath.Walk(path, walkFunc)
	} else {
		// For non-recursive, just read the immediate directory
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory %s: %w", path, err)
		}

		for _, entry := range entries {
			entryPath := filepath.Join(path, entry.Name())
			info, err := entry.Info()
			if err != nil {
				tflog.Warn(ctx, "Failed to get entry info", map[string]interface{}{
					"path":  entryPath,
					"error": err.Error(),
				})
				continue
			}

			// Skip directories in non-recursive mode
			if info.IsDir() {
				continue
			}

			err = walkFunc(entryPath, info, nil)
			if err != nil && err != filepath.SkipDir {
				return nil, err
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", path, err)
	}

	return state, nil
}

// CompareDirectoryStates compares two directory states to determine if they're equivalent.
func CompareDirectoryStates(before, after *DirectoryState) bool {
	if before == nil && after == nil {
		return true
	}
	if before == nil || after == nil {
		return false
	}

	// Basic existence check
	if before.Exists != after.Exists {
		return false
	}

	if !before.Exists && !after.Exists {
		return true // Both don't exist
	}

	// Compare file counts
	if before.FileCount != after.FileCount {
		return false
	}

	// Compare individual files
	if len(before.Files) != len(after.Files) {
		return false
	}

	for relPath, beforeFile := range before.Files {
		afterFile, exists := after.Files[relPath]
		if !exists {
			return false
		}

		if !CompareFileStates(beforeFile, afterFile) {
			return false
		}
	}

	return true
}

// EnsureIdempotentFileOperation ensures a file operation is idempotent by checking current state.
func EnsureIdempotentFileOperation(ctx context.Context, targetPath string, operation func() error) error {
	// Get initial state
	beforeState, err := GetFileState(targetPath)
	if err != nil {
		tflog.Debug(ctx, "Could not get initial file state", map[string]interface{}{
			"path":  targetPath,
			"error": err.Error(),
		})
		// Continue with operation even if we can't get initial state
	}

	// Perform the operation
	err = operation()
	if err != nil {
		return err
	}

	// Get final state
	afterState, err := GetFileState(targetPath)
	if err != nil {
		tflog.Debug(ctx, "Could not get final file state", map[string]interface{}{
			"path":  targetPath,
			"error": err.Error(),
		})
		// Operation succeeded, state check is just informational
		// nolint:nilerr // Intentionally ignoring error as this is post-operation state checking
		return nil
	}

	// Log the state change
	if beforeState != nil {
		statesEqual := CompareFileStates(beforeState, afterState)
		tflog.Debug(ctx, "File operation completed", map[string]interface{}{
			"path":           targetPath,
			"states_equal":   statesEqual,
			"before_exists":  beforeState.Exists,
			"after_exists":   afterState.Exists,
			"before_size":    beforeState.Size,
			"after_size":     afterState.Size,
			"before_symlink": beforeState.IsSymlink,
			"after_symlink":  afterState.IsSymlink,
		})

		if statesEqual {
			tflog.Info(ctx, "File operation was idempotent - no changes made", map[string]interface{}{
				"path": targetPath,
			})
		}
	}

	return nil
}

// EnsureIdempotentDirectoryOperation ensures a directory operation is idempotent.
func EnsureIdempotentDirectoryOperation(ctx context.Context, targetPath string, recursive bool, operation func() error) error {
	// Get initial state
	beforeState, err := GetDirectoryState(ctx, targetPath, recursive)
	if err != nil {
		tflog.Debug(ctx, "Could not get initial directory state", map[string]interface{}{
			"path":  targetPath,
			"error": err.Error(),
		})
		// Continue with operation even if we can't get initial state
	}

	// Perform the operation
	err = operation()
	if err != nil {
		return err
	}

	// Get final state
	afterState, err := GetDirectoryState(ctx, targetPath, recursive)
	if err != nil {
		tflog.Debug(ctx, "Could not get final directory state", map[string]interface{}{
			"path":  targetPath,
			"error": err.Error(),
		})
		// Operation succeeded, state check is just informational
		// nolint:nilerr // Intentionally ignoring error as this is post-operation state checking
		return nil
	}

	// Log the state change
	if beforeState != nil {
		statesEqual := CompareDirectoryStates(beforeState, afterState)
		tflog.Debug(ctx, "Directory operation completed", map[string]interface{}{
			"path":              targetPath,
			"states_equal":      statesEqual,
			"before_exists":     beforeState.Exists,
			"after_exists":      afterState.Exists,
			"before_file_count": beforeState.FileCount,
			"after_file_count":  afterState.FileCount,
		})

		if statesEqual {
			tflog.Info(ctx, "Directory operation was idempotent - no changes made", map[string]interface{}{
				"path": targetPath,
			})
		}
	}

	return nil
}

// IsDryRunMode checks if the operation should be performed in dry-run mode.
func IsDryRunMode(ctx context.Context, dryRun bool) bool {
	if dryRun {
		tflog.Info(ctx, "Operating in dry-run mode - no actual changes will be made")
		return true
	}
	return false
}

// LogDryRunOperation logs what would be done in dry-run mode.
func LogDryRunOperation(ctx context.Context, operation, path string, details map[string]interface{}) {
	logDetails := map[string]interface{}{
		"operation": operation,
		"path":      path,
		"dry_run":   true,
	}

	for k, v := range details {
		logDetails[k] = v
	}

	tflog.Info(ctx, "DRY RUN: Would perform operation", logDetails)
}
