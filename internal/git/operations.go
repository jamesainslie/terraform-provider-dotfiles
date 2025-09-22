// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package git

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

// GitManager handles Git operations for dotfiles repositories.
type GitManager struct {
	auth transport.AuthMethod
}

// RepositoryInfo contains information about a Git repository.
type RepositoryInfo struct {
	URL        string
	LocalPath  string
	Branch     string
	LastCommit string
	LastUpdate time.Time
}

// AuthConfig contains authentication configuration.
type AuthConfig struct {
	// Personal Access Token for HTTPS authentication
	PersonalAccessToken string
	// Username for HTTPS authentication (optional, defaults to token)
	Username string
	// SSH Private Key path for SSH authentication
	SSHPrivateKeyPath string
	// SSH Private Key passphrase
	SSHPassphrase string
}

// NewGitManager creates a new Git manager with authentication.
func NewGitManager(authConfig *AuthConfig) (*GitManager, error) {
	manager := &GitManager{}

	if authConfig != nil {
		auth, err := buildAuthMethod(authConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build authentication: %w", err)
		}
		manager.auth = auth
	}

	return manager, nil
}

// IsGitURL checks if the given string is a valid Git URL.
func IsGitURL(sourceURL string) bool {
	// Match GitHub URLs
	githubPatterns := []string{
		`^https://github\.com/[\w\-\.]+/[\w\-\.]+(?:\.git)?/?$`,
		`^git@github\.com:[\w\-\.]+/[\w\-\.]+(?:\.git)?$`,
		`^github\.com[:/][\w\-\.]+/[\w\-\.]+(?:\.git)?/?$`,
	}

	for _, pattern := range githubPatterns {
		matched, _ := regexp.MatchString(pattern, sourceURL)
		if matched {
			return true
		}
	}

	// Match general Git URLs
	generalPatterns := []string{
		`^https?://.*\.git$`,
		`^git@.*:.*\.git$`,
		`^ssh://.*\.git$`,
	}

	for _, pattern := range generalPatterns {
		matched, _ := regexp.MatchString(pattern, sourceURL)
		if matched {
			return true
		}
	}

	return false
}

// NormalizeGitURL converts various Git URL formats to standard format.
func NormalizeGitURL(sourceURL string) (string, error) {
	// Handle GitHub shorthand formats
	if matched, _ := regexp.MatchString(`^github\.com[:/][\w\-\.]+/[\w\-\.]+`, sourceURL); matched {
		// Convert github.com/user/repo to https://github.com/user/repo.git
		url := strings.TrimPrefix(sourceURL, "github.com/")
		url = strings.TrimPrefix(url, "github.com:")
		if !strings.HasSuffix(url, ".git") {
			url = url + ".git"
		}
		return "https://github.com/" + url, nil
	}

	// Ensure .git suffix for GitHub HTTPS URLs
	if matched, _ := regexp.MatchString(`^https://github\.com/[\w\-\.]+/[\w\-\.]+$`, sourceURL); matched {
		if !strings.HasSuffix(sourceURL, ".git") {
			return sourceURL + ".git", nil
		}
	}

	return sourceURL, nil
}

// CloneRepository clones a Git repository to the specified local path.
func (g *GitManager) CloneRepository(ctx context.Context, repoURL, localPath, branch string) (*RepositoryInfo, error) {
	// Normalize the URL
	normalizedURL, err := NormalizeGitURL(repoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize URL: %w", err)
	}

	// Ensure local path directory exists
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Clone options
	cloneOptions := &git.CloneOptions{
		URL:          normalizedURL,
		Auth:         g.auth,
		RemoteName:   "origin",
		SingleBranch: true,
		Tags:         git.NoTags,
	}

	if branch != "" {
		cloneOptions.ReferenceName = plumbing.ReferenceName("refs/heads/" + branch)
	} else {
		// Let Git determine the default branch
		cloneOptions.SingleBranch = false
	}

	// Perform clone
	repo, err := git.PlainCloneContext(ctx, localPath, false, cloneOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	// Get repository info
	info, err := g.getRepositoryInfo(repo, normalizedURL, localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository info: %w", err)
	}

	return info, nil
}

// UpdateRepository pulls the latest changes from the remote repository.
func (g *GitManager) UpdateRepository(ctx context.Context, localPath string) (*RepositoryInfo, error) {
	// Open existing repository
	repo, err := git.PlainOpen(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Get working tree
	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	// Pull latest changes
	pullOptions := &git.PullOptions{
		RemoteName: "origin",
		Auth:       g.auth,
	}

	err = worktree.PullContext(ctx, pullOptions)
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return nil, fmt.Errorf("failed to pull updates: %w", err)
	}

	// Get remote URL
	remote, err := repo.Remote("origin")
	if err != nil {
		return nil, fmt.Errorf("failed to get remote: %w", err)
	}

	var remoteURL string
	if len(remote.Config().URLs) > 0 {
		remoteURL = remote.Config().URLs[0]
	}

	// Get repository info
	info, err := g.getRepositoryInfo(repo, remoteURL, localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository info: %w", err)
	}

	return info, nil
}

// GetRepositoryInfo returns information about a local Git repository.
func (g *GitManager) GetRepositoryInfo(localPath string) (*RepositoryInfo, error) {
	repo, err := git.PlainOpen(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Get remote URL
	remote, err := repo.Remote("origin")
	if err != nil {
		return nil, fmt.Errorf("failed to get remote: %w", err)
	}

	var remoteURL string
	if len(remote.Config().URLs) > 0 {
		remoteURL = remote.Config().URLs[0]
	}

	return g.getRepositoryInfo(repo, remoteURL, localPath)
}

// ValidateRepository checks if a local path contains a valid Git repository.
func (g *GitManager) ValidateRepository(localPath string) error {
	_, err := git.PlainOpen(localPath)
	if err != nil {
		return fmt.Errorf("invalid Git repository at %s: %w", localPath, err)
	}
	return nil
}

// getRepositoryInfo extracts information from a Git repository.
func (g *GitManager) getRepositoryInfo(repo *git.Repository, url, localPath string) (*RepositoryInfo, error) {
	// Get HEAD reference
	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Get current branch
	var branch string
	if head.Name().IsBranch() {
		branch = head.Name().Short()
	}

	// Get last commit
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	return &RepositoryInfo{
		URL:        url,
		LocalPath:  localPath,
		Branch:     branch,
		LastCommit: commit.Hash.String(),
		LastUpdate: time.Now(),
	}, nil
}

// buildAuthMethod creates appropriate authentication method.
func buildAuthMethod(authConfig *AuthConfig) (transport.AuthMethod, error) {
	// SSH authentication
	if authConfig.SSHPrivateKeyPath != "" {
		sshAuth, err := ssh.NewPublicKeysFromFile("git", authConfig.SSHPrivateKeyPath, authConfig.SSHPassphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to create SSH auth: %w", err)
		}
		return sshAuth, nil
	}

	// HTTPS authentication with PAT
	if authConfig.PersonalAccessToken != "" {
		username := authConfig.Username
		if username == "" {
			username = authConfig.PersonalAccessToken // GitHub allows token as username
		}

		return &http.BasicAuth{
			Username: username,
			Password: authConfig.PersonalAccessToken,
		}, nil
	}

	// No authentication
	return nil, nil
}

// GetLocalCachePath returns the local cache path for a Git repository.
func GetLocalCachePath(cacheRoot, repoURL string) (string, error) {
	// Parse the URL to create a safe directory name
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	// Create safe directory name from URL
	host := parsedURL.Host
	path := strings.TrimPrefix(parsedURL.Path, "/")
	path = strings.TrimSuffix(path, ".git")

	safePath := filepath.Join(host, path)
	// Replace any unsafe characters
	safePath = strings.ReplaceAll(safePath, ":", "_")
	safePath = strings.ReplaceAll(safePath, "?", "_")
	safePath = strings.ReplaceAll(safePath, "*", "_")

	return filepath.Join(cacheRoot, safePath), nil
}
