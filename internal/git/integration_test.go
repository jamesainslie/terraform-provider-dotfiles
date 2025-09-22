// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
)

func TestGitManagerIntegration(t *testing.T) {
	// Create temporary directory for test repositories
	tempDir := t.TempDir()

	t.Run("LocalRepositoryOperations", func(t *testing.T) {
		// Create a local Git repository for testing
		repoDir := filepath.Join(tempDir, "test-repo")
		_, err := createTestRepository(repoDir)
		if err != nil {
			t.Fatalf("Failed to create test repository: %v", err)
		}

		// Test ValidateRepository
		manager, err := NewGitManager(nil)
		if err != nil {
			t.Fatalf("Failed to create Git manager: %v", err)
		}

		err = manager.ValidateRepository(repoDir)
		if err != nil {
			t.Errorf("ValidateRepository failed for valid repository: %v", err)
		}

		// Test ValidateRepository with non-repository
		nonRepoDir := filepath.Join(tempDir, "not-a-repo")
		err = os.MkdirAll(nonRepoDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create non-repo directory: %v", err)
		}

		err = manager.ValidateRepository(nonRepoDir)
		if err == nil {
			t.Error("ValidateRepository should fail for non-repository directory")
		}

		// Note: GetRepositoryInfo requires a repository with proper remotes
		// This will be tested in full integration tests when we have actual Git operations
		t.Log("GetRepositoryInfo will be tested with complete Git operations in integration tests")
	})

	t.Run("GetLocalCachePath", func(t *testing.T) {
		cacheRoot := filepath.Join(tempDir, "cache")

		testCases := []struct {
			name     string
			repoURL  string
			expected string
		}{
			{
				name:     "GitHub repository",
				repoURL:  "https://github.com/user/repo.git",
				expected: filepath.Join(cacheRoot, "github.com", "user", "repo"),
			},
			{
				name:     "GitLab repository",
				repoURL:  "https://gitlab.com/user/repo.git",
				expected: filepath.Join(cacheRoot, "gitlab.com", "user", "repo"),
			},
			{
				name:     "Enterprise GitHub",
				repoURL:  "https://github.enterprise.com/company/repo.git",
				expected: filepath.Join(cacheRoot, "github.enterprise.com", "company", "repo"),
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := GetLocalCachePath(cacheRoot, tc.repoURL)
				if err != nil {
					t.Errorf("GetLocalCachePath failed: %v", err)
				}

				if result != tc.expected {
					t.Errorf("Expected cache path %s, got %s", tc.expected, result)
				}
			})
		}
	})

	t.Run("AuthConfigValidation", func(t *testing.T) {
		testCases := []struct {
			name       string
			authConfig *AuthConfig
			expectAuth bool
			expectErr  bool
		}{
			{
				name:       "No authentication",
				authConfig: &AuthConfig{},
				expectAuth: false,
				expectErr:  false,
			},
			{
				name: "PAT authentication",
				authConfig: &AuthConfig{
					PersonalAccessToken: "ghp_test_token",
					Username:            "testuser",
				},
				expectAuth: true,
				expectErr:  false,
			},
			{
				name: "PAT without username",
				authConfig: &AuthConfig{
					PersonalAccessToken: "ghp_test_token",
				},
				expectAuth: true,
				expectErr:  false,
			},
			{
				name: "SSH with non-existent key",
				authConfig: &AuthConfig{
					SSHPrivateKeyPath: "/nonexistent/key",
				},
				expectAuth: false,
				expectErr:  true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				manager, err := NewGitManager(tc.authConfig)

				if tc.expectErr {
					if err == nil {
						t.Error("Expected error but got none")
					}
					return
				}

				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}

				hasAuth := manager.auth != nil
				if hasAuth != tc.expectAuth {
					t.Errorf("Expected auth=%v, got auth=%v", tc.expectAuth, hasAuth)
				}
			})
		}
	})
}

// createTestRepository creates a local Git repository for testing.
func createTestRepository(repoPath string) (*git.Repository, error) {
	// Initialize repository
	repo, err := git.PlainInit(repoPath, false)
	if err != nil {
		return nil, err
	}

	// Create a test file
	testFile := filepath.Join(repoPath, "README.md")
	err = os.WriteFile(testFile, []byte("# Test Repository\n\nThis is a test repository for integration testing."), 0644)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

func TestGitURLValidation(t *testing.T) {
	// Test comprehensive URL validation
	testCases := []struct {
		name    string
		url     string
		isValid bool
	}{
		// Valid Git URLs
		{"GitHub HTTPS", "https://github.com/user/repo.git", true},
		{"GitHub HTTPS no .git", "https://github.com/user/repo", true},
		{"GitHub SSH", "git@github.com:user/repo.git", true},
		{"GitHub SSH no .git", "git@github.com:user/repo", true},
		{"GitHub shorthand", "github.com/user/repo", true},
		{"GitHub shorthand colon", "github.com:user/repo", true},
		{"GitLab HTTPS", "https://gitlab.com/user/repo.git", true},
		{"GitLab SSH", "git@gitlab.com:user/repo.git", true},
		{"Generic Git HTTPS", "https://git.example.com/user/repo.git", true},
		{"Generic Git SSH", "git@git.example.com:user/repo.git", true},
		{"SSH protocol", "ssh://git@example.com/user/repo.git", true},

		// Invalid URLs (should be treated as local paths)
		{"Local absolute path", "/path/to/local/repo", false},
		{"Local relative path", "./local/repo", false},
		{"Home path", "~/dotfiles", false},
		{"Plain HTTPS", "https://example.com/page", false},
		{"Invalid format", "not-a-url", false},
		{"Empty string", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsGitURL(tc.url)
			if result != tc.isValid {
				t.Errorf("IsGitURL(%q) = %v, expected %v", tc.url, result, tc.isValid)
			}
		})
	}
}

func TestGitURLNormalization(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "GitHub shorthand slash",
			input:    "github.com/user/repo",
			expected: "https://github.com/user/repo.git",
		},
		{
			name:     "GitHub shorthand colon",
			input:    "github.com:user/repo",
			expected: "https://github.com/user/repo.git",
		},
		{
			name:     "GitHub HTTPS add .git",
			input:    "https://github.com/user/repo",
			expected: "https://github.com/user/repo.git",
		},
		{
			name:     "GitHub HTTPS with .git unchanged",
			input:    "https://github.com/user/repo.git",
			expected: "https://github.com/user/repo.git",
		},
		{
			name:     "SSH URL unchanged",
			input:    "git@github.com:user/repo.git",
			expected: "git@github.com:user/repo.git",
		},
		{
			name:     "Other Git URL unchanged",
			input:    "https://gitlab.com/user/repo.git",
			expected: "https://gitlab.com/user/repo.git",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := NormalizeGitURL(tc.input)

			if tc.wantErr {
				if err == nil {
					t.Errorf("Expected error for input %q", tc.input)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for input %q: %v", tc.input, err)
				return
			}

			if result != tc.expected {
				t.Errorf("NormalizeGitURL(%q) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}
