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

// DarwinProvider implements PlatformProvider for macOS.
type DarwinProvider struct {
	BasePlatform
}

// GetHomeDir returns the user's home directory.
func (p *DarwinProvider) GetHomeDir() (string, error) {
	return os.UserHomeDir()
}

// GetConfigDir returns the user's config directory.
func (p *DarwinProvider) GetConfigDir() (string, error) {
	home, err := p.GetHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config"), nil
}

// GetAppSupportDir returns the user's Application Support directory.
func (p *DarwinProvider) GetAppSupportDir() (string, error) {
	home, err := p.GetHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Application Support"), nil
}

// ResolvePath resolves a path to its absolute form.
func (p *DarwinProvider) ResolvePath(path string) (string, error) {
	return p.ExpandPath(path)
}

// ExpandPath expands ~ and environment variables in the path.
func (p *DarwinProvider) ExpandPath(path string) (string, error) {
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
func (p *DarwinProvider) CreateSymlink(source, target string) error {
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
	if info, err := os.Lstat(expandedTarget); err == nil {
		// Handle both files and directories properly
		if info.IsDir() {
			if err := os.RemoveAll(expandedTarget); err != nil {
				return fmt.Errorf("unable to remove existing directory: %w", err)
			}
		} else {
			if err := os.Remove(expandedTarget); err != nil {
				return fmt.Errorf("unable to remove existing file: %w", err)
			}
		}
	}

	// Create the symlink
	if err := os.Symlink(expandedSource, expandedTarget); err != nil {
		return fmt.Errorf("unable to create symlink: %w", err)
	}

	return nil
}

// CopyFile copies a file from source to target.
func (p *DarwinProvider) CopyFile(source, target string) error {
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
		// Best effort close - errors are non-critical
		_ = sourceFile.Close()
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
		// Best effort close - errors are non-critical
		_ = targetFile.Close()
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
func (p *DarwinProvider) SetPermissions(path string, mode os.FileMode) error {
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
func (p *DarwinProvider) GetFileInfo(path string) (os.FileInfo, error) {
	expandedPath, err := p.ExpandPath(path)
	if err != nil {
		return nil, fmt.Errorf("unable to expand path: %w", err)
	}

	return os.Stat(expandedPath)
}

// DetectApplication detects if an application is installed.
func (p *DarwinProvider) DetectApplication(name string) (*ApplicationInfo, error) {
	info := &ApplicationInfo{
		Name:      name,
		Installed: false,
	}

	// Check common application locations
	var titleName string
	if len(name) > 0 {
		titleName = strings.ToUpper(name[:1]) + name[1:]
	}

	appPaths := []string{
		fmt.Sprintf("/Applications/%s.app", name),
		fmt.Sprintf("/Applications/%s.app", titleName),
		fmt.Sprintf("/System/Applications/%s.app", name),
		fmt.Sprintf("/System/Applications/%s.app", titleName),
	}

	for _, appPath := range appPaths {
		if _, err := os.Stat(appPath); err == nil {
			info.Installed = true
			info.InstallationPath = appPath
			break
		}
	}

	// Check if command exists in PATH
	if !info.Installed {
		if execPath, err := exec.LookPath(name); err == nil {
			info.Installed = true
			info.ExecutablePath = execPath
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
func (p *DarwinProvider) GetApplicationPaths(name string) (map[string]string, error) {
	paths := make(map[string]string)

	home, err := p.GetHomeDir()
	if err != nil {
		return nil, err
	}

	appSupport, err := p.GetAppSupportDir()
	if err != nil {
		return nil, err
	}

	config, err := p.GetConfigDir()
	if err != nil {
		return nil, err
	}

	// Common application path patterns
	paths["config"] = filepath.Join(config, name)
	paths["data"] = filepath.Join(appSupport, name)
	paths["cache"] = filepath.Join(home, "Library", "Caches", name)
	paths["preferences"] = filepath.Join(home, "Library", "Preferences", fmt.Sprintf("com.%s.plist", name))

	// Special cases for known applications
	switch strings.ToLower(name) {
	case "cursor":
		paths["config"] = filepath.Join(appSupport, "Cursor", "User")
		paths["data"] = filepath.Join(appSupport, "Cursor")
	case "vscode":
		paths["config"] = filepath.Join(appSupport, "Code", "User")
		paths["data"] = filepath.Join(appSupport, "Code")
	case "git":
		paths["config"] = filepath.Join(home, ".gitconfig")
	case "ssh":
		paths["config"] = filepath.Join(home, ".ssh")
	case "gpg":
		paths["config"] = filepath.Join(home, ".gnupg")
	}

	return paths, nil
}
