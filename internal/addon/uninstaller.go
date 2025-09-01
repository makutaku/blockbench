package addon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/makutaku/blockbench/internal/minecraft"
	"github.com/makutaku/blockbench/pkg/filesystem"
)

// UninstallOptions contains options for addon uninstallation
type UninstallOptions struct {
	DryRun    bool
	Verbose   bool
	BackupDir string
	ByUUID    bool
}

// UninstallResult contains the result of an uninstallation
type UninstallResult struct {
	Success        bool
	RemovedPacks   []string
	BackupMetadata *filesystem.BackupMetadata
	Errors         []string
	Warnings       []string
}

// Uninstaller handles addon uninstallation operations
type Uninstaller struct {
	server        *minecraft.Server
	backupManager *BackupManager
}

// NewUninstaller creates a new addon uninstaller
func NewUninstaller(server *minecraft.Server, backupDir string) *Uninstaller {
	return &Uninstaller{
		server:        server,
		backupManager: NewBackupManager(server, backupDir),
	}
}

// UninstallAddon removes an addon with validation and rollback support
func (u *Uninstaller) UninstallAddon(identifier string, options UninstallOptions) (*UninstallResult, error) {
	result := &UninstallResult{
		RemovedPacks: make([]string, 0),
		Errors:       make([]string, 0),
		Warnings:     make([]string, 0),
	}

	if options.Verbose {
		if options.ByUUID {
			fmt.Printf("Starting uninstallation of addon with UUID: %s\n", identifier)
		} else {
			fmt.Printf("Starting uninstallation of addon: %s\n", identifier)
		}
	}

	// Step 1: Find the addon to uninstall
	packToRemove, err := u.findAddonPack(identifier, options.ByUUID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to find addon: %v", err))
		return result, err
	}

	if options.Verbose {
		fmt.Printf("Found pack: %s (UUID: %s, Type: %s)\n", 
			packToRemove.Name, packToRemove.PackID, packToRemove.Type)
	}

	if options.DryRun {
		if options.Verbose {
			fmt.Println("DRY RUN: Pack found, uninstallation would proceed")
		}
		result.Success = true
		result.RemovedPacks = append(result.RemovedPacks, packToRemove.Name)
		return result, nil
	}

	// Step 2: Check for dependencies
	dependents, err := u.checkDependencies(packToRemove.PackID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Dependency check failed: %v", err))
		return result, err
	}

	if len(dependents) > 0 {
		for _, dependent := range dependents {
			result.Warnings = append(result.Warnings, 
				fmt.Sprintf("Pack %s depends on the pack being removed", dependent))
		}
		// For now, we'll allow removal but warn the user
	}

	// Step 3: Create backup
	if options.Verbose {
		fmt.Println("Creating backup before uninstallation...")
	}

	backup, err := u.backupManager.CreateUninstallBackup(packToRemove.Name, packToRemove.PackID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Backup creation failed: %v", err))
		return result, err
	}
	result.BackupMetadata = backup

	// Step 4: Uninstall the pack (with rollback on failure)
	if err := u.server.UninstallPack(packToRemove.PackID); err != nil {
		if options.Verbose {
			fmt.Println("Uninstallation failed, rolling back...")
		}
		
		// Rollback on failure
		if rollbackErr := u.backupManager.RestoreBackup(backup.ID); rollbackErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Rollback failed: %v", rollbackErr))
		} else if options.Verbose {
			fmt.Println("Successfully rolled back changes")
		}
		
		result.Errors = append(result.Errors, fmt.Sprintf("Uninstallation failed: %v", err))
		return result, err
	}

	// Step 5: Post-uninstallation validation
	if err := u.postUninstallValidation(packToRemove.PackID); err != nil {
		if options.Verbose {
			fmt.Println("Post-uninstallation validation failed, rolling back...")
		}
		
		// Rollback on validation failure
		if rollbackErr := u.backupManager.RestoreBackup(backup.ID); rollbackErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Rollback failed: %v", rollbackErr))
		}
		
		result.Errors = append(result.Errors, fmt.Sprintf("Post-uninstallation validation failed: %v", err))
		return result, err
	}

	// Success!
	result.RemovedPacks = append(result.RemovedPacks, packToRemove.Name)
	result.Success = true

	if options.Verbose {
		fmt.Printf("Successfully uninstalled pack: %s\n", packToRemove.Name)
	}

	return result, nil
}

// findAddonPack finds an addon pack by name or UUID
func (u *Uninstaller) findAddonPack(identifier string, byUUID bool) (*minecraft.InstalledPack, error) {
	installedPacks, err := u.server.ListInstalledPacks()
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packs: %w", err)
	}

	if byUUID {
		// Search by UUID
		for _, pack := range installedPacks {
			if pack.PackID == identifier {
				return &pack, nil
			}
		}
		return nil, fmt.Errorf("no pack found with UUID: %s", identifier)
	}

	// Search by name (case-insensitive partial match)
	var matches []minecraft.InstalledPack
	for _, pack := range installedPacks {
		if containsIgnoreCase(pack.Name, identifier) {
			matches = append(matches, pack)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no pack found with name containing: %s", identifier)
	}

	if len(matches) > 1 {
		var names []string
		for _, match := range matches {
			names = append(names, match.Name)
		}
		return nil, fmt.Errorf("multiple packs found matching '%s': %v. Use UUID for precise identification", identifier, names)
	}

	return &matches[0], nil
}

// checkDependencies checks if other packs depend on the pack being removed
func (u *Uninstaller) checkDependencies(packID string) ([]string, error) {
	var dependents []string

	installedPacks, err := u.server.ListInstalledPacks()
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packs: %w", err)
	}

	// For each installed pack, check if it depends on the pack being removed
	for _, pack := range installedPacks {
		if pack.PackID == packID {
			continue // Skip the pack being removed
		}

		// Try to load the pack's manifest to check dependencies
		manifest, err := u.loadPackManifest(pack.PackID, pack.Type)
		if err != nil {
			// If we can't load the manifest, we can't check dependencies
			continue
		}

		// Check if this pack depends on the one being removed
		for _, dep := range manifest.Dependencies {
			if dep.UUID == packID {
				dependents = append(dependents, pack.Name)
				break
			}
		}
	}

	return dependents, nil
}

// loadPackManifest loads a manifest for an installed pack
func (u *Uninstaller) loadPackManifest(packID string, packType minecraft.PackType) (*minecraft.Manifest, error) {
	var baseDir string
	
	switch packType {
	case minecraft.PackTypeBehavior:
		baseDir = u.server.Paths.BehaviorPacksDir
	case minecraft.PackTypeResource:
		baseDir = u.server.Paths.ResourcePacksDir
	default:
		return nil, fmt.Errorf("unknown pack type: %s", packType)
	}

	return u.findAndLoadManifest(baseDir, packID)
}

// findAndLoadManifest finds and loads a manifest by pack ID
func (u *Uninstaller) findAndLoadManifest(baseDir, packID string) (*minecraft.Manifest, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", baseDir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		
		manifestPath := filepath.Join(baseDir, entry.Name(), "manifest.json")
		manifest, err := minecraft.ParseManifest(manifestPath)
		if err != nil {
			continue // Skip if can't read manifest
		}
		
		if manifest.Header.UUID == packID {
			return manifest, nil
		}
	}

	return nil, fmt.Errorf("manifest not found for pack ID %s", packID)
}

// postUninstallValidation validates that the pack was successfully removed
func (u *Uninstaller) postUninstallValidation(packID string) error {
	installedPacks, err := u.server.ListInstalledPacks()
	if err != nil {
		return fmt.Errorf("failed to list installed packs for validation: %w", err)
	}

	// Check that the pack is no longer in the installed list
	for _, pack := range installedPacks {
		if pack.PackID == packID {
			return fmt.Errorf("pack %s still appears in installed packs after removal", packID)
		}
	}

	return nil
}

// containsIgnoreCase performs case-insensitive substring matching
func containsIgnoreCase(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	return strings.Contains(s, substr)
}