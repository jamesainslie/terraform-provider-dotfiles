// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DotfilesConfig holds the provider configuration.
type DotfilesConfig struct {
	DotfilesRoot       string
	BackupEnabled      bool
	BackupDirectory    string
	Strategy           string
	ConflictResolution string
	DryRun             bool
	AutoDetectPlatform bool
	TargetPlatform     string
	TemplateEngine     string
	LogLevel           string
}

// SetDefaults sets default values for the provider configuration.
func (c *DotfilesConfig) SetDefaults() error {
	// Set default dotfiles root
	if c.DotfilesRoot == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("unable to get user home directory for dotfiles_root: %w", err)
		}
		c.DotfilesRoot = filepath.Join(homeDir, "dotfiles")
	}

	// Set default backup directory
	if c.BackupDirectory == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("unable to get user home directory for backup_directory: %w", err)
		}
		c.BackupDirectory = filepath.Join(homeDir, ".dotfiles-backups")
	}

	// Set other defaults
	if c.Strategy == "" {
		c.Strategy = "symlink"
	}
	if c.ConflictResolution == "" {
		c.ConflictResolution = "backup"
	}
	if c.TargetPlatform == "" {
		c.TargetPlatform = "auto"
	}
	if c.TemplateEngine == "" {
		c.TemplateEngine = "go"
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}

	return nil
}

// Validate validates the provider configuration and expands paths.
// Call SetDefaults() before calling this method.
func (c *DotfilesConfig) Validate() error {
	var errs []string

	// Validate dotfiles root
	if c.DotfilesRoot == "" {
		errs = append(errs, "dotfiles_root cannot be empty")
	} else {
		// Expand path
		if strings.HasPrefix(c.DotfilesRoot, "~") {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				errs = append(errs, fmt.Sprintf("unable to expand dotfiles_root path: %v", err))
			} else {
				c.DotfilesRoot = filepath.Join(homeDir, c.DotfilesRoot[1:])
			}
		}

		// Convert to absolute path
		absPath, err := filepath.Abs(c.DotfilesRoot)
		if err != nil {
			errs = append(errs, fmt.Sprintf("invalid dotfiles_root path: %v", err))
		} else {
			c.DotfilesRoot = absPath

			// Validate that DotfilesRoot is writable
			if err := c.validateWritablePath(c.DotfilesRoot, "dotfiles_root"); err != nil {
				errs = append(errs, err.Error())
			}
		}
	}

	// Validate backup directory if backups are enabled
	if c.BackupEnabled && c.BackupDirectory != "" {
		if strings.HasPrefix(c.BackupDirectory, "~") {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				errs = append(errs, fmt.Sprintf("unable to expand backup_directory path: %v", err))
			} else {
				c.BackupDirectory = filepath.Join(homeDir, c.BackupDirectory[1:])
			}
		}

		absPath, err := filepath.Abs(c.BackupDirectory)
		if err != nil {
			errs = append(errs, fmt.Sprintf("invalid backup_directory path: %v", err))
		} else {
			c.BackupDirectory = absPath

			// Validate that BackupDirectory is writable
			if err := c.validateWritablePath(c.BackupDirectory, "backup_directory"); err != nil {
				errs = append(errs, err.Error())
			}
		}
	}

	// Validate strategy
	validStrategies := []string{"symlink", "copy", "template"}
	if !contains(validStrategies, c.Strategy) {
		errs = append(errs, fmt.Sprintf("invalid strategy '%s', must be one of: %v", c.Strategy, validStrategies))
	}

	// Validate conflict resolution
	validConflictResolutions := []string{"backup", "overwrite", "skip", "prompt"}
	if !contains(validConflictResolutions, c.ConflictResolution) {
		errs = append(errs, fmt.Sprintf("invalid conflict_resolution '%s', must be one of: %v", c.ConflictResolution, validConflictResolutions))
	}

	// Validate target platform
	validPlatforms := []string{"auto", "macos", "linux", "windows"}
	if !contains(validPlatforms, c.TargetPlatform) {
		errs = append(errs, fmt.Sprintf("invalid target_platform '%s', must be one of: %v", c.TargetPlatform, validPlatforms))
	}

	// Validate template engine
	validEngines := []string{"go", "handlebars", "none"}
	if !contains(validEngines, c.TemplateEngine) {
		errs = append(errs, fmt.Sprintf("invalid template_engine '%s', must be one of: %v", c.TemplateEngine, validEngines))
	}

	// Validate log level
	validLogLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLogLevels, c.LogLevel) {
		errs = append(errs, fmt.Sprintf("invalid log_level '%s', must be one of: %v", c.LogLevel, validLogLevels))
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

// contains checks if a slice contains a string.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// validateWritablePath checks if a path is writable, or if its parent directory is writable for creation.
func (c *DotfilesConfig) validateWritablePath(path, pathType string) error {
	// Check if path exists
	if info, err := os.Stat(path); err == nil {
		// Path exists, check if it's writable
		if info.IsDir() {
			// For directories, try to create a test file
			testFile := filepath.Join(path, ".terraform-provider-dotfiles-write-test")
			if file, err := os.Create(testFile); err != nil {
				return fmt.Errorf("%s directory '%s' is not writable: %w", pathType, path, err)
			} else {
				file.Close()
				os.Remove(testFile) // Clean up test file
			}
		} else {
			return fmt.Errorf("%s path '%s' exists but is not a directory", pathType, path)
		}
	} else if os.IsNotExist(err) {
		// Path doesn't exist, check if parent directory is writable
		parentDir := filepath.Dir(path)
		if parentInfo, err := os.Stat(parentDir); err != nil {
			return fmt.Errorf("%s parent directory '%s' does not exist or is not accessible: %w", pathType, parentDir, err)
		} else if !parentInfo.IsDir() {
			return fmt.Errorf("%s parent path '%s' is not a directory", pathType, parentDir)
		} else {
			// Try to create the directory to test writability
			if err := os.MkdirAll(path, 0755); err != nil {
				return fmt.Errorf("cannot create %s directory '%s': %w", pathType, path, err)
			}
			// Directory created successfully, it's writable
		}
	} else {
		return fmt.Errorf("cannot access %s path '%s': %w", pathType, path, err)
	}

	return nil
}
