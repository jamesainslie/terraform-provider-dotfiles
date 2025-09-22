// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fileops

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/platform"
)

func TestPermissionConfig(t *testing.T) {
	t.Run("ValidatePermissionConfig", func(t *testing.T) {
		testCases := []struct {
			name        string
			config      *PermissionConfig
			expectError bool
		}{
			{
				name:        "nil config",
				config:      nil,
				expectError: false,
			},
			{
				name: "valid config",
				config: &PermissionConfig{
					DirectoryMode: "0755",
					FileMode:      "0644",
					Recursive:     true,
					Rules: map[string]string{
						"*.pub": "0644",
						"id_*":  "0600",
					},
				},
				expectError: false,
			},
			{
				name: "invalid directory mode",
				config: &PermissionConfig{
					DirectoryMode: "999",
					FileMode:      "0644",
				},
				expectError: true,
			},
			{
				name: "invalid file mode",
				config: &PermissionConfig{
					DirectoryMode: "0755",
					FileMode:      "abc",
				},
				expectError: true,
			},
			{
				name: "invalid rule permission",
				config: &PermissionConfig{
					DirectoryMode: "0755",
					FileMode:      "0644",
					Rules: map[string]string{
						"*.pub": "999",
					},
				},
				expectError: true,
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := ValidatePermissionConfig(tc.config)
				if tc.expectError && err == nil {
					t.Error("Expected error but got none")
				}
				if !tc.expectError && err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			})
		}
	})
}

func TestParsePermissionString(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expected    os.FileMode
		expectError bool
	}{
		{
			name:        "empty string defaults to 0644",
			input:       "",
			expected:    0644,
			expectError: false,
		},
		{
			name:        "valid octal with leading zero",
			input:       "0755",
			expected:    0755,
			expectError: false,
		},
		{
			name:        "valid octal without leading zero",
			input:       "644",
			expected:    0644,
			expectError: false,
		},
		{
			name:        "valid strict permission",
			input:       "0600",
			expected:    0600,
			expectError: false,
		},
		{
			name:        "invalid permission out of range",
			input:       "999",
			expectError: true,
		},
		{
			name:        "invalid non-numeric",
			input:       "abc",
			expectError: true,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parsePermissionString(tc.input)
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for input %s, but got none", tc.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %s: %v", tc.input, err)
				}
				if result != tc.expected {
					t.Errorf("Expected %o, got %o", tc.expected, result)
				}
			}
		})
	}
}

func TestMatchPattern(t *testing.T) {
	testCases := []struct {
		name     string
		pattern  string
		filename string
		expected bool
	}{
		{
			name:     "exact match",
			pattern:  "config.fish",
			filename: "config.fish",
			expected: true,
		},
		{
			name:     "glob star match",
			pattern:  "id_*",
			filename: "id_rsa",
			expected: true,
		},
		{
			name:     "extension match",
			pattern:  "*.pub",
			filename: "id_rsa.pub",
			expected: true,
		},
		{
			name:     "no match",
			pattern:  "id_*",
			filename: "config.fish",
			expected: false,
		},
		{
			name:     "question mark match",
			pattern:  "file?.txt",
			filename: "file1.txt",
			expected: true,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := matchPattern(tc.pattern, tc.filename)
			if result != tc.expected {
				t.Errorf("Pattern %s with filename %s: expected %v, got %v",
					tc.pattern, tc.filename, tc.expected, result)
			}
		})
	}
}

func TestFileManagerPermissions(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "fileops-permissions-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a file manager with a real platform provider
	platformProvider := platform.DetectPlatform()
	fm := NewFileManager(platformProvider, false)
	
	t.Run("ApplyFilePermissions", func(t *testing.T) {
		// Create a test file
		testFile := filepath.Join(tempDir, "test.txt")
		err := os.WriteFile(testFile, []byte("test content"), 0666)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		
		config := &PermissionConfig{
			FileMode: "0600",
		}
		
		err = fm.ApplyPermissions(testFile, config)
		if err != nil {
			t.Fatalf("Failed to apply permissions: %v", err)
		}
		
		// Check that permissions were applied
		info, err := os.Stat(testFile)
		if err != nil {
			t.Fatalf("Failed to stat file: %v", err)
		}
		
		expectedMode := os.FileMode(0600)
		if info.Mode().Perm() != expectedMode {
			t.Errorf("Expected permission %o, got %o", expectedMode, info.Mode().Perm())
		}
	})
	
	t.Run("ApplyDirectoryPermissions", func(t *testing.T) {
		// Create a test directory
		testDir := filepath.Join(tempDir, "testdir")
		err := os.Mkdir(testDir, 0777)
		if err != nil {
			t.Fatalf("Failed to create test directory: %v", err)
		}
		
		config := &PermissionConfig{
			DirectoryMode: "0700",
		}
		
		err = fm.ApplyPermissions(testDir, config)
		if err != nil {
			t.Fatalf("Failed to apply permissions: %v", err)
		}
		
		// Check that permissions were applied
		info, err := os.Stat(testDir)
		if err != nil {
			t.Fatalf("Failed to stat directory: %v", err)
		}
		
		expectedMode := os.FileMode(0700)
		if info.Mode().Perm() != expectedMode {
			t.Errorf("Expected permission %o, got %o", expectedMode, info.Mode().Perm())
		}
	})
	
	t.Run("ApplyPermissionsWithRules", func(t *testing.T) {
		// Create test files with different patterns
		testFiles := map[string]string{
			"id_rsa":       "0600", // Should match id_* rule
			"id_rsa.pub":   "0644", // Should match *.pub rule
			"config.fish":  "0644", // Should use default
		}
		
		for filename := range testFiles {
			testFile := filepath.Join(tempDir, filename)
			err := os.WriteFile(testFile, []byte("test"), 0666)
			if err != nil {
				t.Fatalf("Failed to create test file %s: %v", filename, err)
			}
		}
		
		config := &PermissionConfig{
			FileMode: "0644", // default
			Rules: map[string]string{
				"id_*":  "0600",
				"*.pub": "0644",
			},
		}
		
		for filename, expectedPerm := range testFiles {
			testFile := filepath.Join(tempDir, filename)
			err = fm.ApplyPermissions(testFile, config)
			if err != nil {
				t.Fatalf("Failed to apply permissions to %s: %v", filename, err)
			}
			
			// Check permissions
			info, err := os.Stat(testFile)
			if err != nil {
				t.Fatalf("Failed to stat file %s: %v", filename, err)
			}
			
			expectedMode, _ := parsePermissionString(expectedPerm)
			if info.Mode().Perm() != expectedMode {
				t.Errorf("File %s: expected permission %o, got %o",
					filename, expectedMode, info.Mode().Perm())
			}
		}
	})
}

func TestDryRun(t *testing.T) {
	platformProvider := platform.DetectPlatform()
	fm := NewFileManager(platformProvider, true)
	
	config := &PermissionConfig{
		DirectoryMode: "0700",
		FileMode:      "0644",
		Recursive:     true,
	}
	
	// This should not fail even with invalid path in dry run mode
	err := fm.ApplyPermissions("/nonexistent/path", config)
	if err != nil {
		t.Errorf("Dry run should not fail with nonexistent path: %v", err)
	}
}
