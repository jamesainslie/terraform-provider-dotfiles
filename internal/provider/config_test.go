// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDotfilesConfigComprehensive(t *testing.T) {
	t.Run("Path expansion and validation", func(t *testing.T) {
		config := &DotfilesConfig{
			DotfilesRoot:    "~/test-dotfiles",
			BackupDirectory: "~/test-backups",
		}

		// Create test directories to validate against
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("Failed to get home directory: %v", err)
		}

		testDotfilesDir := filepath.Join(homeDir, "test-dotfiles")
		testBackupDir := filepath.Join(homeDir, "test-backups")

		// Create the directories so validation doesn't fail for missing paths
		err = os.MkdirAll(testDotfilesDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create test dotfiles dir: %v", err)
		}
		defer os.RemoveAll(testDotfilesDir)

		config.BackupEnabled = true
		config.Strategy = "symlink"
		config.ConflictResolution = "backup"
		config.TargetPlatform = "auto"
		config.TemplateEngine = "go"
		config.LogLevel = "info"

		err = config.Validate()
		if err != nil {
			t.Errorf("Validation failed: %v", err)
		}

		// Verify paths were expanded
		if config.DotfilesRoot != testDotfilesDir {
			t.Errorf("DotfilesRoot not expanded correctly: expected %s, got %s", testDotfilesDir, config.DotfilesRoot)
		}

		if config.BackupDirectory != testBackupDir {
			t.Errorf("BackupDirectory not expanded correctly: expected %s, got %s", testBackupDir, config.BackupDirectory)
		}
	})

	t.Run("All validation error cases", func(t *testing.T) {
		testCases := []struct {
			name   string
			config *DotfilesConfig
		}{
			{
				name: "Empty dotfiles root",
				config: &DotfilesConfig{
					DotfilesRoot: "",
				},
			},
			{
				name: "Invalid strategy",
				config: &DotfilesConfig{
					DotfilesRoot: "/tmp/test",
					Strategy:     "invalid_strategy",
				},
			},
			{
				name: "Invalid conflict resolution",
				config: &DotfilesConfig{
					DotfilesRoot:       "/tmp/test",
					ConflictResolution: "invalid_resolution",
				},
			},
			{
				name: "Invalid target platform",
				config: &DotfilesConfig{
					DotfilesRoot:   "/tmp/test",
					TargetPlatform: "invalid_platform",
				},
			},
			{
				name: "Invalid template engine",
				config: &DotfilesConfig{
					DotfilesRoot:   "/tmp/test",
					TemplateEngine: "invalid_engine",
				},
			},
			{
				name: "Invalid log level",
				config: &DotfilesConfig{
					DotfilesRoot: "/tmp/test",
					LogLevel:     "invalid_level",
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.config.Validate()
				if err == nil {
					t.Error("Expected validation error but got none")
				}
			})
		}
	})

	t.Run("Default value setting", func(t *testing.T) {
		config := &DotfilesConfig{
			DotfilesRoot: "/tmp/test",
			// Leave all other values empty to test default setting
		}

		err := config.SetDefaults()
		if err != nil {
			t.Errorf("Setting defaults failed: %v", err)
		}
		err = config.Validate()
		if err != nil {
			t.Errorf("Validation failed: %v", err)
		}

		// Verify all defaults were set
		expectedDefaults := map[string]string{
			"Strategy":           "symlink",
			"ConflictResolution": "backup",
			"TargetPlatform":     "auto",
			"TemplateEngine":     "go",
			"LogLevel":           "info",
		}

		actualValues := map[string]string{
			"Strategy":           config.Strategy,
			"ConflictResolution": config.ConflictResolution,
			"TargetPlatform":     config.TargetPlatform,
			"TemplateEngine":     config.TemplateEngine,
			"LogLevel":           config.LogLevel,
		}

		for field, expected := range expectedDefaults {
			if actual := actualValues[field]; actual != expected {
				t.Errorf("Default not set correctly for %s: expected %s, got %s", field, expected, actual)
			}
		}
	})

	t.Run("Contains function", func(t *testing.T) {
		// Test the contains helper function
		slice := []string{"a", "b", "c"}

		if !contains(slice, "a") {
			t.Error("contains should return true for existing item 'a'")
		}
		if !contains(slice, "b") {
			t.Error("contains should return true for existing item 'b'")
		}
		if !contains(slice, "c") {
			t.Error("contains should return true for existing item 'c'")
		}
		if contains(slice, "d") {
			t.Error("contains should return false for non-existing item 'd'")
		}
		if contains(slice, "") {
			t.Error("contains should return false for empty string")
		}

		// Test with empty slice
		emptySlice := []string{}
		if contains(emptySlice, "a") {
			t.Error("contains should return false for empty slice")
		}

		// Test with slice containing empty string
		sliceWithEmpty := []string{"", "a", "b"}
		if !contains(sliceWithEmpty, "") {
			t.Error("contains should return true for empty string when slice contains it")
		}
	})
}
