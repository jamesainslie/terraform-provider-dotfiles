// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fileops

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// EnhancedBackupConfig represents enhanced backup configuration
type EnhancedBackupConfig struct {
	Enabled         bool
	Directory       string
	RetentionPolicy string
	Compression     bool
	Incremental     bool
	MaxBackups      int64
	BackupFormat    string
	BackupMetadata  bool
	BackupIndex     bool
}

// BackupMetadata represents metadata for a backup file
type BackupMetadata struct {
	OriginalPath string    `json:"original_path"`
	BackupPath   string    `json:"backup_path"`
	Timestamp    time.Time `json:"timestamp"`
	Checksum     string    `json:"checksum"`
	Compressed   bool      `json:"compressed"`
	OriginalSize int64     `json:"original_size"`
	BackupSize   int64     `json:"backup_size"`
	FileMode     string    `json:"file_mode"`
}

// BackupIndex represents an index of all backups
type BackupIndex struct {
	LastUpdated time.Time        `json:"last_updated"`
	Backups     []BackupMetadata `json:"backups"`
}

// CreateEnhancedBackup creates a backup with enhanced features
func (fm *FileManager) CreateEnhancedBackup(filePath string, config *EnhancedBackupConfig) (string, error) {
	if config == nil || !config.Enabled {
		return "", nil
	}

	if fm.dryRun {
		return fmt.Sprintf("%s/enhanced-backup-dry-run", config.Directory), nil
	}

	// Check if incremental backup is needed
	if config.Incremental && config.BackupIndex {
		if shouldSkipBackup, err := fm.shouldSkipIncrementalBackup(filePath, config); err != nil {
			return "", fmt.Errorf("failed to check incremental backup: %w", err)
		} else if shouldSkipBackup {
			return "", nil // Skip backup - content hasn't changed
		}
	}

	// Create backup directory
	err := os.MkdirAll(config.Directory, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate backup filename based on format
	backupPath, err := fm.generateBackupPath(filePath, config)
	if err != nil {
		return "", fmt.Errorf("failed to generate backup path: %w", err)
	}

	// Create the backup
	if config.Compression {
		err = fm.createCompressedBackup(filePath, backupPath)
	} else {
		err = fm.platform.CopyFile(filePath, backupPath)
	}
	if err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	// Create metadata if requested
	if config.BackupMetadata {
		err = fm.createBackupMetadata(filePath, backupPath, config)
		if err != nil {
			return "", fmt.Errorf("failed to create backup metadata: %w", err)
		}
	}

	// Update backup index if requested
	if config.BackupIndex {
		err = fm.updateBackupIndex(filePath, backupPath, config)
		if err != nil {
			return "", fmt.Errorf("failed to update backup index: %w", err)
		}
	}

	// Apply retention policy
	if config.MaxBackups > 0 {
		err = fm.applyRetentionPolicy(filePath, config)
		if err != nil {
			return "", fmt.Errorf("failed to apply retention policy: %w", err)
		}
	}

	return backupPath, nil
}

// generateBackupPath generates backup file path based on format
func (fm *FileManager) generateBackupPath(filePath string, config *EnhancedBackupConfig) (string, error) {
	fileName := filepath.Base(filePath)
	var backupName string

	switch config.BackupFormat {
	case "timestamped":
		timestamp := time.Now().Format("2006-01-02-150405")
		backupName = fmt.Sprintf("%s.backup.%s", fileName, timestamp)
	case "numbered":
		backupName = fm.generateNumberedBackupName(fileName, config.Directory)
	case "git_style":
		checksum, err := fm.calculateFileChecksum(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to calculate checksum: %w", err)
		}
		backupName = fmt.Sprintf("%s.backup.%s", fileName, checksum[:8])
	default:
		return "", fmt.Errorf("unsupported backup format: %s", config.BackupFormat)
	}

	backupPath := filepath.Join(config.Directory, backupName)
	if config.Compression {
		backupPath += ".gz"
	}

	return backupPath, nil
}

// generateNumberedBackupName generates a numbered backup name
func (fm *FileManager) generateNumberedBackupName(fileName, backupDir string) string {
	pattern := filepath.Join(backupDir, fileName+".backup.*")
	existing, _ := filepath.Glob(pattern)

	nextNumber := 1
	if len(existing) > 0 {
		maxNumber := 0
		for _, path := range existing {
			base := filepath.Base(path)
			// Remove .gz extension if present for parsing (safe unconditionally)
			base = strings.TrimSuffix(base, ".gz")

			parts := strings.Split(base, ".")
			if len(parts) >= 3 {
				// The number should be the last part after removing .gz
				numberPart := parts[len(parts)-1]
				if num, err := strconv.Atoi(numberPart); err == nil {
					if num > maxNumber {
						maxNumber = num
					}
				}
			}
		}
		nextNumber = maxNumber + 1
	}

	return fmt.Sprintf("%s.backup.%03d", fileName, nextNumber)
}

// createCompressedBackup creates a compressed backup using gzip
func (fm *FileManager) createCompressedBackup(sourcePath, backupPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	backupFile, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer backupFile.Close()

	gzWriter := gzip.NewWriter(backupFile)
	defer gzWriter.Close()

	_, err = io.Copy(gzWriter, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to compress file: %w", err)
	}

	return nil
}

// createBackupMetadata creates metadata for a backup
func (fm *FileManager) createBackupMetadata(originalPath, backupPath string, config *EnhancedBackupConfig) error {
	// Get file info
	originalInfo, err := os.Stat(originalPath)
	if err != nil {
		return fmt.Errorf("failed to stat original file: %w", err)
	}

	backupInfo, err := os.Stat(backupPath)
	if err != nil {
		return fmt.Errorf("failed to stat backup file: %w", err)
	}

	// Calculate checksum
	checksum, err := fm.calculateFileChecksum(originalPath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Create metadata
	metadata := BackupMetadata{
		OriginalPath: originalPath,
		BackupPath:   backupPath,
		Timestamp:    time.Now(),
		Checksum:     checksum,
		Compressed:   config.Compression,
		OriginalSize: originalInfo.Size(),
		BackupSize:   backupInfo.Size(),
		FileMode:     originalInfo.Mode().String(),
	}

	// Write metadata to file
	metadataPath := backupPath + ".meta"
	metadataFile, err := os.Create(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to create metadata file: %w", err)
	}
	defer metadataFile.Close()

	encoder := json.NewEncoder(metadataFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(metadata); err != nil {
		return fmt.Errorf("failed to encode metadata: %w", err)
	}

	return nil
}

// updateBackupIndex updates the backup index
func (fm *FileManager) updateBackupIndex(originalPath, backupPath string, config *EnhancedBackupConfig) error {
	indexPath := filepath.Join(config.Directory, ".backup_index.json")

	// Load existing index or create new one
	index := &BackupIndex{
		LastUpdated: time.Now(),
		Backups:     make([]BackupMetadata, 0),
	}

	if _, err := os.Stat(indexPath); err == nil {
		if loadedIndex, loadErr := LoadBackupIndex(indexPath); loadErr == nil {
			index = loadedIndex
		}
	}

	// Calculate checksum for new backup
	checksum, err := fm.calculateFileChecksum(originalPath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Add new backup to index
	originalInfo, _ := os.Stat(originalPath)
	backupInfo, _ := os.Stat(backupPath)

	newBackup := BackupMetadata{
		OriginalPath: originalPath,
		BackupPath:   backupPath,
		Timestamp:    time.Now(),
		Checksum:     checksum,
		Compressed:   config.Compression,
		OriginalSize: originalInfo.Size(),
		BackupSize:   backupInfo.Size(),
		FileMode:     originalInfo.Mode().String(),
	}

	index.Backups = append(index.Backups, newBackup)
	index.LastUpdated = time.Now()

	// Write updated index
	return fm.saveBackupIndex(index, indexPath)
}

// shouldSkipIncrementalBackup checks if incremental backup should be skipped
func (fm *FileManager) shouldSkipIncrementalBackup(filePath string, config *EnhancedBackupConfig) (bool, error) {
	indexPath := filepath.Join(config.Directory, ".backup_index.json")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return false, nil // No index exists, create first backup
	}

	// Load index
	index, loadErr := LoadBackupIndex(indexPath)
	if loadErr != nil {
		// Treat as empty index and proceed to create a backup.
		index = &BackupIndex{LastUpdated: time.Now(), Backups: nil}
	}

	// Calculate current file checksum
	currentChecksum, err := fm.calculateFileChecksum(filePath)
	if err != nil {
		return false, err
	}

	// Check if we already have a backup with this checksum
	for _, backup := range index.Backups {
		if backup.OriginalPath == filePath && backup.Checksum == currentChecksum {
			return true, nil // Skip backup - content hasn't changed
		}
	}

	return false, nil // Content has changed, create backup
}

// applyRetentionPolicy removes old backups based on retention policy
func (fm *FileManager) applyRetentionPolicy(originalPath string, config *EnhancedBackupConfig) error {
	fileName := filepath.Base(originalPath)
	pattern := filepath.Join(config.Directory, fileName+".backup.*")
	backupFiles, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to list backup files: %w", err)
	}

	// Filter to get only actual backup files (not metadata files)
	actualBackups := make([]string, 0)
	for _, path := range backupFiles {
		// Skip metadata files
		if strings.HasSuffix(path, ".meta") {
			continue
		}

		// Include both compressed and uncompressed backup files
		baseName := filepath.Base(path)
		if strings.Contains(baseName, ".backup.") {
			actualBackups = append(actualBackups, path)
		}
	}

	if len(actualBackups) <= int(config.MaxBackups) {
		return nil // Within retention limit
	}

	// Sort backups by modification time (oldest first)
	sort.Slice(actualBackups, func(i, j int) bool {
		infoI, errI := os.Stat(actualBackups[i])
		infoJ, errJ := os.Stat(actualBackups[j])
		if errI != nil || errJ != nil {
			return false
		}
		return infoI.ModTime().Before(infoJ.ModTime())
	})

	// Remove oldest backups beyond retention limit
	toRemove := len(actualBackups) - int(config.MaxBackups)
	for i := 0; i < toRemove; i++ {
		backupPath := actualBackups[i]

		// Remove backup file
		if err := os.Remove(backupPath); err != nil {
			return fmt.Errorf("failed to remove old backup %s: %w", backupPath, err)
		}

		// Remove associated metadata file if it exists
		metadataPath := backupPath + ".meta"
		if _, err := os.Stat(metadataPath); err == nil {
			_ = os.Remove(metadataPath)
		}
	}

	return nil
}

// calculateFileChecksum calculates SHA256 checksum of a file
func (fm *FileManager) calculateFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to calculate hash: %w", err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// saveBackupIndex saves backup index to file
func (fm *FileManager) saveBackupIndex(index *BackupIndex, indexPath string) error {
	indexFile, err := os.Create(indexPath)
	if err != nil {
		return fmt.Errorf("failed to create index file: %w", err)
	}
	defer indexFile.Close()

	encoder := json.NewEncoder(indexFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(index); err != nil {
		return fmt.Errorf("failed to encode index: %w", err)
	}

	return nil
}

// LoadBackupMetadata loads backup metadata from file
func LoadBackupMetadata(metadataPath string) (*BackupMetadata, error) {
	file, err := os.Open(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open metadata file: %w", err)
	}
	defer file.Close()

	var metadata BackupMetadata
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	return &metadata, nil
}

// LoadBackupIndex loads backup index from file
func LoadBackupIndex(indexPath string) (*BackupIndex, error) {
	file, err := os.Open(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open index file: %w", err)
	}
	defer file.Close()

	var index BackupIndex
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&index); err != nil {
		return nil, fmt.Errorf("failed to decode index: %w", err)
	}

	return &index, nil
}

// ValidateEnhancedBackupConfig validates enhanced backup configuration
func ValidateEnhancedBackupConfig(config *EnhancedBackupConfig) error {
	if config == nil {
		return nil
	}

	// Validate retention policy format
	if config.RetentionPolicy != "" {
		if !isValidRetentionPolicy(config.RetentionPolicy) {
			return fmt.Errorf("invalid retention policy format: %s", config.RetentionPolicy)
		}
	}

	// Validate backup format
	validFormats := []string{"timestamped", "numbered", "git_style"}
	if config.BackupFormat != "" {
		valid := false
		for _, format := range validFormats {
			if config.BackupFormat == format {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid backup format: %s (must be one of: %v)", config.BackupFormat, validFormats)
		}
	}

	// Validate max backups
	if config.MaxBackups < 0 {
		return fmt.Errorf("max_backups cannot be negative: %d", config.MaxBackups)
	}

	return nil
}

// isValidRetentionPolicy checks if retention policy is valid
func isValidRetentionPolicy(policy string) bool {
	if len(policy) < 2 {
		return false
	}

	unit := policy[len(policy)-1]
	value := policy[:len(policy)-1]

	// Check if value is numeric
	if _, err := strconv.Atoi(value); err != nil {
		return false
	}

	// Check if unit is valid
	validUnits := []byte{'d', 'w', 'm', 'y'}
	for _, validUnit := range validUnits {
		if unit == validUnit {
			return true
		}
	}

	return false
}
