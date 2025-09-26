// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package idempotency

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetFileState(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World!"

	// Create test file
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Get file state
	state, err := GetFileState(testFile)
	if err != nil {
		t.Fatalf("GetFileState failed: %v", err)
	}

	if state == nil {
		t.Fatal("Expected file state to be returned")
	}

	if state.Path != testFile {
		t.Errorf("Expected path %s, got %s", testFile, state.Path)
	}

	if state.Size != int64(len(testContent)) {
		t.Errorf("Expected size %d, got %d", len(testContent), state.Size)
	}

	if state.Mode != 0644 {
		t.Errorf("Expected mode 0644, got %o", state.Mode)
	}

	if state.ContentHash == "" {
		t.Error("Expected content hash to be calculated")
	}

	if state.ModTime.IsZero() {
		t.Error("Expected modification time to be set")
	}

	if !state.Exists {
		t.Error("Expected Exists to be true for existing file")
	}
}

func TestGetFileStateNonexistent(t *testing.T) {
	state, err := GetFileState("/nonexistent/file.txt")

	if err == nil {
		t.Error("Expected error for nonexistent file")
	}

	if state != nil {
		t.Error("Expected nil state for nonexistent file")
	}
}

func TestGetFileStateDirectory(t *testing.T) {
	tempDir := t.TempDir()

	state, err := GetFileState(tempDir)
	if err != nil {
		t.Fatalf("GetFileState failed for directory: %v", err)
	}

	if state == nil {
		t.Fatal("Expected directory state to be returned")
	}

	if state.Size != 0 {
		t.Error("Expected size 0 for directory")
	}

	if state.ContentHash != "" {
		t.Error("Expected empty content hash for directory")
	}

	if !state.Exists {
		t.Error("Expected Exists to be true for existing directory")
	}
}

func TestGetDirectoryState(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	files := []string{"file1.txt", "file2.txt", "subdir/file3.txt"}
	for _, file := range files {
		fullPath := filepath.Join(tempDir, file)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		err = os.WriteFile(fullPath, []byte("content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Get directory state (recursive)
	ctx := context.Background()
	state, err := GetDirectoryState(ctx, tempDir, true)
	if err != nil {
		t.Fatalf("GetDirectoryState failed: %v", err)
	}

	if state == nil {
		t.Fatal("Expected directory state to be returned")
	}

	if state.Path != tempDir {
		t.Errorf("Expected path %s, got %s", tempDir, state.Path)
	}

	if len(state.Files) == 0 {
		t.Error("Expected files to be found in directory state")
	}

	// Should find all files (including in subdirectory)
	expectedFiles := 3
	if len(state.Files) != expectedFiles {
		t.Errorf("Expected %d files, got %d", expectedFiles, len(state.Files))
	}

	if state.FileCount != int64(expectedFiles) {
		t.Errorf("Expected file count %d, got %d", expectedFiles, state.FileCount)
	}

	if state.ModTime.IsZero() {
		t.Error("Expected modification time to be set")
	}
}

func TestGetDirectoryStateEmpty(t *testing.T) {
	tempDir := t.TempDir()

	ctx := context.Background()
	state, err := GetDirectoryState(ctx, tempDir, true)
	if err != nil {
		t.Fatalf("GetDirectoryState failed for empty directory: %v", err)
	}

	if state == nil {
		t.Fatal("Expected directory state to be returned")
	}

	if len(state.Files) != 0 {
		t.Errorf("Expected 0 files in empty directory, got %d", len(state.Files))
	}

	if state.FileCount != 0 {
		t.Errorf("Expected file count 0, got %d", state.FileCount)
	}
}

func TestGetDirectoryStateNonRecursive(t *testing.T) {
	tempDir := t.TempDir()

	// Create files in root and subdirectory
	rootFile := filepath.Join(tempDir, "root.txt")
	err := os.WriteFile(rootFile, []byte("root content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create root file: %v", err)
	}

	subdir := filepath.Join(tempDir, "subdir")
	err = os.MkdirAll(subdir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	subFile := filepath.Join(subdir, "sub.txt")
	err = os.WriteFile(subFile, []byte("sub content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create sub file: %v", err)
	}

	// Get directory state (non-recursive)
	ctx := context.Background()
	state, err := GetDirectoryState(ctx, tempDir, false)
	if err != nil {
		t.Fatalf("GetDirectoryState failed: %v", err)
	}

	// Should only find the root file, not the subdirectory file
	if len(state.Files) != 1 {
		t.Errorf("Expected 1 file in non-recursive scan, got %d", len(state.Files))
	}
}

func TestGetDirectoryStateNonexistent(t *testing.T) {
	ctx := context.Background()
	state, err := GetDirectoryState(ctx, "/nonexistent/directory", true)

	if err == nil {
		t.Error("Expected error for nonexistent directory")
	}

	if state != nil {
		t.Error("Expected nil state for nonexistent directory")
	}
}

func TestCompareFileStates(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	// Create initial file
	initialContent := "initial content"
	err := os.WriteFile(testFile, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Get initial state
	state1, err := GetFileState(testFile)
	if err != nil {
		t.Fatalf("Failed to get initial file state: %v", err)
	}

	// Wait a moment to ensure different modification time
	time.Sleep(10 * time.Millisecond)

	// Modify file
	modifiedContent := "modified content"
	err = os.WriteFile(testFile, []byte(modifiedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Get modified state
	state2, err := GetFileState(testFile)
	if err != nil {
		t.Fatalf("Failed to get modified file state: %v", err)
	}

	// Compare states - should detect changes
	hasChanged := CompareFileStates(state1, state2)
	if !hasChanged {
		t.Error("Expected changes to be detected")
	}
}

func TestCompareFileStatesIdentical(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	// Create file
	content := "test content"
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Get state twice
	state1, err := GetFileState(testFile)
	if err != nil {
		t.Fatalf("Failed to get first file state: %v", err)
	}

	state2, err := GetFileState(testFile)
	if err != nil {
		t.Fatalf("Failed to get second file state: %v", err)
	}

	// Compare identical states
	hasChanged := CompareFileStates(state1, state2)
	if hasChanged {
		t.Error("Expected no changes for identical states")
	}
}

func TestCompareFileStatesNil(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	// Create file and get state
	err := os.WriteFile(testFile, []byte("content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	state, err := GetFileState(testFile)
	if err != nil {
		t.Fatalf("Failed to get file state: %v", err)
	}

	// Compare with nil states
	hasChanged1 := CompareFileStates(nil, state)
	if !hasChanged1 {
		t.Error("Expected changes when comparing nil to valid state")
	}

	hasChanged2 := CompareFileStates(state, nil)
	if !hasChanged2 {
		t.Error("Expected changes when comparing valid state to nil")
	}

	hasChanged3 := CompareFileStates(nil, nil)
	if hasChanged3 {
		t.Error("Expected no changes when comparing nil to nil")
	}
}

func TestCompareDirectoryStates(t *testing.T) {
	tempDir := t.TempDir()

	// Create initial files
	initialFiles := []string{"file1.txt", "file2.txt"}
	for _, file := range initialFiles {
		fullPath := filepath.Join(tempDir, file)
		err := os.WriteFile(fullPath, []byte("content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create initial file %s: %v", file, err)
		}
	}

	// Get initial state
	ctx := context.Background()
	state1, err := GetDirectoryState(ctx, tempDir, true)
	if err != nil {
		t.Fatalf("Failed to get initial directory state: %v", err)
	}

	// Add another file
	newFile := filepath.Join(tempDir, "file3.txt")
	err = os.WriteFile(newFile, []byte("new content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create new file: %v", err)
	}

	// Get modified state
	state2, err := GetDirectoryState(ctx, tempDir, true)
	if err != nil {
		t.Fatalf("Failed to get modified directory state: %v", err)
	}

	// Compare states - should detect changes
	hasChanged := CompareDirectoryStates(state1, state2)
	if !hasChanged {
		t.Error("Expected changes to be detected")
	}
}

func TestCompareDirectoryStatesIdentical(t *testing.T) {
	tempDir := t.TempDir()

	// Create files
	files := []string{"file1.txt", "file2.txt"}
	for _, file := range files {
		fullPath := filepath.Join(tempDir, file)
		err := os.WriteFile(fullPath, []byte("content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}

	// Get state twice
	ctx := context.Background()
	state1, err := GetDirectoryState(ctx, tempDir, true)
	if err != nil {
		t.Fatalf("Failed to get first directory state: %v", err)
	}

	state2, err := GetDirectoryState(ctx, tempDir, true)
	if err != nil {
		t.Fatalf("Failed to get second directory state: %v", err)
	}

	// Compare identical states
	hasChanged := CompareDirectoryStates(state1, state2)
	if hasChanged {
		t.Error("Expected no changes for identical directory states")
	}
}

func TestFileStateWithSymlink(t *testing.T) {
	tempDir := t.TempDir()
	targetFile := filepath.Join(tempDir, "target.txt")
	symlinkFile := filepath.Join(tempDir, "link.txt")

	// Create target file
	err := os.WriteFile(targetFile, []byte("target content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	// Create symlink
	err = os.Symlink(targetFile, symlinkFile)
	if err != nil {
		t.Skipf("Skipping symlink test (may not be supported): %v", err)
	}

	// Get symlink state
	state, err := GetFileState(symlinkFile)
	if err != nil {
		t.Fatalf("Failed to get symlink state: %v", err)
	}

	if state == nil {
		t.Fatal("Expected symlink state to be returned")
	}

	if !state.IsSymlink {
		t.Error("Expected IsSymlink to be true")
	}

	if state.SymlinkTarget != targetFile {
		t.Errorf("Expected symlink target %s, got %s", targetFile, state.SymlinkTarget)
	}

	t.Logf("Symlink state: size=%d, mode=%o, target=%s", state.Size, state.Mode, state.SymlinkTarget)
}

func TestLargeDirectoryState(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large directory test in short mode")
	}

	tempDir := t.TempDir()
	numFiles := 100

	// Create many files
	for i := 0; i < numFiles; i++ {
		fileName := filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i))
		content := fmt.Sprintf("content for file %d", i)
		err := os.WriteFile(fileName, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create file %d: %v", i, err)
		}
	}

	// Get directory state
	start := time.Now()
	ctx := context.Background()
	state, err := GetDirectoryState(ctx, tempDir, true)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to get large directory state: %v", err)
	}

	if state.FileCount != int64(numFiles) {
		t.Errorf("Expected file count %d, got %d", numFiles, state.FileCount)
	}

	t.Logf("Processed %d files in %v", numFiles, duration)
}

func TestContentHashConsistency(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "checksum_test.txt")
	content := "consistent content for checksum testing"

	// Create file
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Get state multiple times
	hashes := make([]string, 3)
	for i := 0; i < 3; i++ {
		state, err := GetFileState(testFile)
		if err != nil {
			t.Fatalf("Failed to get file state %d: %v", i, err)
		}
		hashes[i] = state.ContentHash
	}

	// All hashes should be identical
	for i := 1; i < len(hashes); i++ {
		if hashes[i] != hashes[0] {
			t.Errorf("Content hash inconsistency: %s != %s", hashes[i], hashes[0])
		}
	}

	// Hash should be non-empty and reasonable length
	if len(hashes[0]) < 32 {
		t.Errorf("Content hash seems too short: %s (length %d)", hashes[0], len(hashes[0]))
	}
}

func TestEdgeCaseFilenames(t *testing.T) {
	tempDir := t.TempDir()

	// Test various edge case filenames
	edgeCases := []string{
		"normal.txt",
		"file with spaces.txt",
		"file-with-dashes.txt",
		"file_with_underscores.txt",
		"file.with.dots.txt",
		"UPPERCASE.TXT",
		"MiXeDcAsE.TxT",
	}

	// Add Unicode filenames if supported
	unicodeFiles := []string{
		"café.txt",
		"naïve.txt",
		"résumé.txt",
	}

	// Test creating and getting state for each filename
	for _, filename := range append(edgeCases, unicodeFiles...) {
		t.Run(filename, func(t *testing.T) {
			fullPath := filepath.Join(tempDir, filename)
			content := fmt.Sprintf("content for %s", filename)

			err := os.WriteFile(fullPath, []byte(content), 0644)
			if err != nil {
				t.Logf("Skipping file %s (creation failed): %v", filename, err)
				return
			}

			state, err := GetFileState(fullPath)
			if err != nil {
				t.Errorf("Failed to get state for file %s: %v", filename, err)
				return
			}

			if state.Path != fullPath {
				t.Errorf("Path mismatch for %s: expected %s, got %s", filename, fullPath, state.Path)
			}

			if state.Size != int64(len(content)) {
				t.Errorf("Size mismatch for %s: expected %d, got %d", filename, len(content), state.Size)
			}
		})
	}
}
