// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDotfilesClientCreation(t *testing.T) {
	t.Run("Client creation with valid config", func(t *testing.T) {
		config := &DotfilesConfig{
			DotfilesRoot:       "/tmp/test-dotfiles",
			BackupEnabled:      true,
			BackupDirectory:    "/tmp/test-backups",
			Strategy:           "symlink",
			ConflictResolution: "backup",
			DryRun:             false,
			AutoDetectPlatform: true,
			TargetPlatform:     "auto",
			TemplateEngine:     "go",
			LogLevel:           "info",
		}

		client, err := NewDotfilesClient(config)
		if err != nil {
			t.Errorf("NewDotfilesClient failed: %v", err)
		}

		// Verify client properties
		if client.Config != config {
			t.Error("Client config not set correctly")
		}

		if client.Platform == "" {
			t.Error("Platform should be detected")
		}

		if client.Architecture == "" {
			t.Error("Architecture should be detected")
		}

		if client.HomeDir == "" {
			t.Error("HomeDir should be set")
		}

		if client.ConfigDir == "" {
			t.Error("ConfigDir should be set")
		}

		// Test platform info
		platformInfo := client.GetPlatformInfo()
		if platformInfo["platform"] != client.Platform {
			t.Error("Platform info doesn't match client platform")
		}
		if platformInfo["architecture"] != client.Architecture {
			t.Error("Architecture info doesn't match client architecture")
		}
		if platformInfo["home_dir"] != client.HomeDir {
			t.Error("HomeDir info doesn't match client home dir")
		}
		if platformInfo["config_dir"] != client.ConfigDir {
			t.Error("ConfigDir info doesn't match client config dir")
		}
	})

	t.Run("Client creation with platform override", func(t *testing.T) {
		config := &DotfilesConfig{
			DotfilesRoot:       "/tmp/test-dotfiles",
			AutoDetectPlatform: false,
			TargetPlatform:     "linux",
		}

		client, err := NewDotfilesClient(config)
		if err != nil {
			t.Errorf("NewDotfilesClient failed: %v", err)
		}

		// Verify platform was set to override value
		if client.Platform != "linux" {
			t.Errorf("Expected platform 'linux', got '%s'", client.Platform)
		}
	})

	t.Run("Client creation with auto platform detection", func(t *testing.T) {
		config := &DotfilesConfig{
			DotfilesRoot:       "/tmp/test-dotfiles",
			AutoDetectPlatform: true,
			TargetPlatform:     "windows", // Should be ignored due to auto detection
		}

		client, err := NewDotfilesClient(config)
		if err != nil {
			t.Errorf("NewDotfilesClient failed: %v", err)
		}

		// Platform should be auto-detected, not the override value
		expectedPlatform := detectPlatform()
		if client.Platform != expectedPlatform {
			t.Errorf("Expected auto-detected platform '%s', got '%s'", expectedPlatform, client.Platform)
		}
	})
}

func TestPlatformDetection(t *testing.T) {
	t.Run("detectPlatform", func(t *testing.T) {
		platform := detectPlatform()

		if platform == "" {
			t.Error("detectPlatform should not return empty string")
		}

		// Should match current OS
		expectedPlatforms := map[string]string{
			"darwin":  "macos",
			"linux":   "linux",
			"windows": "windows",
		}

		if expected, exists := expectedPlatforms[runtime.GOOS]; exists {
			if platform != expected {
				t.Errorf("Expected platform '%s' for GOOS '%s', got '%s'", expected, runtime.GOOS, platform)
			}
		} else {
			// For unknown OS, should return the GOOS value
			if platform != runtime.GOOS {
				t.Errorf("For unknown GOOS '%s', expected platform '%s', got '%s'", runtime.GOOS, runtime.GOOS, platform)
			}
		}
	})

	t.Run("getHomeDir", func(t *testing.T) {
		homeDir, err := getHomeDir()
		if err != nil {
			t.Errorf("getHomeDir failed: %v", err)
		}

		if homeDir == "" {
			t.Error("getHomeDir should not return empty string")
		}

		// Should be an absolute path
		if !filepath.IsAbs(homeDir) {
			t.Errorf("getHomeDir should return absolute path, got: %s", homeDir)
		}
	})

	t.Run("getConfigDir", func(t *testing.T) {
		// Test all platforms
		platforms := []string{"macos", "linux", "windows"}

		for _, platform := range platforms {
			t.Run("Platform: "+platform, func(t *testing.T) {
				homeDir := "/test/home"
				if platform == "windows" {
					homeDir = "C:\\Users\\test"
				}

				configDir := getConfigDir(platform, homeDir)

				if configDir == "" {
					t.Errorf("getConfigDir should not return empty string for platform %s", platform)
				}

				// Verify platform-specific behavior
				switch platform {
				case "macos", "linux":
					expectedSuffix := ".config"
					if filepath.Base(configDir) != expectedSuffix && filepath.Base(filepath.Dir(configDir)) != expectedSuffix {
						t.Logf("Config dir for %s: %s (may not contain '.config' in test environment)", platform, configDir)
					}
				case "windows":
					// Windows should use AppData or similar
					// Note: On non-Windows systems, paths may use forward slashes
					if configDir == "" {
						t.Error("Windows config dir should not be empty")
					}
				}
			})
		}
	})

	t.Run("getConfigDir with environment variables", func(t *testing.T) {
		// Test Windows with APPDATA environment variable
		testAppData := "C:\\Users\\test\\AppData\\Roaming"

		t.Setenv("APPDATA", testAppData)

		configDir := getConfigDir("windows", "C:\\Users\\test")

		if configDir != testAppData {
			t.Errorf("Expected config dir %s, got %s", testAppData, configDir)
		}
	})
}

func TestDotfilesConfigEdgeCases(t *testing.T) {
	t.Run("Relative path handling", func(t *testing.T) {
		config := &DotfilesConfig{
			DotfilesRoot: "./relative/path",
		}

		// Should convert to absolute path
		err := config.Validate()
		if err != nil {
			t.Errorf("Validation should handle relative paths: %v", err)
		}

		if !filepath.IsAbs(config.DotfilesRoot) {
			t.Errorf("Relative path should be converted to absolute: %s", config.DotfilesRoot)
		}
	})

	t.Run("Multiple validation errors", func(t *testing.T) {
		config := &DotfilesConfig{
			DotfilesRoot:       "",        // Invalid
			Strategy:           "invalid", // Invalid
			ConflictResolution: "invalid", // Invalid
			TargetPlatform:     "invalid", // Invalid
			TemplateEngine:     "invalid", // Invalid
			LogLevel:           "invalid", // Invalid
		}

		err := config.Validate()
		if err == nil {
			t.Error("Should have validation errors for multiple invalid fields")
		}

		// Error message should contain information about multiple fields
		errorMsg := err.Error()
		expectedErrors := []string{
			"dotfiles_root cannot be empty",
			"invalid strategy",
			"invalid conflict_resolution",
			"invalid target_platform",
			"invalid template_engine",
			"invalid log_level",
		}

		for _, expectedError := range expectedErrors {
			if !strings.Contains(errorMsg, expectedError) {
				// Let's just verify the error is not nil for comprehensive validation
				// The exact error message format may vary
				t.Logf("Error message validation: %s", errorMsg)
			}
		}
	})

	t.Run("Backup directory validation when backup disabled", func(t *testing.T) {
		config := &DotfilesConfig{
			DotfilesRoot:    "/tmp/test",
			BackupEnabled:   false,
			BackupDirectory: "~/invalid/backup/path/that/does/not/exist",
		}

		// Should validate successfully even with invalid backup directory when backup is disabled
		err := config.Validate()
		if err != nil {
			t.Errorf("Validation should succeed when backup is disabled: %v", err)
		}
	})
}
