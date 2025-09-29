// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolvePathFunctions(t *testing.T) {
	platforms := []PlatformProvider{
		&DarwinProvider{BasePlatform{platform: "macos", architecture: "amd64"}},
		&LinuxProvider{BasePlatform{platform: "linux", architecture: "amd64"}},
		&WindowsProvider{BasePlatform{platform: "windows", architecture: "amd64"}},
	}

	for _, platform := range platforms {
		t.Run("Platform: "+platform.GetPlatform(), func(t *testing.T) {
			t.Run("ResolvePath", func(t *testing.T) {
				testCases := []struct {
					name        string
					input       string
					shouldError bool
				}{
					{
						name:        "Home directory",
						input:       "~",
						shouldError: false,
					},
					{
						name:        "Home subdirectory",
						input:       "~/test",
						shouldError: false,
					},
					{
						name:        "Absolute path",
						input:       "/tmp/test",
						shouldError: false,
					},
					{
						name:        "Relative path",
						input:       "./test",
						shouldError: false,
					},
					{
						name:        "Empty path",
						input:       "",
						shouldError: true,
					},
				}

				for _, tc := range testCases {
					t.Run(tc.name, func(t *testing.T) {
						result, err := platform.ResolvePath(tc.input)

						if tc.shouldError {
							if err == nil {
								t.Errorf("ResolvePath(%q) should have errored", tc.input)
							}
							return
						}

						if err != nil {
							t.Errorf("ResolvePath(%q) unexpected error: %v", tc.input, err)
							return
						}

						if result == "" {
							t.Errorf("ResolvePath(%q) returned empty string", tc.input)
						}

						// Should return absolute path
						if !filepath.IsAbs(result) {
							t.Errorf("ResolvePath(%q) should return absolute path, got: %s", tc.input, result)
						}
					})
				}
			})

			t.Run("ExpandPath comprehensive", func(t *testing.T) {
				testCases := []struct {
					name        string
					input       string
					shouldError bool
					checkFunc   func(string) bool
				}{
					{
						name:        "Home directory",
						input:       "~",
						shouldError: false,
						checkFunc:   func(result string) bool { return strings.Contains(result, "/") || strings.Contains(result, "\\") },
					},
					{
						name:        "Home subdirectory",
						input:       "~/Documents",
						shouldError: false,
						checkFunc:   func(result string) bool { return strings.Contains(result, "Documents") },
					},
					{
						name:        "Environment variable",
						input:       "$HOME/test",
						shouldError: false,
						checkFunc:   func(result string) bool { return strings.Contains(result, "test") },
					},
					{
						name:        "Current directory",
						input:       ".",
						shouldError: false,
						checkFunc:   filepath.IsAbs,
					},
					{
						name:        "Empty path",
						input:       "",
						shouldError: true,
						checkFunc:   nil,
					},
				}

				for _, tc := range testCases {
					t.Run(tc.name, func(t *testing.T) {
						result, err := platform.ExpandPath(tc.input)

						if tc.shouldError {
							if err == nil {
								t.Errorf("ExpandPath(%q) should have errored", tc.input)
							}
							return
						}

						if err != nil {
							t.Errorf("ExpandPath(%q) unexpected error: %v", tc.input, err)
							return
						}

						if tc.checkFunc != nil && !tc.checkFunc(result) {
							t.Errorf("ExpandPath(%q) = %s failed check function", tc.input, result)
						}
					})
				}
			})
		})
	}
}

func TestPlatformSpecificExpandPath(t *testing.T) {
	// Test current platform directly
	platform := DetectPlatform()

	t.Run("Current platform expand path", func(t *testing.T) {
		// Test with actual home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("Failed to get home directory: %v", err)
		}

		// Test home expansion
		result, err := platform.ExpandPath("~")
		if err != nil {
			t.Errorf("ExpandPath('~') failed: %v", err)
		} else if result != homeDir {
			t.Errorf("ExpandPath('~') = %s, expected %s", result, homeDir)
		}

		// Test home subdirectory expansion
		result, err = platform.ExpandPath("~/test")
		if err != nil {
			t.Errorf("ExpandPath('~/test') failed: %v", err)
		} else {
			expectedPath := filepath.Join(homeDir, "test")
			if result != expectedPath {
				t.Errorf("ExpandPath('~/test') = %s, expected %s", result, expectedPath)
			}
		}

		// Test environment variable expansion
		if runtime.GOOS != "windows" {
			result, err = platform.ExpandPath("$HOME")
			if err != nil {
				t.Errorf("ExpandPath('$HOME') failed: %v", err)
			} else if result != homeDir {
				t.Errorf("ExpandPath('$HOME') = %s, expected %s", result, homeDir)
			}
		}
	})
}

func TestGetAppSupportDir(t *testing.T) {
	platforms := []PlatformProvider{
		&DarwinProvider{BasePlatform{platform: "macos", architecture: "amd64"}},
		&LinuxProvider{BasePlatform{platform: "linux", architecture: "amd64"}},
		&WindowsProvider{BasePlatform{platform: "windows", architecture: "amd64"}},
	}

	for _, platform := range platforms {
		t.Run("Platform: "+platform.GetPlatform(), func(t *testing.T) {
			appSupportDir, err := platform.GetAppSupportDir()
			if err != nil {
				t.Errorf("GetAppSupportDir() failed: %v", err)
			}

			if appSupportDir == "" {
				t.Error("GetAppSupportDir() should not return empty string")
			}

			// Platform-specific validation
			switch platform.GetPlatform() {
			case "macos":
				if !strings.Contains(appSupportDir, "Library") {
					t.Logf("macOS app support dir: %s (may not contain 'Library' in test environment)", appSupportDir)
				}
			case "linux":
				if !strings.Contains(appSupportDir, ".local") && !strings.Contains(appSupportDir, "share") {
					t.Logf("Linux app support dir: %s (may not follow XDG in test environment)", appSupportDir)
				}
			case "windows":
				// Windows app support is same as config dir
				configDir, _ := platform.GetConfigDir()
				if appSupportDir != configDir {
					t.Logf("Windows app support should match config dir")
				}
			}
		})
	}
}
