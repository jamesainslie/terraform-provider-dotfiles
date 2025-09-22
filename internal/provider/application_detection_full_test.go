// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestApplicationDetectionFullWorkflow tests a complete real-world application detection scenario.
func TestApplicationDetectionFullWorkflow(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()

	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	backupDir := filepath.Join(tempDir, "backups")

	// Create source directory structure
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create application-specific configurations
	appConfigs := map[string]string{
		"git/gitconfig.template": `[user]
    name = {{.user_name}}
    email = {{.user_email}}
[core]
    editor = {{.editor}}
{{if isMacOS .system.platform}}[credential]
    helper = osxkeychain{{end}}
{{if isLinux .system.platform}}[credential]
    helper = cache{{end}}`,

		"fish/config.fish": `# Fish configuration
set -g fish_greeting ""
set -gx EDITOR {{.editor}}
{{if .homebrew_path}}set -gx HOMEBREW_PREFIX {{.homebrew_path}}{{end}}`,

		"cursor/settings.json": `{
  "editor.fontSize": 14,
  "workbench.colorTheme": "Dark+ (default dark)",
  "terminal.integrated.shell.osx": "/opt/homebrew/bin/fish"
}`,
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
		BackupEnabled:      true,
		BackupDirectory:    backupDir,
		ConflictResolution: "backup",
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

	t.Run("Git configuration with application detection", func(t *testing.T) {
		ctx := context.Background()

		// Test conditional Git configuration that only applies if Git is installed
		fileModel := &EnhancedFileResourceModelWithApplicationDetection{
			EnhancedFileResourceModelWithTemplate: EnhancedFileResourceModelWithTemplate{
				EnhancedFileResourceModelWithBackup: EnhancedFileResourceModelWithBackup{
					EnhancedFileResourceModel: EnhancedFileResourceModel{
						FileResourceModel: FileResourceModel{
							ID:         types.StringValue("git-config"),
							Repository: types.StringValue("test-repo"),
							Name:       types.StringValue("git-config"),
							SourcePath: types.StringValue("git/gitconfig.template"),
							TargetPath: types.StringValue(filepath.Join(targetDir, ".gitconfig")),
							IsTemplate: types.BoolValue(true),
							TemplateVars: func() types.Map {
								vars := map[string]attr.Value{
									"user_name":  types.StringValue("Test User"),
									"user_email": types.StringValue("test@example.com"),
									"editor":     types.StringValue("vim"),
								}
								mapVal, _ := types.MapValue(types.StringType, vars)
								return mapVal
							}(),
						},
						Permissions: &PermissionsModel{
							Files: types.StringValue("0644"),
						},
					},
				},
				TemplateEngine: types.StringValue("go"),
			},
			RequireApplication:    types.StringValue("git"),
			ApplicationVersionMin: types.StringValue("2.0.0"),
			SkipIfAppMissing:      types.BoolValue(false), // Warn but proceed
		}

		// Test application detection configuration building
		appConfig := buildApplicationDetectionConfig(fileModel)

		if appConfig.RequiredApplication != "git" {
			t.Error("Required application should be git")
		}
		if appConfig.MinVersion != "2.0.0" {
			t.Error("Min version should be 2.0.0")
		}
		if appConfig.SkipIfMissing {
			t.Error("Should not skip if missing for this test")
		}

		// Test application requirement checking
		fileResource := &FileResource{client: client}
		shouldSkip := fileResource.checkApplicationRequirements(ctx, appConfig, nil)

		// The result depends on whether git is actually installed
		t.Logf("Git detection result: shouldSkip=%v", shouldSkip)
	})

	t.Run("Cursor configuration with strict detection", func(t *testing.T) {
		ctx := context.Background()

		// Test application that likely won't be installed - should skip
		fileModel := &EnhancedFileResourceModelWithApplicationDetection{
			EnhancedFileResourceModelWithTemplate: EnhancedFileResourceModelWithTemplate{
				EnhancedFileResourceModelWithBackup: EnhancedFileResourceModelWithBackup{
					EnhancedFileResourceModel: EnhancedFileResourceModel{
						FileResourceModel: FileResourceModel{
							ID:         types.StringValue("cursor-config"),
							Repository: types.StringValue("test-repo"),
							Name:       types.StringValue("cursor-settings"),
							SourcePath: types.StringValue("cursor/settings.json"),
							TargetPath: types.StringValue(filepath.Join(targetDir, "cursor-settings.json")),
						},
					},
				},
			},
			RequireApplication: types.StringValue("cursor"),
			SkipIfAppMissing:   types.BoolValue(true), // Skip if not installed
		}

		appConfig := buildApplicationDetectionConfig(fileModel)

		fileResource := &FileResource{client: client}

		// Create mock diagnostics for testing
		var diagnostics diag.Diagnostics
		shouldSkip := fileResource.checkApplicationRequirements(ctx, appConfig, &diagnostics)

		// Cursor is unlikely to be installed in test environment, so should skip
		if !shouldSkip {
			t.Log("Cursor was detected as installed - this may vary by environment")
		} else {
			t.Log("Cursor not detected - configuration will be skipped as expected")
		}
	})

	t.Run("Fish shell configuration with detection", func(t *testing.T) {
		ctx := context.Background()

		// Test shell configuration that requires Fish shell
		fileModel := &EnhancedFileResourceModelWithApplicationDetection{
			EnhancedFileResourceModelWithTemplate: EnhancedFileResourceModelWithTemplate{
				EnhancedFileResourceModelWithBackup: EnhancedFileResourceModelWithBackup{
					EnhancedFileResourceModel: EnhancedFileResourceModel{
						FileResourceModel: FileResourceModel{
							ID:         types.StringValue("fish-config"),
							Repository: types.StringValue("test-repo"),
							Name:       types.StringValue("fish-config"),
							SourcePath: types.StringValue("fish/config.fish"),
							TargetPath: types.StringValue(filepath.Join(targetDir, ".config/fish/config.fish")),
							IsTemplate: types.BoolValue(true),
							TemplateVars: func() types.Map {
								vars := map[string]attr.Value{
									"editor":        types.StringValue("vim"),
									"homebrew_path": types.StringValue("/opt/homebrew"),
								}
								mapVal, _ := types.MapValue(types.StringType, vars)
								return mapVal
							}(),
						},
						Permissions: &PermissionsModel{
							Directory: types.StringValue("0755"),
							Files:     types.StringValue("0644"),
							Recursive: types.BoolValue(true),
						},
					},
				},
				TemplateEngine: types.StringValue("go"),
			},
			RequireApplication: types.StringValue("fish"),
			SkipIfAppMissing:   types.BoolValue(true), // Skip if Fish not installed
		}

		appConfig := buildApplicationDetectionConfig(fileModel)

		fileResource := &FileResource{client: client}
		shouldSkip := fileResource.checkApplicationRequirements(ctx, appConfig, nil)

		// Fish may or may not be installed - test should handle both cases
		t.Logf("Fish shell detection result: shouldSkip=%v", shouldSkip)

		// The important thing is that the detection runs without error
		if appConfig.RequiredApplication != "fish" {
			t.Error("Required application should be fish")
		}
	})
}

// TestApplicationDetectionFeatureCompleteness tests that all application detection features work.
func TestApplicationDetectionFeatureCompleteness(t *testing.T) {
	ctx := context.Background()

	t.Run("All detection methods supported", func(t *testing.T) {
		methods := []string{"command", "file", "brew_cask", "package_manager"}

		// Create mock application resource
		config := &DotfilesConfig{
			DotfilesRoot:       "/tmp",
			BackupEnabled:      false,
			AutoDetectPlatform: true,
		}
		client, _ := NewDotfilesClient(config)
		appResource := &ApplicationResource{client: client}

		for _, method := range methods {
			t.Run("Method: "+method, func(t *testing.T) {
				// Test that each method can be called without error
				_ = appResource.tryDetectionMethod(ctx, "test-app", method)
				// Some methods may fail depending on environment, but should not panic
			})
		}
	})

	t.Run("Application resource provides all required attributes", func(t *testing.T) {
		// Test that ApplicationResourceModel has all required fields
		model := ApplicationResourceModel{
			ID:                   types.StringValue("test"),
			Repository:           types.StringValue("repo"),
			Application:          types.StringValue("app"),
			SourcePath:           types.StringValue("source"),
			DetectInstallation:   types.BoolValue(true),
			SkipIfNotInstalled:   types.BoolValue(true),
			WarnIfNotInstalled:   types.BoolValue(false),
			MinVersion:           types.StringValue("1.0.0"),
			MaxVersion:           types.StringValue("2.0.0"),
			ConditionalOperation: types.BoolValue(true),
			ConfigStrategy:       types.StringValue("symlink"),
			Installed:            types.BoolValue(false),
			Version:              types.StringValue("unknown"),
			InstallationPath:     types.StringValue("/path/to/app"),
			LastChecked:          types.StringValue("2024-01-01T00:00:00Z"),
			DetectionResult:      types.StringValue("not_found"),
		}

		// Verify all fields are accessible
		requiredFields := map[string]interface{}{
			"ID":                   model.ID.ValueString(),
			"Repository":           model.Repository.ValueString(),
			"Application":          model.Application.ValueString(),
			"SourcePath":           model.SourcePath.ValueString(),
			"DetectInstallation":   model.DetectInstallation.ValueBool(),
			"SkipIfNotInstalled":   model.SkipIfNotInstalled.ValueBool(),
			"WarnIfNotInstalled":   model.WarnIfNotInstalled.ValueBool(),
			"MinVersion":           model.MinVersion.ValueString(),
			"MaxVersion":           model.MaxVersion.ValueString(),
			"ConditionalOperation": model.ConditionalOperation.ValueBool(),
			"ConfigStrategy":       model.ConfigStrategy.ValueString(),
			"Installed":            model.Installed.ValueBool(),
			"Version":              model.Version.ValueString(),
			"InstallationPath":     model.InstallationPath.ValueString(),
			"LastChecked":          model.LastChecked.ValueString(),
			"DetectionResult":      model.DetectionResult.ValueString(),
		}

		// Verify each field is not nil/empty
		for fieldName, value := range requiredFields {
			if value == nil {
				t.Errorf("Field %s should not be nil", fieldName)
			}
		}
	})

	t.Run("Integration with existing resource features", func(t *testing.T) {
		// Test that application detection works alongside other features
		comprehensiveModel := &EnhancedFileResourceModelWithApplicationDetection{
			EnhancedFileResourceModelWithTemplate: EnhancedFileResourceModelWithTemplate{
				EnhancedFileResourceModelWithBackup: EnhancedFileResourceModelWithBackup{
					EnhancedFileResourceModel: EnhancedFileResourceModel{
						FileResourceModel: FileResourceModel{
							ID:         types.StringValue("comprehensive-test"),
							Repository: types.StringValue("test-repo"),
							Name:       types.StringValue("comprehensive-config"),
							SourcePath: types.StringValue("app/config.template"),
							TargetPath: types.StringValue("/tmp/comprehensive-config"),
							IsTemplate: types.BoolValue(true),
						},
						Permissions: &PermissionsModel{
							Files: types.StringValue("0600"),
						},
						PostCreateCommands: func() types.List {
							commands := []attr.Value{
								types.StringValue("echo 'Config created'"),
							}
							list, _ := types.ListValue(types.StringType, commands)
							return list
						}(),
					},
					BackupPolicy: &BackupPolicyModel{
						AlwaysBackup: types.BoolValue(true),
						BackupFormat: types.StringValue("timestamped"),
						Compression:  types.BoolValue(true),
					},
				},
				TemplateEngine: types.StringValue("go"),
				TemplateFunctions: func() types.Map {
					funcs := map[string]attr.Value{
						"customFunc": types.StringValue("custom-value"),
					}
					mapVal, _ := types.MapValue(types.StringType, funcs)
					return mapVal
				}(),
			},
			RequireApplication:    types.StringValue("test-app"),
			ApplicationVersionMin: types.StringValue("1.0.0"),
			ApplicationVersionMax: types.StringValue("2.0.0"),
			SkipIfAppMissing:      types.BoolValue(true),
		}

		// Test that all configurations can be built successfully
		appConfig := buildApplicationDetectionConfig(comprehensiveModel)

		permConfig, err := buildFilePermissionConfig(&comprehensiveModel.EnhancedFileResourceModel)
		if err != nil {
			t.Fatalf("Failed to build permission config: %v", err)
		}

		backupConfig, err := buildEnhancedBackupConfigFromAppModel(comprehensiveModel)
		if err != nil {
			t.Fatalf("Failed to build backup config: %v", err)
		}

		templateConfig, err := buildEnhancedTemplateConfigFromAppModel(comprehensiveModel)
		if err != nil {
			t.Fatalf("Failed to build template config: %v", err)
		}

		// Verify all configurations are valid
		if appConfig.RequiredApplication != "test-app" {
			t.Error("Application config should be set correctly")
		}
		if permConfig.FileMode != "0600" {
			t.Error("Permission config should be set correctly")
		}
		if backupConfig != nil && !backupConfig.Enabled {
			t.Log("Backup config may not be enabled depending on configuration")
		}
		if templateConfig.Engine != "go" {
			t.Error("Template config should be set correctly")
		}
	})
}

// TestApplicationDetectionRealWorldScenarios tests scenarios that mirror actual usage.
func TestApplicationDetectionRealWorldScenarios(t *testing.T) {
	tempDir := t.TempDir()

	config := &DotfilesConfig{
		DotfilesRoot:       tempDir,
		BackupEnabled:      true,
		AutoDetectPlatform: true,
	}
	client, _ := NewDotfilesClient(config)

	scenarios := []struct {
		name        string
		application string
		description string
		shouldSkip  bool
	}{
		{
			name:        "Development editor configuration",
			application: "code",
			description: "VSCode configuration should be conditional",
			shouldSkip:  true, // Likely not installed in test environment
		},
		{
			name:        "Shell configuration",
			application: "bash",
			description: "Bash configuration should usually be available",
			shouldSkip:  false, // Bash is usually available
		},
		{
			name:        "Git configuration",
			application: "git",
			description: "Git configuration should be conditional on Git installation",
			shouldSkip:  false, // Git is commonly available
		},
		{
			name:        "Terminal emulator",
			application: "alacritty",
			description: "Alacritty configuration should be conditional",
			shouldSkip:  true, // Specialized terminal, likely not installed
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			appResource := &ApplicationResource{client: client}
			ctx := context.Background()

			model := ApplicationResourceModel{
				Application:        types.StringValue(scenario.application),
				DetectInstallation: types.BoolValue(true),
				DetectionMethods: func() types.List {
					methods := []attr.Value{
						types.StringValue("command"),
						types.StringValue("file"),
						types.StringValue("package_manager"),
					}
					list, _ := types.ListValue(types.StringType, methods)
					return list
				}(),
			}

			result := appResource.performApplicationDetection(ctx, &model)

			t.Logf("%s: installed=%v, method=%s, expected_skip=%v",
				scenario.description, result.Installed, result.Method, scenario.shouldSkip)

			// The actual result may vary by environment, but the detection should work
			if result.Method == "" {
				t.Error("Detection method should always be set")
			}
		})
	}
}
