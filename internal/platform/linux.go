// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package platform

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// LinuxProvider implements PlatformProvider for Linux.
type LinuxProvider struct {
	BasePlatform
}

// GetHomeDir returns the user's home directory.
func (p *LinuxProvider) GetHomeDir() (string, error) {
	return os.UserHomeDir()
}

// GetConfigDir returns the user's config directory.
func (p *LinuxProvider) GetConfigDir() (string, error) {
	// Follow XDG Base Directory specification
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return configHome, nil
	}

	home, err := p.GetHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config"), nil
}

// GetAppSupportDir returns the user's data directory.
func (p *LinuxProvider) GetAppSupportDir() (string, error) {
	// Follow XDG Base Directory specification
	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		return dataHome, nil
	}

	home, err := p.GetHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share"), nil
}

// ResolvePath resolves a path to its absolute form.
func (p *LinuxProvider) ResolvePath(path string) (string, error) {
	return p.ExpandPath(path)
}

// ExpandPath expands ~ and environment variables in the path.
func (p *LinuxProvider) ExpandPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}

	// Handle ~ expansion
	if strings.HasPrefix(path, "~/") {
		home, err := p.GetHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to get home directory: %w", err)
		}
		path = filepath.Join(home, path[2:])
	} else if path == "~" {
		home, err := p.GetHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to get home directory: %w", err)
		}
		path = home
	}

	// Expand environment variables
	path = os.ExpandEnv(path)

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("unable to convert to absolute path: %w", err)
	}

	return absPath, nil
}

// CreateSymlink creates a symbolic link.
func (p *LinuxProvider) CreateSymlink(source, target string) error {
	// Expand paths
	expandedSource, err := p.ExpandPath(source)
	if err != nil {
		return fmt.Errorf("unable to expand source path: %w", err)
	}

	expandedTarget, err := p.ExpandPath(target)
	if err != nil {
		return fmt.Errorf("unable to expand target path: %w", err)
	}

	// Check if source exists
	if _, err := os.Stat(expandedSource); os.IsNotExist(err) {
		return fmt.Errorf("source file does not exist: %s", expandedSource)
	}

	// Create parent directories for target
	targetDir := filepath.Dir(expandedTarget)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("unable to create target directory: %w", err)
	}

	// Remove existing target if it exists
	if _, err := os.Lstat(expandedTarget); err == nil {
		if err := os.Remove(expandedTarget); err != nil {
			return fmt.Errorf("unable to remove existing target: %w", err)
		}
	}

	// Create the symlink
	if err := os.Symlink(expandedSource, expandedTarget); err != nil {
		return fmt.Errorf("unable to create symlink: %w", err)
	}

	return nil
}

// CopyFile copies a file from source to target.
func (p *LinuxProvider) CopyFile(source, target string) error {
	// Expand paths
	expandedSource, err := p.ExpandPath(source)
	if err != nil {
		return fmt.Errorf("unable to expand source path: %w", err)
	}

	expandedTarget, err := p.ExpandPath(target)
	if err != nil {
		return fmt.Errorf("unable to expand target path: %w", err)
	}

	// Open source file
	sourceFile, err := os.Open(expandedSource)
	if err != nil {
		return fmt.Errorf("unable to open source file: %w", err)
	}
	defer func() {
		if err := sourceFile.Close(); err != nil {
			// Log error but don't fail the operation
			fmt.Printf("Warning: failed to close source file: %v\n", err)
		}
	}()

	// Get source file info
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("unable to get source file info: %w", err)
	}

	// Create parent directories for target
	targetDir := filepath.Dir(expandedTarget)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("unable to create target directory: %w", err)
	}

	// Create target file
	targetFile, err := os.Create(expandedTarget)
	if err != nil {
		return fmt.Errorf("unable to create target file: %w", err)
	}
	defer func() {
		if err := targetFile.Close(); err != nil {
			// Log error but don't fail the operation
		}
	}()

	// Copy file contents
	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		return fmt.Errorf("unable to copy file contents: %w", err)
	}

	// Copy permissions
	if err := targetFile.Chmod(sourceInfo.Mode()); err != nil {
		return fmt.Errorf("unable to copy file permissions: %w", err)
	}

	return nil
}

// SetPermissions sets file permissions.
func (p *LinuxProvider) SetPermissions(path string, mode os.FileMode) error {
	expandedPath, err := p.ExpandPath(path)
	if err != nil {
		return fmt.Errorf("unable to expand path: %w", err)
	}

	if err := os.Chmod(expandedPath, mode); err != nil {
		return fmt.Errorf("unable to set permissions: %w", err)
	}

	return nil
}

// GetFileInfo returns file information.
func (p *LinuxProvider) GetFileInfo(path string) (os.FileInfo, error) {
	expandedPath, err := p.ExpandPath(path)
	if err != nil {
		return nil, fmt.Errorf("unable to expand path: %w", err)
	}

	return os.Stat(expandedPath)
}

// DetectApplication detects if an application is installed.
func (p *LinuxProvider) DetectApplication(name string) (*ApplicationInfo, error) {
	info := &ApplicationInfo{
		Name:      name,
		Installed: false,
	}

	// Check if command exists in PATH
	if execPath, err := exec.LookPath(name); err == nil {
		info.Installed = true
		info.ExecutablePath = execPath
	}

	// Check common application installation directories
	if !info.Installed {
		appPaths := []string{
			fmt.Sprintf("/usr/bin/%s", name),
			fmt.Sprintf("/usr/local/bin/%s", name),
			fmt.Sprintf("/bin/%s", name),
			fmt.Sprintf("/sbin/%s", name),
			fmt.Sprintf("/snap/bin/%s", name),
			fmt.Sprintf("/var/lib/flatpak/exports/bin/%s", name),
		}

		for _, appPath := range appPaths {
			if _, err := os.Stat(appPath); err == nil {
				info.Installed = true
				info.ExecutablePath = appPath
				break
			}
		}
	}

	// Get application configuration paths if installed
	if info.Installed {
		configPaths, err := p.GetApplicationPaths(name)
		if err == nil {
			if configPath, exists := configPaths["config"]; exists {
				info.ConfigPaths = append(info.ConfigPaths, configPath)
			}
			if dataPath, exists := configPaths["data"]; exists {
				info.DataPaths = append(info.DataPaths, dataPath)
			}
		}
	}

	return info, nil
}

// GetApplicationPaths returns application-specific paths.
func (p *LinuxProvider) GetApplicationPaths(name string) (map[string]string, error) {
	paths := make(map[string]string)

	home, err := p.GetHomeDir()
	if err != nil {
		return nil, err
	}

	config, err := p.GetConfigDir()
	if err != nil {
		return nil, err
	}

	data, err := p.GetAppSupportDir()
	if err != nil {
		return nil, err
	}

	// XDG Cache directory
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		cacheDir = filepath.Join(home, ".cache")
	}

	// Common application path patterns
	paths["config"] = filepath.Join(config, name)
	paths["data"] = filepath.Join(data, name)
	paths["cache"] = filepath.Join(cacheDir, name)

	// Special cases for known applications
	switch strings.ToLower(name) {
	case "cursor":
		paths["config"] = filepath.Join(config, "Cursor", "User")
		paths["data"] = filepath.Join(data, "Cursor")
	case "code", "vscode":
		paths["config"] = filepath.Join(config, "Code", "User")
		paths["data"] = filepath.Join(data, "Code")
	case "git":
		paths["config"] = filepath.Join(home, ".gitconfig")
	case "ssh":
		paths["config"] = filepath.Join(home, ".ssh")
	case "gpg":
		paths["config"] = filepath.Join(home, ".gnupg")
	case "fish":
		paths["config"] = filepath.Join(config, "fish")
	case "zsh":
		paths["config"] = filepath.Join(home, ".zshrc")
	case "bash":
		paths["config"] = filepath.Join(home, ".bashrc")
	case "vim":
		paths["config"] = filepath.Join(home, ".vimrc")
	case "nvim", "neovim":
		paths["config"] = filepath.Join(config, "nvim")
	}

	return paths, nil
}
