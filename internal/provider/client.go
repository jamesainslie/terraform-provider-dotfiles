// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/services"
)

// DotfilesClient provides the client interface for dotfiles operations.
type DotfilesClient struct {
	Config       *DotfilesConfig
	Platform     string
	Architecture string
	HomeDir      string
	ConfigDir    string

	// Services
	Services *services.ServiceRegistry

	// Concurrency management
	ConcurrencyManager *services.ConcurrencyManager
}

// NewDotfilesClient creates a new dotfiles client with the provided configuration.
func NewDotfilesClient(config *DotfilesConfig) (*DotfilesClient, error) {
	client := &DotfilesClient{
		Config:       config,
		Architecture: runtime.GOARCH,
	}

	// Determine platform
	if config.AutoDetectPlatform || config.TargetPlatform == "auto" {
		client.Platform = detectPlatform()
	} else {
		client.Platform = config.TargetPlatform
	}

	// Get home directory
	homeDir, err := getHomeDir()
	if err != nil {
		return nil, fmt.Errorf("unable to determine home directory: %w", err)
	}
	client.HomeDir = homeDir

	// Get config directory
	client.ConfigDir = getConfigDir(client.Platform, homeDir)

	// Initialize concurrency manager
	client.ConcurrencyManager = services.NewConcurrencyManager(DefaultMaxConcurrency)

	// Initialize services
	serviceConfig := services.ServiceConfig{
		DryRun: config.DryRun,
		PlatformProvider: &ClientPlatformProvider{
			client: client,
		},
	}
	client.Services = services.NewServiceRegistry(serviceConfig)

	return client, nil
}

// detectPlatform detects the current platform.
func detectPlatform() string {
	switch runtime.GOOS {
	case "darwin":
		return "macos"
	case "linux":
		return "linux"
	case "windows":
		return "windows"
	default:
		return runtime.GOOS
	}
}

// getHomeDir returns the user's home directory.
func getHomeDir() (string, error) {
	// This is a placeholder - will be replaced with platform-specific implementation
	// For now, use the OS package
	return os.UserHomeDir()
}

// getConfigDir returns the user's config directory based on platform.
func getConfigDir(platform, homeDir string) string {
	switch platform {
	case "macos", "linux":
		return filepath.Join(homeDir, ".config")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData != "" {
			return appData
		}
		return filepath.Join(homeDir, "AppData", "Roaming")
	default:
		return filepath.Join(homeDir, ".config")
	}
}

// GetPlatformInfo returns platform information.
func (c *DotfilesClient) GetPlatformInfo() map[string]interface{} {
	return map[string]interface{}{
		"platform":     c.Platform,
		"architecture": c.Architecture,
		"home_dir":     c.HomeDir,
		"config_dir":   c.ConfigDir,
	}
}

// ClientPlatformProvider implements the platform provider interfaces for services.
type ClientPlatformProvider struct {
	client *DotfilesClient
}

// CopyFile implements services.PlatformProvider.CopyFile.
func (p *ClientPlatformProvider) CopyFile(src, dst string, mode os.FileMode) error {
	// Implementation would use platform-specific file operations
	return nil
}

// CreateDirectory implements services.PlatformProvider.CreateDirectory.
func (p *ClientPlatformProvider) CreateDirectory(path string, mode os.FileMode) error {
	return os.MkdirAll(path, mode)
}

// GetFileInfo implements services.PlatformProvider.GetFileInfo.
func (p *ClientPlatformProvider) GetFileInfo(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// CalculateChecksum implements services.PlatformProvider.CalculateChecksum.
func (p *ClientPlatformProvider) CalculateChecksum(path string) (string, error) {
	// Implementation would calculate file checksum
	return "", nil
}

// ReadFile implements services.TemplatePlatformProvider.ReadFile.
func (p *ClientPlatformProvider) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile implements services.TemplatePlatformProvider.WriteFile.
func (p *ClientPlatformProvider) WriteFile(path string, content []byte, mode uint32) error {
	return os.WriteFile(path, content, os.FileMode(mode))
}

// GetPlatformInfo implements services.TemplatePlatformProvider.GetPlatformInfo.
func (p *ClientPlatformProvider) GetPlatformInfo() map[string]interface{} {
	return map[string]interface{}{
		"platform":     p.client.Platform,
		"architecture": p.client.Architecture,
		"home_dir":     p.client.HomeDir,
		"config_dir":   p.client.ConfigDir,
	}
}

// ExpandPath implements services.TemplatePlatformProvider.ExpandPath.
func (p *ClientPlatformProvider) ExpandPath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}

	// Handle tilde expansion
	if len(path) > 0 && path[0] == '~' {
		return filepath.Join(p.client.HomeDir, path[1:]), nil
	}

	return path, nil
}
