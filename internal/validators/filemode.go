// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package validators

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// FileModeValidator validates that a string represents a valid file mode.
type FileModeValidator struct{}

// Description returns a description of the validator.
func (v FileModeValidator) Description(_ context.Context) string {
	return "value must be a valid file mode (e.g., '0644', '0755', '644', '755')"
}

// MarkdownDescription returns a markdown description of the validator.
func (v FileModeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString performs the validation.
func (v FileModeValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value := request.ConfigValue.ValueString()
	if value == "" {
		return // Empty values are handled by other validators
	}

	// Parse the file mode
	mode, err := parseFileMode(value)
	if err != nil {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Invalid File Mode",
			fmt.Sprintf("Invalid file mode '%s': %s", value, err),
		)
		return
	}

	// Validate that the mode is reasonable (not too permissive or restrictive)
	if mode > 0777 {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Invalid File Mode",
			fmt.Sprintf("File mode '%s' is too permissive (maximum is 0777)", value),
		)
		return
	}

	if mode == 0 {
		response.Diagnostics.AddAttributeWarning(
			request.Path,
			"Restrictive File Mode",
			fmt.Sprintf("File mode '%s' grants no permissions to anyone", value),
		)
	}

	// Warn about potentially problematic permissions
	if mode&0002 != 0 { // World writable
		response.Diagnostics.AddAttributeWarning(
			request.Path,
			"World Writable File Mode",
			fmt.Sprintf("File mode '%s' is world-writable, which may be a security risk", value),
		)
	}

	if mode&0044 == 0 && mode&0004 == 0 { // Not readable by owner or group
		response.Diagnostics.AddAttributeWarning(
			request.Path,
			"Unreadable File Mode",
			fmt.Sprintf("File mode '%s' may make the file unreadable", value),
		)
	}
}

// parseFileMode parses a file mode string into os.FileMode.
func parseFileMode(modeStr string) (os.FileMode, error) {
	if modeStr == "" {
		return 0644, nil // default
	}

	// Check if it looks like an octal mode (only valid octal digits 0-7)
	octalPattern := regexp.MustCompile(`^0[0-7]{3}$|^[0-7]{3}$`)
	if octalPattern.MatchString(modeStr) {
		// Parse as octal
		var base int
		var modeValue string
		if modeStr[0] == '0' && len(modeStr) > 1 {
			// Explicit octal (e.g., "0644")
			base = 8
			modeValue = modeStr
		} else if len(modeStr) == 3 || len(modeStr) == 4 {
			// Implicit octal (e.g., "644")
			base = 8
			modeValue = modeStr
		} else {
			return 0, fmt.Errorf("ambiguous file mode format")
		}

		mode, err := strconv.ParseUint(modeValue, base, 32)
		if err != nil {
			return 0, fmt.Errorf("invalid octal file mode: %w", err)
		}
		if mode > 0777 {
			return 0, fmt.Errorf("file mode %s is too permissive (maximum is 0777)", modeValue)
		}
		return os.FileMode(mode), nil
	}

	// If it doesn't match octal pattern, reject it
	return 0, fmt.Errorf("invalid file mode format '%s' (expected octal format like '0644' or '644' with digits 0-7 only)", modeStr)
}

// ValidFileMode returns a validator which ensures that the provided value is a valid file mode.
func ValidFileMode() validator.String {
	return FileModeValidator{}
}

// SecureFileMode returns a validator which ensures that the provided file mode is secure.
func SecureFileMode() validator.String {
	return secureFileModeValidator{}
}

// secureFileModeValidator validates that a file mode is secure (not world-readable/writable).
type secureFileModeValidator struct{}

func (v secureFileModeValidator) Description(_ context.Context) string {
	return "value must be a secure file mode (not world-readable or world-writable)"
}

func (v secureFileModeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v secureFileModeValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value := request.ConfigValue.ValueString()
	if value == "" {
		return // Empty values are handled by other validators
	}

	// Parse the file mode
	mode, err := parseFileMode(value)
	if err != nil {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Invalid File Mode",
			fmt.Sprintf("Invalid file mode '%s': %s", value, err),
		)
		return
	}

	// Check for world-readable
	if mode&0004 != 0 {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Insecure File Mode",
			fmt.Sprintf("File mode '%s' is world-readable, which may be a security risk for sensitive files", value),
		)
		return
	}

	// Check for world-writable
	if mode&0002 != 0 {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Insecure File Mode",
			fmt.Sprintf("File mode '%s' is world-writable, which is a security risk", value),
		)
		return
	}

	// Check for group-readable for very sensitive files
	if mode&0040 != 0 {
		response.Diagnostics.AddAttributeWarning(
			request.Path,
			"Group-Readable File Mode",
			fmt.Sprintf("File mode '%s' is group-readable, consider using more restrictive permissions for sensitive files", value),
		)
	}
}

// DirectoryFileMode returns a validator for directory file modes.
func DirectoryFileMode() validator.String {
	return directoryFileModeValidator{}
}

// directoryFileModeValidator validates directory-specific file modes.
type directoryFileModeValidator struct{}

func (v directoryFileModeValidator) Description(_ context.Context) string {
	return "value must be a valid directory file mode with execute permissions"
}

func (v directoryFileModeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v directoryFileModeValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value := request.ConfigValue.ValueString()
	if value == "" {
		return // Empty values are handled by other validators
	}

	// Parse the file mode
	mode, err := parseFileMode(value)
	if err != nil {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Invalid Directory Mode",
			fmt.Sprintf("Invalid directory mode '%s': %s", value, err),
		)
		return
	}

	// Directories need execute permission to be accessible
	if mode&0100 == 0 { // Owner execute
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Invalid Directory Mode",
			fmt.Sprintf("Directory mode '%s' lacks owner execute permission, directory will not be accessible", value),
		)
		return
	}

	// Warn if directory is not readable by owner
	if mode&0400 == 0 { // Owner read
		response.Diagnostics.AddAttributeWarning(
			request.Path,
			"Directory Not Readable",
			fmt.Sprintf("Directory mode '%s' lacks owner read permission, directory contents may not be listable", value),
		)
	}

	// Common directory modes validation
	commonDirModes := []os.FileMode{0755, 0750, 0700, 0711}
	isCommon := false
	for _, common := range commonDirModes {
		if mode == common {
			isCommon = true
			break
		}
	}

	if !isCommon {
		response.Diagnostics.AddAttributeWarning(
			request.Path,
			"Unusual Directory Mode",
			fmt.Sprintf("Directory mode '%s' is unusual, common modes are 0755, 0750, 0700, or 0711", value),
		)
	}
}
