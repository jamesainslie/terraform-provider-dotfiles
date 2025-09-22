// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/template"
)

// TestEnhancedTemplateSchema tests that enhanced template features are in the schema
func TestEnhancedTemplateSchema(t *testing.T) {
	t.Run("FileResource should support enhanced template features", func(t *testing.T) {
		r := NewFileResource()
		ctx := context.Background()

		req := resource.SchemaRequest{}
		resp := &resource.SchemaResponse{}

		r.Schema(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Fatalf("Schema validation failed: %v", resp.Diagnostics)
		}

		// Check for template_engine attribute
		if _, exists := resp.Schema.Attributes["template_engine"]; !exists {
			t.Error("template_engine attribute should be defined in file resource schema")
		}

		// Check for platform_template_vars attribute
		if _, exists := resp.Schema.Attributes["platform_template_vars"]; !exists {
			t.Error("platform_template_vars attribute should be defined in file resource schema")
		}

		// Check for template_functions attribute
		if _, exists := resp.Schema.Attributes["template_functions"]; !exists {
			t.Error("template_functions attribute should be defined in file resource schema")
		}
	})
}

// TestEnhancedTemplateModel tests the enhanced template resource model
func TestEnhancedTemplateModel(t *testing.T) {
	// Test enhanced file model with template features
	model := &EnhancedFileResourceModelWithTemplate{
		EnhancedFileResourceModelWithBackup: EnhancedFileResourceModelWithBackup{
			EnhancedFileResourceModel: EnhancedFileResourceModel{
				FileResourceModel: FileResourceModel{
					ID:         types.StringValue("test-template"),
					Repository: types.StringValue("test-repo"),
					Name:       types.StringValue("git-config"),
					SourcePath: types.StringValue("git/gitconfig.template"),
					TargetPath: types.StringValue("~/.gitconfig"),
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
			},
		},
		TemplateEngine: types.StringValue("go"),
		PlatformTemplateVars: func() types.Map {
			platformVars := map[string]attr.Value{
				"macos": func() types.Object {
					attrs := map[string]attr.Value{
						"credential_helper": types.StringValue("osxkeychain"),
						"diff_tool":         types.StringValue("opendiff"),
					}
					objType := map[string]attr.Type{
						"credential_helper": types.StringType,
						"diff_tool":         types.StringType,
					}
					objVal, _ := types.ObjectValue(objType, attrs)
					return objVal
				}(),
				"linux": func() types.Object {
					attrs := map[string]attr.Value{
						"credential_helper": types.StringValue("cache"),
						"diff_tool":         types.StringValue("vimdiff"),
					}
					objType := map[string]attr.Type{
						"credential_helper": types.StringType,
						"diff_tool":         types.StringType,
					}
					objVal, _ := types.ObjectValue(objType, attrs)
					return objVal
				}(),
			}
			mapVal, _ := types.MapValue(
				types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"credential_helper": types.StringType,
						"diff_tool":         types.StringType,
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

	// Verify template engine
	if model.TemplateEngine.ValueString() != "go" {
		t.Error("Template engine should be 'go'")
	}

	// Verify template vars
	templateVars := model.TemplateVars.Elements()
	if len(templateVars) != 3 {
		t.Errorf("Expected 3 template vars, got %d", len(templateVars))
	}

	// Verify platform template vars structure
	platformVars := model.PlatformTemplateVars.Elements()
	if len(platformVars) != 2 {
		t.Errorf("Expected 2 platform configurations, got %d", len(platformVars))
	}

	// Verify template functions
	templateFuncs := model.TemplateFunctions.Elements()
	if len(templateFuncs) != 2 {
		t.Errorf("Expected 2 template functions, got %d", len(templateFuncs))
	}
}

// TestTemplateEngineSelection tests template engine selection functionality
func TestTemplateEngineSelection(t *testing.T) {
	testCases := []struct {
		name         string
		engine       string
		expectError  bool
		expectedType string
	}{
		{
			name:         "Go template engine",
			engine:       "go",
			expectError:  false,
			expectedType: "*template.GoTemplateEngine",
		},
		{
			name:         "Handlebars template engine",
			engine:       "handlebars",
			expectError:  false,
			expectedType: "*template.HandlebarsTemplateEngine",
		},
		{
			name:         "Mustache template engine",
			engine:       "mustache",
			expectError:  false,
			expectedType: "*template.MustacheTemplateEngine",
		},
		{
			name:        "Invalid template engine",
			engine:      "invalid",
			expectError: true,
		},
		{
			name:         "Default to Go when empty",
			engine:       "",
			expectError:  false,
			expectedType: "*template.GoTemplateEngine",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			engine, err := CreateTemplateEngine(tc.engine)
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for engine %s, but got none", tc.engine)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for engine %s, but got: %v", tc.engine, err)
				}
				if engine == nil {
					t.Error("Template engine should not be nil")
				}
			}
		})
	}
}

// TestPlatformSpecificTemplateVars tests platform-specific template variable handling
func TestPlatformSpecificTemplateVars(t *testing.T) {
	platformVars := map[string]map[string]interface{}{
		"macos": {
			"credential_helper": "osxkeychain",
			"diff_tool":         "opendiff",
			"homebrew_path":     "/opt/homebrew",
		},
		"linux": {
			"credential_helper": "cache",
			"diff_tool":         "vimdiff",
			"homebrew_path":     "/home/linuxbrew/.linuxbrew",
		},
		"windows": {
			"credential_helper": "manager",
			"diff_tool":         "vimdiff",
			"homebrew_path":     "",
		},
	}

	testCases := []struct {
		name             string
		currentPlatform  string
		expectedHelper   string
		expectedDiffTool string
	}{
		{
			name:             "macOS platform",
			currentPlatform:  "macos",
			expectedHelper:   "osxkeychain",
			expectedDiffTool: "opendiff",
		},
		{
			name:             "Linux platform",
			currentPlatform:  "linux",
			expectedHelper:   "cache",
			expectedDiffTool: "vimdiff",
		},
		{
			name:             "Windows platform",
			currentPlatform:  "windows",
			expectedHelper:   "manager",
			expectedDiffTool: "vimdiff",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			context := BuildPlatformAwareTemplateContext(
				map[string]interface{}{"platform": tc.currentPlatform},
				map[string]interface{}{"user_name": "Test User"},
				map[string]interface{}{"platform_vars": platformVars},
			)

			// Verify platform-specific variables are included
			if context["credential_helper"] != tc.expectedHelper {
				t.Errorf("Expected credential_helper %s, got %v",
					tc.expectedHelper, context["credential_helper"])
			}
			if context["diff_tool"] != tc.expectedDiffTool {
				t.Errorf("Expected diff_tool %s, got %v",
					tc.expectedDiffTool, context["diff_tool"])
			}
		})
	}
}

// TestCustomTemplateFunctions tests custom template function support
func TestCustomTemplateFunctions(t *testing.T) {
	t.Run("Custom function registration", func(t *testing.T) {
		customFunctions := map[string]interface{}{
			"homebrewPrefix": func() string { return "/opt/homebrew" },
			"configPath":     func(app string) string { return "~/.config/" + app },
			"camelCase":      func(s string) string { return toCamelCase(s) },
		}

		engine, err := NewGoTemplateEngineWithFunctions(customFunctions)
		if err != nil {
			t.Fatalf("Failed to create template engine with custom functions: %v", err)
		}

		templateContent := `Homebrew: {{homebrewPrefix}}
Config: {{configPath "myapp"}}
Camel: {{camelCase "hello_world"}}`

		result, err := engine.ProcessTemplate(templateContent, map[string]interface{}{})
		if err != nil {
			t.Fatalf("Template processing with custom functions failed: %v", err)
		}

		if !containsStringTemplate(result, "/opt/homebrew") {
			t.Error("Custom homebrewPrefix function should work")
		}
		if !containsStringTemplate(result, "~/.config/myapp") {
			t.Error("Custom configPath function should work")
		}
		if !containsStringTemplate(result, "helloWorld") {
			t.Error("Custom camelCase function should work")
		}
	})
}

// Helper functions that need to be implemented
func CreateTemplateEngine(engineType string) (template.TemplateEngine, error) {
	// This function should be implemented in the template package
	switch engineType {
	case "", "go":
		return template.NewGoTemplateEngine()
	case "handlebars":
		return template.NewHandlebarsTemplateEngine()
	case "mustache":
		return template.NewMustacheTemplateEngine()
	default:
		return nil, fmt.Errorf("unsupported template engine: %s", engineType)
	}
}

func BuildPlatformAwareTemplateContext(systemInfo, userVars, contextVars map[string]interface{}) map[string]interface{} {
	// This function should be implemented
	context := make(map[string]interface{})

	// Add user vars
	for k, v := range userVars {
		context[k] = v
	}

	// Add system info
	context["system"] = systemInfo

	// Add platform-specific vars based on current platform
	if platform, ok := systemInfo["platform"].(string); ok {
		if platformVarsInterface, exists := contextVars["platform_vars"]; exists {
			if platformVars, ok := platformVarsInterface.(map[string]map[string]interface{}); ok {
				if platformSpecific, exists := platformVars[platform]; exists {
					for k, v := range platformSpecific {
						context[k] = v
					}
				}
			}
		}
	}

	return context
}

func NewGoTemplateEngineWithFunctions(customFunctions map[string]interface{}) (template.TemplateEngine, error) {
	return template.NewGoTemplateEngineWithFunctions(customFunctions)
}

func NewHandlebarsTemplateEngine() (template.TemplateEngine, error) {
	return template.NewHandlebarsTemplateEngine()
}

func NewMustacheTemplateEngine() (template.TemplateEngine, error) {
	return template.NewMustacheTemplateEngine()
}

func toCamelCase(s string) string {
	// Helper function for testing
	parts := strings.Split(s, "_")
	if len(parts) == 0 {
		return s
	}

	result := parts[0]
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			result += strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return result
}

func containsStringTemplate(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
