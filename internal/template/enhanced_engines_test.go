// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEnhancedTemplateEngines tests multiple template engine support
func TestEnhancedTemplateEngines(t *testing.T) {
	testTemplate := `Hello {{.user_name}}!
Your email is {{.user_email}}.
{{if .editor}}Editor: {{.editor}}{{end}}`

	testContext := map[string]interface{}{
		"user_name":  "Test User",
		"user_email": "test@example.com",
		"editor":     "vim",
	}

	expectedContent := `Hello Test User!
Your email is test@example.com.
Editor: vim`

	t.Run("Go template engine", func(t *testing.T) {
		engine, err := CreateTemplateEngine("go")
		if err != nil {
			t.Fatalf("Failed to create Go template engine: %v", err)
		}

		result, err := engine.ProcessTemplate(testTemplate, testContext)
		if err != nil {
			t.Fatalf("Go template processing failed: %v", err)
		}

		if strings.TrimSpace(result) != strings.TrimSpace(expectedContent) {
			t.Errorf("Go template result mismatch.\nExpected:\n%s\nGot:\n%s", expectedContent, result)
		}
	})

	t.Run("Handlebars template engine", func(t *testing.T) {
		engine, err := CreateTemplateEngine("handlebars")
		if err != nil {
			t.Fatalf("Failed to create Handlebars template engine: %v", err)
		}

		result, err := engine.ProcessTemplate(testTemplate, testContext)
		if err != nil {
			t.Fatalf("Handlebars template processing failed: %v", err)
		}

		if strings.TrimSpace(result) != strings.TrimSpace(expectedContent) {
			t.Errorf("Handlebars template result mismatch.\nExpected:\n%s\nGot:\n%s", expectedContent, result)
		}
	})

	t.Run("Mustache template engine", func(t *testing.T) {
		engine, err := CreateTemplateEngine("mustache")
		if err != nil {
			t.Fatalf("Failed to create Mustache template engine: %v", err)
		}

		result, err := engine.ProcessTemplate(testTemplate, testContext)
		if err != nil {
			t.Fatalf("Mustache template processing failed: %v", err)
		}

		if strings.TrimSpace(result) != strings.TrimSpace(expectedContent) {
			t.Errorf("Mustache template result mismatch.\nExpected:\n%s\nGot:\n%s", expectedContent, result)
		}
	})

	t.Run("Invalid template engine", func(t *testing.T) {
		_, err := CreateTemplateEngine("invalid")
		if err == nil {
			t.Error("Should return error for invalid template engine")
		}
		if !strings.Contains(err.Error(), "unsupported template engine") {
			t.Errorf("Error should mention unsupported engine, got: %v", err)
		}
	})
}

// TestCustomTemplateFunctions tests custom function support
func TestCustomTemplateFunctions(t *testing.T) {
	t.Run("Go template engine with custom functions", func(t *testing.T) {
		customFunctions := map[string]interface{}{
			"homebrewPrefix": func() string { 
				return "/opt/homebrew" 
			},
			"configPath": func(app string) string { 
				return "~/.config/" + app 
			},
			"camelCase": func(s string) string {
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
			},
		}

		engine, err := NewGoTemplateEngineWithFunctions(customFunctions)
		if err != nil {
			t.Fatalf("Failed to create Go template engine with custom functions: %v", err)
		}

		templateContent := `Homebrew: {{homebrewPrefix}}
Config: {{configPath "myapp"}}
Camel: {{camelCase "hello_world"}}`

		result, err := engine.ProcessTemplate(templateContent, map[string]interface{}{})
		if err != nil {
			t.Fatalf("Template processing with custom functions failed: %v", err)
		}

		if !strings.Contains(result, "/opt/homebrew") {
			t.Error("Custom homebrewPrefix function should work")
		}
		if !strings.Contains(result, "~/.config/myapp") {
			t.Error("Custom configPath function should work")
		}
		if !strings.Contains(result, "helloWorld") {
			t.Error("Custom camelCase function should work")
		}
	})

	t.Run("Template engine with functions factory", func(t *testing.T) {
		customFunctions := map[string]interface{}{
			"testFunc": func() string { return "test-result" },
		}

		engine, err := CreateTemplateEngineWithFunctions("go", customFunctions)
		if err != nil {
			t.Fatalf("Failed to create template engine with functions: %v", err)
		}

		result, err := engine.ProcessTemplate("{{testFunc}}", map[string]interface{}{})
		if err != nil {
			t.Fatalf("Template processing failed: %v", err)
		}

		if !strings.Contains(result, "test-result") {
			t.Error("Custom function should be available")
		}
	})
}

// TestPlatformAwareTemplateContext tests platform-specific template variable support
func TestPlatformAwareTemplateContext(t *testing.T) {
	systemInfo := map[string]interface{}{
		"platform":     "macos",
		"architecture": "arm64",
		"home_dir":     "/Users/test",
	}

	userVars := map[string]interface{}{
		"user_name":  "Test User",
		"user_email": "test@example.com",
	}

	platformVars := map[string]map[string]interface{}{
		"macos": {
			"credential_helper": "osxkeychain",
			"diff_tool":        "opendiff",
			"homebrew_path":    "/opt/homebrew",
		},
		"linux": {
			"credential_helper": "cache",
			"diff_tool":        "vimdiff",
			"homebrew_path":    "/home/linuxbrew/.linuxbrew",
		},
	}

	t.Run("Build platform-aware template context", func(t *testing.T) {
		context := BuildPlatformAwareTemplateContext(systemInfo, userVars, platformVars)

		// Verify user vars are included
		if context["user_name"] != "Test User" {
			t.Error("User variables should be included in context")
		}

		// Verify system info is included
		if systemData, ok := context["system"].(map[string]interface{}); ok {
			if systemData["platform"] != "macos" {
				t.Error("System info should be included in context")
			}
		} else {
			t.Error("System info should be included in context")
		}

		// Verify platform-specific vars are included at root level
		if context["credential_helper"] != "osxkeychain" {
			t.Error("Platform-specific credential_helper should be included")
		}
		if context["diff_tool"] != "opendiff" {
			t.Error("Platform-specific diff_tool should be included")
		}
		if context["homebrew_path"] != "/opt/homebrew" {
			t.Error("Platform-specific homebrew_path should be included")
		}
	})

	t.Run("Template with platform-specific variables", func(t *testing.T) {
		engine, err := NewGoTemplateEngine()
		if err != nil {
			t.Fatalf("Failed to create template engine: %v", err)
		}

		templateContent := `[credential]
    helper = {{.credential_helper}}
[diff]
    tool = {{.diff_tool}}
[homebrew]
    prefix = {{.homebrew_path}}`

		context := BuildPlatformAwareTemplateContext(systemInfo, userVars, platformVars)
		result, err := engine.ProcessTemplate(templateContent, context)
		if err != nil {
			t.Fatalf("Template processing failed: %v", err)
		}

		if !strings.Contains(result, "osxkeychain") {
			t.Error("Template should contain macOS credential helper")
		}
		if !strings.Contains(result, "opendiff") {
			t.Error("Template should contain macOS diff tool")
		}
		if !strings.Contains(result, "/opt/homebrew") {
			t.Error("Template should contain macOS homebrew path")
		}
	})
}

// TestTemplateFileProcessing tests file-based template processing with enhanced features
func TestTemplateFileProcessing(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("Process platform-specific template file", func(t *testing.T) {
		// Create template file
		templatePath := filepath.Join(tempDir, "gitconfig.template")
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
    tool = {{.diff_tool}}`

		err := os.WriteFile(templatePath, []byte(templateContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create template file: %v", err)
		}

		// Create context with platform-specific variables
		systemInfo := map[string]interface{}{
			"platform": "macos",
		}
		userVars := map[string]interface{}{
			"user_name":    "Test User",
			"user_email":   "test@example.com",
			"editor":       "vim",
			"signing_key":  "ABC123",
		}
		platformVars := map[string]map[string]interface{}{
			"macos": {
				"credential_helper": "osxkeychain",
				"diff_tool":        "opendiff",
			},
		}

		context := BuildPlatformAwareTemplateContext(systemInfo, userVars, platformVars)

		// Process template file
		engine, err := NewGoTemplateEngine()
		if err != nil {
			t.Fatalf("Failed to create template engine: %v", err)
		}

		outputPath := filepath.Join(tempDir, "gitconfig")
		err = engine.ProcessTemplateFile(templatePath, outputPath, context, "0644")
		if err != nil {
			t.Fatalf("ProcessTemplateFile failed: %v", err)
		}

		// Verify output file exists and contains expected content
		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read output file: %v", err)
		}

		processedContent := string(content)
		if !strings.Contains(processedContent, "Test User") {
			t.Error("Processed template should contain user name")
		}
		if !strings.Contains(processedContent, "osxkeychain") {
			t.Error("Processed template should contain macOS credential helper")
		}
		if !strings.Contains(processedContent, "opendiff") {
			t.Error("Processed template should contain macOS diff tool")
		}
		if !strings.Contains(processedContent, "signingkey = ABC123") {
			t.Error("Processed template should contain conditional signing key")
		}
	})
}
