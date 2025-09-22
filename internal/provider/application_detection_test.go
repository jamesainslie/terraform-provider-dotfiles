// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/platform"
)

// TestApplicationDetectionMethods tests the application detection functionality.
func TestApplicationDetectionMethods(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Create a mock application structure
	mockAppPath := filepath.Join(tempDir, "TestApp.app")
	if err := os.MkdirAll(mockAppPath, 0755); err != nil {
		t.Fatalf("Failed to create mock app: %v", err)
	}

	// Create ApplicationResource with test client
	config := &DotfilesConfig{
		DotfilesRoot:       tempDir,
		BackupEnabled:      false,
		BackupDirectory:    filepath.Join(tempDir, "backups"),
		ConflictResolution: "skip",
		DryRun:             false,
		AutoDetectPlatform: true,
		TargetPlatform:     "auto",
		TemplateEngine:     "go",
		LogLevel:           "info",
	}

	client, err := NewDotfilesClient(config)
	if err != nil {
		t.Fatalf("Failed to create dotfiles client: %v", err)
	}

	appResource := &ApplicationResource{client: client}
	ctx := context.Background()

	t.Run("Command detection", func(t *testing.T) {
		// Test detection of common system commands
		testCases := []struct {
			name        string
			command     string
			shouldExist bool
		}{
			{
				name:        "echo command exists",
				command:     "echo",
				shouldExist: true,
			},
			{
				name:        "ls command exists",
				command:     "ls",
				shouldExist: true,
			},
			{
				name:        "nonexistent command",
				command:     "thiscommanddoesnotexist12345",
				shouldExist: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := appResource.detectByCommand(ctx, tc.command)

				if result.Installed != tc.shouldExist {
					t.Errorf("Expected command %s existence to be %v, got %v",
						tc.command, tc.shouldExist, result.Installed)
				}

				if result.Installed && result.Method != "command" {
					t.Error("Detection method should be 'command'")
				}
			})
		}
	})

	t.Run("File detection", func(t *testing.T) {
		platformProvider := platform.DetectPlatform()

		// Create the test app in the working directory for detection
		workingDir, _ := os.Getwd()
		testAppPath := filepath.Join(workingDir, "TestApp.app")
		err := os.MkdirAll(testAppPath, 0755)
		if err != nil {
			t.Fatalf("Failed to create test app in working directory: %v", err)
		}
		defer os.RemoveAll(testAppPath)

		// Test file detection with existing file
		result := appResource.detectByFile(ctx, "TestApp", platformProvider)

		// Should detect the test app we created (on macOS)
		if platformProvider.GetPlatform() == "macos" {
			if !result.Installed {
				t.Error("Should detect test application on macOS")
			}
			if result.Method != "file" {
				t.Error("Detection method should be 'file'")
			}
			if result.InstallationPath != "./TestApp.app" {
				t.Errorf("Expected installation path ./TestApp.app, got %s", result.InstallationPath)
			}
		}

		// Test file detection with non-existent app
		result2 := appResource.detectByFile(ctx, "NonExistentApp", platformProvider)

		if result2.Installed {
			t.Error("Should not detect non-existent application")
		}
	})

	t.Run("Package manager detection", func(t *testing.T) {
		platformProvider := platform.DetectPlatform()

		// Test with a known package (git is usually installed)
		result := appResource.detectByPackageManager(ctx, "git", platformProvider)

		// git might or might not be installed via package manager, but should not error
		if result.Method != "package_manager" && result.Installed {
			t.Error("Detection method should be 'package_manager' if installed")
		}

		// Test with non-existent package
		result2 := appResource.detectByPackageManager(ctx, "thispackagedoesnotexist12345", platformProvider)

		if result2.Installed {
			t.Error("Should not detect non-existent package")
		}
	})
}

// TestApplicationDetectionIntegration tests complete application detection workflow.
func TestApplicationDetectionIntegration(t *testing.T) {
	tempDir := t.TempDir()

	// Create test client
	config := &DotfilesConfig{
		DotfilesRoot:       tempDir,
		BackupEnabled:      false,
		BackupDirectory:    filepath.Join(tempDir, "backups"),
		ConflictResolution: "skip",
		DryRun:             false,
		AutoDetectPlatform: true,
		TargetPlatform:     "auto",
		TemplateEngine:     "go",
		LogLevel:           "info",
	}

	client, err := NewDotfilesClient(config)
	if err != nil {
		t.Fatalf("Failed to create dotfiles client: %v", err)
	}

	appResource := &ApplicationResource{client: client}
	ctx := context.Background()

	t.Run("Complete application detection workflow", func(t *testing.T) {
		// Create application resource model
		model := ApplicationResourceModel{
			Repository:         types.StringValue("test-repo"),
			Application:        types.StringValue("git"),
			SourcePath:         types.StringValue("git"),
			DetectInstallation: types.BoolValue(true),
			SkipIfNotInstalled: types.BoolValue(false),
			WarnIfNotInstalled: types.BoolValue(true),
			DetectionMethods: func() types.List {
				methods := []attr.Value{
					types.StringValue("command"),
					types.StringValue("package_manager"),
				}
				list, _ := types.ListValue(types.StringType, methods)
				return list
			}(),
		}

		// Perform application detection
		result := appResource.performApplicationDetection(ctx, &model)

		// git should typically be available on most systems
		if result.Method == "not_found" {
			t.Log("git not detected - this may be expected in some test environments")
		} else {
			if !result.Installed {
				t.Error("Detection should report installation status correctly")
			}
			if result.Method == "" {
				t.Error("Detection should report which method succeeded")
			}
		}
	})

	t.Run("Detection with skip behavior", func(t *testing.T) {
		// Test skip behavior when application is not installed
		model := ApplicationResourceModel{
			Repository:         types.StringValue("test-repo"),
			Application:        types.StringValue("nonexistentapp12345"),
			SourcePath:         types.StringValue("nonexistent"),
			DetectInstallation: types.BoolValue(true),
			SkipIfNotInstalled: types.BoolValue(true),
			WarnIfNotInstalled: types.BoolValue(false),
			DetectionMethods: func() types.List {
				methods := []attr.Value{
					types.StringValue("command"),
				}
				list, _ := types.ListValue(types.StringType, methods)
				return list
			}(),
		}

		// Perform detection
		result := appResource.performApplicationDetection(ctx, &model)

		// Should not be installed
		if result.Installed {
			t.Error("Non-existent application should not be detected as installed")
		}
		if result.Method != "not_found" {
			t.Error("Detection method should be 'not_found' for non-existent app")
		}
	})

	t.Run("Version compatibility checking", func(t *testing.T) {
		// Test version compatibility logic
		testCases := []struct {
			name       string
			detected   string
			minVersion string
			maxVersion string
			compatible bool
		}{
			{
				name:       "Version within range",
				detected:   "1.5.0",
				minVersion: "1.0.0",
				maxVersion: "2.0.0",
				compatible: true,
			},
			{
				name:       "Version too old",
				detected:   "0.9.0",
				minVersion: "1.0.0",
				maxVersion: "2.0.0",
				compatible: false,
			},
			{
				name:       "Version too new",
				detected:   "2.1.0",
				minVersion: "1.0.0",
				maxVersion: "2.0.0",
				compatible: false,
			},
			{
				name:       "No version constraints",
				detected:   "1.5.0",
				minVersion: "",
				maxVersion: "",
				compatible: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				compatible := isVersionCompatible(tc.detected, tc.minVersion, tc.maxVersion)
				if compatible != tc.compatible {
					t.Errorf("Version compatibility for %s (min:%s, max:%s) expected %v, got %v",
						tc.detected, tc.minVersion, tc.maxVersion, tc.compatible, compatible)
				}
			})
		}
	})
}

// TestApplicationDetectionUtilities tests utility functions for application detection.
func TestApplicationDetectionUtilities(t *testing.T) {
	t.Run("capitalizeFirst", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"cursor", "Cursor"},
			{"vscode", "Vscode"},
			{"", ""},
			{"A", "A"},
			{"already-capitalized", "Already-capitalized"},
		}

		for _, tc := range testCases {
			result := capitalizeFirst(tc.input)
			if result != tc.expected {
				t.Errorf("capitalizeFirst(%q) = %q, expected %q", tc.input, result, tc.expected)
			}
		}
	})

	t.Run("Version parsing", func(t *testing.T) {
		// Test version parsing utilities
		testCases := []struct {
			version string
			valid   bool
		}{
			{"1.0.0", true},
			{"2.1.3", true},
			{"0.17.0", true},
			{"invalid", false},
			{"", false},
		}

		for _, tc := range testCases {
			valid := isValidVersion(tc.version)
			if valid != tc.valid {
				t.Errorf("isValidVersion(%q) = %v, expected %v", tc.version, valid, tc.valid)
			}
		}
	})
}

// (isVersionCompatible is implemented in file_resource.go).

func isValidVersion(version string) bool {
	// Simplified version validation
	if version == "" {
		return false
	}

	// Basic check for version-like string (contains dots and numbers)
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}

	for _, part := range parts {
		if len(part) == 0 {
			return false
		}
		// Check if part contains only digits
		for _, char := range part {
			if char < '0' || char > '9' {
				return false
			}
		}
	}

	return true
}
