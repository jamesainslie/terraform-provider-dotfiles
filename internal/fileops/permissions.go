// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package fileops

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/utils"
)

// PermissionConfig represents permission configuration for operations.
type PermissionConfig struct {
	DirectoryMode string
	FileMode      string
	Recursive     bool
	Rules         map[string]string // pattern -> permission mapping
}

// ApplyPermissions applies permission configuration to a path.
func (fm *FileManager) ApplyPermissions(targetPath string, config *PermissionConfig) error {
	if config == nil {
		return nil
	}

	if fm.dryRun {
		fmt.Printf("DRY RUN: Would apply permissions to %s (dir: %s, file: %s, recursive: %v)\n",
			targetPath, config.DirectoryMode, config.FileMode, config.Recursive)
		return nil
	}

	// Apply permissions based on whether target is directory or file
	info, err := os.Stat(targetPath)
	if err != nil {
		return fmt.Errorf("failed to stat target path: %w", err)
	}

	if info.IsDir() {
		return fm.applyDirectoryPermissions(targetPath, config)
	}

	return fm.applyFilePermissions(targetPath, config)
}

// applyDirectoryPermissions applies permissions to a directory.
func (fm *FileManager) applyDirectoryPermissions(dirPath string, config *PermissionConfig) error {
	// Apply directory permission to the directory itself
	if config.DirectoryMode != "" {
		mode, err := parsePermissionString(config.DirectoryMode)
		if err != nil {
			return fmt.Errorf("invalid directory permission %s: %w", config.DirectoryMode, err)
		}

		err = fm.platform.SetPermissions(dirPath, mode)
		if err != nil {
			return fmt.Errorf("failed to set directory permissions: %w", err)
		}
	}

	if config.Recursive {
		return fm.applyPermissionsRecursively(dirPath, config)
	}

	return nil
}

// applyFilePermissions applies permissions to a single file.
func (fm *FileManager) applyFilePermissions(filePath string, config *PermissionConfig) error {
	fileName := filepath.Base(filePath)

	// Check for specific rules first - find the most specific match
	permission := config.FileMode // default
	if config.Rules != nil {
		bestMatch := ""
		bestPerm := ""

		// Find the most specific pattern match
		for pattern, perm := range config.Rules {
			if matched := matchPattern(pattern, fileName); matched {
				// Prefer more specific patterns (patterns with more characters are generally more specific)
				if bestMatch == "" || len(pattern) > len(bestMatch) || isMoreSpecific(pattern, bestMatch) {
					bestMatch = pattern
					bestPerm = perm
				}
			}
		}

		if bestMatch != "" {
			permission = bestPerm
		}
	}

	if permission == "" {
		return nil // No permission to apply
	}

	mode, err := parsePermissionString(permission)
	if err != nil {
		return fmt.Errorf("invalid file permission %s: %w", permission, err)
	}

	return fm.platform.SetPermissions(filePath, mode)
}

// applyPermissionsRecursively applies permissions recursively to directory contents.
func (fm *FileManager) applyPermissionsRecursively(dirPath string, config *PermissionConfig) error {
	return filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory as it's handled separately
		if path == dirPath {
			return nil
		}

		if d.IsDir() {
			if config.DirectoryMode != "" {
				mode, parseErr := parsePermissionString(config.DirectoryMode)
				if parseErr != nil {
					return fmt.Errorf("invalid directory permission %s: %w", config.DirectoryMode, parseErr)
				}

				if setErr := fm.platform.SetPermissions(path, mode); setErr != nil {
					return fmt.Errorf("failed to set directory permissions for %s: %w", path, setErr)
				}
			}
		} else {
			// Apply file permissions
			if applyErr := fm.applyFilePermissions(path, config); applyErr != nil {
				return fmt.Errorf("failed to apply file permissions for %s: %w", path, applyErr)
			}
		}

		return nil
	})
}

// CreateSymlinkWithPermissions creates a symlink and applies permissions to the source.
func (fm *FileManager) CreateSymlinkWithPermissions(sourcePath, targetPath string, config *PermissionConfig) error {
	if fm.dryRun {
		fmt.Printf("DRY RUN: Would create symlink %s -> %s with permissions\n", targetPath, sourcePath)
		return nil
	}

	// Create the symlink first
	err := fm.CreateSymlink(sourcePath, targetPath)
	if err != nil {
		return err
	}

	// Apply permissions to the source (since symlinks typically inherit permissions)
	if config != nil && utils.PathExists(sourcePath) {
		return fm.ApplyPermissions(sourcePath, config)
	}

	return nil
}

// CreateSymlinkWithParentsAndPermissions creates a symlink with parent directories and permissions.
func (fm *FileManager) CreateSymlinkWithParentsAndPermissions(sourcePath, targetPath string, config *PermissionConfig) error {
	if fm.dryRun {
		fmt.Printf("DRY RUN: Would create symlink %s -> %s (with parents and permissions)\n", targetPath, sourcePath)
		return nil
	}

	// Create parent directories with appropriate permissions
	parentDir := filepath.Dir(targetPath)
	if config != nil && config.DirectoryMode != "" {
		mode, err := parsePermissionString(config.DirectoryMode)
		if err != nil {
			return fmt.Errorf("invalid directory permission: %w", err)
		}
		err = os.MkdirAll(parentDir, mode)
		if err != nil {
			return fmt.Errorf("failed to create parent directories: %w", err)
		}
	} else {
		err := os.MkdirAll(parentDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create parent directories: %w", err)
		}
	}

	// Create symlink and apply permissions
	return fm.CreateSymlinkWithPermissions(sourcePath, targetPath, config)
}

// CopyFileWithPermissions copies a file and applies comprehensive permission configuration.
func (fm *FileManager) CopyFileWithPermissions(sourcePath, targetPath string, config *PermissionConfig) error {
	if fm.dryRun {
		fmt.Printf("DRY RUN: Would copy %s to %s with permissions\n", sourcePath, targetPath)
		return nil
	}

	// Use platform provider to copy file
	err := fm.platform.CopyFile(sourcePath, targetPath)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Apply permissions
	if config != nil {
		err = fm.ApplyPermissions(targetPath, config)
		if err != nil {
			return fmt.Errorf("failed to apply permissions: %w", err)
		}
	}

	return nil
}

// parsePermissionString parses a permission string to os.FileMode.
func parsePermissionString(perm string) (os.FileMode, error) {
	if perm == "" {
		return 0644, nil
	}

	// Remove leading zeros for parsing
	trimmed := strings.TrimLeft(perm, "0")
	if trimmed == "" {
		trimmed = "0"
	}

	// Parse as octal
	parsed, err := strconv.ParseUint(trimmed, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid permission format %q: %w", perm, err)
	}

	// Validate permission range
	if parsed > 0777 {
		return 0, fmt.Errorf("permission %q is out of valid range (0-777)", perm)
	}

	return os.FileMode(parsed), nil
}

// matchPattern matches a filename against a pattern (simple glob support).
func matchPattern(pattern, filename string) bool {
	matched, err := filepath.Match(pattern, filename)
	if err != nil {
		return false
	}
	return matched
}

// isMoreSpecific determines if pattern1 is more specific than pattern2.
func isMoreSpecific(pattern1, pattern2 string) bool {
	// Count wildcards - fewer wildcards means more specific
	wildcards1 := strings.Count(pattern1, "*") + strings.Count(pattern1, "?")
	wildcards2 := strings.Count(pattern2, "*") + strings.Count(pattern2, "?")

	if wildcards1 != wildcards2 {
		return wildcards1 < wildcards2
	}

	// If same number of wildcards, prefer exact character matches
	exactChars1 := len(pattern1) - wildcards1
	exactChars2 := len(pattern2) - wildcards2

	return exactChars1 > exactChars2
}

// ValidatePermissionConfig validates a permission configuration.
func ValidatePermissionConfig(config *PermissionConfig) error {
	if config == nil {
		return nil
	}

	// Validate directory mode
	if config.DirectoryMode != "" {
		_, err := parsePermissionString(config.DirectoryMode)
		if err != nil {
			return fmt.Errorf("invalid directory permission: %w", err)
		}
	}

	// Validate file mode
	if config.FileMode != "" {
		_, err := parsePermissionString(config.FileMode)
		if err != nil {
			return fmt.Errorf("invalid file permission: %w", err)
		}
	}

	// Validate rules
	for pattern, perm := range config.Rules {
		if _, err := parsePermissionString(perm); err != nil {
			return fmt.Errorf("invalid permission %s for pattern %s: %w", perm, pattern, err)
		}
	}

	return nil
}
