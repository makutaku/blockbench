package addon

import (
	"fmt"
	"os"

	"github.com/makutaku/blockbench/internal/minecraft"
	"github.com/makutaku/blockbench/pkg/filesystem"
)

// RollbackManager handles rollback operations for failed addon installations/uninstallations
type RollbackManager struct {
	server        *minecraft.Server
	backupManager *BackupManager
}

// NewRollbackManager creates a new rollback manager
func NewRollbackManager(server *minecraft.Server, backupDir string) *RollbackManager {
	return &RollbackManager{
		server:        server,
		backupManager: NewBackupManager(server, backupDir),
	}
}

// RollbackOptions contains options for rollback operations
type RollbackOptions struct {
	Verbose bool
	DryRun  bool
}

// RollbackResult contains the result of a rollback operation
type RollbackResult struct {
	Success       bool
	BackupID      string
	RestoredFiles []string
	Errors        []string
}

// RollbackToBackup performs a rollback to a specific backup
func (rm *RollbackManager) RollbackToBackup(backupID string, options RollbackOptions) (*RollbackResult, error) {
	result := &RollbackResult{
		BackupID:      backupID,
		RestoredFiles: make([]string, 0),
		Errors:        make([]string, 0),
	}

	if options.Verbose {
		fmt.Printf("Starting rollback to backup: %s\n", backupID)
	}

	// Load backup metadata
	metadata, err := rm.backupManager.LoadMetadata(backupID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to load backup metadata: %v", err))
		return result, err
	}

	if options.Verbose {
		fmt.Printf("Backup found: %s (created: %s)\n", metadata.Description, metadata.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("Files to restore: %d\n", len(metadata.Files))
	}

	if options.DryRun {
		if options.Verbose {
			fmt.Println("DRY RUN: Would restore the following files:")
			for _, file := range metadata.Files {
				fmt.Printf("  - %s\n", file)
			}
		}
		result.Success = true
		result.RestoredFiles = metadata.Files
		return result, nil
	}

	// Perform the rollback
	if err := rm.backupManager.RestoreBackup(backupID); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Rollback failed: %v", err))
		return result, err
	}

	result.Success = true
	result.RestoredFiles = metadata.Files

	if options.Verbose {
		fmt.Printf("Successfully rolled back %d files\n", len(result.RestoredFiles))
	}

	return result, nil
}

// ListAvailableBackups returns a list of available backups for rollback
func (rm *RollbackManager) ListAvailableBackups() ([]filesystem.BackupMetadata, error) {
	return rm.backupManager.ListBackups()
}

// GetBackupInfo returns detailed information about a specific backup
func (rm *RollbackManager) GetBackupInfo(backupID string) (*filesystem.BackupMetadata, error) {
	backups, err := rm.ListAvailableBackups()
	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}

	for _, backup := range backups {
		if backup.ID == backupID {
			return &backup, nil
		}
	}

	return nil, fmt.Errorf("backup not found: %s", backupID)
}

// ValidateBackup checks if a backup is valid and can be restored
func (rm *RollbackManager) ValidateBackup(backupID string) error {
	metadata, err := rm.GetBackupInfo(backupID)
	if err != nil {
		return fmt.Errorf("backup validation failed: %w", err)
	}

	// Check if backup directory still exists
	if _, err := os.Stat(metadata.BackupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup directory no longer exists: %s", metadata.BackupPath)
	}

	// Additional validation could be added here
	return nil
}
