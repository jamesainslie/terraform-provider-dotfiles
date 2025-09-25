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
	// More robust conversion for Handlebars compatibility
	result := content

	// Convert {{#if var}} to {{if .var}}
	result = strings.ReplaceAll(result, "{{#if ", "{{if .")
	result = strings.ReplaceAll(result, "{{/if}}", "{{end}}")

	// Convert {{#unless var}} to {{if not .var}}
	result = strings.ReplaceAll(result, "{{#unless ", "{{if not .")
	result = strings.ReplaceAll(result, "{{/unless}}", "{{end}}")

	// Convert {{#each items}} to {{range .items}}
	result = strings.ReplaceAll(result, "{{#each ", "{{range .")
	result = strings.ReplaceAll(result, "{{/each}}", "{{end}}")

	// Convert {{#with obj}} to {{with .obj}}
	result = strings.ReplaceAll(result, "{{#with ", "{{with .")
	result = strings.ReplaceAll(result, "{{/with}}", "{{end}}")

	// Convert simple variables: {{var}} to {{.var}} (but not if already has dot or contains spaces/operators)
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		// Find all {{...}} patterns
		start := 0
		for {
			openIdx := strings.Index(line[start:], "{{")
			if openIdx == -1 {
				break
			}
			openIdx += start

			closeIdx := strings.Index(line[openIdx:], "}}")
			if closeIdx == -1 {
				break
			}
			closeIdx += openIdx + 2

			// Extract the content between {{ and }}
			content := line[openIdx+2 : closeIdx-2]
			content = strings.TrimSpace(content)

			// Skip if it's a control structure, already has dot, or contains operators
			if !strings.HasPrefix(content, ".") &&
				!strings.Contains(content, " ") &&
				!strings.Contains(content, "if") &&
				!strings.Contains(content, "range") &&
				!strings.Contains(content, "with") &&
				!strings.Contains(content, "end") &&
				!strings.Contains(content, "not") &&
				content != "" {
				// Replace {{var}} with {{.var}}
				newContent := "{{." + content + "}}"
				line = line[:openIdx] + newContent + line[closeIdx:]
				start = openIdx + len(newContent)
			} else {
				start = closeIdx
			}
		}
		lines[i] = line
	}

	return strings.Join(lines, "\n")
}

// convertMustacheToGo converts Mustache syntax to Go template syntax.
func convertMustacheToGo(content string) string {
	// More robust conversion for Mustache compatibility
	result := content

	// Convert {{#section}} to {{with .section}} (for object context)
	// Convert {{#items}} to {{range .items}} (for array iteration)
	// This is a simplified approach - real Mustache would need context analysis

	// Handle sections that look like arrays (plural names often indicate arrays)
	result = strings.ReplaceAll(result, "{{#items}}", "{{range .items}}")
	result = strings.ReplaceAll(result, "{{#users}}", "{{range .users}}")
	result = strings.ReplaceAll(result, "{{#files}}", "{{range .files}}")
	result = strings.ReplaceAll(result, "{{#configs}}", "{{range .configs}}")

	// Handle sections that look like objects
	result = strings.ReplaceAll(result, "{{#user}}", "{{with .user}}")
	result = strings.ReplaceAll(result, "{{#config}}", "{{with .config}}")
	result = strings.ReplaceAll(result, "{{#settings}}", "{{with .settings}}")

	// Convert closing tags
	result = strings.ReplaceAll(result, "{{/items}}", "{{end}}")
	result = strings.ReplaceAll(result, "{{/users}}", "{{end}}")
	result = strings.ReplaceAll(result, "{{/files}}", "{{end}}")
	result = strings.ReplaceAll(result, "{{/configs}}", "{{end}}")
	result = strings.ReplaceAll(result, "{{/user}}", "{{end}}")
	result = strings.ReplaceAll(result, "{{/config}}", "{{end}}")
	result = strings.ReplaceAll(result, "{{/settings}}", "{{end}}")

	// Convert simple variables: {{var}} to {{.var}} (similar to Handlebars)
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		// Find all {{...}} patterns
		start := 0
		for {
			openIdx := strings.Index(line[start:], "{{")
			if openIdx == -1 {
				break
			}
			openIdx += start

			closeIdx := strings.Index(line[openIdx:], "}}")
			if closeIdx == -1 {
				break
			}
			closeIdx += openIdx + 2

			// Extract the content between {{ and }}
			content := line[openIdx+2 : closeIdx-2]
			content = strings.TrimSpace(content)

			// Skip if it's a control structure, already has dot, or contains operators
			if !strings.HasPrefix(content, ".") &&
				!strings.Contains(content, " ") &&
				!strings.Contains(content, "range") &&
				!strings.Contains(content, "with") &&
				!strings.Contains(content, "end") &&
				!strings.HasPrefix(content, "#") &&
				!strings.HasPrefix(content, "/") &&
				content != "" {
				// Replace {{var}} with {{.var}}
				newContent := "{{." + content + "}}"
				line = line[:openIdx] + newContent + line[closeIdx:]
				start = openIdx + len(newContent)
			} else {
				start = closeIdx
			}
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
		engine, err := NewHandlebarsTemplateEngine()
		if err != nil {
			return nil, err
		}
		// Add custom functions to the Handlebars engine
		for name, fn := range customFunctions {
			engine.functions[name] = fn
		}
		return engine, nil
	case "mustache":
		engine, err := NewMustacheTemplateEngine()
		if err != nil {
			return nil, err
		}
		// Add custom functions to the Mustache engine
		for name, fn := range customFunctions {
			engine.functions[name] = fn
		}
		return engine, nil
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
