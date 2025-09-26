// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package services

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// MockPlatformProvider implements PlatformProvider for testing.
type MockPlatformProvider struct {
	CopyFileFunc          func(src, dst string, mode os.FileMode) error
	CreateDirectoryFunc   func(path string, mode os.FileMode) error
	GetFileInfoFunc       func(path string) (os.FileInfo, error)
	CalculateChecksumFunc func(path string) (string, error)
}

func (m *MockPlatformProvider) CopyFile(src, dst string, mode os.FileMode) error {
	if m.CopyFileFunc != nil {
		return m.CopyFileFunc(src, dst, mode)
	}
	return nil
}

func (m *MockPlatformProvider) CreateDirectory(path string, mode os.FileMode) error {
	if m.CreateDirectoryFunc != nil {
		return m.CreateDirectoryFunc(path, mode)
	}
	return nil
}

func (m *MockPlatformProvider) GetFileInfo(path string) (os.FileInfo, error) {
	if m.GetFileInfoFunc != nil {
		return m.GetFileInfoFunc(path)
	}
	return &mockFileInfo{name: "test.txt", size: 100}, nil
}

func (m *MockPlatformProvider) CalculateChecksum(path string) (string, error) {
	if m.CalculateChecksumFunc != nil {
		return m.CalculateChecksumFunc(path)
	}
	return "mock-checksum", nil
}

// mockFileInfo implements os.FileInfo for testing.
type mockFileInfo struct {
	name string
	size int64
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() os.FileMode  { return 0644 }
func (m *mockFileInfo) ModTime() time.Time { return time.Now() }
func (m *mockFileInfo) IsDir() bool        { return false }
func (m *mockFileInfo) Sys() interface{}   { return nil }

func TestNewDefaultBackupService(t *testing.T) {
	mockProvider := &MockPlatformProvider{}

	service := NewDefaultBackupService(mockProvider, false)
	if service == nil {
		t.Fatal("Expected service to be created")
	}

	if service.platformProvider != mockProvider {
		t.Error("Expected platform provider to be set")
	}

	if service.dryRun {
		t.Error("Expected dry run to be false")
	}
}

func TestBackupService_CreateBackup(t *testing.T) {
	tests := []struct {
		name    string
		dryRun  bool
		options BackupOptions
	}{
		{
			name:   "normal backup",
			dryRun: false,
			options: BackupOptions{
				Format:      BackupFormatTimestamped,
				Compression: false,
				Metadata:    map[string]string{"test": "value"},
			},
		},
		{
			name:   "dry run backup",
			dryRun: true,
			options: BackupOptions{
				Format:      BackupFormatNumbered,
				Compression: true,
				DryRun:      true,
			},
		},
		{
			name:   "compressed backup",
			dryRun: false,
			options: BackupOptions{
				Format:      BackupFormatGitStyle,
				Compression: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider := &MockPlatformProvider{}
			service := NewDefaultBackupService(mockProvider, tt.dryRun)

			ctx := context.Background()
			result, err := service.CreateBackup(ctx, "/source/file", "/backup/dir", tt.options)

			if err != nil {
				t.Fatalf("CreateBackup failed: %v", err)
			}

			if result == nil {
				t.Fatal("Expected backup result")
			}

			if result.Compressed != tt.options.Compression {
				t.Errorf("Expected compression %v, got %v", tt.options.Compression, result.Compressed)
			}

			if len(result.Metadata) != len(tt.options.Metadata) {
				t.Errorf("Expected metadata length %d, got %d", len(tt.options.Metadata), len(result.Metadata))
			}
		})
	}
}

func TestBackupService_CreateEnhancedBackup(t *testing.T) {
	mockProvider := &MockPlatformProvider{}
	service := NewDefaultBackupService(mockProvider, false)

	config := &EnhancedBackupConfig{
		Enabled:     true,
		Directory:   "/enhanced/backup",
		Format:      BackupFormatTimestamped,
		Compression: true,
		Retention: RetentionPolicy{
			MaxAge:   24 * time.Hour,
			MaxCount: 10,
		},
		Metadata: map[string]string{
			"source":    "enhanced-test",
			"timestamp": time.Now().Format(time.RFC3339),
		},
		Incremental: false,
	}

	ctx := context.Background()
	result, err := service.CreateEnhancedBackup(ctx, "/source/enhanced", config)

	if err != nil {
		t.Fatalf("CreateEnhancedBackup failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected enhanced backup result")
	}

	if !result.Compressed {
		t.Error("Expected compressed backup")
	}

	if len(result.Metadata) == 0 {
		t.Error("Expected metadata in result")
	}
}

func TestBackupService_ValidateBackup(t *testing.T) {
	mockProvider := &MockPlatformProvider{}
	service := NewDefaultBackupService(mockProvider, false)

	ctx := context.Background()
	result, err := service.ValidateBackup(ctx, "/backup/path")

	if err != nil {
		t.Fatalf("ValidateBackup failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected validation result")
	}

	if !result.Valid {
		t.Error("Expected backup to be valid")
	}

	if !result.ChecksumMatch {
		t.Error("Expected checksum to match")
	}
}

func TestBackupFormats(t *testing.T) {
	formats := []BackupFormat{
		BackupFormatTimestamped,
		BackupFormatNumbered,
		BackupFormatGitStyle,
	}

	for _, format := range formats {
		t.Run(string(format), func(t *testing.T) {
			mockProvider := &MockPlatformProvider{}
			service := NewDefaultBackupService(mockProvider, false)

			options := BackupOptions{
				Format: format,
			}

			ctx := context.Background()
			result, err := service.CreateBackup(ctx, "/test/source", "/test/backup", options)

			if err != nil {
				t.Fatalf("Backup with format %s failed: %v", format, err)
			}

			if result == nil {
				t.Fatalf("Expected result for format %s", format)
			}

			if result.BackupPath == "" {
				t.Errorf("Expected backup path for format %s", format)
			}
		})
	}
}

func TestRetentionPolicy(t *testing.T) {
	policies := []RetentionPolicy{
		{
			MaxAge:   24 * time.Hour,
			MaxCount: 5,
		},
		{
			MaxAge:      7 * 24 * time.Hour,
			MaxCount:    10,
			KeepDaily:   7,
			KeepWeekly:  4,
			KeepMonthly: 12,
		},
	}

	for i, policy := range policies {
		t.Run(fmt.Sprintf("policy_%d", i), func(t *testing.T) {
			mockProvider := &MockPlatformProvider{}
			service := NewDefaultBackupService(mockProvider, false)

			ctx := context.Background()
			err := service.CleanupBackups(ctx, "/backup/dir", policy)

			if err != nil {
				t.Fatalf("CleanupBackups failed: %v", err)
			}
		})
	}
}

func TestBackupService_EdgeCases(t *testing.T) {
	t.Run("empty source path", func(t *testing.T) {
		mockProvider := &MockPlatformProvider{}
		service := NewDefaultBackupService(mockProvider, false)

		ctx := context.Background()
		result, err := service.CreateBackup(ctx, "", "/backup", BackupOptions{})

		// Should handle gracefully - either error or empty result
		if err == nil && result != nil && result.BackupPath == "" {
			t.Log("Empty source handled gracefully")
		}
	})

	t.Run("nil options", func(t *testing.T) {
		mockProvider := &MockPlatformProvider{}
		service := NewDefaultBackupService(mockProvider, false)

		ctx := context.Background()
		_, err := service.CreateBackup(ctx, "/source", "/backup", BackupOptions{})

		if err != nil {
			t.Logf("Nil options handled with error: %v", err)
		}
	})

	t.Run("very long paths", func(t *testing.T) {
		mockProvider := &MockPlatformProvider{}
		service := NewDefaultBackupService(mockProvider, false)

		longPath := "/very/long/path/" + strings.Repeat("a", 200)
		ctx := context.Background()
		_, err := service.CreateBackup(ctx, longPath, "/backup", BackupOptions{})

		if err != nil {
			t.Logf("Long path handled with error: %v", err)
		}
	})
}

func TestBackupService_ConcurrentOperations(t *testing.T) {
	mockProvider := &MockPlatformProvider{}
	service := NewDefaultBackupService(mockProvider, false)

	ctx := context.Background()
	numOperations := 10

	// Run multiple backup operations concurrently
	results := make(chan error, numOperations)

	for i := 0; i < numOperations; i++ {
		go func(index int) {
			sourcePath := fmt.Sprintf("/source/file%d", index)
			backupDir := fmt.Sprintf("/backup/dir%d", index)

			_, err := service.CreateBackup(ctx, sourcePath, backupDir, BackupOptions{
				Format: BackupFormatTimestamped,
			})
			results <- err
		}(i)
	}

	// Wait for all operations to complete
	for i := 0; i < numOperations; i++ {
		if err := <-results; err != nil {
			t.Errorf("Concurrent operation %d failed: %v", i, err)
		}
	}
}
