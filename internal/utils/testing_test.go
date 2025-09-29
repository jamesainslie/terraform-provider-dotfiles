// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFileMode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected os.FileMode
		wantErr  bool
	}{
		{
			name:     "Empty string defaults to 0600",
			input:    "",
			expected: 0600,
			wantErr:  false,
		},
		{
			name:     "Octal format 0600",
			input:    "0600",
			expected: 0600,
			wantErr:  false,
		},
		{
			name:     "Octal format 0755",
			input:    "0755",
			expected: 0755,
			wantErr:  false,
		},
		{
			name:     "Decimal format 644",
			input:    "644",
			expected: 644,
			wantErr:  false,
		},
		{
			name:     "Invalid format",
			input:    "invalid",
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseFileMode(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseFileMode(%q) expected error, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseFileMode(%q) unexpected error: %v", tt.input, err)
				return
			}

			if result != tt.expected {
				t.Errorf("ParseFileMode(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatFileMode(t *testing.T) {
	tests := []struct {
		name     string
		input    os.FileMode
		expected string
	}{
		{
			name:     "Mode 0600",
			input:    0600,
			expected: "0600",
		},
		{
			name:     "Mode 0755",
			input:    0755,
			expected: "0755",
		},
		{
			name:     "Mode 0600",
			input:    0600,
			expected: "0600",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatFileMode(tt.input)
			if result != tt.expected {
				t.Errorf("FormatFileMode(%v) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPathExists(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test existing file
	if !PathExists(testFile) {
		t.Error("PathExists should return true for existing file")
	}

	// Test existing directory
	if !PathExists(tempDir) {
		t.Error("PathExists should return true for existing directory")
	}

	// Test non-existent path
	nonExistentPath := filepath.Join(tempDir, "nonexistent.txt")
	if PathExists(nonExistentPath) {
		t.Error("PathExists should return false for non-existent path")
	}
}

func TestIsSymlink(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test regular file
	if IsSymlink(testFile) {
		t.Error("IsSymlink should return false for regular file")
	}

	// Test directory
	if IsSymlink(tempDir) {
		t.Error("IsSymlink should return false for directory")
	}

	// Test non-existent file
	nonExistentPath := filepath.Join(tempDir, "nonexistent.txt")
	if IsSymlink(nonExistentPath) {
		t.Error("IsSymlink should return false for non-existent file")
	}

	// Create symlink (if supported)
	symlinkPath := filepath.Join(tempDir, "symlink.txt")
	err = os.Symlink(testFile, symlinkPath)
	if err != nil {
		t.Logf("Symlink creation failed (may be expected on this platform): %v", err)
		return
	}

	// Test symlink
	if !IsSymlink(symlinkPath) {
		t.Error("IsSymlink should return true for symbolic link")
	}
}

func TestCreateTempDotfilesRepo(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Create test dotfiles repo
	repoPath, err := CreateTempDotfilesRepo(tempDir)
	if err != nil {
		t.Fatalf("CreateTempDotfilesRepo failed: %v", err)
	}

	// Verify repository was created
	if !PathExists(repoPath) {
		t.Error("Repository directory was not created")
	}

	// Verify directories were created
	expectedDirs := []string{"git", "fish", "ssh", "tools"}
	for _, dir := range expectedDirs {
		dirPath := filepath.Join(repoPath, dir)
		if !PathExists(dirPath) {
			t.Errorf("Directory %s was not created", dir)
		}
	}

	// Verify files were created
	expectedFiles := map[string]string{
		"git/gitconfig":     "Test User",
		"fish/config.fish":  "Fish configuration",
		"ssh/config":        "Host github.com",
		"tools/cursor.json": "editor.fontSize",
	}

	for filePath, expectedContent := range expectedFiles {
		fullPath := filepath.Join(repoPath, filePath)
		if !PathExists(fullPath) {
			t.Errorf("File %s was not created", filePath)
			continue
		}

		content, err := os.ReadFile(fullPath)
		if err != nil {
			t.Errorf("Failed to read file %s: %v", filePath, err)
			continue
		}

		if !strings.Contains(string(content), expectedContent) {
			t.Errorf("File %s does not contain expected content '%s'", filePath, expectedContent)
		}
	}
}

func TestGenerateTestID(t *testing.T) {
	prefix := "test"
	id := GenerateTestID(prefix)

	if !strings.HasPrefix(id, prefix) {
		t.Errorf("Generated ID %q should have prefix %q", id, prefix)
	}

	if len(id) <= len(prefix) {
		t.Error("Generated ID should be longer than prefix")
	}

	// Test with different prefixes
	id2 := GenerateTestID("other")
	if !strings.HasPrefix(id2, "other") {
		t.Error("Generated ID should have correct prefix")
	}

	// Test that format is consistent (contains the process ID)
	if !strings.Contains(id, "-") {
		t.Error("Generated ID should contain separator")
	}
}

func TestCompareFileContent(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Create test files
	file1 := filepath.Join(tempDir, "file1.txt")
	file2 := filepath.Join(tempDir, "file2.txt")
	file3 := filepath.Join(tempDir, "file3.txt")

	content1 := "identical content"
	content2 := "identical content"
	content3 := "different content"

	err := os.WriteFile(file1, []byte(content1), 0600)
	if err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}

	err = os.WriteFile(file2, []byte(content2), 0600)
	if err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	err = os.WriteFile(file3, []byte(content3), 0600)
	if err != nil {
		t.Fatalf("Failed to create file3: %v", err)
	}

	// Test identical files
	same, err := CompareFileContent(file1, file2)
	if err != nil {
		t.Errorf("CompareFileContent failed: %v", err)
	}
	if !same {
		t.Error("Identical files should be reported as same")
	}

	// Test different files
	same, err = CompareFileContent(file1, file3)
	if err != nil {
		t.Errorf("CompareFileContent failed: %v", err)
	}
	if same {
		t.Error("Different files should be reported as different")
	}

	// Test non-existent file
	nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")
	_, err = CompareFileContent(file1, nonExistentFile)
	if err == nil {
		t.Error("CompareFileContent should error with non-existent file")
	}
}
