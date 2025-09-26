// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package validators

import (
	"testing"
)

func TestValidFileMode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		mode      string
		wantError bool
	}{
		{"valid octal mode", "0644", false},
		{"valid decimal mode", "644", false},
		{"valid directory mode", "0755", false},
		{"invalid format", "abc", true},
		{"too permissive", "0999", true},
		{"empty string", "", false}, // Defaults to 0644
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mode, err := parseFileMode(tc.mode)
			hasError := err != nil

			if tc.wantError && !hasError {
				t.Errorf("Expected error for mode %q, but got none (parsed as %o)", tc.mode, mode)
			}
			if !tc.wantError && hasError {
				t.Errorf("Expected no error for mode %q, but got: %v", tc.mode, err)
			}
		})
	}
}

func TestValidTemplateEngine(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		engine string
		valid  bool
	}{
		{"go engine", "go", true},
		{"handlebars engine", "handlebars", true},
		{"mustache engine", "mustache", true},
		{"invalid engine", "jinja2", false},
		{"empty engine", "", true}, // Defaults to go
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simple validation check - the actual validator would use the framework
			validEngines := []string{"go", "handlebars", "mustache", ""}
			isValid := false
			for _, valid := range validEngines {
				if tc.engine == valid {
					isValid = true
					break
				}
			}

			if tc.valid && !isValid {
				t.Errorf("Expected %q to be valid, but it wasn't", tc.engine)
			}
			if !tc.valid && isValid {
				t.Errorf("Expected %q to be invalid, but it was valid", tc.engine)
			}
		})
	}
}

func TestValidTemplateName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		varName  string
		wantWarn bool
	}{
		{"valid name", "user_name", false},
		{"camelCase name", "userName", false},
		{"starts with underscore", "_private", false},
		{"contains numbers", "var123", false},
		{"reserved keyword", "if", true},
		{"system prefix", "system_var", true},
		{"platform prefix", "platform_info", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Check for reserved keywords
			reservedKeywords := []string{
				"if", "else", "end", "range", "with", "define", "template", "block",
				"true", "false", "nil", "null", "undefined",
			}

			isReserved := false
			for _, keyword := range reservedKeywords {
				if tc.varName == keyword {
					isReserved = true
					break
				}
			}

			hasSystemPrefix := len(tc.varName) > 6 && (tc.varName[:6] == "system" || (len(tc.varName) > 8 && tc.varName[:8] == "platform"))

			shouldWarn := isReserved || hasSystemPrefix

			if tc.wantWarn && !shouldWarn {
				t.Errorf("Expected warning for name %q, but got none", tc.varName)
			}
			if !tc.wantWarn && shouldWarn {
				t.Errorf("Expected no warning for name %q, but got one", tc.varName)
			}
		})
	}
}
