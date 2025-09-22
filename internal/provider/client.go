// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// DotfilesClient provides the client interface for dotfiles operations
type DotfilesClient struct {
	Config       *DotfilesConfig
	Platform     string
	Architecture string
	HomeDir      string
	ConfigDir    string
}

// NewDotfilesClient creates a new dotfiles client with the provided configuration
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
	configDir, err := getConfigDir(client.Platform, homeDir)
	if err != nil {
		return nil, fmt.Errorf("unable to determine config directory: %w", err)
	}
	client.ConfigDir = configDir

	return client, nil
}

// detectPlatform detects the current platform
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

// getHomeDir returns the user's home directory
func getHomeDir() (string, error) {
	// This is a placeholder - will be replaced with platform-specific implementation
	// For now, use the OS package
	return os.UserHomeDir()
}

// getConfigDir returns the user's config directory based on platform
func getConfigDir(platform, homeDir string) (string, error) {
	switch platform {
	case "macos", "linux":
		return filepath.Join(homeDir, ".config"), nil
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData != "" {
			return appData, nil
		}
		return filepath.Join(homeDir, "AppData", "Roaming"), nil
	default:
		return filepath.Join(homeDir, ".config"), nil
	}
}

// GetPlatformInfo returns platform information
func (c *DotfilesClient) GetPlatformInfo() map[string]interface{} {
	return map[string]interface{}{
		"platform":     c.Platform,
		"architecture": c.Architecture,
		"home_dir":     c.HomeDir,
		"config_dir":   c.ConfigDir,
	}
}
