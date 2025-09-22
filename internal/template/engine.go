// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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

// TemplateEngine defines the interface for template processing.
type TemplateEngine interface {
	ProcessTemplate(templateContent string, context map[string]interface{}) (string, error)
	ProcessTemplateFile(templatePath, outputPath string, context map[string]interface{}, fileMode string) error
	ValidateTemplate(templateContent string) error
}

// GoTemplateEngine implements TemplateEngine using Go templates.
type GoTemplateEngine struct {
	functions template.FuncMap
}

// NewGoTemplateEngine creates a new Go template engine with custom functions.
func NewGoTemplateEngine() (*GoTemplateEngine, error) {
	engine := &GoTemplateEngine{
		functions: template.FuncMap{
			// Path helper functions
			"configPath": func(app string) string {
				return filepath.Join("~/.config", app)
			},
			"homebrewBin": func(prefix string) string {
				return filepath.Join(prefix, "bin")
			},

			// String helper functions
			"upper": strings.ToUpper,
			"lower": strings.ToLower,
			"title": func(s string) string {
				if len(s) == 0 {
					return s
				}
				return strings.ToUpper(s[:1]) + s[1:]
			},

			// Conditional helpers
			"default": func(defaultValue, value interface{}) interface{} {
				if value == nil || value == "" {
					return defaultValue
				}
				return value
			},

			// Platform helpers
			"isLinux":   func(platform string) bool { return platform == "linux" },
			"isMacOS":   func(platform string) bool { return platform == "macos" },
			"isWindows": func(platform string) bool { return platform == "windows" },
		},
	}

	return engine, nil
}

// ProcessTemplate processes a template string with the given context.
func (e *GoTemplateEngine) ProcessTemplate(templateContent string, context map[string]interface{}) (string, error) {
	// Create template with custom functions
	tmpl, err := template.New("template").Funcs(e.functions).Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Execute template
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, context)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// ProcessTemplateFile processes a template file and writes the result to output file.
func (e *GoTemplateEngine) ProcessTemplateFile(templatePath, outputPath string, context map[string]interface{}, fileMode string) error {
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

// ValidateTemplate validates template syntax without executing it.
func (e *GoTemplateEngine) ValidateTemplate(templateContent string) error {
	_, err := template.New("validation").Funcs(e.functions).Parse(templateContent)
	if err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}
	return nil
}

// CreateTemplateContext creates a template context with system and user variables.
func CreateTemplateContext(systemInfo, userVars map[string]interface{}) map[string]interface{} {
	context := make(map[string]interface{})

	// Add system information
	context["system"] = systemInfo

	// Add user variables at root level
	for key, value := range userVars {
		context[key] = value
	}

	return context
}

// CreateTemplateContextWithFeatures creates a template context with system, user, and feature variables.
func CreateTemplateContextWithFeatures(systemInfo, userVars, features map[string]interface{}) map[string]interface{} {
	context := CreateTemplateContext(systemInfo, userVars)

	// Add features
	context["features"] = features

	return context
}
