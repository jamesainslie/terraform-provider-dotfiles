// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fileops

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/platform"
)

// TestEnhancedBackupConfiguration tests the enhanced backup configuration
func TestEnhancedBackupConfiguration(t *testing.T) {
	t.Run("EnhancedBackupConfig validation", func(t *testing.T) {
		testCases := []struct {
			name        string
			config      *EnhancedBackupConfig
			expectError bool
		}{
			{
				name:        "nil config",
				config:      nil,
				expectError: false,
			},
			{
				name: "valid config",
				config: &EnhancedBackupConfig{
					Enabled:         true,
					Directory:       "/tmp/backups",
					RetentionPolicy: "30d",
					Compression:     true,
					MaxBackups:      50,
					BackupFormat:    "timestamped",
				},
				expectError: false,
			},
			{
				name: "invalid retention policy",
				config: &EnhancedBackupConfig{
					Enabled:         true,
					Directory:       "/tmp/backups",
					RetentionPolicy: "invalid",
					MaxBackups:      50,
					BackupFormat:    "timestamped",
				},
				expectError: true,
			},
			{
				name: "invalid backup format",
				config: &EnhancedBackupConfig{
					Enabled:         true,
					Directory:       "/tmp/backups",
					RetentionPolicy: "30d",
					MaxBackups:      50,
					BackupFormat:    "invalid",
				},
				expectError: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := ValidateEnhancedBackupConfig(tc.config)
				if tc.expectError && err == nil {
					t.Error("Expected error but got none")
				}
				if !tc.expectError && err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			})
		}
	})
}

// TestBackupFormats tests different backup naming formats
func TestBackupFormats(t *testing.T) {
	tempDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tempDir, "test.conf")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	backupDir := filepath.Join(tempDir, "backups")
	platformProvider := platform.DetectPlatform()
	fm := NewFileManager(platformProvider, false)

	testCases := []struct {
		name         string
		format       string
		expectedName string
	}{
		{
			name:         "timestamped format",
			format:       "timestamped",
			expectedName: "test.conf.backup.",
		},
		{
			name:         "numbered format",
			format:       "numbered",
			expectedName: "test.conf.backup.001",
		},
		{
			name:         "git-style format",
			format:       "git_style",
			expectedName: "test.conf.backup.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &EnhancedBackupConfig{
				Enabled:      true,
				Directory:    backupDir,
				BackupFormat: tc.format,
				MaxBackups:   10,
			}

			backupPath, err := fm.CreateEnhancedBackup(testFile, config)
			if err != nil {
				t.Errorf("CreateEnhancedBackup failed: %v", err)
				return
			}

			backupName := filepath.Base(backupPath)
			if tc.format == "numbered" {
				if backupName != tc.expectedName {
					t.Errorf("Expected backup name %s, got %s", tc.expectedName, backupName)
				}
			} else {
				if !strings.Contains(backupName, tc.expectedName) {
					t.Errorf("Expected backup name to contain %s, got %s", tc.expectedName, backupName)
				}
			}

			// Verify backup exists
			if !PathExists(backupPath) {
				t.Error("Backup file should exist")
			}
		})
	}
}

// TestBackupCompression tests backup compression functionality
func TestBackupCompression(t *testing.T) {
	tempDir := t.TempDir()

	// Create test file with some content
	testContent := strings.Repeat("This is test content for compression testing. ", 100)
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	backupDir := filepath.Join(tempDir, "backups")
	platformProvider := platform.DetectPlatform()
	fm := NewFileManager(platformProvider, false)

	t.Run("Compressed backup", func(t *testing.T) {
		config := &EnhancedBackupConfig{
			Enabled:      true,
			Directory:    backupDir,
			BackupFormat: "timestamped",
			Compression:  true,
			MaxBackups:   10,
		}

		backupPath, err := fm.CreateEnhancedBackup(testFile, config)
		if err != nil {
			t.Fatalf("CreateEnhancedBackup failed: %v", err)
		}

		// Backup should have .gz extension
		if !strings.HasSuffix(backupPath, ".gz") {
			t.Error("Compressed backup should have .gz extension")
		}

		// Verify we can decompress it
		backupFile, err := os.Open(backupPath)
		if err != nil {
			t.Fatalf("Failed to open backup file: %v", err)
		}
		defer backupFile.Close()

		gzReader, err := gzip.NewReader(backupFile)
		if err != nil {
			t.Fatalf("Failed to create gzip reader: %v", err)
		}
		defer gzReader.Close()

		decompressed, err := io.ReadAll(gzReader)
		if err != nil {
			t.Fatalf("Failed to read decompressed content: %v", err)
		}

		if string(decompressed) != testContent {
			t.Error("Decompressed content does not match original")
		}
	})

	t.Run("Uncompressed backup", func(t *testing.T) {
		config := &EnhancedBackupConfig{
			Enabled:      true,
			Directory:    backupDir,
			BackupFormat: "timestamped",
			Compression:  false,
			MaxBackups:   10,
		}

		backupPath, err := fm.CreateEnhancedBackup(testFile, config)
		if err != nil {
			t.Fatalf("CreateEnhancedBackup failed: %v", err)
		}

		// Backup should not have .gz extension
		if strings.HasSuffix(backupPath, ".gz") {
			t.Error("Uncompressed backup should not have .gz extension")
		}

		// Verify content is readable directly
		backupContent, err := os.ReadFile(backupPath)
		if err != nil {
			t.Fatalf("Failed to read backup file: %v", err)
		}

		if string(backupContent) != testContent {
			t.Error("Backup content does not match original")
		}
	})
}

// TestBackupRetention tests backup retention management
func TestBackupRetention(t *testing.T) {
	tempDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tempDir, "config.txt")
	err = os.WriteFile(testFile, []byte("initial content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	backupDir := filepath.Join(tempDir, "backups")
	platformProvider := platform.DetectPlatform()
	fm := NewFileManager(platformProvider, false)

	config := &EnhancedBackupConfig{
		Enabled:      true,
		Directory:    backupDir,
		BackupFormat: "numbered",
		MaxBackups:   3, // Keep only 3 backups
	}

	// Create multiple backups
	backupPaths := make([]string, 0)
	for i := 1; i <= 5; i++ {
		// Update file content
		content := fmt.Sprintf("content version %d", i)
		err = os.WriteFile(testFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to update test file: %v", err)
		}

		backupPath, err := fm.CreateEnhancedBackup(testFile, config)
		if err != nil {
			t.Fatalf("CreateEnhancedBackup failed: %v", err)
		}
		backupPaths = append(backupPaths, backupPath)

		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	// Check retention - should only have last 3 backups
	backupFiles, err := filepath.Glob(filepath.Join(backupDir, "config.txt.backup.*"))
	if err != nil {
		t.Fatalf("Failed to list backup files: %v", err)
	}

	if len(backupFiles) != 3 {
		t.Errorf("Expected 3 backup files after retention, got %d", len(backupFiles))
	}

	// Verify the newest backups are retained
	for _, backupPath := range backupPaths[2:] { // Last 3 backups
		if !PathExists(backupPath) {
			t.Errorf("Expected backup file %s to exist after retention", backupPath)
		}
	}

	// Verify the oldest backups are removed
	for _, backupPath := range backupPaths[:2] { // First 2 backups
		if PathExists(backupPath) {
			t.Errorf("Expected backup file %s to be removed by retention", backupPath)
		}
	}
}

// TestBackupMetadata tests backup metadata storage and retrieval
func TestBackupMetadata(t *testing.T) {
	tempDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tempDir, "test.conf")
	testContent := "test configuration content"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	backupDir := filepath.Join(tempDir, "backups")
	platformProvider := platform.DetectPlatform()
	fm := NewFileManager(platformProvider, false)

	config := &EnhancedBackupConfig{
		Enabled:        true,
		Directory:      backupDir,
		BackupFormat:   "timestamped",
		BackupMetadata: true,
		BackupIndex:    true,
	}

	t.Run("Create backup with metadata", func(t *testing.T) {
		backupPath, err := fm.CreateEnhancedBackup(testFile, config)
		if err != nil {
			t.Fatalf("CreateEnhancedBackup failed: %v", err)
		}

		// Check if metadata file exists
		metadataPath := backupPath + ".meta"
		if !PathExists(metadataPath) {
			t.Error("Backup metadata file should exist")
		}

		// Verify metadata content
		metadata, err := LoadBackupMetadata(metadataPath)
		if err != nil {
			t.Fatalf("Failed to load backup metadata: %v", err)
		}

		if metadata.OriginalPath != testFile {
			t.Errorf("Expected original path %s, got %s", testFile, metadata.OriginalPath)
		}
		if metadata.BackupPath != backupPath {
			t.Errorf("Expected backup path %s, got %s", backupPath, metadata.BackupPath)
		}
		if metadata.Checksum == "" {
			t.Error("Backup metadata should include checksum")
		}
	})

	t.Run("Backup index management", func(t *testing.T) {
		// Create multiple backups
		for i := 0; i < 3; i++ {
			content := fmt.Sprintf("content version %d", i)
			err = os.WriteFile(testFile, []byte(content), 0644)
			if err != nil {
				t.Fatalf("Failed to update test file: %v", err)
			}

			_, err = fm.CreateEnhancedBackup(testFile, config)
			if err != nil {
				t.Fatalf("CreateEnhancedBackup failed: %v", err)
			}
		}

		// Check if backup index exists
		indexPath := filepath.Join(backupDir, ".backup_index.json")
		if !PathExists(indexPath) {
			t.Error("Backup index file should exist")
		}

		// Verify index content
		index, err := LoadBackupIndex(indexPath)
		if err != nil {
			t.Fatalf("Failed to load backup index: %v", err)
		}

		if len(index.Backups) != 4 { // 3 from loop + 1 from previous test
			t.Errorf("Expected 4 backups in index, got %d", len(index.Backups))
		}
	})
}

// TestIncrementalBackup tests incremental backup functionality
func TestIncrementalBackup(t *testing.T) {
	tempDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tempDir, "config.txt")
	originalContent := "original content"
	err = os.WriteFile(testFile, []byte(originalContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	backupDir := filepath.Join(tempDir, "backups")
	platformProvider := platform.DetectPlatform()
	fm := NewFileManager(platformProvider, false)

	config := &EnhancedBackupConfig{
		Enabled:      true,
		Directory:    backupDir,
		BackupFormat: "timestamped",
		Incremental:  true,
		BackupIndex:  true,
	}

	t.Run("First backup should always be created", func(t *testing.T) {
		backupPath, err := fm.CreateEnhancedBackup(testFile, config)
		if err != nil {
			t.Fatalf("CreateEnhancedBackup failed: %v", err)
		}

		if backupPath == "" {
			t.Error("First backup should always be created")
		}
		if !PathExists(backupPath) {
			t.Error("Backup file should exist")
		}
	})

	t.Run("Duplicate backup should be skipped", func(t *testing.T) {
		// Try to backup same content again
		backupPath, err := fm.CreateEnhancedBackup(testFile, config)
		if err != nil {
			t.Fatalf("CreateEnhancedBackup failed: %v", err)
		}

		// Should return empty path indicating no backup was needed
		if backupPath != "" {
			t.Error("Incremental backup should skip duplicate content")
		}
	})

	t.Run("Changed content should create new backup", func(t *testing.T) {
		// Change file content
		newContent := "modified content"
		err = os.WriteFile(testFile, []byte(newContent), 0644)
		if err != nil {
			t.Fatalf("Failed to update test file: %v", err)
		}

		backupPath, err := fm.CreateEnhancedBackup(testFile, config)
		if err != nil {
			t.Fatalf("CreateEnhancedBackup failed: %v", err)
		}

		if backupPath == "" {
			t.Error("Backup should be created when content changes")
		}
		if !PathExists(backupPath) {
			t.Error("New backup file should exist")
		}
	})
}

// Helper function to check if path exists
func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
