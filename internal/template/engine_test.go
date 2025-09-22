// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package template

import (
	"os"
	"path/filepath"
	"testing"
)

// TestTemplateEngine tests Go template processing using TDD
func TestTemplateEngine(t *testing.T) {
	t.Run("Create Go template engine", func(t *testing.T) {
		engine, err := NewGoTemplateEngine()
		if err != nil {
			t.Errorf("NewGoTemplateEngine failed: %v", err)
		}
		
		if engine == nil {
			t.Error("Template engine should not be nil")
		}
	})
	
	t.Run("Process simple template", func(t *testing.T) {
		engine, err := NewGoTemplateEngine()
		if err != nil {
			t.Fatalf("Failed to create template engine: %v", err)
		}
		
		templateContent := `Hello {{.name}}!
Your email is {{.email}}.`
		
		context := map[string]interface{}{
			"name":  "Test User",
			"email": "test@example.com",
		}
		
		result, err := engine.ProcessTemplate(templateContent, context)
		if err != nil {
			t.Errorf("ProcessTemplate failed: %v", err)
		}
		
		expected := `Hello Test User!
Your email is test@example.com.`
		
		if result != expected {
			t.Errorf("Template processing failed.\nExpected:\n%s\nGot:\n%s", expected, result)
		}
	})
	
	t.Run("Process template with conditionals", func(t *testing.T) {
		engine, err := NewGoTemplateEngine()
		if err != nil {
			t.Fatalf("Failed to create template engine: %v", err)
		}
		
		templateContent := `Config for {{.app}}:
{{if .features.enabled}}Feature is enabled{{else}}Feature is disabled{{end}}
{{if .debug}}Debug mode: ON{{end}}
{{range .plugins}}Plugin: {{.}}
{{end}}`
		
		context := map[string]interface{}{
			"app": "cursor",
			"features": map[string]interface{}{
				"enabled": true,
			},
			"debug":   true,
			"plugins": []string{"go", "terraform", "docker"},
		}
		
		result, err := engine.ProcessTemplate(templateContent, context)
		if err != nil {
			t.Errorf("ProcessTemplate with conditionals failed: %v", err)
		}
		
		// Verify conditional content
		if !containsString(result, "Feature is enabled") {
			t.Error("Template should contain 'Feature is enabled'")
		}
		if !containsString(result, "Debug mode: ON") {
			t.Error("Template should contain debug mode text")
		}
		if !containsString(result, "Plugin: go") {
			t.Error("Template should contain plugin list")
		}
		if containsString(result, "{{") {
			t.Error("Template should not contain unprocessed variables")
		}
	})
	
	t.Run("Process template with custom functions", func(t *testing.T) {
		engine, err := NewGoTemplateEngine()
		if err != nil {
			t.Fatalf("Failed to create template engine: %v", err)
		}
		
		templateContent := `Config path: {{configPath "cursor"}}
Homebrew bin: {{homebrewBin "/opt/homebrew"}}
Upper name: {{upper .name}}`
		
		context := map[string]interface{}{
			"name": "test user",
		}
		
		result, err := engine.ProcessTemplate(templateContent, context)
		if err != nil {
			t.Errorf("ProcessTemplate with custom functions failed: %v", err)
		}
		
		// Verify custom functions worked
		if !containsString(result, ".config/cursor") {
			t.Error("Template should contain processed configPath function")
		}
		if !containsString(result, "/opt/homebrew/bin") {
			t.Error("Template should contain processed homebrewBin function")
		}
		if !containsString(result, "TEST USER") {
			t.Error("Template should contain uppercased name")
		}
	})
	
	t.Run("Validate template syntax", func(t *testing.T) {
		engine, err := NewGoTemplateEngine()
		if err != nil {
			t.Fatalf("Failed to create template engine: %v", err)
		}
		
		// Valid template
		validTemplate := `Hello {{.name}}!`
		err = engine.ValidateTemplate(validTemplate)
		if err != nil {
			t.Errorf("Valid template should not error: %v", err)
		}
		
		// Invalid template
		invalidTemplate := `Hello {{.name}!` // Missing closing brace
		err = engine.ValidateTemplate(invalidTemplate)
		if err == nil {
			t.Error("Invalid template should return error")
		}
	})
	
	t.Run("Template error handling", func(t *testing.T) {
		engine, err := NewGoTemplateEngine()
		if err != nil {
			t.Fatalf("Failed to create template engine: %v", err)
		}
		
		// Template with missing variable
		templateContent := `Hello {{.missing_variable}}!`
		
		context := map[string]interface{}{
			"name": "Test User",
		}
		
		result, err := engine.ProcessTemplate(templateContent, context)
		if err != nil {
			// This behavior depends on template engine configuration
			t.Logf("Template with missing variable failed as expected: %v", err)
		} else {
			// Some template engines might render missing variables as empty
			if containsString(result, "{{") {
				t.Error("Template should not contain unprocessed variables")
			}
		}
	})
}

func TestTemplateProcessFile(t *testing.T) {
	tempDir := t.TempDir()
	
	t.Run("Process template file to output file", func(t *testing.T) {
		engine, err := NewGoTemplateEngine()
		if err != nil {
			t.Fatalf("Failed to create template engine: %v", err)
		}
		
		// Create template file
		templatePath := filepath.Join(tempDir, "config.template")
		templateContent := `[user]
    name = {{.user_name}}
    email = {{.user_email}}
[core]
    editor = {{.editor}}
{{if .gpg_key}}[user]
    signingkey = {{.gpg_key}}{{end}}`
		
		err = os.WriteFile(templatePath, []byte(templateContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create template file: %v", err)
		}
		
		// Process template to output file
		outputPath := filepath.Join(tempDir, "processed-config")
		context := map[string]interface{}{
			"user_name":  "Test User",
			"user_email": "test@example.com",
			"editor":     "vim",
			"gpg_key":    "ABC123DEF456",
		}
		
		err = engine.ProcessTemplateFile(templatePath, outputPath, context, "0644")
		if err != nil {
			t.Errorf("ProcessTemplateFile failed: %v", err)
		}
		
		// Verify output file exists
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Error("Output file should be created")
		}
		
		// Verify template was processed
		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
		} else {
			processedContent := string(content)
			if !containsString(processedContent, "Test User") {
				t.Error("Processed content should contain user name")
			}
			if !containsString(processedContent, "test@example.com") {
				t.Error("Processed content should contain email")
			}
			if !containsString(processedContent, "signingkey = ABC123DEF456") {
				t.Error("Processed content should contain conditional GPG key")
			}
			if containsString(processedContent, "{{") {
				t.Error("Processed content should not contain template variables")
			}
		}
		
		// Verify file permissions
		info, err := os.Stat(outputPath)
		if err != nil {
			t.Errorf("Failed to stat output file: %v", err)
		} else {
			expectedMode := os.FileMode(0644)
			if info.Mode().Perm() != expectedMode {
				t.Errorf("File permissions not set correctly: expected %v, got %v", expectedMode, info.Mode().Perm())
			}
		}
	})
	
	t.Run("Template file error cases", func(t *testing.T) {
		engine, err := NewGoTemplateEngine()
		if err != nil {
			t.Fatalf("Failed to create template engine: %v", err)
		}
		
		// Non-existent template file
		err = engine.ProcessTemplateFile("/nonexistent/template.tmpl", filepath.Join(tempDir, "output"), map[string]interface{}{}, "0644")
		if err == nil {
			t.Error("ProcessTemplateFile should error with non-existent template")
		}
		
		// Invalid permissions
		validTemplate := filepath.Join(tempDir, "valid.template")
		err = os.WriteFile(validTemplate, []byte("Hello {{.name}}"), 0644)
		if err != nil {
			t.Fatalf("Failed to create valid template: %v", err)
		}
		
		err = engine.ProcessTemplateFile(validTemplate, filepath.Join(tempDir, "output"), map[string]interface{}{"name": "test"}, "invalid")
		if err == nil {
			t.Error("ProcessTemplateFile should error with invalid permissions")
		}
	})
}

func TestTemplateContext(t *testing.T) {
	t.Run("Create template context", func(t *testing.T) {
		systemInfo := map[string]interface{}{
			"platform":     "macos",
			"architecture": "arm64",
			"home_dir":     "/Users/test",
			"config_dir":   "/Users/test/.config",
		}
		
		userVars := map[string]interface{}{
			"name":  "Test User",
			"email": "test@example.com",
		}
		
		context := CreateTemplateContext(systemInfo, userVars)
		if context == nil {
			t.Error("Template context should not be nil")
		}
		
		// Verify system info is included
		if context["system"] == nil {
			t.Error("Template context should include system info")
		}
		
		// Verify user variables are included
		if context["name"] != "Test User" {
			t.Error("Template context should include user variables")
		}
		if context["email"] != "test@example.com" {
			t.Error("Template context should include user email")
		}
	})
	
	t.Run("Template context with features", func(t *testing.T) {
		features := map[string]interface{}{
			"docker_enabled": true,
			"k8s_tools":      false,
			"tide_prompt":    true,
		}
		
		context := CreateTemplateContextWithFeatures(map[string]interface{}{}, map[string]interface{}{}, features)
		if context == nil {
			t.Error("Template context should not be nil")
		}
		
		// Verify features are included
		if context["features"] == nil {
			t.Error("Template context should include features")
		}
		
		featuresMap, ok := context["features"].(map[string]interface{})
		if !ok {
			t.Error("Features should be a map")
		} else {
			if featuresMap["docker_enabled"] != true {
				t.Error("Docker feature should be enabled")
			}
			if featuresMap["k8s_tools"] != false {
				t.Error("K8s tools feature should be disabled")
			}
		}
	})
}

// Helper function to check if string contains substring
func containsString(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
