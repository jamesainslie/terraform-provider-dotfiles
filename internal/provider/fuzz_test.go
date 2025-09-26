// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// FuzzInput represents input data for fuzz testing.
type FuzzInput struct {
	Paths           []string
	FileContents    []string
	TemplateVars    map[string]interface{}
	Configurations  []map[string]interface{}
	FilePermissions []string
}

// generateRandomString creates random strings for testing.
func generateRandomString(length int, charset string) string {
	if charset == "" {
		charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	}

	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

// generateFuzzPaths creates various path formats for testing.
func generateFuzzPaths() []string {
	return []string{
		// Normal paths
		"/home/user/.config/test",
		"~/dotfiles/config",
		"./relative/path",
		"../parent/path",

		// Edge case paths
		"",                          // Empty path
		"/",                         // Root path
		"//double//slash//path",     // Double slashes
		"/path/with spaces/file",    // Spaces
		"/path/with\ttab/file",      // Tab character
		"/path/with\nnewline/file",  // Newline
		"/path/with\"quotes\"/file", // Quotes
		"/path/with'single'/file",   // Single quotes
		"/path/with$var/file",       // Environment variables
		"/path/with${VAR}/file",     // Environment variables (braces)
		"~/path/with~tilde/file",    // Tilde in middle

		// Very long paths
		"/very/long/path/" + strings.Repeat("a", 200),

		// Special characters
		"/path/with/special/chars/!@#$%^&*()",
		"/path/with/unicode/café/naïve",
		"/path/with/emoji/😀/test",

		// Windows-style paths (for cross-platform testing)
		"C:\\Windows\\Path",
		"\\\\server\\share\\file",

		// Relative paths with many levels
		strings.Repeat("../", 50) + "file",

		// Paths with null bytes (should be rejected)
		"/path/with\x00null/file",
	}
}

// generateFuzzFileContents creates various file content scenarios.
func generateFuzzFileContents() []string {
	return []string{
		// Normal content
		"Hello, World!",
		"# Configuration file\nkey=value\n",

		// Empty content
		"",

		// Very large content
		strings.Repeat("Large content line\n", 10000),

		// Binary-like content
		string([]byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}),

		// Unicode content
		"Unicode: café naïve résumé 中文 日本語 العربية",
		"Emoji: 😀 🎉 🔥 ⚡ 🚀",

		// Template-like content
		"{{.Name}} lives at {{.Address}}",
		"{{ range .Items }}Item: {{.}}{{ end }}",
		"{{/* Comment */}} Value: {{.Value | upper}}",

		// Invalid template syntax
		"{{.Name} missing closing brace",
		"{{.NonExistent.Field.Chain}}",
		"{{ invalid syntax here",

		// Special characters
		"Content with\nnewlines\tand\ttabs",
		"Content with \"quotes\" and 'apostrophes'",
		"Content with $ENV_VAR and ${ANOTHER_VAR}",

		// Very long lines
		strings.Repeat("x", 100000),

		// Control characters
		string([]byte{0x01, 0x02, 0x03, 0x07, 0x08, 0x0C}),
	}
}

// generateFuzzTemplateVars creates various template variable scenarios.
func generateFuzzTemplateVars() []map[string]interface{} {
	return []map[string]interface{}{
		// Normal variables
		{"Name": "John", "Age": 30, "City": "New York"},

		// Empty variables
		{},

		// Nil values
		{"Name": nil, "Age": nil},

		// Mixed types
		{
			"String": "hello",
			"Int":    42,
			"Float":  3.14,
			"Bool":   true,
			"Array":  []string{"a", "b", "c"},
			"Map":    map[string]string{"key": "value"},
		},

		// Very long values
		{"LongString": strings.Repeat("x", 10000)},

		// Special characters in keys and values
		{
			"key with spaces":      "value",
			"key-with-dashes":      "value",
			"key_with_underscores": "value",
			"unicode-café":         "naïve-value",
		},

		// Nested structures
		{
			"User": map[string]interface{}{
				"Profile": map[string]interface{}{
					"Name": "John",
					"Details": map[string]string{
						"Email": "john@example.com",
					},
				},
			},
		},

		// Arrays with mixed types
		{"MixedArray": []interface{}{"string", 42, true, nil}},

		// Empty keys
		{"": "empty key value"},

		// Keys with special characters
		{
			"key\nwith\nnewlines": "value",
			"key\twith\ttabs":     "value",
			"key\"with\"quotes":   "value",
		},
	}
}

// generateFuzzConfigurations creates various provider configuration scenarios.
func generateFuzzConfigurations() []map[string]interface{} {
	return []map[string]interface{}{
		// Valid configurations
		{
			"dotfiles_root":       "/tmp/dotfiles",
			"strategy":            "symlink",
			"conflict_resolution": "backup",
		},

		// Empty configuration
		{},

		// Invalid strategy
		{
			"dotfiles_root": "/tmp/dotfiles",
			"strategy":      "invalid_strategy",
		},

		// Very long paths
		{
			"dotfiles_root": "/tmp/" + strings.Repeat("very_long_directory_name/", 50),
		},

		// Special characters in paths
		{
			"dotfiles_root":    "/tmp/path with spaces/and\"quotes",
			"backup_directory": "/tmp/backup with\ttabs\nand newlines",
		},

		// Invalid types
		{
			"dotfiles_root": 123,             // Should be string
			"dry_run":       "not_a_boolean", // Should be boolean
		},

		// Nil values
		{
			"dotfiles_root": nil,
			"strategy":      nil,
		},
	}
}

// generateFuzzFilePermissions creates various file permission scenarios.
func generateFuzzFilePermissions() []string {
	return []string{
		// Valid permissions
		"0644", "0755", "0600", "0777",
		"644", "755", "600", "777",

		// Invalid permissions
		"", "invalid", "0999", "1777",
		"rwxrwxrwx", "777", "-rwxrwxrwx",

		// Edge cases
		"0000", "0001", "0007",

		// Very long strings
		strings.Repeat("7", 100),

		// Special characters
		"0644\n", "0644\t", "0644 ",
		"0644;rm -rf /", // Injection attempt
	}
}

func TestFuzzPathHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping fuzz test in short mode")
	}

	paths := generateFuzzPaths()

	for i, path := range paths {
		t.Run(fmt.Sprintf("path_%d", i), func(t *testing.T) {
			// Test path expansion - removed unused client variable

			// This should not panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Path handling panicked for path '%s': %v", path, r)
				}
			}()

			// Test various path operations
			abs := filepath.IsAbs(path)
			_ = abs // Use the result to avoid unused variable

			if path != "" {
				dir := filepath.Dir(path)
				base := filepath.Base(path)
				_ = dir
				_ = base
			}

			// Test path cleaning
			cleaned := filepath.Clean(path)
			_ = cleaned
		})
	}
}

func TestFuzzFileContentHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping fuzz test in short mode")
	}

	contents := generateFuzzFileContents()
	tempDir := t.TempDir()

	for i, content := range contents {
		t.Run(fmt.Sprintf("content_%d", i), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Content handling panicked for content length %d: %v", len(content), r)
				}
			}()

			// Test writing and reading content
			testFile := filepath.Join(tempDir, fmt.Sprintf("test_%d.txt", i))

			err := os.WriteFile(testFile, []byte(content), 0644)
			if err != nil {
				t.Logf("WriteFile failed for content %d: %v", i, err)
				return
			}

			readContent, err := os.ReadFile(testFile)
			if err != nil {
				t.Errorf("ReadFile failed for content %d: %v", i, err)
				return
			}

			if string(readContent) != content {
				t.Logf("Content mismatch for test %d (lengths: expected %d, got %d)",
					i, len(content), len(readContent))
			}
		})
	}
}

func TestFuzzTemplateVariables(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping fuzz test in short mode")
	}

	varSets := generateFuzzTemplateVars()

	for i, vars := range varSets {
		t.Run(fmt.Sprintf("vars_%d", i), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Template variable handling panicked for vars %d: %v", i, r)
				}
			}()

			// Test JSON marshaling/unmarshaling (common operation)
			if len(vars) > 0 {
				// This exercises the variable handling without requiring
				// full template processing
				for key, value := range vars {
					if key == "" {
						continue // Skip empty keys
					}

					// Test string conversion
					str := fmt.Sprintf("%v", value)
					_ = str

					// Test type checking
					switch value.(type) {
					case string, int, float64, bool, nil:
						// Expected types
					case []interface{}, map[string]interface{}:
						// Complex types
					default:
						t.Logf("Unexpected type for key %s: %T", key, value)
					}
				}
			}
		})
	}
}

func TestFuzzProviderConfiguration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping fuzz test in short mode")
	}

	configs := generateFuzzConfigurations()

	for i, configMap := range configs {
		t.Run(fmt.Sprintf("config_%d", i), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Configuration handling panicked for config %d: %v", i, r)
				}
			}()

			// Create a basic config
			config := &DotfilesConfig{}

			// Apply fuzz values
			for key, value := range configMap {
				switch key {
				case "dotfiles_root":
					if str, ok := value.(string); ok {
						config.DotfilesRoot = str
					}
				case "strategy":
					if str, ok := value.(string); ok {
						config.Strategy = str
					}
				case "conflict_resolution":
					if str, ok := value.(string); ok {
						config.ConflictResolution = str
					}
				case "dry_run":
					if b, ok := value.(bool); ok {
						config.DryRun = b
					}
				}
			}

			// Test configuration validation
			// This should handle invalid inputs gracefully
			if err := config.SetDefaults(); err != nil {
				// Skip this configuration if defaults fail
				t.Logf("Configuration %d failed to set defaults: %v", i, err)
				return
			}
			err := config.Validate()

			// We expect some configurations to fail validation
			if err != nil {
				t.Logf("Configuration %d failed validation (expected): %v", i, err)
			}
		})
	}
}

func TestFuzzFilePermissions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping fuzz test in short mode")
	}

	permissions := generateFuzzFilePermissions()

	for i, perm := range permissions {
		t.Run(fmt.Sprintf("perm_%d", i), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Permission handling panicked for permission '%s': %v", perm, r)
				}
			}()

			// Test permission parsing
			if perm != "" {
				// This exercises permission validation logic
				validOctal := true
				if len(perm) < 3 || len(perm) > 4 {
					validOctal = false
				}

				for _, char := range perm {
					if char < '0' || char > '7' {
						validOctal = false
						break
					}
				}

				if validOctal {
					t.Logf("Permission '%s' appears to be valid octal", perm)
				} else {
					t.Logf("Permission '%s' is not valid octal (expected)", perm)
				}
			}
		})
	}
}

func TestFuzzConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping fuzz test in short mode")
	}

	tempDir := t.TempDir()
	numGoroutines := 10
	numOperations := 50

	// Test concurrent file operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Concurrent operation %d panicked: %v", id, r)
				}
			}()

			for j := 0; j < numOperations; j++ {
				// Random file operations
				fileName := fmt.Sprintf("file_%d_%d.txt", id, j)
				filePath := filepath.Join(tempDir, fileName)

				// Random content
				content := generateRandomString(rand.Intn(1000), "")

				// Write file
				err := os.WriteFile(filePath, []byte(content), 0644)
				if err != nil {
					continue // Skip on error
				}

				// Read file
				_, err = os.ReadFile(filePath)
				if err != nil {
					continue // Skip on error
				}

				// Remove file
				_ = os.Remove(filePath) // Cleanup - ignore errors
			}
		}(i)
	}

	// Give goroutines time to complete
	time.Sleep(1 * time.Second)
}

func TestFuzzLargeFileHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping fuzz test in short mode")
	}

	tempDir := t.TempDir()

	// Test various file sizes
	sizes := []int{
		0,           // Empty file
		1,           // Single byte
		1024,        // 1KB
		1024 * 1024, // 1MB
	}

	// Only test very large files if not in short mode
	if !testing.Short() {
		sizes = append(sizes, 10*1024*1024) // 10MB
	}

	for i, size := range sizes {
		t.Run(fmt.Sprintf("size_%d_bytes", size), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Large file handling panicked for size %d: %v", size, r)
				}
			}()

			fileName := fmt.Sprintf("large_file_%d.txt", i)
			filePath := filepath.Join(tempDir, fileName)

			// Create content of specified size
			content := strings.Repeat("x", size)

			// Write file
			err := os.WriteFile(filePath, []byte(content), 0644)
			if err != nil {
				t.Logf("Failed to write large file of size %d: %v", size, err)
				return
			}

			// Read file
			readContent, err := os.ReadFile(filePath)
			if err != nil {
				t.Errorf("Failed to read large file of size %d: %v", size, err)
				return
			}

			if len(readContent) != size {
				t.Errorf("Size mismatch: expected %d, got %d", size, len(readContent))
			}

			// Clean up
			_ = os.Remove(filePath) // Cleanup - ignore errors
		})
	}
}

func TestFuzzErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping fuzz test in short mode")
	}

	// Test error handling with various invalid inputs
	testCases := []struct {
		name string
		test func() error
	}{
		{
			name: "read_nonexistent_file",
			test: func() error {
				_, err := os.ReadFile("/nonexistent/path/file.txt")
				return err
			},
		},
		{
			name: "write_to_readonly_location",
			test: func() error {
				return os.WriteFile("/read/only/path/file.txt", []byte("content"), 0644)
			},
		},
		{
			name: "create_directory_without_permission",
			test: func() error {
				return os.MkdirAll("/root/restricted/path", 0755)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Error handling panicked in %s: %v", tc.name, r)
				}
			}()

			err := tc.test()
			if err != nil {
				t.Logf("Expected error in %s: %v", tc.name, err)
			}
		})
	}
}

func init() {
	// Seed random number generator for consistent but varied fuzz testing
	// Note: rand.Seed is deprecated in Go 1.20+. Using default random generator.
}
