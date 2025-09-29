// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package utils

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ParseFileMode parses a file mode string like "0644" into os.FileMode.
func ParseFileMode(modeStr string) (os.FileMode, error) {
	if modeStr == "" {
		return 0600, nil // default (secure permissions)
	}

	// Handle octal mode strings
	if strings.HasPrefix(modeStr, "0") {
		mode, err := strconv.ParseUint(modeStr, 8, 32)
		if err != nil {
			return 0, err
		}
		return os.FileMode(mode), nil
	}

	// Handle decimal mode
	mode, err := strconv.ParseUint(modeStr, 10, 32)
	if err != nil {
		return 0, err
	}
	return os.FileMode(mode), nil
}

// FormatFileMode formats an os.FileMode as an octal string like "0644".
func FormatFileMode(mode os.FileMode) string {
	return "0" + strconv.FormatUint(uint64(mode.Perm()), 8)
}

// PathExists checks if a path exists.
func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsSymlink checks if a path is a symbolic link.
func IsSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

// CreateTempDotfilesRepo creates a temporary dotfiles repository for testing.
func CreateTempDotfilesRepo(tempDir string) (string, error) {
	repoDir := filepath.Join(tempDir, "dotfiles")
	err := os.MkdirAll(repoDir, 0755)
	if err != nil {
		return "", err
	}

	// Create basic dotfiles structure
	dirs := []string{
		"git",
		"fish",
		"ssh",
		"tools",
		"templates",
	}

	for _, dir := range dirs {
		err := os.MkdirAll(filepath.Join(repoDir, dir), 0755)
		if err != nil {
			return "", err
		}
	}

	// Create test files
	files := map[string]string{
		"git/gitconfig": `[user]
    name = Test User
    email = test@example.com
[core]
    editor = vim`,
		"fish/config.fish": `# Fish configuration
set -g fish_greeting ""
set -gx EDITOR vim`,
		"ssh/config": `Host github.com
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_ed25519`,
		"tools/cursor.json": `{
    "editor.fontSize": 14,
    "workbench.colorTheme": "Dark+ (default dark)"
}`,
	}

	for filePath, content := range files {
		fullPath := filepath.Join(repoDir, filePath)
		err := os.WriteFile(fullPath, []byte(content), 0600)
		if err != nil {
			return "", err
		}
	}

	return repoDir, nil
}

// GenerateTestID generates a unique test ID.
func GenerateTestID(prefix string) string {
	return prefix + "-" + strconv.Itoa(os.Getpid())
}

// CompareFileContent compares the content of two files.
func CompareFileContent(file1, file2 string) (bool, error) {
	content1, err := os.ReadFile(file1)
	if err != nil {
		return false, err
	}

	content2, err := os.ReadFile(file2)
	if err != nil {
		return false, err
	}

	return string(content1) == string(content2), nil
}
