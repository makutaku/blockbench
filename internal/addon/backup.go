package addon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/makutaku/blockbench/internal/minecraft"
	"github.com/makutaku/blockbench/pkg/filesystem"
)

// BackupManager wraps the filesystem backup manager with addon-specific functionality
type BackupManager struct {
	*filesystem.BackupManager
	server *minecraft.Server
}

// NewBackupManager creates a new addon backup manager
func NewBackupManager(server *minecraft.Server, backupRoot string) *BackupManager {
	return &BackupManager{
		BackupManager: filesystem.NewBackupManager(backupRoot),
		server:        server,
	}
}

// CreateInstallBackup creates a backup before installing an addon
func (bm *BackupManager) CreateInstallBackup(addonName, addonUUID string) (*filesystem.BackupMetadata, error) {
	files := []string{
		bm.server.Paths.WorldBehaviorPacks,
		bm.server.Paths.WorldResourcePacks,
		bm.server.Paths.WorldBehaviorHistory,
		bm.server.Paths.WorldResourceHistory,
	}

	description := fmt.Sprintf("Before installing addon: %s", addonName)

	metadata, err := bm.CreateBackup("install", description, files)
	if err != nil {
		return nil, err
	}

	metadata.AddonName = addonName
	metadata.AddonUUID = addonUUID
	metadata.ServerPath = bm.server.Paths.ServerRoot

	return metadata, nil
}

// CreateUninstallBackup creates a backup before uninstalling an addon
func (bm *BackupManager) CreateUninstallBackup(addonName, addonUUID string) (*filesystem.BackupMetadata, error) {
	files := []string{
		bm.server.Paths.WorldBehaviorPacks,
		bm.server.Paths.WorldResourcePacks,
		bm.server.Paths.WorldBehaviorHistory,
		bm.server.Paths.WorldResourceHistory,
	}

	// Also backup the addon directory itself
	addonDirs, err := bm.findAddonDirectories(addonUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to find addon directories: %w", err)
	}

	files = append(files, addonDirs...)

	description := fmt.Sprintf("Before uninstalling addon: %s", addonName)

	metadata, err := bm.CreateBackup("uninstall", description, files)
	if err != nil {
		return nil, err
	}

	metadata.AddonName = addonName
	metadata.AddonUUID = addonUUID
	metadata.ServerPath = bm.server.Paths.ServerRoot

	return metadata, nil
}

// findAddonDirectories finds the directories for a specific addon
func (bm *BackupManager) findAddonDirectories(addonUUID string) ([]string, error) {
	var dirs []string
	var errors []string

	// Check behavior packs directory
	behaviorDir, err := bm.findAddonInDirectory(bm.server.Paths.BehaviorPacksDir, addonUUID)
	if err == nil {
		dirs = append(dirs, behaviorDir)
	} else {
		// Only track non-"not found" errors
		if !os.IsNotExist(err) && !strings.Contains(err.Error(), "not found") {
			errors = append(errors, fmt.Sprintf("behavior packs: %v", err))
		}
	}

	// Check resource packs directory
	resourceDir, err := bm.findAddonInDirectory(bm.server.Paths.ResourcePacksDir, addonUUID)
	if err == nil {
		dirs = append(dirs, resourceDir)
	} else {
		// Only track non-"not found" errors
		if !os.IsNotExist(err) && !strings.Contains(err.Error(), "not found") {
			errors = append(errors, fmt.Sprintf("resource packs: %v", err))
		}
	}

	// If we found no directories and had errors, return the errors
	if len(dirs) == 0 && len(errors) > 0 {
		return nil, fmt.Errorf("failed to find addon directories: %s", strings.Join(errors, "; "))
	}

	// If we found no directories and had no errors, addon doesn't exist
	if len(dirs) == 0 {
		return nil, fmt.Errorf("addon with UUID %s not found in any pack directory", addonUUID)
	}

	return dirs, nil
}

// LoadMetadata loads backup metadata by ID
func (bm *BackupManager) LoadMetadata(backupID string) (*filesystem.BackupMetadata, error) {
	backups, err := bm.ListBackups()
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

// findAddonInDirectory searches for an addon directory by UUID
func (bm *BackupManager) findAddonInDirectory(baseDir, addonUUID string) (string, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return "", fmt.Errorf("failed to read directory %s: %w", baseDir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		packPath := filepath.Join(baseDir, entry.Name())
		manifestPath := filepath.Join(packPath, "manifest.json")

		manifest, err := minecraft.ParseManifest(manifestPath)
		if err != nil {
			continue // Skip if can't read manifest
		}

		if manifest.Header.UUID == addonUUID {
			return packPath, nil
		}
	}

	return "", fmt.Errorf("addon directory not found for UUID: %s", addonUUID)
}
