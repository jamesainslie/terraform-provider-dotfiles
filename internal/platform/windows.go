// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package platform

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// WindowsProvider implements PlatformProvider for Windows.
type WindowsProvider struct {
	BasePlatform
}

// GetHomeDir returns the user's home directory.
func (p *WindowsProvider) GetHomeDir() (string, error) {
	return os.UserHomeDir()
}

// GetConfigDir returns the user's config directory.
func (p *WindowsProvider) GetConfigDir() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData != "" {
		return appData, nil
	}

	home, err := p.GetHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "AppData", "Roaming"), nil
}

// GetAppSupportDir returns the user's application data directory.
func (p *WindowsProvider) GetAppSupportDir() (string, error) {
	return p.GetConfigDir() // Same as config on Windows
}

// ResolvePath resolves a path to its absolute form.
func (p *WindowsProvider) ResolvePath(path string) (string, error) {
	return p.ExpandPath(path)
}

// ExpandPath expands ~ and environment variables in the path.
func (p *WindowsProvider) ExpandPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}

	// Handle ~ expansion
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
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

	// Convert to absolute path and normalize separators
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("unable to convert to absolute path: %w", err)
	}

	return filepath.FromSlash(absPath), nil
}

// CreateSymlink creates a symbolic link (requires elevated privileges on Windows).
func (p *WindowsProvider) CreateSymlink(source, target string) error {
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
	sourceInfo, err := os.Stat(expandedSource)
	if os.IsNotExist(err) {
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
	// Note: This requires developer mode or elevated privileges on Windows
	if err := os.Symlink(expandedSource, expandedTarget); err != nil {
		// If symlink fails, fall back to copying (for Windows compatibility)
		fmt.Printf("Warning: Symlink creation failed, falling back to copy: %v\n", err)
		return p.CopyFile(source, target)
	}

	// For directories, we may need to handle differently on Windows
	if sourceInfo.IsDir() {
		// Directory symlinks work differently on Windows
		// For now, we use the same approach as files
		// TODO: Implement directory-specific symlink handling for Windows
	}

	return nil
}

// CopyFile copies a file from source to target.
func (p *WindowsProvider) CopyFile(source, target string) error {
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
	defer sourceFile.Close()

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
	defer targetFile.Close()

	// Copy file contents
	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		return fmt.Errorf("unable to copy file contents: %w", err)
	}

	// Copy permissions (simplified for Windows)
	if err := targetFile.Chmod(sourceInfo.Mode()); err != nil {
		// Windows permission handling is different, so we'll log but not fail
		fmt.Printf("Warning: Unable to copy file permissions on Windows: %v\n", err)
	}

	return nil
}

// SetPermissions sets file permissions (simplified for Windows).
func (p *WindowsProvider) SetPermissions(path string, mode os.FileMode) error {
	expandedPath, err := p.ExpandPath(path)
	if err != nil {
		return fmt.Errorf("unable to expand path: %w", err)
	}

	// Windows doesn't have the same permission model as Unix
	// This is a simplified implementation
	if err := os.Chmod(expandedPath, mode); err != nil {
		// Don't fail on Windows permission errors, just warn
		fmt.Printf("Warning: Unable to set permissions on Windows: %v\n", err)
	}

	return nil
}

// GetFileInfo returns file information.
func (p *WindowsProvider) GetFileInfo(path string) (os.FileInfo, error) {
	expandedPath, err := p.ExpandPath(path)
	if err != nil {
		return nil, fmt.Errorf("unable to expand path: %w", err)
	}

	return os.Stat(expandedPath)
}

// DetectApplication detects if an application is installed.
func (p *WindowsProvider) DetectApplication(name string) (*ApplicationInfo, error) {
	info := &ApplicationInfo{
		Name:      name,
		Installed: false,
	}

	// Check if command exists in PATH
	if execPath, err := exec.LookPath(name + ".exe"); err == nil {
		info.Installed = true
		info.ExecutablePath = execPath
	} else if execPath, err := exec.LookPath(name); err == nil {
		info.Installed = true
		info.ExecutablePath = execPath
	}

	// Check common Windows application directories
	if !info.Installed {
		programFiles := os.Getenv("PROGRAMFILES")
		programFilesX86 := os.Getenv("PROGRAMFILES(X86)")

		var titleName string
		if len(name) > 0 {
			titleName = strings.ToUpper(name[:1]) + name[1:]
		}
		
		checkPaths := []string{}
		if programFiles != "" {
			checkPaths = append(checkPaths, 
				filepath.Join(programFiles, name),
				filepath.Join(programFiles, titleName),
			)
		}
		if programFilesX86 != "" {
			checkPaths = append(checkPaths,
				filepath.Join(programFilesX86, name),
				filepath.Join(programFilesX86, titleName),
			)
		}

		for _, appPath := range checkPaths {
			if _, err := os.Stat(appPath); err == nil {
				info.Installed = true
				info.InstallationPath = appPath
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
func (p *WindowsProvider) GetApplicationPaths(name string) (map[string]string, error) {
	paths := make(map[string]string)

	home, err := p.GetHomeDir()
	if err != nil {
		return nil, err
	}

	appData, err := p.GetConfigDir()
	if err != nil {
		return nil, err
	}

	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		localAppData = filepath.Join(home, "AppData", "Local")
	}

	// Common application path patterns
	paths["config"] = filepath.Join(appData, name)
	paths["data"] = filepath.Join(localAppData, name)
	paths["cache"] = filepath.Join(localAppData, name, "Cache")

	// Special cases for known applications
	switch strings.ToLower(name) {
	case "cursor":
		paths["config"] = filepath.Join(appData, "Cursor", "User")
		paths["data"] = filepath.Join(localAppData, "Cursor")
	case "code", "vscode":
		paths["config"] = filepath.Join(appData, "Code", "User")
		paths["data"] = filepath.Join(localAppData, "Code")
	case "git":
		paths["config"] = filepath.Join(home, ".gitconfig")
	case "ssh":
		paths["config"] = filepath.Join(home, ".ssh")
	case "gpg":
		paths["config"] = filepath.Join(appData, "gnupg")
	case "powershell":
		paths["config"] = filepath.Join(home, "Documents", "PowerShell")
	}

	return paths, nil
}
