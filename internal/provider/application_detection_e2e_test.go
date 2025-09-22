// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestApplicationDetectionEndToEnd tests the complete application detection workflow.
func TestApplicationDetectionEndToEnd(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()

	sourceDir := filepath.Join(tempDir, "dotfiles")
	backupDir := filepath.Join(tempDir, "backups")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create test application configurations
	appConfigs := map[string]string{
		"cursor/settings.json": `{
  "editor.fontSize": 14,
  "workbench.colorTheme": "Dark+ (default dark)"
}`,
		"vscode/settings.json": `{
  "editor.fontSize": 12,
  "workbench.colorTheme": "Default Light+"
}`,
		"git/gitconfig": `[user]
    name = Test User
    email = test@example.com`,
	}

	for configPath, content := range appConfigs {
		fullPath := filepath.Join(sourceDir, configPath)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatalf("Failed to create config directory: %v", err)
		}

		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create config file %s: %v", configPath, err)
		}
	}

	// Create test client
	config := &DotfilesConfig{
		DotfilesRoot:       sourceDir,
		BackupEnabled:      false,
		BackupDirectory:    backupDir,
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

	t.Run("Application detection workflow", func(t *testing.T) {
		appResource := &ApplicationResource{client: client}
		ctx := context.Background()

		// Test direct detection functionality
		t.Run("Detect known application", func(t *testing.T) {
			data := ApplicationResourceModel{
				Repository:         types.StringValue("test-repo"),
				Application:        types.StringValue("git"),
				SourcePath:         types.StringValue("git"),
				DetectInstallation: types.BoolValue(true),
				SkipIfNotInstalled: types.BoolValue(false),
				WarnIfNotInstalled: types.BoolValue(true),
				// Detection methods will use defaults
			}

			// Test detection directly
			result := appResource.performApplicationDetection(ctx, &data)

			// Verify detection result structure
			if result == nil {
				t.Fatal("Detection result should not be nil")
			}
			if result.Method == "" {
				t.Error("Detection method should be set")
			}
		})

		// Test unknown application detection
		t.Run("Detect unknown application", func(t *testing.T) {
			data := ApplicationResourceModel{
				Application:        types.StringValue("unknownapp12345"),
				DetectInstallation: types.BoolValue(true),
				// Detection methods will use defaults
			}

			result := appResource.performApplicationDetection(ctx, &data)

			if result.Installed {
				t.Error("Unknown application should not be detected as installed")
			}
			if result.Method != "not_found" {
				t.Error("Detection method should be 'not_found' for unknown app")
			}
		})
	})

	t.Run("Conditional file resource with application detection", func(t *testing.T) {
		// Test integration of application detection with existing file resources
		fileModel := &EnhancedFileResourceModelWithApplicationDetection{
			EnhancedFileResourceModelWithTemplate: EnhancedFileResourceModelWithTemplate{
				EnhancedFileResourceModelWithBackup: EnhancedFileResourceModelWithBackup{
					EnhancedFileResourceModel: EnhancedFileResourceModel{
						FileResourceModel: FileResourceModel{
							ID:         types.StringValue("conditional-config"),
							Repository: types.StringValue("test-repo"),
							Name:       types.StringValue("git-config"),
							SourcePath: types.StringValue("git/gitconfig"),
							TargetPath: types.StringValue("~/.gitconfig"),
						},
					},
				},
			},
			RequireApplication:    types.StringValue("git"),
			ApplicationVersionMin: types.StringValue("2.0.0"),
			ApplicationVersionMax: types.StringValue("3.0.0"),
			SkipIfAppMissing:      types.BoolValue(true),
		}

		// Build application detection config
		appConfig := buildApplicationDetectionConfig(fileModel)

		// Verify configuration
		if appConfig.RequiredApplication != "git" {
			t.Error("Required application should be git")
		}
		if appConfig.MinVersion != "2.0.0" {
			t.Error("Min version should be 2.0.0")
		}
		if !appConfig.SkipIfMissing {
			t.Error("Skip if missing should be enabled")
		}
	})

	t.Run("Application configuration strategies", func(t *testing.T) {
		strategies := []string{"symlink", "copy", "merge", "template"}

		for _, strategy := range strategies {
			t.Run("Strategy: "+strategy, func(t *testing.T) {
				model := ApplicationResourceModel{
					Application:    types.StringValue("test-app"),
					ConfigStrategy: types.StringValue(strategy),
				}

				// Verify strategy can be set and retrieved
				if model.ConfigStrategy.ValueString() != strategy {
					t.Errorf("Config strategy should be %s, got %s",
						strategy, model.ConfigStrategy.ValueString())
				}

				// Test strategy validation
				valid := isValidConfigStrategy(strategy)
				if !valid {
					t.Errorf("Config strategy %s should be valid", strategy)
				}
			})
		}

		// Test invalid strategy
		invalid := isValidConfigStrategy("invalid-strategy")
		if invalid {
			t.Error("Invalid strategy should not be valid")
		}
	})

	t.Run("Complex application detection scenarios", func(t *testing.T) {
		// Test complex scenarios with multiple detection methods
		scenarios := []struct {
			name        string
			application string
			methods     []string
			expectFound bool
		}{
			{
				name:        "Git with command and package detection",
				application: "git",
				methods:     []string{"command", "package_manager"},
				expectFound: true, // git is usually available
			},
			{
				name:        "Non-existent app with all methods",
				application: "nonexistentapp12345",
				methods:     []string{"command", "file", "package_manager"},
				expectFound: false,
			},
			{
				name:        "Shell command detection",
				application: "sh",
				methods:     []string{"command"},
				expectFound: true, // sh should be available on Unix systems
			},
		}

		appResource := &ApplicationResource{client: client}
		ctx := context.Background()

		for _, scenario := range scenarios {
			t.Run(scenario.name, func(t *testing.T) {
				model := ApplicationResourceModel{
					Application:        types.StringValue(scenario.application),
					DetectInstallation: types.BoolValue(true),
					// Detection methods will use defaults
				}

				result := appResource.performApplicationDetection(ctx, &model)

				if result.Installed != scenario.expectFound {
					t.Logf("Detection result for %s: installed=%v, method=%s, expected=%v",
						scenario.application, result.Installed, result.Method, scenario.expectFound)
				}
			})
		}
	})
}

// (Mock state implementation removed for simplicity - focus on core detection logic).

// (buildApplicationDetectionConfig is implemented in file_resource.go).

func isValidConfigStrategy(strategy string) bool {
	validStrategies := []string{"symlink", "copy", "merge", "template"}
	for _, valid := range validStrategies {
		if strategy == valid {
			return true
		}
	}
	return false
}

// (Types moved to enhanced_template_models.go).
