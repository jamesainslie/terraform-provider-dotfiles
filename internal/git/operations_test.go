// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package git

import (
	"testing"
)

func TestIsGitURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "GitHub HTTPS URL",
			url:      "https://github.com/user/repo.git",
			expected: true,
		},
		{
			name:     "GitHub HTTPS URL without .git",
			url:      "https://github.com/user/repo",
			expected: true,
		},
		{
			name:     "GitHub SSH URL",
			url:      "git@github.com:user/repo.git",
			expected: true,
		},
		{
			name:     "GitHub SSH URL without .git",
			url:      "git@github.com:user/repo",
			expected: true,
		},
		{
			name:     "GitHub shorthand",
			url:      "github.com/user/repo",
			expected: true,
		},
		{
			name:     "GitHub shorthand with colon",
			url:      "github.com:user/repo",
			expected: true,
		},
		{
			name:     "Generic HTTPS Git URL",
			url:      "https://gitlab.com/user/repo.git",
			expected: true,
		},
		{
			name:     "Generic SSH Git URL",
			url:      "git@gitlab.com:user/repo.git",
			expected: true,
		},
		{
			name:     "SSH Git URL with ssh://",
			url:      "ssh://git@gitlab.com/user/repo.git",
			expected: true,
		},
		{
			name:     "Local path",
			url:      "/path/to/local/repo",
			expected: false,
		},
		{
			name:     "Relative path",
			url:      "./local/repo",
			expected: false,
		},
		{
			name:     "Home path",
			url:      "~/dotfiles",
			expected: false,
		},
		{
			name:     "HTTPS URL without .git",
			url:      "https://example.com/path",
			expected: false,
		},
		{
			name:     "Invalid URL",
			url:      "not-a-url",
			expected: false,
		},
		{
			name:     "Empty string",
			url:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsGitURL(tt.url)
			if result != tt.expected {
				t.Errorf("IsGitURL(%q) = %v, expected %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestNormalizeGitURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "GitHub shorthand with slash",
			input:    "github.com/user/repo",
			expected: "https://github.com/user/repo.git",
			wantErr:  false,
		},
		{
			name:     "GitHub shorthand with colon",
			input:    "github.com:user/repo",
			expected: "https://github.com/user/repo.git",
			wantErr:  false,
		},
		{
			name:     "GitHub shorthand with .git",
			input:    "github.com/user/repo.git",
			expected: "https://github.com/user/repo.git",
			wantErr:  false,
		},
		{
			name:     "GitHub HTTPS without .git",
			input:    "https://github.com/user/repo",
			expected: "https://github.com/user/repo.git",
			wantErr:  false,
		},
		{
			name:     "GitHub HTTPS with .git",
			input:    "https://github.com/user/repo.git",
			expected: "https://github.com/user/repo.git",
			wantErr:  false,
		},
		{
			name:     "SSH URL unchanged",
			input:    "git@github.com:user/repo.git",
			expected: "git@github.com:user/repo.git",
			wantErr:  false,
		},
		{
			name:     "Other HTTPS Git URL unchanged",
			input:    "https://gitlab.com/user/repo.git",
			expected: "https://gitlab.com/user/repo.git",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeGitURL(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NormalizeGitURL(%q) expected error, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("NormalizeGitURL(%q) unexpected error: %v", tt.input, err)
				return
			}

			if result != tt.expected {
				t.Errorf("NormalizeGitURL(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetLocalCachePath(t *testing.T) {
	tests := []struct {
		name      string
		cacheRoot string
		repoURL   string
		expected  string
		wantErr   bool
	}{
		{
			name:      "GitHub HTTPS URL",
			cacheRoot: "/tmp/cache",
			repoURL:   "https://github.com/user/repo.git",
			expected:  "/tmp/cache/github.com/user/repo",
			wantErr:   false,
		},
		{
			name:      "GitHub HTTPS URL without .git",
			cacheRoot: "/tmp/cache",
			repoURL:   "https://github.com/user/repo",
			expected:  "/tmp/cache/github.com/user/repo",
			wantErr:   false,
		},
		{
			name:      "GitLab URL",
			cacheRoot: "/tmp/cache",
			repoURL:   "https://gitlab.com/user/repo.git",
			expected:  "/tmp/cache/gitlab.com/user/repo",
			wantErr:   false,
		},
		{
			name:      "Enterprise GitHub",
			cacheRoot: "/tmp/cache",
			repoURL:   "https://github.enterprise.com/company/repo.git",
			expected:  "/tmp/cache/github.enterprise.com/company/repo",
			wantErr:   false,
		},
		{
			name:      "URL with port",
			cacheRoot: "/tmp/cache",
			repoURL:   "https://git.example.com:8080/user/repo.git",
			expected:  "/tmp/cache/git.example.com_8080/user/repo",
			wantErr:   false,
		},
		{
			name:      "Invalid URL",
			cacheRoot: "/tmp/cache",
			repoURL:   "not-a-valid-url",
			expected:  "/tmp/cache/not-a-valid-url", // url.Parse is lenient
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetLocalCachePath(tt.cacheRoot, tt.repoURL)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetLocalCachePath(%q, %q) expected error, got nil", tt.cacheRoot, tt.repoURL)
				}
				return
			}

			if err != nil {
				t.Errorf("GetLocalCachePath(%q, %q) unexpected error: %v", tt.cacheRoot, tt.repoURL, err)
				return
			}

			if result != tt.expected {
				t.Errorf("GetLocalCachePath(%q, %q) = %q, expected %q", tt.cacheRoot, tt.repoURL, result, tt.expected)
			}
		})
	}
}

func TestNewGitManager(t *testing.T) {
	tests := []struct {
		name       string
		authConfig *AuthConfig
		wantErr    bool
	}{
		{
			name:       "Nil auth config",
			authConfig: nil,
			wantErr:    false,
		},
		{
			name:       "Empty auth config",
			authConfig: &AuthConfig{},
			wantErr:    false,
		},
		{
			name: "PAT auth config",
			authConfig: &AuthConfig{
				PersonalAccessToken: "ghp_test_token",
				Username:            "testuser",
			},
			wantErr: false,
		},
		{
			name: "PAT without username",
			authConfig: &AuthConfig{
				PersonalAccessToken: "ghp_test_token",
			},
			wantErr: false,
		},
		{
			name: "SSH auth config with non-existent key",
			authConfig: &AuthConfig{
				SSHPrivateKeyPath: "/path/to/nonexistent/key",
			},
			wantErr: true, // Should fail because key doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewGitManager(tt.authConfig)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewGitManager() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewGitManager() unexpected error: %v", err)
				return
			}

			if manager == nil {
				t.Error("NewGitManager() returned nil manager")
			}
		})
	}
}

func TestBuildAuthMethod(t *testing.T) {
	tests := []struct {
		name       string
		authConfig *AuthConfig
		expectAuth bool
		wantErr    bool
	}{
		{
			name:       "No authentication",
			authConfig: &AuthConfig{},
			expectAuth: false,
			wantErr:    false,
		},
		{
			name: "PAT authentication",
			authConfig: &AuthConfig{
				PersonalAccessToken: "ghp_test_token",
				Username:            "testuser",
			},
			expectAuth: true,
			wantErr:    false,
		},
		{
			name: "PAT without username",
			authConfig: &AuthConfig{
				PersonalAccessToken: "ghp_test_token",
			},
			expectAuth: true,
			wantErr:    false,
		},
		{
			name: "SSH with non-existent key",
			authConfig: &AuthConfig{
				SSHPrivateKeyPath: "/path/to/nonexistent/key",
			},
			expectAuth: false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth, err := buildAuthMethod(tt.authConfig)

			if tt.wantErr {
				if err == nil {
					t.Errorf("buildAuthMethod() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("buildAuthMethod() unexpected error: %v", err)
				return
			}

			hasAuth := auth != nil
			if hasAuth != tt.expectAuth {
				t.Errorf("buildAuthMethod() auth = %v, expected auth = %v", hasAuth, tt.expectAuth)
			}
		})
	}
}
