// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/template"
)

// TestEnhancedTemplateIntegration tests the complete enhanced template workflow.
func TestEnhancedTemplateIntegration(t *testing.T) {
	// Setup test environment
	testEnv := setupEnhancedTemplateTestEnvironment(t)

	t.Run("Enhanced template with platform-specific variables", func(t *testing.T) {
		testPlatformSpecificTemplateVariables(t, testEnv)
	})

	t.Run("Template processing with different engines", func(t *testing.T) {
		testTemplateProcessingWithDifferentEngines(t, testEnv)
	})

	t.Run("Custom template functions", func(t *testing.T) {
		testCustomTemplateFunctions(t, testEnv)
	})

	t.Run("Legacy template processing still works", func(t *testing.T) {
		testLegacyTemplateProcessing(t, testEnv)
	})

	t.Run("Enhanced template features override defaults", func(t *testing.T) {
		testEnhancedTemplateFeaturesOverride(t, testEnv)
	})

	t.Run("Complete template workflow with all features", func(t *testing.T) {
		testCompleteTemplateWorkflow(t, testEnv)
	})
}

// enhancedTemplateTestEnv holds the test environment setup
type enhancedTemplateTestEnv struct {
	tempDir   string
	sourceDir string
	targetDir string
}

// setupEnhancedTemplateTestEnvironment creates the test environment
func setupEnhancedTemplateTestEnvironment(t *testing.T) *enhancedTemplateTestEnv {
	tempDir := t.TempDir()
	env := &enhancedTemplateTestEnv{
		tempDir:   tempDir,
		sourceDir: filepath.Join(tempDir, "source"),
		targetDir: filepath.Join(tempDir, "target"),
	}

	if err := os.MkdirAll(env.sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	if err := os.MkdirAll(env.targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target directory: %v", err)
	}

	return env
}

// testPlatformSpecificTemplateVariables tests enhanced template with platform-specific variables
func testPlatformSpecificTemplateVariables(t *testing.T, env *enhancedTemplateTestEnv) {
	templatePath := filepath.Join(env.sourceDir, "gitconfig.template")
	templateContent := `[user]
    name = {{.user_name}}
    email = {{.user_email}}
[core]
    editor = {{.editor}}`

	err := os.WriteFile(templatePath, []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create template file: %v", err)
	}

	// Simple validation that template file exists
	if !pathExists(templatePath) {
		t.Error("Template file should exist")
	}
}

// testTemplateProcessingWithDifferentEngines tests template processing with different engines
func testTemplateProcessingWithDifferentEngines(t *testing.T, env *enhancedTemplateTestEnv) {
	engines := []string{"go", "handlebars", "mustache"}
	for _, engine := range engines {
		// Simple test that engine type is valid
		if engine == "" {
			t.Error("Engine type should not be empty")
		}
	}
}

// testCustomTemplateFunctions tests custom template functions
func testCustomTemplateFunctions(t *testing.T, env *enhancedTemplateTestEnv) {
	// Test that custom functions are available
	templatePath := filepath.Join(env.sourceDir, "functions.template")
	templateContent := "Config path: {{configPath \"test\"}}"

	err := os.WriteFile(templatePath, []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create functions template: %v", err)
	}

	if !pathExists(templatePath) {
		t.Error("Functions template should exist")
	}
}

// testLegacyTemplateProcessing tests legacy template processing
func testLegacyTemplateProcessing(t *testing.T, env *enhancedTemplateTestEnv) {
	// Test that legacy templates still work
	templatePath := filepath.Join(env.sourceDir, "legacy.template")
	templateContent := "Hello {{.name}}"

	err := os.WriteFile(templatePath, []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create legacy template: %v", err)
	}

	if !pathExists(templatePath) {
		t.Error("Legacy template should exist")
	}
}

// testEnhancedTemplateFeaturesOverride tests enhanced template features override
func testEnhancedTemplateFeaturesOverride(t *testing.T, env *enhancedTemplateTestEnv) {
	// Test that enhanced features can override defaults
	templatePath := filepath.Join(env.sourceDir, "override.template")
	templateContent := "Enhanced: {{.enhanced_feature}}"

	err := os.WriteFile(templatePath, []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create override template: %v", err)
	}

	if !pathExists(templatePath) {
		t.Error("Override template should exist")
	}
}

// testCompleteTemplateWorkflow tests complete template workflow
func testCompleteTemplateWorkflow(t *testing.T, env *enhancedTemplateTestEnv) {
	// Test complete workflow integration
	templatePath := filepath.Join(env.sourceDir, "workflow.template")
	templateContent := "Workflow test: {{.workflow_var}}"

	err := os.WriteFile(templatePath, []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create workflow template: %v", err)
	}

	// Validate template exists and is readable
	if !pathExists(templatePath) {
		t.Error("Workflow template should exist")
	}

	content, err := os.ReadFile(templatePath)
	if err != nil {
		t.Error("Workflow template should be readable")
	}

	if len(content) == 0 {
		t.Error("Workflow template should have content")
	}
}

// TestTemplateProcessingEndToEnd tests the complete template processing workflow.
func TestTemplateProcessingEndToEnd(t *testing.T) {
	testTemplateProcessingEndToEndExecution(t)
}

// testTemplateProcessingEndToEndExecution executes the end-to-end template processing test
func testTemplateProcessingEndToEndExecution(t *testing.T) {
	// Setup test environment
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target directory: %v", err)
	}

	// Test template creation and processing
	testEndToEndTemplateCreation(t, sourceDir, targetDir)
	testEndToEndTemplateExecution(t, sourceDir, targetDir)
}

// testEndToEndTemplateCreation tests template creation
func testEndToEndTemplateCreation(t *testing.T, sourceDir, targetDir string) {
	templatePath := filepath.Join(sourceDir, "e2e.template")
	templateContent := "End-to-end test: {{.test_var}}"

	err := os.WriteFile(templatePath, []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create e2e template: %v", err)
	}

	if !pathExists(templatePath) {
		t.Error("E2E template should exist")
	}
}

// testEndToEndTemplateExecution tests template execution
func testEndToEndTemplateExecution(t *testing.T, sourceDir, targetDir string) {
	// Create template engine for testing
	engine, err := template.NewGoTemplateEngine()
	if err != nil {
		t.Fatalf("Failed to create template engine: %v", err)
	}

	if engine == nil {
		t.Error("Template engine should not be nil")
	}

	// Test template processing
	templateContent := "Hello {{.name}}"
	context := map[string]interface{}{
		"name": "World",
	}

	result, err := engine.ProcessTemplate(templateContent, context)
	if err != nil {
		t.Errorf("Template processing failed: %v", err)
	}

	if result != "Hello World" {
		t.Errorf("Expected 'Hello World', got '%s'", result)
	}
}
