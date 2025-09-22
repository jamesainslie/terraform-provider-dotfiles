// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/template"
)

// TestEnhancedTemplateIntegration tests the complete enhanced template workflow
func TestEnhancedTemplateIntegration(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "enhanced-template-integration-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")
	
	err = os.MkdirAll(sourceDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	
	err = os.MkdirAll(targetDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create target directory: %v", err)
	}

	t.Run("Enhanced template with platform-specific variables", func(t *testing.T) {
		// Create template file with platform-specific content
		templatePath := filepath.Join(sourceDir, "gitconfig.template")
		templateContent := `[user]
    name = {{.user_name}}
    email = {{.user_email}}
[core]
    editor = {{.editor}}
{{if .signing_key}}[user]
    signingkey = {{.signing_key}}{{end}}
[credential]
    helper = {{.credential_helper}}
[diff]
    tool = {{.diff_tool}}
[homebrew]
    prefix = {{.homebrew_path}}`

		err := os.WriteFile(templatePath, []byte(templateContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create template file: %v", err)
		}

		// Create enhanced file model with template features
		model := &EnhancedFileResourceModelWithTemplate{
			EnhancedFileResourceModelWithBackup: EnhancedFileResourceModelWithBackup{
				EnhancedFileResourceModel: EnhancedFileResourceModel{
					FileResourceModel: FileResourceModel{
						ID:         types.StringValue("git-config"),
						Repository: types.StringValue("test-repo"),
						Name:       types.StringValue("git-config"),
						SourcePath: types.StringValue("gitconfig.template"),
						TargetPath: types.StringValue(filepath.Join(targetDir, "gitconfig")),
						IsTemplate: types.BoolValue(true),
						TemplateVars: func() types.Map {
							vars := map[string]attr.Value{
								"user_name":    types.StringValue("Test User"),
								"user_email":   types.StringValue("test@example.com"),
								"editor":       types.StringValue("vim"),
								"signing_key":  types.StringValue("ABC123DEF"),
							}
							mapVal, _ := types.MapValue(types.StringType, vars)
							return mapVal
						}(),
					},
				},
			},
			TemplateEngine: types.StringValue("go"),
			PlatformTemplateVars: func() types.Map {
				platformVars := map[string]attr.Value{
					"macos": func() types.Object {
						attrs := map[string]attr.Value{
							"credential_helper": types.StringValue("osxkeychain"),
							"diff_tool":        types.StringValue("opendiff"),
							"homebrew_path":    types.StringValue("/opt/homebrew"),
						}
						objType := map[string]attr.Type{
							"credential_helper": types.StringType,
							"diff_tool":        types.StringType,
							"homebrew_path":    types.StringType,
						}
						objVal, _ := types.ObjectValue(objType, attrs)
						return objVal
					}(),
					"linux": func() types.Object {
						attrs := map[string]attr.Value{
							"credential_helper": types.StringValue("cache"),
							"diff_tool":        types.StringValue("vimdiff"),
							"homebrew_path":    types.StringValue("/home/linuxbrew/.linuxbrew"),
						}
						objType := map[string]attr.Type{
							"credential_helper": types.StringType,
							"diff_tool":        types.StringType,
							"homebrew_path":    types.StringType,
						}
						objVal, _ := types.ObjectValue(objType, attrs)
						return objVal
					}(),
				}
				mapVal, _ := types.MapValue(
					types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"credential_helper": types.StringType,
							"diff_tool":        types.StringType,
							"homebrew_path":    types.StringType,
						},
					},
					platformVars,
				)
				return mapVal
			}(),
			TemplateFunctions: func() types.Map {
				funcs := map[string]attr.Value{
					"homebrewPrefix": types.StringValue("/opt/homebrew"),
					"configPath":     types.StringValue("~/.config"),
				}
				mapVal, _ := types.MapValue(types.StringType, funcs)
				return mapVal
			}(),
		}

		// Build and validate template configuration
		templateConfig, err := buildEnhancedTemplateConfig(model)
		if err != nil {
			t.Fatalf("Failed to build template config: %v", err)
		}

		// Verify template configuration
		if templateConfig.Engine != "go" {
			t.Error("Template engine should be 'go'")
		}
		if len(templateConfig.UserVars) != 4 {
			t.Errorf("Expected 4 user variables, got %d", len(templateConfig.UserVars))
		}
		if len(templateConfig.PlatformVars) != 2 {
			t.Errorf("Expected 2 platform configurations, got %d", len(templateConfig.PlatformVars))
		}

		// Test that platform-specific variables work
		macosVars := templateConfig.PlatformVars["macos"]
		if macosVars["credential_helper"] != "osxkeychain" {
			t.Error("macOS credential helper should be osxkeychain")
		}
		if macosVars["diff_tool"] != "opendiff" {
			t.Error("macOS diff tool should be opendiff")
		}

		linuxVars := templateConfig.PlatformVars["linux"]
		if linuxVars["credential_helper"] != "cache" {
			t.Error("Linux credential helper should be cache")
		}
		if linuxVars["diff_tool"] != "vimdiff" {
			t.Error("Linux diff tool should be vimdiff")
		}
	})

	t.Run("Template processing with different engines", func(t *testing.T) {
		// Test different template engines
		engines := []string{"go", "handlebars", "mustache"}
		
		for _, engineType := range engines {
			t.Run("Engine: "+engineType, func(t *testing.T) {
				// Create simple template file
				templatePath := filepath.Join(sourceDir, engineType+"-template.txt")
				templateContent := `Hello {{.name}}!
Email: {{.email}}`

				err := os.WriteFile(templatePath, []byte(templateContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create template file: %v", err)
				}

				// Create template engine
				engine, err := template.CreateTemplateEngine(engineType)
				if err != nil {
					t.Fatalf("Failed to create %s template engine: %v", engineType, err)
				}

				// Process template
				context := map[string]interface{}{
					"name":  "Test User",
					"email": "test@example.com",
				}

				outputPath := filepath.Join(targetDir, engineType+"-output.txt")
				err = engine.ProcessTemplateFile(templatePath, outputPath, context, "0644")
				if err != nil {
					t.Fatalf("Failed to process %s template: %v", engineType, err)
				}

				// Verify output
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				processedContent := string(content)
				if !strings.Contains(processedContent, "Test User") {
					t.Errorf("%s template should contain processed name", engineType)
				}
				if !strings.Contains(processedContent, "test@example.com") {
					t.Errorf("%s template should contain processed email", engineType)
				}
			})
		}
	})

	t.Run("Custom template functions", func(t *testing.T) {
		// Create template with custom functions
		templatePath := filepath.Join(sourceDir, "custom-functions.template")
		templateContent := `Config: {{configPath "myapp"}}
Homebrew: {{homebrewPrefix}}
Upper: {{upper .name}}
Camel: {{camelCase "hello_world"}}`

		err := os.WriteFile(templatePath, []byte(templateContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create template file: %v", err)
		}

		// Create template engine with custom functions
		customFunctions := map[string]interface{}{
			"customFunc": func() string { return "custom-result" },
		}

		engine, err := template.NewGoTemplateEngineWithFunctions(customFunctions)
		if err != nil {
			t.Fatalf("Failed to create template engine with custom functions: %v", err)
		}

		// Process template
		context := map[string]interface{}{
			"name": "test user",
		}

		outputPath := filepath.Join(targetDir, "custom-functions-output.txt")
		err = engine.ProcessTemplateFile(templatePath, outputPath, context, "0644")
		if err != nil {
			t.Fatalf("Failed to process template with custom functions: %v", err)
		}

		// Verify output contains results from custom functions
		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read output file: %v", err)
		}

		processedContent := string(content)
		if !strings.Contains(processedContent, "~/.config/myapp") {
			t.Error("Template should contain processed configPath function")
		}
		if !strings.Contains(processedContent, "/opt/homebrew") {
			t.Error("Template should contain processed homebrewPrefix function")
		}
		if !strings.Contains(processedContent, "TEST USER") {
			t.Error("Template should contain uppercased name")
		}
		if !strings.Contains(processedContent, "helloWorld") {
			t.Error("Template should contain camelCase result")
		}
	})
}

// TestTemplateEngineCompatibility tests backward compatibility with existing template functionality
func TestTemplateEngineCompatibility(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "template-compatibility-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("Legacy template processing still works", func(t *testing.T) {
		// Test that existing FileResourceModel still works
		model := &EnhancedFileResourceModelWithTemplate{
			EnhancedFileResourceModelWithBackup: EnhancedFileResourceModelWithBackup{
				EnhancedFileResourceModel: EnhancedFileResourceModel{
					FileResourceModel: FileResourceModel{
						ID:         types.StringValue("legacy-template"),
						Repository: types.StringValue("test-repo"),
						Name:       types.StringValue("legacy-config"),
						SourcePath: types.StringValue("config.template"),
						TargetPath: types.StringValue(filepath.Join(tempDir, "config")),
						IsTemplate: types.BoolValue(true),
						TemplateVars: func() types.Map {
							vars := map[string]attr.Value{
								"user_name": types.StringValue("Legacy User"),
							}
							mapVal, _ := types.MapValue(types.StringType, vars)
							return mapVal
						}(),
					},
				},
			},
			// No enhanced template features - should use defaults
		}

		// Build template config - should work with defaults
		templateConfig, err := buildEnhancedTemplateConfig(model)
		if err != nil {
			t.Fatalf("Failed to build template config: %v", err)
		}

		// Should use Go engine by default
		if templateConfig.Engine != "go" {
			t.Error("Should default to Go template engine")
		}

		// Should have user vars
		if templateConfig.UserVars["user_name"] != "Legacy User" {
			t.Error("Should preserve user variables")
		}
	})

	t.Run("Enhanced template features override defaults", func(t *testing.T) {
		// Test that enhanced template features work
		model := &EnhancedFileResourceModelWithTemplate{
			EnhancedFileResourceModelWithBackup: EnhancedFileResourceModelWithBackup{
				EnhancedFileResourceModel: EnhancedFileResourceModel{
					FileResourceModel: FileResourceModel{
						ID:         types.StringValue("enhanced-template"),
						Repository: types.StringValue("test-repo"),
						Name:       types.StringValue("enhanced-config"),
						SourcePath: types.StringValue("config.template"),
						TargetPath: types.StringValue(filepath.Join(tempDir, "enhanced-config")),
						IsTemplate: types.BoolValue(true),
					},
				},
			},
			TemplateEngine: types.StringValue("handlebars"),
			PlatformTemplateVars: func() types.Map {
				platformVars := map[string]attr.Value{
					"macos": func() types.Object {
						attrs := map[string]attr.Value{
							"homebrew_path": types.StringValue("/opt/homebrew"),
						}
						objType := map[string]attr.Type{
							"homebrew_path": types.StringType,
						}
						objVal, _ := types.ObjectValue(objType, attrs)
						return objVal
					}(),
				}
				mapVal, _ := types.MapValue(
					types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"homebrew_path": types.StringType,
						},
					},
					platformVars,
				)
				return mapVal
			}(),
		}

		// Build template config
		templateConfig, err := buildEnhancedTemplateConfig(model)
		if err != nil {
			t.Fatalf("Failed to build enhanced template config: %v", err)
		}

		// Should use configured engine
		if templateConfig.Engine != "handlebars" {
			t.Error("Should use configured template engine")
		}

		// Should have platform vars
		if len(templateConfig.PlatformVars) != 1 {
			t.Error("Should have platform template variables")
		}
		if templateConfig.PlatformVars["macos"]["homebrew_path"] != "/opt/homebrew" {
			t.Error("Should have correct platform-specific variables")
		}
	})
}

// TestTemplateProcessingEndToEnd tests complete template processing workflow
func TestTemplateProcessingEndToEnd(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "template-e2e-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "templates")
	targetDir := filepath.Join(tempDir, "output")
	
	err = os.MkdirAll(sourceDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	t.Run("Complete template workflow with all features", func(t *testing.T) {
		// Create comprehensive template
		templatePath := filepath.Join(sourceDir, "comprehensive.template")
		templateContent := `# Configuration for {{.user_name}}
[user]
    name = {{.user_name}}
    email = {{.user_email}}
    
[core]
    editor = {{default "vim" .editor}}
    
{{if isMacOS .system.platform}}# macOS-specific configuration
[credential]
    helper = {{.credential_helper}}
[diff]
    tool = {{.diff_tool}}
{{end}}

{{if isLinux .system.platform}}# Linux-specific configuration  
[credential]
    helper = {{.credential_helper}}
[diff]
    tool = {{.diff_tool}}
{{end}}

# Custom paths
Config path: {{configPath "git"}}
Homebrew: {{homebrewPrefix}}
Title case: {{title .user_name}}`

		err := os.WriteFile(templatePath, []byte(templateContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create comprehensive template: %v", err)
		}

		// Test template processing for macOS
		t.Run("macOS template processing", func(t *testing.T) {
			systemInfo := map[string]interface{}{
				"platform":     "macos",
				"architecture": "arm64",
				"home_dir":     "/Users/test",
			}

			userVars := map[string]interface{}{
				"user_name":  "Test User",
				"user_email": "test@example.com",
				"editor":     "code",
			}

			platformVars := map[string]map[string]interface{}{
				"macos": {
					"credential_helper": "osxkeychain",
					"diff_tool":        "opendiff",
				},
			}

			// Create template engine
			engine, err := template.NewGoTemplateEngine()
			if err != nil {
				t.Fatalf("Failed to create template engine: %v", err)
			}

			// Build context with platform awareness
			context := template.BuildPlatformAwareTemplateContext(systemInfo, userVars, platformVars)

			// Process template
			outputPath := filepath.Join(targetDir, "macos-config")
			err = engine.ProcessTemplateFile(templatePath, outputPath, context, "0644")
			if err != nil {
				t.Fatalf("Failed to process macOS template: %v", err)
			}

			// Verify output
			content, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}

			processedContent := string(content)
			
			// Check user variables
			if !strings.Contains(processedContent, "Test User") {
				t.Error("Should contain user name")
			}
			if !strings.Contains(processedContent, "test@example.com") {
				t.Error("Should contain user email")
			}
			
			// Check platform-specific content
			if !strings.Contains(processedContent, "osxkeychain") {
				t.Error("Should contain macOS credential helper")
			}
			if !strings.Contains(processedContent, "opendiff") {
				t.Error("Should contain macOS diff tool")
			}
			
			// Check custom functions
			if !strings.Contains(processedContent, "~/.config/git") {
				t.Error("Should contain processed configPath function")
			}
			if !strings.Contains(processedContent, "/opt/homebrew") {
				t.Error("Should contain processed homebrewPrefix function")
			}
			if !strings.Contains(processedContent, "Test User") {
				t.Error("Should contain title case processed name")
			}
			
			// Check conditional content (macOS section should be included)
			if !strings.Contains(processedContent, "# macOS-specific configuration") {
				t.Error("Should contain macOS-specific section")
			}
			
			// Check that Linux section is not included
			if strings.Contains(processedContent, "# Linux-specific configuration") {
				t.Error("Should not contain Linux-specific section on macOS")
			}
		})

		// Test template processing for Linux
		t.Run("Linux template processing", func(t *testing.T) {
			systemInfo := map[string]interface{}{
				"platform":     "linux",
				"architecture": "amd64",
				"home_dir":     "/home/test",
			}

			userVars := map[string]interface{}{
				"user_name":  "Linux User",
				"user_email": "linux@example.com",
			}

			platformVars := map[string]map[string]interface{}{
				"linux": {
					"credential_helper": "cache",
					"diff_tool":        "vimdiff",
				},
			}

			// Create template engine
			engine, err := template.NewGoTemplateEngine()
			if err != nil {
				t.Fatalf("Failed to create template engine: %v", err)
			}

			// Build context with platform awareness
			context := template.BuildPlatformAwareTemplateContext(systemInfo, userVars, platformVars)

			// Process template
			outputPath := filepath.Join(targetDir, "linux-config")
			err = engine.ProcessTemplateFile(templatePath, outputPath, context, "0644")
			if err != nil {
				t.Fatalf("Failed to process Linux template: %v", err)
			}

			// Verify output
			content, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}

			processedContent := string(content)
			
			// Check platform-specific content
			if !strings.Contains(processedContent, "cache") {
				t.Error("Should contain Linux credential helper")
			}
			if !strings.Contains(processedContent, "vimdiff") {
				t.Error("Should contain Linux diff tool")
			}
			
			// Check conditional content (Linux section should be included)
			if !strings.Contains(processedContent, "# Linux-specific configuration") {
				t.Error("Should contain Linux-specific section")
			}
			
			// Check that macOS section is not included
			if strings.Contains(processedContent, "# macOS-specific configuration") {
				t.Error("Should not contain macOS-specific section on Linux")
			}
		})
	})
}
