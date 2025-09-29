// Copyright (c) HashCorp, Inc.
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
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
)

// GitManager handles Git operations for dotfiles repositories.
type GitManager struct {
	auth   transport.AuthMethod
	config *AuthConfig
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
	// SSH Known Hosts file path
	SSHKnownHostsPath string
	// Skip SSH host key verification (insecure)
	SSHSkipHostKeyVerification bool
	// Authentication method preference
	AuthMethod string
}

// NewGitManager creates a new Git manager with authentication.
func NewGitManager(authConfig *AuthConfig) (*GitManager, error) {
	manager := &GitManager{
		config: authConfig,
	}

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
			url += ".git"
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

// CloneOptions contains options for cloning repositories.
type CloneOptions struct {
	// URL is the repository URL to clone
	URL string
	// LocalPath is where to clone the repository
	LocalPath string
	// Branch is the specific branch to clone (optional)
	Branch string
	// Depth limits the clone depth (0 for full clone)
	Depth int
	// RecurseSubmodules indicates whether to clone submodules
	RecurseSubmodules bool
	// SingleBranch indicates whether to clone only the specified branch
	SingleBranch bool
	// Progress callback for clone progress
	Progress func(message string)
}

// CloneRepositoryWithOptions clones a repository with enhanced options.
func (gm *GitManager) CloneRepositoryWithOptions(ctx context.Context, options CloneOptions) (*RepositoryInfo, error) {
	cloneOptions := &git.CloneOptions{
		URL:      options.URL,
		Auth:     gm.auth,
		Progress: nil, // We'll handle progress separately
	}

	// Set branch if specified
	if options.Branch != "" {
		cloneOptions.ReferenceName = plumbing.ReferenceName("refs/heads/" + options.Branch)
		cloneOptions.SingleBranch = options.SingleBranch
	}

	// Set depth if specified
	if options.Depth > 0 {
		cloneOptions.Depth = options.Depth
	}

	// Set submodule recursion
	if options.RecurseSubmodules {
		cloneOptions.RecurseSubmodules = git.DefaultSubmoduleRecursionDepth
	}

	// Clone the repository
	repo, err := git.PlainCloneContext(ctx, options.LocalPath, false, cloneOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	// Get repository information
	return gm.getRepositoryInfoFromRepo(repo, options.URL, options.LocalPath)
}

// UpdateSubmodules updates all submodules in a repository.
func (gm *GitManager) UpdateSubmodules(ctx context.Context, localPath string) error {
	repo, err := git.PlainOpen(localPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Get submodules and update them individually
	submodules, err := worktree.Submodules()
	if err != nil {
		return fmt.Errorf("failed to get submodules: %w", err)
	}

	// Update each submodule
	for _, submodule := range submodules {
		err = submodule.Update(&git.SubmoduleUpdateOptions{
			Init: true,
			Auth: gm.auth,
		})
		if err != nil {
			return fmt.Errorf("failed to update submodule %s: %w", submodule.Config().Name, err)
		}
	}

	return nil
}

// ListSubmodules lists all submodules in a repository.
func (gm *GitManager) ListSubmodules(localPath string) ([]*SubmoduleInfo, error) {
	repo, err := git.PlainOpen(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	submodules, err := worktree.Submodules()
	if err != nil {
		return nil, fmt.Errorf("failed to get submodules: %w", err)
	}

	submoduleInfos := make([]*SubmoduleInfo, 0, len(submodules))
	for _, submodule := range submodules {
		config := submodule.Config()
		status, err := submodule.Status()
		if err != nil {
			// Continue with other submodules if one fails
			continue
		}

		info := &SubmoduleInfo{
			Name:   config.Name,
			Path:   config.Path,
			URL:    config.URL,
			Branch: config.Branch,
			Status: status.Current.String(),
		}
		submoduleInfos = append(submoduleInfos, info)
	}

	return submoduleInfos, nil
}

// SubmoduleInfo contains information about a Git submodule.
type SubmoduleInfo struct {
	Name   string
	Path   string
	URL    string
	Branch string
	Status string
}

// ValidateRepositoryDetailed validates that a repository is properly configured and returns detailed results.
func (gm *GitManager) ValidateRepositoryDetailed(localPath string) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:    true,
		Warnings: []string{},
		Errors:   []string{},
	}

	// Check if directory exists
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		result.Valid = false
		result.Errors = append(result.Errors, "Repository directory does not exist")
		return result, nil
	}

	// Check if it's a Git repository
	repo, err := git.PlainOpen(localPath)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, "Not a valid Git repository: "+err.Error())
		return result, err
	}

	// Check for uncommitted changes
	worktree, err := repo.Worktree()
	if err != nil {
		result.Warnings = append(result.Warnings, "Could not check worktree status: "+err.Error())
		return result, err
	} else {
		status, err := worktree.Status()
		if err != nil {
			result.Warnings = append(result.Warnings, "Could not get repository status: "+err.Error())
		} else if !status.IsClean() {
			result.Warnings = append(result.Warnings, "Repository has uncommitted changes")
		}
	}

	// Check for submodules and their status
	if worktree != nil {
		submodules, err := worktree.Submodules()
		if err == nil && len(submodules) > 0 {
			for _, submodule := range submodules {
				status, err := submodule.Status()
				if err != nil {
					result.Warnings = append(result.Warnings,
						fmt.Sprintf("Could not check submodule %s status: %s",
							submodule.Config().Name, err.Error()))
				} else if !status.IsClean() {
					result.Warnings = append(result.Warnings,
						fmt.Sprintf("Submodule %s has uncommitted changes",
							submodule.Config().Name))
				}
			}
		}
	}

	return result, nil
}

// ValidationResult contains the result of repository validation.
type ValidationResult struct {
	Valid    bool
	Warnings []string
	Errors   []string
}

// GetRemoteInfo gets information about the remote repository.
func (gm *GitManager) GetRemoteInfo(ctx context.Context, repoURL string) (*RemoteInfo, error) {
	// Create a memory storage for remote operations
	storage := memory.NewStorage()

	// Create remote
	remote := git.NewRemote(storage, &config.RemoteConfig{
		Name: "origin",
		URLs: []string{repoURL},
	})

	// List references
	refs, err := remote.List(&git.ListOptions{
		Auth: gm.auth,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list remote references: %w", err)
	}

	info := &RemoteInfo{
		URL:      repoURL,
		Branches: []string{},
		Tags:     []string{},
	}

	// Parse references
	for _, ref := range refs {
		name := ref.Name()
		if name.IsBranch() {
			branchName := name.Short()
			info.Branches = append(info.Branches, branchName)
			if branchName == "main" || branchName == "master" {
				info.DefaultBranch = branchName
			}
		} else if name.IsTag() {
			info.Tags = append(info.Tags, name.Short())
		}
	}

	// Set default branch if not found
	if info.DefaultBranch == "" && len(info.Branches) > 0 {
		info.DefaultBranch = info.Branches[0]
	}

	return info, nil
}

// RemoteInfo contains information about a remote repository.
type RemoteInfo struct {
	URL           string
	DefaultBranch string
	Branches      []string
	Tags          []string
}

// Helper method to get repository info from a git.Repository.
func (gm *GitManager) getRepositoryInfoFromRepo(repo *git.Repository, url, localPath string) (*RepositoryInfo, error) {
	// Get HEAD reference
	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Get branch name
	var branchName string
	if head.Name().IsBranch() {
		branchName = head.Name().Short()
	} else {
		branchName = "detached"
	}

	// Get last commit
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	return &RepositoryInfo{
		URL:        url,
		LocalPath:  localPath,
		Branch:     branchName,
		LastCommit: commit.Hash.String(),
		LastUpdate: time.Now(),
	}, nil
}
