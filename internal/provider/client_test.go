// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"testing"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/platform"
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
	// Setup test environment
	testEnv := setupPlatformDetectionTestEnvironment(t)

	// Test platform detection
	testDetectionFunctionality(t, testEnv)

	// Test platform validation
	testPlatformValidation(t, testEnv)
}

// platformDetectionTestEnv holds the test environment setup
type platformDetectionTestEnv struct {
	provider platform.PlatformProvider
}

// setupPlatformDetectionTestEnvironment creates the test environment
func setupPlatformDetectionTestEnvironment(t *testing.T) *platformDetectionTestEnv {
	_ = t // Test parameter not used in this setup function
	provider := platform.DetectPlatform()
	return &platformDetectionTestEnv{
		provider: provider,
	}
}

// testDetectionFunctionality tests detection functionality
func testDetectionFunctionality(t *testing.T, env *platformDetectionTestEnv) {
	if env.provider == nil {
		t.Error("Platform provider should not be nil")
	}
}

// testPlatformValidation tests platform validation
func testPlatformValidation(t *testing.T, env *platformDetectionTestEnv) {
	// Simple validation test
	if env.provider == nil {
		t.Error("Provider should be detected")
	}
}
