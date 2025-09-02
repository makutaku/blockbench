package filesystem

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// BackupMetadata contains information about a backup
type BackupMetadata struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Operation   string    `json:"operation"`
	AddonName   string    `json:"addon_name,omitempty"`
	AddonUUID   string    `json:"addon_uuid,omitempty"`
	ServerPath  string    `json:"server_path"`
	BackupPath  string    `json:"backup_path"`
	Files       []string  `json:"files"`
	Description string    `json:"description,omitempty"`
}

// BackupManager handles backup operations
type BackupManager struct {
	BackupRoot string
	metadata   []BackupMetadata
}

// NewBackupManager creates a new backup manager
func NewBackupManager(backupRoot string) *BackupManager {
	return &BackupManager{
		BackupRoot: backupRoot,
		metadata:   make([]BackupMetadata, 0),
	}
}

// CreateBackup creates a backup of specified files/directories
func (bm *BackupManager) CreateBackup(operation, description string, files []string) (*BackupMetadata, error) {
	// Generate backup ID
	backupID := generateBackupID()

	// Create backup directory
	backupDir := filepath.Join(bm.BackupRoot, backupID)
	if err := os.MkdirAll(backupDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create metadata
	metadata := BackupMetadata{
		ID:          backupID,
		Timestamp:   time.Now(),
		Operation:   operation,
		BackupPath:  backupDir,
		Files:       make([]string, 0),
		Description: description,
	}

	// Backup each file/directory
	for _, file := range files {
		if err := bm.backupFile(file, backupDir); err != nil {
			// Cleanup on error
			if rmErr := os.RemoveAll(backupDir); rmErr != nil {
				// Log cleanup failure but don't override original error
				fmt.Fprintf(os.Stderr, "Warning: Failed to cleanup backup directory: %v\n", rmErr)
			}
			return nil, fmt.Errorf("failed to backup %s: %w", file, err)
		}
		metadata.Files = append(metadata.Files, file)
	}

	// Save metadata
	if err := bm.saveMetadata(&metadata); err != nil {
		if rmErr := os.RemoveAll(backupDir); rmErr != nil {
			// Log cleanup failure but don't override original error
			fmt.Fprintf(os.Stderr, "Warning: Failed to cleanup backup directory: %v\n", rmErr)
		}
		return nil, fmt.Errorf("failed to save backup metadata: %w", err)
	}

	return &metadata, nil
}

// RestoreBackup restores files from a backup
func (bm *BackupManager) RestoreBackup(backupID string) error {
	metadata, err := bm.loadMetadata(backupID)
	if err != nil {
		return fmt.Errorf("failed to load backup metadata: %w", err)
	}

	// Restore each backed up file
	for _, originalFile := range metadata.Files {
		if err := bm.restoreFile(originalFile, metadata.BackupPath); err != nil {
			return fmt.Errorf("failed to restore %s: %w", originalFile, err)
		}
	}

	return nil
}

// DeleteBackup removes a backup and its metadata
func (bm *BackupManager) DeleteBackup(backupID string) error {
	// Load metadata to get backup path
	metadata, err := bm.loadMetadata(backupID)
	if err != nil {
		return fmt.Errorf("failed to load backup metadata: %w", err)
	}

	// Remove backup directory
	if err := os.RemoveAll(metadata.BackupPath); err != nil {
		return fmt.Errorf("failed to remove backup directory: %w", err)
	}

	// Remove metadata file
	metadataFile := filepath.Join(bm.BackupRoot, fmt.Sprintf("%s.json", backupID))
	if err := os.Remove(metadataFile); err != nil {
		return fmt.Errorf("failed to remove metadata file: %w", err)
	}

	return nil
}

// ListBackups returns a list of all backups
func (bm *BackupManager) ListBackups() ([]BackupMetadata, error) {
	if err := os.MkdirAll(bm.BackupRoot, 0750); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	entries, err := os.ReadDir(bm.BackupRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []BackupMetadata
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			backupID := entry.Name()[:len(entry.Name())-5] // Remove .json extension
			metadata, err := bm.loadMetadata(backupID)
			if err != nil {
				continue // Skip corrupted metadata
			}
			backups = append(backups, *metadata)
		}
	}

	return backups, nil
}

// backupFile backs up a single file or directory
func (bm *BackupManager) backupFile(source, backupDir string) error {
	// Get relative path for backup structure
	basename := filepath.Base(source)
	backupPath := filepath.Join(backupDir, basename)

	// Check if source exists
	sourceInfo, err := os.Stat(source)
	if os.IsNotExist(err) {
		// Create empty marker file for non-existent files
		markerFile := backupPath + ".missing"
		return os.WriteFile(markerFile, []byte(""), 0600)
	}
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	if sourceInfo.IsDir() {
		return copyDir(source, backupPath)
	}

	return copyFile(source, backupPath)
}

// restoreFile restores a single file or directory
func (bm *BackupManager) restoreFile(originalPath, backupDir string) error {
	basename := filepath.Base(originalPath)
	backupPath := filepath.Join(backupDir, basename)

	// Check if this was a missing file
	markerFile := backupPath + ".missing"
	if _, err := os.Stat(markerFile); err == nil {
		// File was missing in original, remove it if it exists now
		if _, err := os.Stat(originalPath); err == nil {
			return os.RemoveAll(originalPath)
		}
		return nil
	}

	// Check if backup exists
	backupInfo, err := os.Stat(backupPath)
	if err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	if backupInfo.IsDir() {
		// Remove existing directory if it exists
		if _, err := os.Stat(originalPath); err == nil {
			if err := os.RemoveAll(originalPath); err != nil {
				return fmt.Errorf("failed to remove existing directory: %w", err)
			}
		}
		return copyDir(backupPath, originalPath)
	}

	return copyFile(backupPath, originalPath)
}

// saveMetadata saves backup metadata to a JSON file
func (bm *BackupManager) saveMetadata(metadata *BackupMetadata) error {
	metadataFile := filepath.Join(bm.BackupRoot, fmt.Sprintf("%s.json", metadata.ID))

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return os.WriteFile(metadataFile, data, 0600)
}

// loadMetadata loads backup metadata from a JSON file
func (bm *BackupManager) loadMetadata(backupID string) (*BackupMetadata, error) {
	metadataFile := filepath.Join(bm.BackupRoot, fmt.Sprintf("%s.json", backupID))

	data, err := os.ReadFile(metadataFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata BackupMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &metadata, nil
}

// generateBackupID generates a unique backup ID
func generateBackupID() string {
	return fmt.Sprintf("backup_%d", time.Now().Unix())
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(dst), 0750); err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := srcFile.WriteTo(dstFile); err != nil {
		return err
	}

	// Copy file permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	return os.Chmod(dst, srcInfo.Mode())
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}
