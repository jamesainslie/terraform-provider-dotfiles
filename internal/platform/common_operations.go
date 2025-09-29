package platform

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// PlatformExpander defines the interface for path expansion functionality
type PlatformExpander interface {
	ExpandPath(path string) (string, error)
}

// CreateSymlinkCommon creates a symbolic link using the platform's path expansion.
func CreateSymlinkCommon(expander PlatformExpander, source, target string) error {
	// Expand paths
	expandedSource, err := expander.ExpandPath(source)
	if err != nil {
		return fmt.Errorf("unable to expand source path: %w", err)
	}

	expandedTarget, err := expander.ExpandPath(target)
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

// CopyFileCommon copies a file from source to target using the platform's path expansion.
func CopyFileCommon(expander PlatformExpander, source, target string) error {
	// Expand paths
	expandedSource, err := expander.ExpandPath(source)
	if err != nil {
		return fmt.Errorf("unable to expand source path: %w", err)
	}

	expandedTarget, err := expander.ExpandPath(target)
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
