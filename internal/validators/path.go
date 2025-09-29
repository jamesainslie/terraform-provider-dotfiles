// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
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

	// Basic validation
	if !v.validateBasicPath(value, request.Path, response) {
		return
	}

	// Path format validation
	if !v.validatePathFormat(value, request.Path, response) {
		return
	}

	// Path expansion and resolution
	expandedPath, ok := v.expandAndResolvePath(value, request.Path, response)
	if !ok {
		return
	}

	// Existence and type checks
	v.validatePathProperties(expandedPath, request.Path, response)
}

// validateBasicPath performs basic path validation (empty check)
func (v PathValidator) validateBasicPath(value string, path path.Path, response *validator.StringResponse) bool {
	if value == "" {
		response.Diagnostics.AddAttributeError(
			path,
			"Invalid Path",
			"Path cannot be empty",
		)
		return false
	}
	return true
}

// validatePathFormat validates path format and characters
func (v PathValidator) validatePathFormat(value string, path path.Path, response *validator.StringResponse) bool {
	// Check for invalid characters
	if strings.Contains(value, "\x00") {
		response.Diagnostics.AddAttributeError(
			path,
			"Invalid Path",
			"Path cannot contain null characters",
		)
		return false
	}

	// Check if absolute path is required
	if !v.allowRelative && !filepath.IsAbs(value) && !strings.HasPrefix(value, "~") {
		response.Diagnostics.AddAttributeError(
			path,
			"Invalid Path",
			fmt.Sprintf("Path must be absolute, got: %s", value),
		)
		return false
	}

	return true
}

// expandAndResolvePath expands and resolves the path
func (v PathValidator) expandAndResolvePath(value string, pathAttr path.Path, response *validator.StringResponse) (string, bool) {
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
				pathAttr,
				"Invalid Path",
				fmt.Sprintf("Cannot resolve path: %s", err),
			)
			return "", false
		}
		expandedPath = absPath
	}

	return expandedPath, true
}

// validatePathProperties validates path existence and type requirements
func (v PathValidator) validatePathProperties(expandedPath string, pathAttr path.Path, response *validator.StringResponse) {
	// Check if path must exist
	if v.mustExist {
		if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
			response.Diagnostics.AddAttributeError(
				pathAttr,
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
					pathAttr,
					"Invalid Path Type",
					fmt.Sprintf("Path must be a directory, got file: %s", expandedPath),
				)
				return
			}
		} else if !os.IsNotExist(err) {
			response.Diagnostics.AddAttributeError(
				pathAttr,
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

	// Basic validation
	if !v.validateBasicWritablePath(value, request.Path, response) {
		return
	}

	// Path expansion and resolution
	absPath, ok := v.expandAndResolveWritablePath(value, request.Path, response)
	if !ok {
		return
	}

	// Directory writability checks
	v.validateDirectoryWritability(absPath, request.Path, response)
}

// validateBasicWritablePath performs basic validation for writable directory paths
func (v writableDirectoryValidator) validateBasicWritablePath(value string, pathAttr path.Path, response *validator.StringResponse) bool {
	if value == "" {
		response.Diagnostics.AddAttributeError(
			pathAttr,
			"Invalid Path",
			"Path cannot be empty",
		)
		return false
	}
	return true
}

// expandAndResolveWritablePath expands and resolves the path for writability checking
func (v writableDirectoryValidator) expandAndResolveWritablePath(value string, pathAttr path.Path, response *validator.StringResponse) (string, bool) {
	// Expand tilde if present
	expandedPath := value
	if strings.HasPrefix(value, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			response.Diagnostics.AddAttributeError(
				pathAttr,
				"Path Expansion Error",
				fmt.Sprintf("Cannot expand home directory: %s", err),
			)
			return "", false
		}
		expandedPath = filepath.Join(homeDir, value[1:])
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(expandedPath)
	if err != nil {
		response.Diagnostics.AddAttributeError(
			pathAttr,
			"Invalid Path",
			fmt.Sprintf("Cannot resolve path: %s", err),
		)
		return "", false
	}

	return absPath, true
}

// validateDirectoryWritability validates directory existence and writability
func (v writableDirectoryValidator) validateDirectoryWritability(absPath string, pathAttr path.Path, response *validator.StringResponse) {
	if info, err := os.Stat(absPath); err == nil {
		v.validateExistingDirectory(absPath, info, pathAttr, response)
	} else if os.IsNotExist(err) {
		v.validateParentDirectoryWritability(absPath, pathAttr, response)
	} else {
		response.Diagnostics.AddAttributeError(
			pathAttr,
			"Path Access Error",
			fmt.Sprintf("Cannot access path: %s", err),
		)
	}
}

// validateExistingDirectory validates an existing directory's type and writability
func (v writableDirectoryValidator) validateExistingDirectory(absPath string, info os.FileInfo, pathAttr path.Path, response *validator.StringResponse) {
	// Directory exists, check if it's actually a directory
	if !info.IsDir() {
		response.Diagnostics.AddAttributeError(
			pathAttr,
			"Invalid Path Type",
			fmt.Sprintf("Path must be a directory, got file: %s", absPath),
		)
		return
	}

	// Test write access by creating a temporary file
	v.testDirectoryWriteAccess(absPath, pathAttr, response)
}

// validateParentDirectoryWritability validates parent directory for creating new directories
func (v writableDirectoryValidator) validateParentDirectoryWritability(absPath string, pathAttr path.Path, response *validator.StringResponse) {
	parentDir := filepath.Dir(absPath)
	if parentInfo, err := os.Stat(parentDir); err != nil {
		response.Diagnostics.AddAttributeError(
			pathAttr,
			"Parent Directory Not Found",
			fmt.Sprintf("Parent directory does not exist: %s", parentDir),
		)
	} else if !parentInfo.IsDir() {
		response.Diagnostics.AddAttributeError(
			pathAttr,
			"Invalid Parent Path",
			fmt.Sprintf("Parent path is not a directory: %s", parentDir),
		)
	} else {
		// Test write access in parent directory
		v.testDirectoryWriteAccess(parentDir, pathAttr, response)
	}
}

// testDirectoryWriteAccess tests write access by creating and cleaning up a temporary file
func (v writableDirectoryValidator) testDirectoryWriteAccess(dirPath string, pathAttr path.Path, response *validator.StringResponse) {
	tempFile := filepath.Join(dirPath, ".terraform-provider-dotfiles-write-test")
	file, err := os.Create(tempFile)
	if err != nil {
		errorMsg := fmt.Sprintf("Directory is not writable: %s", dirPath)
		if dirPath != filepath.Dir(dirPath) { // if this is a parent dir check
			errorMsg = fmt.Sprintf("Cannot create directory in parent: %s", dirPath)
		}
		response.Diagnostics.AddAttributeError(
			pathAttr,
			"Directory Not Writable",
			errorMsg,
		)
		return
	}

	// Clean up temporary file
	if err := file.Close(); err != nil {
		// Log error but continue
		fmt.Printf("Warning: failed to close temp file: %v\n", err)
	}
	if err := os.Remove(tempFile); err != nil {
		// Log error but continue - this is just cleanup
		fmt.Printf("Warning: failed to remove temp file: %v\n", err)
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
