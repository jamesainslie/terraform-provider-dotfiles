// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package platform

import (
	"os"
	"runtime"
)

// PlatformProvider defines the interface for platform-specific operations.
type PlatformProvider interface {
	// Platform information
	GetPlatform() string
	GetArchitecture() string

	// Directory operations
	GetHomeDir() (string, error)
	GetConfigDir() (string, error)
	GetAppSupportDir() (string, error)

	// Path operations
	ResolvePath(path string) (string, error)
	ExpandPath(path string) (string, error)
	GetPathSeparator() string

	// File operations
	CreateSymlink(source, target string) error
	CopyFile(source, target string) error
	SetPermissions(path string, mode os.FileMode) error
	GetFileInfo(path string) (os.FileInfo, error)

	// Application detection
	DetectApplication(name string) (*ApplicationInfo, error)
	GetApplicationPaths(name string) (map[string]string, error)
}

// ApplicationInfo represents information about an installed application.
type ApplicationInfo struct {
	Name             string
	Installed        bool
	Version          string
	InstallationPath string
	ExecutablePath   string
	ConfigPaths      []string
	DataPaths        []string
}

// BasePlatform provides common functionality for all platforms.
type BasePlatform struct {
	platform     string
	architecture string
}

// GetPlatform returns the platform name.
func (p *BasePlatform) GetPlatform() string {
	return p.platform
}

// GetArchitecture returns the architecture.
func (p *BasePlatform) GetArchitecture() string {
	return p.architecture
}

// GetPathSeparator returns the path separator for the platform.
func (p *BasePlatform) GetPathSeparator() string {
	if p.platform == "windows" {
		return ";"
	}
	return ":"
}

// DetectPlatform detects and returns the appropriate platform provider.
func DetectPlatform() PlatformProvider {
	switch runtime.GOOS {
	case "darwin":
		return &DarwinProvider{BasePlatform{platform: "macos", architecture: runtime.GOARCH}}
	case "linux":
		return &LinuxProvider{BasePlatform{platform: "linux", architecture: runtime.GOARCH}}
	case "windows":
		return &WindowsProvider{BasePlatform{platform: "windows", architecture: runtime.GOARCH}}
	default:
		// Fallback to Linux provider for unknown platforms
		return &LinuxProvider{BasePlatform{platform: runtime.GOOS, architecture: runtime.GOARCH}}
	}
}
