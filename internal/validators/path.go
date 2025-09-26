// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// PathValidator validates that a string is a valid path.
type PathValidator struct {
	allowRelative bool
	mustExist     bool
	mustBeDir     bool
}

// Description returns a description of the validator.
func (v PathValidator) Description(_ context.Context) string {
	desc := "value must be a valid path"
	if !v.allowRelative {
		desc += " (absolute path required)"
	}
	if v.mustExist {
		desc += " and must exist"
	}
	if v.mustBeDir {
		desc += " and must be a directory"
	}
	return desc
}

// MarkdownDescription returns a markdown description of the validator.
func (v PathValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString performs the validation.
func (v PathValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value := request.ConfigValue.ValueString()
	if value == "" {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Invalid Path",
			"Path cannot be empty",
		)
		return
	}

	// Check for invalid characters
	if strings.Contains(value, "\x00") {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Invalid Path",
			"Path cannot contain null characters",
		)
		return
	}

	// Check if absolute path is required
	if !v.allowRelative && !filepath.IsAbs(value) && !strings.HasPrefix(value, "~") {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Invalid Path",
			fmt.Sprintf("Path must be absolute, got: %s", value),
		)
		return
	}

	// Expand tilde if present (for existence checks)
	expandedPath := value
	if strings.HasPrefix(value, "~") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			expandedPath = filepath.Join(homeDir, value[1:])
		}
	}

	// Convert to absolute path for existence checks
	if v.mustExist || v.mustBeDir {
		absPath, err := filepath.Abs(expandedPath)
		if err != nil {
			response.Diagnostics.AddAttributeError(
				request.Path,
				"Invalid Path",
				fmt.Sprintf("Cannot resolve path: %s", err),
			)
			return
		}
		expandedPath = absPath
	}

	// Check if path must exist
	if v.mustExist {
		if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
			response.Diagnostics.AddAttributeError(
				request.Path,
				"Path Not Found",
				fmt.Sprintf("Path does not exist: %s", expandedPath),
			)
			return
		}
	}

	// Check if path must be a directory
	if v.mustBeDir {
		if info, err := os.Stat(expandedPath); err == nil {
			if !info.IsDir() {
				response.Diagnostics.AddAttributeError(
					request.Path,
					"Invalid Path Type",
					fmt.Sprintf("Path must be a directory, got file: %s", expandedPath),
				)
				return
			}
		} else if !os.IsNotExist(err) {
			response.Diagnostics.AddAttributeError(
				request.Path,
				"Path Access Error",
				fmt.Sprintf("Cannot access path: %s", err),
			)
			return
		}
	}
}

// ValidPath returns a validator which ensures that the provided value is a valid path.
func ValidPath() validator.String {
	return PathValidator{
		allowRelative: true,
		mustExist:     false,
		mustBeDir:     false,
	}
}

// AbsolutePath returns a validator which ensures that the provided value is an absolute path.
func AbsolutePath() validator.String {
	return PathValidator{
		allowRelative: false,
		mustExist:     false,
		mustBeDir:     false,
	}
}

// ExistingPath returns a validator which ensures that the provided path exists.
func ExistingPath() validator.String {
	return PathValidator{
		allowRelative: true,
		mustExist:     true,
		mustBeDir:     false,
	}
}

// ExistingDirectory returns a validator which ensures that the provided path exists and is a directory.
func ExistingDirectory() validator.String {
	return PathValidator{
		allowRelative: true,
		mustExist:     true,
		mustBeDir:     true,
	}
}

// WritableDirectory returns a validator which ensures that the provided path is a writable directory.
func WritableDirectory() validator.String {
	return writableDirectoryValidator{}
}

// writableDirectoryValidator validates that a path is a writable directory.
type writableDirectoryValidator struct{}

func (v writableDirectoryValidator) Description(_ context.Context) string {
	return "value must be a writable directory path"
}

func (v writableDirectoryValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v writableDirectoryValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value := request.ConfigValue.ValueString()
	if value == "" {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Invalid Path",
			"Path cannot be empty",
		)
		return
	}

	// Expand tilde if present
	expandedPath := value
	if strings.HasPrefix(value, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			response.Diagnostics.AddAttributeError(
				request.Path,
				"Path Expansion Error",
				fmt.Sprintf("Cannot expand home directory: %s", err),
			)
			return
		}
		expandedPath = filepath.Join(homeDir, value[1:])
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(expandedPath)
	if err != nil {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Invalid Path",
			fmt.Sprintf("Cannot resolve path: %s", err),
		)
		return
	}

	// Check if directory exists or can be created
	if info, err := os.Stat(absPath); err == nil {
		// Directory exists, check if it's writable
		if !info.IsDir() {
			response.Diagnostics.AddAttributeError(
				request.Path,
				"Invalid Path Type",
				fmt.Sprintf("Path must be a directory, got file: %s", absPath),
			)
			return
		}

		// Test write access by creating a temporary file
		tempFile := filepath.Join(absPath, ".terraform-provider-dotfiles-write-test")
		if file, err := os.Create(tempFile); err != nil {
			response.Diagnostics.AddAttributeError(
				request.Path,
				"Directory Not Writable",
				fmt.Sprintf("Directory is not writable: %s", absPath),
			)
			return
		} else {
			file.Close()
			os.Remove(tempFile) // Clean up
		}
	} else if os.IsNotExist(err) {
		// Directory doesn't exist, check if parent is writable
		parentDir := filepath.Dir(absPath)
		if parentInfo, err := os.Stat(parentDir); err != nil {
			response.Diagnostics.AddAttributeError(
				request.Path,
				"Parent Directory Not Found",
				fmt.Sprintf("Parent directory does not exist: %s", parentDir),
			)
			return
		} else if !parentInfo.IsDir() {
			response.Diagnostics.AddAttributeError(
				request.Path,
				"Invalid Parent Path",
				fmt.Sprintf("Parent path is not a directory: %s", parentDir),
			)
			return
		}

		// Test write access in parent directory
		tempFile := filepath.Join(parentDir, ".terraform-provider-dotfiles-write-test")
		if file, err := os.Create(tempFile); err != nil {
			response.Diagnostics.AddAttributeError(
				request.Path,
				"Parent Directory Not Writable",
				fmt.Sprintf("Cannot create directory in parent: %s", parentDir),
			)
			return
		} else {
			file.Close()
			os.Remove(tempFile) // Clean up
		}
	} else {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Path Access Error",
			fmt.Sprintf("Cannot access path: %s", err),
		)
		return
	}
}

// EnvironmentVariableExpansion returns a validator that supports environment variable expansion.
func EnvironmentVariableExpansion() validator.String {
	return envVarValidator{}
}

// envVarValidator validates and expands environment variables in paths.
type envVarValidator struct{}

func (v envVarValidator) Description(_ context.Context) string {
	return "value supports environment variable expansion (e.g., $HOME, ${HOME})"
}

func (v envVarValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v envVarValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value := request.ConfigValue.ValueString()
	if value == "" {
		return // Empty values are handled by other validators
	}

	// Check for valid environment variable syntax
	// Matches $VAR or ${VAR} patterns
	envVarPattern := regexp.MustCompile(`\$\{?[A-Za-z_][A-Za-z0-9_]*\}?`)
	matches := envVarPattern.FindAllString(value, -1)

	for _, match := range matches {
		// Extract variable name
		varName := match
		if strings.HasPrefix(match, "${") && strings.HasSuffix(match, "}") {
			varName = match[2 : len(match)-1]
		} else if strings.HasPrefix(match, "$") {
			varName = match[1:]
		}

		// Check if environment variable exists (warning only)
		if os.Getenv(varName) == "" {
			response.Diagnostics.AddAttributeWarning(
				request.Path,
				"Environment Variable Not Set",
				fmt.Sprintf("Environment variable '%s' is not set, expansion may result in empty string", varName),
			)
		}
	}
}
