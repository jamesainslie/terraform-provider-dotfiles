// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package validators

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// TemplateVariableNameValidator validates that a string is a valid template variable name.
type TemplateVariableNameValidator struct{}

// Description returns a description of the validator.
func (v TemplateVariableNameValidator) Description(_ context.Context) string {
	return "value must be a valid template variable name (alphanumeric and underscores, starting with letter or underscore)"
}

// MarkdownDescription returns a markdown description of the validator.
func (v TemplateVariableNameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString performs the validation.
func (v TemplateVariableNameValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value := request.ConfigValue.ValueString()
	if value == "" {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Invalid Template Variable Name",
			"Template variable name cannot be empty",
		)
		return
	}

	// Check if it's a valid identifier
	validName := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	if !validName.MatchString(value) {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Invalid Template Variable Name",
			fmt.Sprintf("Template variable name '%s' must start with a letter or underscore and contain only letters, numbers, and underscores", value),
		)
		return
	}

	// Check for reserved keywords that might conflict with template engines
	reservedKeywords := []string{
		"if", "else", "end", "range", "with", "define", "template", "block",
		"true", "false", "nil", "null", "undefined",
		"index", "len", "print", "printf", "println",
	}

	lowerValue := strings.ToLower(value)
	for _, keyword := range reservedKeywords {
		if lowerValue == keyword {
			response.Diagnostics.AddAttributeWarning(
				request.Path,
				"Reserved Template Variable Name",
				fmt.Sprintf("Template variable name '%s' is a reserved keyword and may cause conflicts", value),
			)
			break
		}
	}

	// Warn about potentially confusing names
	if strings.HasPrefix(lowerValue, "system") || strings.HasPrefix(lowerValue, "platform") {
		response.Diagnostics.AddAttributeWarning(
			request.Path,
			"Potentially Conflicting Variable Name",
			fmt.Sprintf("Template variable name '%s' may conflict with built-in system/platform variables", value),
		)
	}
}

// ValidTemplateVariableName returns a validator which ensures that the provided value is a valid template variable name.
func ValidTemplateVariableName() validator.String {
	return TemplateVariableNameValidator{}
}

// TemplateEngineValidator validates that a string is a supported template engine.
type TemplateEngineValidator struct{}

// Description returns a description of the validator.
func (v TemplateEngineValidator) Description(_ context.Context) string {
	return "value must be a supported template engine: 'go', 'handlebars', or 'mustache'"
}

// MarkdownDescription returns a markdown description of the validator.
func (v TemplateEngineValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString performs the validation.
func (v TemplateEngineValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value := request.ConfigValue.ValueString()
	if value == "" {
		return // Empty values default to "go"
	}

	supportedEngines := []string{"go", "handlebars", "mustache"}
	for _, engine := range supportedEngines {
		if value == engine {
			return // Valid engine
		}
	}

	response.Diagnostics.AddAttributeError(
		request.Path,
		"Unsupported Template Engine",
		fmt.Sprintf("Template engine '%s' is not supported. Supported engines are: %s", value, strings.Join(supportedEngines, ", ")),
	)
}

// ValidTemplateEngine returns a validator which ensures that the provided value is a supported template engine.
func ValidTemplateEngine() validator.String {
	return TemplateEngineValidator{}
}

// TemplateSyntaxValidator validates basic template syntax.
type TemplateSyntaxValidator struct {
	engine string
}

// Description returns a description of the validator.
func (v TemplateSyntaxValidator) Description(_ context.Context) string {
	return fmt.Sprintf("value must contain valid %s template syntax", v.engine)
}

// MarkdownDescription returns a markdown description of the validator.
func (v TemplateSyntaxValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString performs the validation.
func (v TemplateSyntaxValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value := request.ConfigValue.ValueString()
	if value == "" {
		return
	}

	// Basic syntax validation for template delimiters
	if !v.validateBasicSyntax(value) {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Invalid Template Syntax",
			fmt.Sprintf("Template contains unmatched delimiters or invalid syntax for %s engine", v.engine),
		)
		return
	}

	// Engine-specific validation
	switch v.engine {
	case "go":
		v.validateGoTemplate(value, request, response)
	case "handlebars":
		v.validateHandlebarsTemplate(value, request, response)
	case "mustache":
		v.validateMustacheTemplate(value, request, response)
	}
}

// validateBasicSyntax checks for balanced delimiters.
func (v TemplateSyntaxValidator) validateBasicSyntax(content string) bool {
	openCount := strings.Count(content, "{{")
	closeCount := strings.Count(content, "}}")
	return openCount == closeCount
}

// validateGoTemplate validates Go template syntax.
func (v TemplateSyntaxValidator) validateGoTemplate(content string, request validator.StringRequest, response *validator.StringResponse) {
	// Check for common Go template patterns
	goPatterns := []string{`\{\{\.`, `\{\{range`, `\{\{if`, `\{\{with`, `\{\{end\}\}`}
	hasGoSyntax := false
	for _, pattern := range goPatterns {
		if matched, _ := regexp.MatchString(pattern, content); matched {
			hasGoSyntax = true
			break
		}
	}

	// If it contains template delimiters but no Go syntax, warn
	if strings.Contains(content, "{{") && !hasGoSyntax {
		response.Diagnostics.AddAttributeWarning(
			request.Path,
			"Potential Template Syntax Issue",
			"Template contains delimiters but no recognizable Go template syntax patterns",
		)
	}
}

// validateHandlebarsTemplate validates Handlebars template syntax.
func (v TemplateSyntaxValidator) validateHandlebarsTemplate(content string, request validator.StringRequest, response *validator.StringResponse) {
	// Check for Handlebars-specific syntax
	handlebarsPatterns := []string{`\{\{#if`, `\{\{#each`, `\{\{#with`, `\{\{/`, `\{\{#unless`}
	hasHandlebarsSyntax := false
	for _, pattern := range handlebarsPatterns {
		if matched, _ := regexp.MatchString(pattern, content); matched {
			hasHandlebarsSyntax = true
			break
		}
	}
	_ = hasHandlebarsSyntax // Reserved for future use

	// Check for unmatched block helpers
	if strings.Contains(content, "{{#") {
		ifCount := strings.Count(content, "{{#if")
		endIfCount := strings.Count(content, "{{/if}}")
		if ifCount != endIfCount {
			response.Diagnostics.AddAttributeWarning(
				request.Path,
				"Unmatched Handlebars Blocks",
				"Template may have unmatched {{#if}}/{{/if}} blocks",
			)
		}

		eachCount := strings.Count(content, "{{#each")
		endEachCount := strings.Count(content, "{{/each}}")
		if eachCount != endEachCount {
			response.Diagnostics.AddAttributeWarning(
				request.Path,
				"Unmatched Handlebars Blocks",
				"Template may have unmatched {{#each}}/{{/each}} blocks",
			)
		}
	}
}

// validateMustacheTemplate validates Mustache template syntax.
func (v TemplateSyntaxValidator) validateMustacheTemplate(content string, request validator.StringRequest, response *validator.StringResponse) {
	// Check for Mustache-specific syntax
	mustachePatterns := []string{`\{\{#`, `\{\{/`, `\{\{\^`, `\{\{&`, `\{\{\{.*\}\}\}`}
	hasMustacheSyntax := false
	for _, pattern := range mustachePatterns {
		if matched, _ := regexp.MatchString(pattern, content); matched {
			hasMustacheSyntax = true
			break
		}
	}
	_ = hasMustacheSyntax // Reserved for future use

	// Check for unmatched sections
	if strings.Contains(content, "{{#") || strings.Contains(content, "{{^") {
		// This is a simplified check - real validation would need proper parsing
		openSections := strings.Count(content, "{{#") + strings.Count(content, "{{^")
		closeSections := strings.Count(content, "{{/")
		if openSections != closeSections {
			response.Diagnostics.AddAttributeWarning(
				request.Path,
				"Unmatched Mustache Sections",
				"Template may have unmatched section tags",
			)
		}
	}
}

// ValidTemplateSyntax returns a validator which ensures that the provided value contains valid template syntax for the specified engine.
func ValidTemplateSyntax(engine string) validator.String {
	return TemplateSyntaxValidator{engine: engine}
}
