// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDotfilesConfigRuntimeValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupFunc     func(t *testing.T) string // Returns a test directory
		config        DotfilesConfig
		expectError   bool
		errorContains string
	}{
		{
			name: "Valid writable dotfiles_root",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				dotfilesDir := filepath.Join(tmpDir, "dotfiles")
				err := os.MkdirAll(dotfilesDir, 0755)
				if err != nil {
					t.Fatalf("Failed to create test directory: %v", err)
				}
				return dotfilesDir
			},
			config: DotfilesConfig{
				BackupEnabled: false,
			},
			expectError: false,
		},
		{
			name: "Non-existent dotfiles_root with writable parent",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "new-dotfiles")
			},
			config: DotfilesConfig{
				BackupEnabled: false,
			},
			expectError: false,
		},
		{
			name: "Non-writable dotfiles_root",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				dotfilesDir := filepath.Join(tmpDir, "dotfiles")
				err := os.MkdirAll(dotfilesDir, 0444) // Read-only
				if err != nil {
					t.Fatalf("Failed to create test directory: %v", err)
				}
				return dotfilesDir
			},
			config: DotfilesConfig{
				BackupEnabled: false,
			},
			expectError:   true,
			errorContains: "not writable",
		},
		{
			name: "dotfiles_root is a file instead of directory",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "dotfiles")
				err := os.WriteFile(filePath, []byte("test"), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				return filePath
			},
			config: DotfilesConfig{
				BackupEnabled: false,
			},
			expectError:   true,
			errorContains: "not a directory",
		},
		{
			name: "Valid backup directory when backup enabled",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				dotfilesDir := filepath.Join(tmpDir, "dotfiles")
				backupDir := filepath.Join(tmpDir, "backups")
				err := os.MkdirAll(dotfilesDir, 0755)
				if err != nil {
					t.Fatalf("Failed to create dotfiles directory: %v", err)
				}
				err = os.MkdirAll(backupDir, 0755)
				if err != nil {
					t.Fatalf("Failed to create backup directory: %v", err)
				}
				return dotfilesDir
			},
			config: DotfilesConfig{
				BackupEnabled: true,
			},
			expectError: false,
		},
		{
			name: "Non-writable backup directory when backup enabled",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				dotfilesDir := filepath.Join(tmpDir, "dotfiles")
				backupDir := filepath.Join(tmpDir, "backups")
				err := os.MkdirAll(dotfilesDir, 0755)
				if err != nil {
					t.Fatalf("Failed to create dotfiles directory: %v", err)
				}
				err = os.MkdirAll(backupDir, 0444) // Read-only
				if err != nil {
					t.Fatalf("Failed to create backup directory: %v", err)
				}
				return dotfilesDir
			},
			config: DotfilesConfig{
				BackupEnabled: true,
			},
			expectError:   true,
			errorContains: "backup_directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			testDir := tt.setupFunc(t)

			// Set up config with test directory
			config := tt.config
			config.DotfilesRoot = testDir
			if config.BackupEnabled {
				config.BackupDirectory = filepath.Join(filepath.Dir(testDir), "backups")
			}

			// Set defaults first
			err := config.SetDefaults()
			if err != nil {
				t.Fatalf("SetDefaults failed: %v", err)
			}

			// Run validation
			err = config.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestFileResourceSourceValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupFunc     func(t *testing.T) string // Returns source file path
		expectError   bool
		errorContains string
	}{
		{
			name: "Valid source file",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				sourceFile := filepath.Join(tmpDir, "config.txt")
				err := os.WriteFile(sourceFile, []byte("test content"), 0644)
				if err != nil {
					t.Fatalf("Failed to create source file: %v", err)
				}
				return sourceFile
			},
			expectError: false,
		},
		{
			name: "Non-existent source file",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "missing.txt")
			},
			expectError:   true,
			errorContains: "does not exist",
		},
		{
			name: "Source path is directory",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				dirPath := filepath.Join(tmpDir, "config-dir")
				err := os.MkdirAll(dirPath, 0755)
				if err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}
				return dirPath
			},
			expectError:   true,
			errorContains: "not a regular file",
		},
		{
			name: "Non-readable source file",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				sourceFile := filepath.Join(tmpDir, "config.txt")
				err := os.WriteFile(sourceFile, []byte("test content"), 0000) // No permissions
				if err != nil {
					t.Fatalf("Failed to create source file: %v", err)
				}
				return sourceFile
			},
			expectError:   true,
			errorContains: "not readable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sourcePath := tt.setupFunc(t)

			// Create a FileResource instance
			resource := &FileResource{}

			// Test the validation method
			err := resource.validateSourceFileExists(context.Background(), sourcePath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}
