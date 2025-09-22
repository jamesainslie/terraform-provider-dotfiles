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

// Validate validates and sets defaults for the provider configuration.
func (c *DotfilesConfig) Validate() error {
	var errs []string

	// Set defaults for empty values
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
