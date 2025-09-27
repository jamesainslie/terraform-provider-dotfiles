package migration

import (
	"bytes"
	"strings"
	"testing"
)

func TestMigrateConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "migrate symlink strategy to dotfiles_symlink",
			input: `resource "dotfiles_file" "test_symlink" {
  repository  = dotfiles_repository.main.id
  name        = "test-config"
  source_path = "config.json"
  target_path = "~/.config/app/config.json"
  strategy    = "symlink"
  file_mode   = "0644"
}`,
			expected: `# MIGRATED: dotfiles_file with strategy=symlink → dotfiles_symlink
resource "dotfiles_symlink" "test_symlink" {
  repository  = dotfiles_repository.main.id
  name        = "test-config"
  source_path = "config.json"
  target_path = "~/.config/app/config.json"
  file_mode   = "0644"
}`,
		},
		{
			name: "migrate copy strategy to dotfiles_file without strategy",
			input: `resource "dotfiles_file" "test_copy" {
  repository  = dotfiles_repository.main.id
  name        = "test-config"
  source_path = "config.json"
  target_path = "~/.config/app/config.json"
  strategy    = "copy"
  file_mode   = "0644"
}`,
			expected: `resource "dotfiles_file" "test_copy" {
  repository  = dotfiles_repository.main.id
  name        = "test-config"
  source_path = "config.json"
  target_path = "~/.config/app/config.json"
  file_mode   = "0644"
}`,
		},
		{
			name: "migrate template strategy to is_template=true",
			input: `resource "dotfiles_file" "test_template" {
  repository  = dotfiles_repository.main.id
  name        = "test-config"
  source_path = "config.json"
  target_path = "~/.config/app/config.json"
  strategy    = "template"
  file_mode   = "0644"
}`,
			expected: `# MIGRATED: strategy=template → is_template=true
resource "dotfiles_file" "test_template" {
  repository  = dotfiles_repository.main.id
  name        = "test-config"
  source_path = "config.json"
  target_path = "~/.config/app/config.json"
  is_template = true
  file_mode   = "0644"
}`,
		},
		{
			name: "no migration needed for file without strategy",
			input: `resource "dotfiles_file" "test_no_strategy" {
  repository  = dotfiles_repository.main.id
  name        = "test-config"
  source_path = "config.json"
  target_path = "~/.config/app/config.json"
  file_mode   = "0644"
}`,
			expected: `resource "dotfiles_file" "test_no_strategy" {
  repository  = dotfiles_repository.main.id
  name        = "test-config"
  source_path = "config.json"
  target_path = "~/.config/app/config.json"
  file_mode   = "0644"
}`,
		},
		{
			name: "preserve non-dotfiles_file resources",
			input: `resource "dotfiles_symlink" "existing_symlink" {
  repository  = dotfiles_repository.main.id
  name        = "existing-config"
  source_path = "config.json"
  target_path = "~/.config/app/config.json"
}`,
			expected: `resource "dotfiles_symlink" "existing_symlink" {
  repository  = dotfiles_repository.main.id
  name        = "existing-config"
  source_path = "config.json"
  target_path = "~/.config/app/config.json"
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.NewReader(tt.input)
			var output bytes.Buffer

			err := migrateConfig(input, &output)
			if err != nil {
				t.Fatalf("migrateConfig failed: %v", err)
			}

			result := strings.TrimSpace(output.String())
			expected := strings.TrimSpace(tt.expected)

			if result != expected {
				t.Errorf("Migration result mismatch\nExpected:\n%s\nGot:\n%s", expected, result)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedCount int
		expectedIssue string
	}{
		{
			name: "detect symlink strategy issue",
			input: `resource "dotfiles_file" "test" {
  strategy = "symlink"
}`,
			expectedCount: 1,
			expectedIssue: "uses deprecated strategy field with value 'symlink'",
		},
		{
			name: "detect copy strategy issue",
			input: `resource "dotfiles_file" "test" {
  strategy = "copy"
}`,
			expectedCount: 1,
			expectedIssue: "uses deprecated strategy field with value 'copy'",
		},
		{
			name: "detect template strategy issue",
			input: `resource "dotfiles_file" "test" {
  strategy = "template"
}`,
			expectedCount: 1,
			expectedIssue: "uses deprecated strategy field with value 'template'",
		},
		{
			name: "no issues for file without strategy",
			input: `resource "dotfiles_file" "test" {
  source_path = "test"
}`,
			expectedCount: 0,
		},
		{
			name: "no issues for non-file resources",
			input: `resource "dotfiles_symlink" "test" {
  strategy = "symlink"
}`,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.NewReader(tt.input)
			issues, err := validateConfig(input)
			if err != nil {
				t.Fatalf("validateConfig failed: %v", err)
			}

			if len(issues) != tt.expectedCount {
				t.Errorf("Expected %d issues, got %d", tt.expectedCount, len(issues))
			}

			if tt.expectedCount > 0 && !strings.Contains(issues[0].Description, tt.expectedIssue) {
				t.Errorf("Expected issue containing '%s', got '%s'", tt.expectedIssue, issues[0].Description)
			}
		})
	}
}

func TestConvertToSymlinkResource(t *testing.T) {
	ctx := &ResourceContext{
		ResourceType: "dotfiles_file",
		ResourceName: "test",
		Strategy:     "symlink",
		Lines: []string{
			`resource "dotfiles_file" "test" {`,
			`  source_path = "config"`,
			`  strategy    = "symlink"`,
			`}`,
		},
	}

	result, err := convertToSymlinkResource(ctx)
	if err != nil {
		t.Fatalf("convertToSymlinkResource failed: %v", err)
	}

	expected := []string{
		`# MIGRATED: dotfiles_file with strategy=symlink → dotfiles_symlink`,
		`resource "dotfiles_symlink" "test" {`,
		`  source_path = "config"`,
		`}`,
	}

	if len(result) != len(expected) {
		t.Fatalf("Expected %d lines, got %d", len(expected), len(result))
	}

	for i, line := range result {
		if line != expected[i] {
			t.Errorf("Line %d: expected '%s', got '%s'", i, expected[i], line)
		}
	}
}

func TestRemoveStrategyField(t *testing.T) {
	lines := []string{
		`resource "dotfiles_file" "test" {`,
		`  source_path = "config"`,
		`  strategy    = "copy"`,
		`  file_mode   = "0644"`,
		`}`,
	}

	result := removeStrategyField(lines)
	expected := []string{
		`resource "dotfiles_file" "test" {`,
		`  source_path = "config"`,
		`  file_mode   = "0644"`,
		`}`,
	}

	if len(result) != len(expected) {
		t.Fatalf("Expected %d lines, got %d", len(expected), len(result))
	}

	for i, line := range result {
		if line != expected[i] {
			t.Errorf("Line %d: expected '%s', got '%s'", i, expected[i], line)
		}
	}
}

func TestRegexPatterns(t *testing.T) {
	tests := []struct {
		name    string
		regex   string
		input   string
		matches bool
	}{
		{
			name:    "file resource regex matches",
			input:   `resource "dotfiles_file" "test" {`,
			matches: true,
		},
		{
			name:    "file resource regex doesn't match other resources",
			input:   `resource "dotfiles_symlink" "test" {`,
			matches: false,
		},
		{
			name:    "strategy field regex matches",
			input:   `  strategy = "symlink"`,
			matches: true,
		},
		{
			name:    "strategy field regex matches with different quotes",
			input:   `  strategy    =    "copy"`,
			matches: true,
		},
		{
			name:    "strategy field regex doesn't match other fields",
			input:   `  source_path = "test"`,
			matches: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var matched bool
			if strings.Contains(tt.name, "file resource") {
				matched = fileResourceRegex.MatchString(tt.input)
			} else {
				matched = strategyFieldRegex.MatchString(tt.input)
			}

			if matched != tt.matches {
				t.Errorf("Expected match=%v, got match=%v for input: %s", tt.matches, matched, tt.input)
			}
		})
	}
}
