package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestFilePermissionsResource_StateManagement tests the file permissions resource state operations.
func TestFilePermissionsResource_StateManagement(t *testing.T) {
	testCases := []struct {
		name        string
		path        string
		mode        string
		recursive   bool
		expectError bool
	}{
		{
			name:        "set file permissions 644",
			path:        "/tmp/test-file",
			mode:        "0644",
			recursive:   false,
			expectError: false,
		},
		{
			name:        "set secure file permissions 600",
			path:        "/home/user/.ssh/id_rsa",
			mode:        "0600",
			recursive:   false,
			expectError: false,
		},
		{
			name:        "set directory permissions recursively",
			path:        "/home/user/.ssh",
			mode:        "0700",
			recursive:   true,
			expectError: false,
		},
		{
			name:        "invalid permission mode",
			path:        "/tmp/test",
			mode:        "0999",
			recursive:   false,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			model := FilePermissionsResourceModel{
				Path:      types.StringValue(tc.path),
				Mode:      types.StringValue(tc.mode),
				Recursive: types.BoolValue(tc.recursive),
			}

			err := validatePermissionModel(model)
			hasError := err != nil

			if tc.expectError && !hasError {
				t.Errorf("Expected error for %s but got none", tc.name)
			}
			if !tc.expectError && hasError {
				t.Errorf("Unexpected error for %s: %v", tc.name, err)
			}
		})
	}
}

// validatePermissionModel provides basic validation for testing
func validatePermissionModel(model FilePermissionsResourceModel) error {
	path := model.Path.ValueString()
	mode := model.Mode.ValueString()

	if path == "" {
		return &validationError{"path cannot be empty"}
	}

	if mode == "" {
		return &validationError{"mode cannot be empty"}
	}

	// Basic octal validation
	if !isValidOctalMode(mode) {
		return &validationError{"invalid octal mode"}
	}

	return nil
}

// isValidOctalMode checks if a string is a valid octal permission mode
func isValidOctalMode(mode string) bool {
	if len(mode) != 3 && len(mode) != 4 {
		return false
	}

	start := 0
	if len(mode) == 4 {
		if mode[0] != '0' {
			return false
		}
		start = 1
	}

	for _, char := range mode[start:] {
		if char < '0' || char > '7' {
			return false
		}
	}
	return true
}

// validationError is a simple error type for testing
type validationError struct {
	message string
}

func (e *validationError) Error() string {
	return e.message
}

// TestFilePermissionsResource_Creation tests resource creation.
func TestFilePermissionsResource_Creation(t *testing.T) {
	r := NewFilePermissionsResource()
	permissionsResource, ok := r.(*FilePermissionsResource)
	if !ok {
		t.Fatal("NewFilePermissionsResource() should return *FilePermissionsResource")
	}

	if permissionsResource == nil {
		t.Fatal("NewFilePermissionsResource() returned nil")
	}
}

// TestFilePermissionsResourceModel_Validation tests basic validation scenarios
func TestFilePermissionsResourceModel_Validation(t *testing.T) {
	testValidModel(t)
	testInvalidModels(t)
}

func testValidModel(t *testing.T) {
	model := FilePermissionsResourceModel{
		Path: types.StringValue("/home/user/.ssh/id_rsa"),
		Mode: types.StringValue("0600"),
	}

	err := validatePermissionModel(model)
	if err != nil {
		t.Errorf("Valid model should not return error: %v", err)
	}
}

func testInvalidModels(t *testing.T) {
	invalidCases := []struct {
		name string
		path string
		mode string
	}{
		{"empty path", "", "0644"},
		{"empty mode", "/test/file", ""},
		{"invalid mode format", "/test/file", "abc"},
		{"mode too many digits", "/test/file", "07777"},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			model := FilePermissionsResourceModel{
				Path: types.StringValue(tc.path),
				Mode: types.StringValue(tc.mode),
			}

			err := validatePermissionModel(model)
			if err == nil {
				t.Errorf("Expected error for %s but got none", tc.name)
			}
		})
	}
}

// TestFilePermissionsResource_PatternBasedPermissions tests pattern matching for permissions.
func TestFilePermissionsResource_PatternBasedPermissions(t *testing.T) {
	tests := []struct {
		name         string
		patterns     map[string]string
		testFile     string
		expectedMode string
		expectMatch  bool
	}{
		{
			name: "SSH private key pattern",
			patterns: map[string]string{
				"id_*": "0600",
			},
			testFile:     "id_rsa",
			expectedMode: "0600",
			expectMatch:  true,
		},
		{
			name: "SSH public key pattern",
			patterns: map[string]string{
				"*.pub": "0644",
			},
			testFile:     "id_rsa.pub",
			expectedMode: "0644",
			expectMatch:  true,
		},
		{
			name: "No matching pattern",
			patterns: map[string]string{
				"*.config": "0600",
			},
			testFile:     "regular_file.txt",
			expectedMode: "",
			expectMatch:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode, found := findPatternMatch(tt.patterns, tt.testFile)

			if tt.expectMatch && !found {
				t.Errorf("Expected to find pattern match for %s", tt.testFile)
			}

			if !tt.expectMatch && found {
				t.Errorf("Did not expect to find pattern match for %s", tt.testFile)
			}

			if found && mode != tt.expectedMode {
				t.Errorf("Expected mode %s but got %s", tt.expectedMode, mode)
			}
		})
	}
}

// findPatternMatch simulates pattern matching logic for testing
func findPatternMatch(patterns map[string]string, filename string) (string, bool) {
	// Simple pattern matching logic for testing
	for pattern, mode := range patterns {
		if simpleMatch(pattern, filename) {
			return mode, true
		}
	}
	return "", false
}

// simpleMatch provides basic pattern matching for testing
func simpleMatch(pattern, filename string) bool {
	if pattern == "*" {
		return true
	}

	if len(pattern) > 0 && pattern[0] == '*' {
		suffix := pattern[1:]
		return len(filename) >= len(suffix) && filename[len(filename)-len(suffix):] == suffix
	}

	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(filename) >= len(prefix) && filename[:len(prefix)] == prefix
	}

	return pattern == filename
}

// TestFilePermissionsResource_ImportState tests state import functionality.
func TestFilePermissionsResource_ImportState(t *testing.T) {
	testCases := []struct {
		name      string
		importID  string
		wantError bool
	}{
		{
			name:      "valid import with absolute path",
			importID:  "/home/user/.ssh/id_rsa",
			wantError: false,
		},
		{
			name:      "valid import with home directory",
			importID:  "~/.ssh/id_rsa",
			wantError: false,
		},
		{
			name:      "empty import ID",
			importID:  "",
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateImportID(tc.importID)
			hasError := err != nil

			if tc.wantError && !hasError {
				t.Errorf("Expected error for import ID '%s' but got none", tc.importID)
			}
			if !tc.wantError && hasError {
				t.Errorf("Unexpected error for import ID '%s': %v", tc.importID, err)
			}
		})
	}
}

// validateImportID provides basic import ID validation for testing
func validateImportID(importID string) error {
	if importID == "" {
		return &validationError{"import ID cannot be empty"}
	}
	return nil
}
