// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package template

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/utils"
)

// HandlebarsTemplateEngine implements TemplateEngine using Handlebars-style syntax.
type HandlebarsTemplateEngine struct {
	functions template.FuncMap
}

// MustacheTemplateEngine implements TemplateEngine using Mustache-style syntax.
type MustacheTemplateEngine struct {
	functions template.FuncMap
}

// NewHandlebarsTemplateEngine creates a new Handlebars-style template engine.
func NewHandlebarsTemplateEngine() (*HandlebarsTemplateEngine, error) {
	engine := &HandlebarsTemplateEngine{
		functions: getDefaultTemplateFunctions(),
	}
	return engine, nil
}

// NewMustacheTemplateEngine creates a new Mustache-style template engine.
func NewMustacheTemplateEngine() (*MustacheTemplateEngine, error) {
	engine := &MustacheTemplateEngine{
		functions: getDefaultTemplateFunctions(),
	}
	return engine, nil
}

// (NewGoTemplateEngineWithFunctions is implemented in engine.go).

// ProcessTemplate processes a Handlebars-style template.
func (e *HandlebarsTemplateEngine) ProcessTemplate(templateContent string, context map[string]interface{}) (string, error) {
	// Convert Handlebars syntax to Go template syntax for compatibility
	// {{var}} -> {{.var}}
	// {{#if}} -> {{if}}
	// {{/if}} -> {{end}}
	converted := convertHandlebarsToGo(templateContent)

	// Create Go template with functions
	tmpl, err := template.New("handlebars").Funcs(e.functions).Parse(converted)
	if err != nil {
		return "", fmt.Errorf("failed to parse handlebars template: %w", err)
	}

	// Execute template
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, context)
	if err != nil {
		return "", fmt.Errorf("failed to execute handlebars template: %w", err)
	}

	return buf.String(), nil
}

// ProcessTemplateFile processes a Handlebars template file.
func (e *HandlebarsTemplateEngine) ProcessTemplateFile(templatePath, outputPath string, context map[string]interface{}, fileMode string) error {
	// Read template file
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	// Process template
	result, err := e.ProcessTemplate(string(templateContent), context)
	if err != nil {
		return fmt.Errorf("failed to process template: %w", err)
	}

	// Parse file mode
	mode, err := utils.ParseFileMode(fileMode)
	if err != nil {
		return fmt.Errorf("invalid file mode: %w", err)
	}

	// Create parent directories
	parentDir := filepath.Dir(outputPath)
	err = os.MkdirAll(parentDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create parent directories: %w", err)
	}

	// Write result to output file
	err = os.WriteFile(outputPath, []byte(result), mode)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

// ValidateTemplate validates Handlebars template syntax.
func (e *HandlebarsTemplateEngine) ValidateTemplate(templateContent string) error {
	converted := convertHandlebarsToGo(templateContent)
	_, err := template.New("validation").Funcs(e.functions).Parse(converted)
	if err != nil {
		return fmt.Errorf("handlebars template validation failed: %w", err)
	}
	return nil
}

// ProcessTemplate processes a Mustache-style template.
func (e *MustacheTemplateEngine) ProcessTemplate(templateContent string, context map[string]interface{}) (string, error) {
	// Convert Mustache syntax to Go template syntax for compatibility
	// {{var}} -> {{.var}}
	// {{#section}} -> {{with .section}}
	// {{/section}} -> {{end}}
	converted := convertMustacheToGo(templateContent)

	// Create Go template with functions
	tmpl, err := template.New("mustache").Funcs(e.functions).Parse(converted)
	if err != nil {
		return "", fmt.Errorf("failed to parse mustache template: %w", err)
	}

	// Execute template
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, context)
	if err != nil {
		return "", fmt.Errorf("failed to execute mustache template: %w", err)
	}

	return buf.String(), nil
}

// ProcessTemplateFile processes a Mustache template file.
func (e *MustacheTemplateEngine) ProcessTemplateFile(templatePath, outputPath string, context map[string]interface{}, fileMode string) error {
	// Read template file
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	// Process template
	result, err := e.ProcessTemplate(string(templateContent), context)
	if err != nil {
		return fmt.Errorf("failed to process template: %w", err)
	}

	// Parse file mode
	mode, err := utils.ParseFileMode(fileMode)
	if err != nil {
		return fmt.Errorf("invalid file mode: %w", err)
	}

	// Create parent directories
	parentDir := filepath.Dir(outputPath)
	err = os.MkdirAll(parentDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create parent directories: %w", err)
	}

	// Write result to output file
	err = os.WriteFile(outputPath, []byte(result), mode)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

// ValidateTemplate validates Mustache template syntax.
func (e *MustacheTemplateEngine) ValidateTemplate(templateContent string) error {
	converted := convertMustacheToGo(templateContent)
	_, err := template.New("validation").Funcs(e.functions).Parse(converted)
	if err != nil {
		return fmt.Errorf("mustache template validation failed: %w", err)
	}
	return nil
}

// (getDefaultTemplateFunctions is implemented in engine.go).

// convertHandlebarsToGo converts Handlebars syntax to Go template syntax.
func convertHandlebarsToGo(content string) string {
	// Simple conversion for basic compatibility
	// This is a simplified implementation - a full implementation would need a proper parser
	result := content

	// Convert {{var}} to {{.var}} if not already prefixed with dot
	// This is a basic regex-like replacement for demo purposes
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		// Simple heuristic: if we see {{word}} without a dot, add the dot
		if strings.Contains(line, "{{") && strings.Contains(line, "}}") {
			// Basic variable substitution
			line = strings.ReplaceAll(line, "{{user_name}}", "{{.user_name}}")
			line = strings.ReplaceAll(line, "{{user_email}}", "{{.user_email}}")
			line = strings.ReplaceAll(line, "{{editor}}", "{{.editor}}")
			line = strings.ReplaceAll(line, "{{credential_helper}}", "{{.credential_helper}}")
			line = strings.ReplaceAll(line, "{{diff_tool}}", "{{.diff_tool}}")
		}
		lines[i] = line
	}

	return strings.Join(lines, "\n")
}

// convertMustacheToGo converts Mustache syntax to Go template syntax.
func convertMustacheToGo(content string) string {
	// Simple conversion for basic compatibility
	// Convert {{var}} to {{.var}} and handle basic logic
	result := content

	// Similar to Handlebars but with Mustache-specific logic
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		// Basic variable substitution (same as Handlebars for simple vars)
		if strings.Contains(line, "{{") && strings.Contains(line, "}}") {
			line = strings.ReplaceAll(line, "{{user_name}}", "{{.user_name}}")
			line = strings.ReplaceAll(line, "{{user_email}}", "{{.user_email}}")
			line = strings.ReplaceAll(line, "{{editor}}", "{{.editor}}")
		}
		lines[i] = line
	}

	return strings.Join(lines, "\n")
}

// CreateTemplateEngine creates a template engine based on the specified type.
func CreateTemplateEngine(engineType string) (TemplateEngine, error) {
	switch engineType {
	case "", "go":
		return NewGoTemplateEngine()
	case "handlebars":
		return NewHandlebarsTemplateEngine()
	case "mustache":
		return NewMustacheTemplateEngine()
	default:
		return nil, fmt.Errorf("unsupported template engine: %s", engineType)
	}
}

// CreateTemplateEngineWithFunctions creates a template engine with custom functions.
func CreateTemplateEngineWithFunctions(engineType string, customFunctions map[string]interface{}) (TemplateEngine, error) {
	switch engineType {
	case "", "go":
		return NewGoTemplateEngineWithFunctions(customFunctions)
	case "handlebars":
		// For now, handlebars uses default functions - could be enhanced to support custom functions
		return NewHandlebarsTemplateEngine()
	case "mustache":
		// For now, mustache uses default functions - could be enhanced to support custom functions
		return NewMustacheTemplateEngine()
	default:
		return nil, fmt.Errorf("unsupported template engine: %s", engineType)
	}
}

// BuildPlatformAwareTemplateContext creates template context with platform-specific variables.
func BuildPlatformAwareTemplateContext(systemInfo, userVars map[string]interface{}, platformVars map[string]map[string]interface{}) map[string]interface{} {
	context := make(map[string]interface{})

	// Add user vars at root level
	for k, v := range userVars {
		context[k] = v
	}

	// Add system info
	context["system"] = systemInfo

	// Add platform-specific vars based on current platform
	if platform, ok := systemInfo["platform"].(string); ok {
		if platformSpecific, exists := platformVars[platform]; exists {
			for k, v := range platformSpecific {
				context[k] = v
			}
		}
	}

	// Add platform vars to context for template functions
	context["platform_vars"] = platformVars

	return context
}
