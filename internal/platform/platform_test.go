// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDetectPlatform(t *testing.T) {
	platform := DetectPlatform()

	if platform == nil {
		t.Fatal("DetectPlatform() returned nil")
	}

	// Test that platform detection works
	detectedPlatform := platform.GetPlatform()
	if detectedPlatform == "" {
		t.Error("Platform detection returned empty string")
	}

	// Should match current OS
	expectedPlatforms := map[string]string{
		"darwin":  "macos",
		"linux":   "linux",
		"windows": "windows",
	}

	if expected, exists := expectedPlatforms[runtime.GOOS]; exists {
		if detectedPlatform != expected {
			t.Errorf("Expected platform '%s' for GOOS '%s', got '%s'", expected, runtime.GOOS, detectedPlatform)
		}
	}

	// Test architecture
	arch := platform.GetArchitecture()
	if arch == "" {
		t.Error("Architecture detection returned empty string")
	}
	if arch != runtime.GOARCH {
		t.Errorf("Expected architecture '%s', got '%s'", runtime.GOARCH, arch)
	}
}

func TestPlatformPathOperations(t *testing.T) {
	platform := DetectPlatform()

	t.Run("GetHomeDir", func(t *testing.T) {
		homeDir, err := platform.GetHomeDir()
		if err != nil {
			t.Errorf("GetHomeDir() failed: %v", err)
		}
		if homeDir == "" {
			t.Error("GetHomeDir() returned empty string")
		}
		if !strings.Contains(homeDir, "/") && !strings.Contains(homeDir, "\\") {
			t.Error("GetHomeDir() doesn't look like a valid path")
		}
	})

	t.Run("GetConfigDir", func(t *testing.T) {
		configDir, err := platform.GetConfigDir()
		if err != nil {
			t.Errorf("GetConfigDir() failed: %v", err)
		}
		if configDir == "" {
			t.Error("GetConfigDir() returned empty string")
		}
	})

	t.Run("GetAppSupportDir", func(t *testing.T) {
		appSupportDir, err := platform.GetAppSupportDir()
		if err != nil {
			t.Errorf("GetAppSupportDir() failed: %v", err)
		}
		if appSupportDir == "" {
			t.Error("GetAppSupportDir() returned empty string")
		}
	})

	t.Run("ExpandPath", func(t *testing.T) {
		testCases := []struct {
			name          string
			input         string
			shouldContain string
		}{
			{
				name:          "Home directory expansion",
				input:         "~",
				shouldContain: "/", // Should contain path separator
			},
			{
				name:          "Home subdirectory expansion",
				input:         "~/test",
				shouldContain: "test",
			},
			{
				name:          "Environment variable expansion",
				input:         "$HOME",
				shouldContain: "/", // Should expand to a path
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := platform.ExpandPath(tc.input)
				if err != nil {
					t.Errorf("ExpandPath('%s') failed: %v", tc.input, err)
				}
				if !strings.Contains(result, tc.shouldContain) {
					t.Errorf("ExpandPath('%s') = '%s', should contain '%s'", tc.input, result, tc.shouldContain)
				}
			})
		}
	})

	t.Run("ExpandPath error cases", func(t *testing.T) {
		testCases := []string{
			"", // Empty path should error
		}

		for _, input := range testCases {
			_, err := platform.ExpandPath(input)
			if err == nil {
				t.Errorf("ExpandPath('%s') should have returned an error", input)
			}
		}
	})
}

func TestPlatformPathSeparator(t *testing.T) {
	platform := DetectPlatform()
	separator := platform.GetPathSeparator()

	// Should be either : or ;
	if separator != ":" && separator != ";" {
		t.Errorf("GetPathSeparator() returned '%s', expected ':' or ';'", separator)
	}

	// Windows should use ;, others should use :
	expectedSeparator := ":"
	if runtime.GOOS == "windows" {
		expectedSeparator = ";"
	}

	if separator != expectedSeparator {
		t.Errorf("Expected path separator '%s' for platform '%s', got '%s'", expectedSeparator, runtime.GOOS, separator)
	}
}

func TestApplicationDetection(t *testing.T) {
	platform := DetectPlatform()

	t.Run("DetectApplication", func(t *testing.T) {
		// Test with a command that should exist on most systems
		testApps := []string{"ls", "cat", "echo"}

		for _, app := range testApps {
			info, err := platform.DetectApplication(app)
			if err != nil {
				t.Errorf("DetectApplication('%s') failed: %v", app, err)
				continue
			}

			if info == nil {
				t.Errorf("DetectApplication('%s') returned nil info", app)
				continue
			}

			if info.Name != app {
				t.Errorf("Expected app name '%s', got '%s'", app, info.Name)
			}

			// On most systems, basic commands like ls, cat, echo should be found
			if !info.Installed {
				t.Logf("Application '%s' not found (this may be normal depending on system)", app)
			}
		}
	})

	t.Run("DetectApplication nonexistent", func(t *testing.T) {
		// Test with an application that definitely doesn't exist
		info, err := platform.DetectApplication("definitely-nonexistent-app-12345")
		if err != nil {
			t.Errorf("DetectApplication() should not error for nonexistent apps: %v", err)
		}

		if info == nil {
			t.Error("DetectApplication() should return info even for nonexistent apps")
		} else if info.Installed {
			t.Error("Nonexistent application should not be marked as installed")
		}
	})
}

func TestApplicationPaths(t *testing.T) {
	platform := DetectPlatform()

	t.Run("GetApplicationPaths", func(t *testing.T) {
		// Test with common application names
		testApps := []string{"git", "ssh", "fish"}

		for _, app := range testApps {
			paths, err := platform.GetApplicationPaths(app)
			if err != nil {
				t.Errorf("GetApplicationPaths('%s') failed: %v", app, err)
				continue
			}

			if len(paths) == 0 {
				t.Errorf("GetApplicationPaths('%s') returned no paths", app)
				continue
			}

			// Should have at least a config path
			if _, exists := paths["config"]; !exists {
				t.Errorf("GetApplicationPaths('%s') missing 'config' path", app)
			}

			// Verify paths are reasonable
			for pathType, path := range paths {
				if path == "" {
					t.Errorf("GetApplicationPaths('%s') returned empty path for '%s'", app, pathType)
				}
			}
		}
	})
}

func TestPlatformSpecificBehavior(t *testing.T) {
	platform := DetectPlatform()

	// Test platform-specific behaviors
	switch runtime.GOOS {
	case "darwin":
		t.Run("macOS specific tests", func(t *testing.T) {
			// Test Application Support directory
			appSupportDir, err := platform.GetAppSupportDir()
			if err != nil {
				t.Errorf("GetAppSupportDir() failed on macOS: %v", err)
			}
			if !strings.Contains(appSupportDir, "Library/Application Support") {
				t.Errorf("Expected macOS app support path to contain 'Library/Application Support', got: %s", appSupportDir)
			}
		})

	case "linux":
		t.Run("Linux specific tests", func(t *testing.T) {
			// Test XDG compliance
			configDir, err := platform.GetConfigDir()
			if err != nil {
				t.Errorf("GetConfigDir() failed on Linux: %v", err)
			}
			if !strings.Contains(configDir, ".config") {
				t.Errorf("Expected Linux config path to contain '.config', got: %s", configDir)
			}
		})

	case "windows":
		t.Run("Windows specific tests", func(t *testing.T) {
			// Test Windows paths
			configDir, err := platform.GetConfigDir()
			if err != nil {
				t.Errorf("GetConfigDir() failed on Windows: %v", err)
			}
			if !strings.Contains(configDir, "AppData") && !strings.Contains(configDir, "APPDATA") {
				t.Errorf("Expected Windows config path to contain 'AppData', got: %s", configDir)
			}
		})
	}
}

func TestPlatformInterfaces(t *testing.T) {
	// Test that all platform providers implement the interface correctly
	platforms := []PlatformProvider{
		&DarwinProvider{BasePlatform{platform: "macos", architecture: "amd64"}},
		&LinuxProvider{BasePlatform{platform: "linux", architecture: "amd64"}},
		&WindowsProvider{BasePlatform{platform: "windows", architecture: "amd64"}},
	}

	for _, p := range platforms {
		t.Run("Platform: "+p.GetPlatform(), func(t *testing.T) {
			// Test that all interface methods are implemented
			if p.GetPlatform() == "" {
				t.Error("GetPlatform() returned empty string")
			}
			if p.GetArchitecture() == "" {
				t.Error("GetArchitecture() returned empty string")
			}
			if p.GetPathSeparator() == "" {
				t.Error("GetPathSeparator() returned empty string")
			}

			// Test path operations (these might fail but shouldn't panic)
			_, err := p.GetHomeDir()
			if err != nil {
				t.Logf("GetHomeDir() error (may be expected in test): %v", err)
			}

			_, err = p.GetConfigDir()
			if err != nil {
				t.Logf("GetConfigDir() error (may be expected in test): %v", err)
			}
		})
	}
}

func TestPlatformFileOperations(t *testing.T) {
	platform := DetectPlatform()

	// Create temporary directory for testing
	tempDir := t.TempDir()

	t.Run("CopyFile", func(t *testing.T) {
		// Create source file
		sourceFile := filepath.Join(tempDir, "source.txt")
		sourceContent := "test content for file operations"

		err := os.WriteFile(sourceFile, []byte(sourceContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		// Test copy operation
		targetFile := filepath.Join(tempDir, "target.txt")
		err = platform.CopyFile(sourceFile, targetFile)
		if err != nil {
			t.Errorf("CopyFile failed: %v", err)
		}

		// Verify target file exists and has correct content
		targetContent, err := os.ReadFile(targetFile)
		if err != nil {
			t.Errorf("Failed to read target file: %v", err)
		}

		if string(targetContent) != sourceContent {
			t.Errorf("Target file content mismatch. Expected %s, got %s", sourceContent, string(targetContent))
		}
	})

	t.Run("CreateSymlink", func(t *testing.T) {
		// Create source file
		sourceFile := filepath.Join(tempDir, "symlink_source.txt")
		err := os.WriteFile(sourceFile, []byte("symlink test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		// Test symlink creation
		symlinkTarget := filepath.Join(tempDir, "test_symlink.txt")
		err = platform.CreateSymlink(sourceFile, symlinkTarget)
		if err != nil {
			// On some systems (like Windows without dev mode), symlinks might fail
			t.Logf("CreateSymlink failed (may be expected on this platform): %v", err)
			return
		}

		// Verify symlink exists
		linkInfo, err := os.Lstat(symlinkTarget)
		if err != nil {
			t.Errorf("Failed to stat symlink: %v", err)
		}

		if linkInfo.Mode()&os.ModeSymlink == 0 {
			t.Error("Created file is not a symlink")
		}
	})

	t.Run("Error cases", func(t *testing.T) {
		// Test CopyFile with non-existent source
		err := platform.CopyFile("/nonexistent/source.txt", filepath.Join(tempDir, "target.txt"))
		if err == nil {
			t.Error("CopyFile should error with non-existent source")
		}

		// Test CreateSymlink with non-existent source
		err = platform.CreateSymlink("/nonexistent/source.txt", filepath.Join(tempDir, "symlink.txt"))
		if err == nil {
			t.Error("CreateSymlink should error with non-existent source")
		}

		// Test SetPermissions with non-existent file
		err = platform.SetPermissions("/nonexistent/file.txt", 0644)
		if err == nil {
			t.Error("SetPermissions should error with non-existent file")
		}

		// Test GetFileInfo with non-existent file
		_, err = platform.GetFileInfo("/nonexistent/file.txt")
		if err == nil {
			t.Error("GetFileInfo should error for non-existent file")
		}

		// Test ExpandPath with empty path
		_, err = platform.ExpandPath("")
		if err == nil {
			t.Error("ExpandPath should error with empty path")
		}
	})
}
