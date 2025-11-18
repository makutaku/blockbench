package addon

import (
	"fmt"
	"strings"

	"github.com/makutaku/blockbench/internal/minecraft"
	"github.com/makutaku/blockbench/pkg/filesystem"
)

// UninstallOptions contains options for addon uninstallation
type UninstallOptions struct {
	DryRun      bool
	Verbose     bool
	BackupDir   string
	ByUUID      bool
	Interactive bool
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
		return u.performDryRunSimulation(packToRemove, options)
	}

	// Step 2: Check for dependencies
	dependents, err := u.checkDependencies(packToRemove.PackID, options.Verbose, result)
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
func (u *Uninstaller) checkDependencies(packID string, verbose bool, result *UninstallResult) ([]string, error) {
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
			// If we can't load the manifest, warn but continue
			// (manifest may not exist if pack is broken or was manually installed)
			warning := fmt.Sprintf("Could not verify dependencies for pack %s (%s): %v", pack.Name, pack.PackID, err)
			if verbose {
				fmt.Printf("Warning: %s\n", warning)
				fmt.Println("  Dependency check for this pack will be incomplete")
			}
			if result != nil {
				result.Warnings = append(result.Warnings, "Incomplete dependency check: "+warning)
			}
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

// loadPackManifest loads a manifest for an installed pack using the shared server method
func (u *Uninstaller) loadPackManifest(packID string, packType minecraft.PackType) (*minecraft.Manifest, error) {
	return u.server.FindAndLoadManifestByUUID(packID, packType)
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

// performDryRunSimulation simulates uninstallation operations and shows detailed information
func (u *Uninstaller) performDryRunSimulation(packToRemove *minecraft.InstalledPack, options UninstallOptions) (*UninstallResult, error) {
	result := &UninstallResult{
		RemovedPacks: make([]string, 0),
		Errors:       make([]string, 0),
		Warnings:     make([]string, 0),
	}

	simulator := NewDryRunSimulator(u.server)

	if options.Verbose {
		fmt.Println("DRY RUN: Simulating uninstallation operations...")
	}

	// Simulate dependency check
	dependents, err := u.checkDependencies(packToRemove.PackID, options.Verbose, result)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Dependency check failed: %v", err))
		return result, err
	}

	dependencyDetails := []string{
		fmt.Sprintf("DRY RUN: Checked dependencies for pack: %s", packToRemove.Name),
	}
	if len(dependents) == 0 {
		dependencyDetails = append(dependencyDetails, "DRY RUN: No dependent packs found - safe to remove")
	} else {
		dependencyDetails = append(dependencyDetails, fmt.Sprintf("DRY RUN: Found %d dependent pack(s):", len(dependents)))
		for _, dependent := range dependents {
			dependencyDetails = append(dependencyDetails, fmt.Sprintf("  • %s depends on this pack", dependent))
			result.Warnings = append(result.Warnings, fmt.Sprintf("Pack %s depends on the pack being removed", dependent))
		}
		dependencyDetails = append(dependencyDetails, "DRY RUN: Would proceed with removal but warn about dependencies")
	}

	// Use the simulator to get detailed uninstallation information
	simulation, err := simulator.SimulatePackUninstallation(packToRemove.PackID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Uninstallation simulation failed: %v", err))
		return result, err
	}

	// Show dependency check results
	if err := showStepResult("Dependency check simulation", dependencyDetails, "Backup simulation", "Simulate creating a backup of current state before removal.", convertToInstallOptions(options)); err != nil {
		return result, err
	}

	// Simulate backup creation
	backupDetails := []string{
		"DRY RUN: Backup would be created with timestamp-based ID",
		fmt.Sprintf("DRY RUN: Backup would be stored in: %s/backups/", u.server.Paths.ServerRoot),
		fmt.Sprintf("DRY RUN: Would backup pack directory: %s", simulation.DirectoryToRemove),
		fmt.Sprintf("DRY RUN: Would backup config file: %s", simulation.ConfigFile),
	}
	if err := showStepResult("Backup simulation", backupDetails, "Uninstallation simulation", "Simulate removing pack directory and updating world configuration files.", convertToInstallOptions(options)); err != nil {
		return result, err
	}

	// Simulate uninstallation
	packTypeStr := "behavior"
	if simulation.PackType == minecraft.PackTypeResource {
		packTypeStr = "resource"
	}

	uninstallationDetails := []string{
		fmt.Sprintf("DRY RUN: Would remove %s pack directory: %s", packTypeStr, simulation.DirectoryToRemove),
		fmt.Sprintf("DRY RUN: Would update config file: %s", simulation.ConfigFile),
		fmt.Sprintf("  • Would remove pack entry: %s (UUID: %s)", simulation.PackName, simulation.PackUUID),
	}

	if len(simulation.DependentPacks) > 0 {
		uninstallationDetails = append(uninstallationDetails, "DRY RUN: Dependent packs would be left with broken dependencies:")
		for _, dependent := range simulation.DependentPacks {
			uninstallationDetails = append(uninstallationDetails, fmt.Sprintf("  • %s", dependent))
		}
	}

	if err := showStepResult("Uninstallation simulation", uninstallationDetails, "Validation simulation", "Simulate post-uninstallation validation to ensure pack is properly removed.", convertToInstallOptions(options)); err != nil {
		return result, err
	}

	// Simulate post-uninstallation validation
	validationDetails := []string{
		fmt.Sprintf("DRY RUN: Would verify pack removal: %s", simulation.PackName),
		"DRY RUN: Pack would no longer be registered with the server",
		"DRY RUN: Pack directory would be completely removed",
	}
	if err := showStepResult("Validation simulation", validationDetails, "", "", convertToInstallOptions(options)); err != nil {
		return result, err
	}

	// Summary
	result.RemovedPacks = append(result.RemovedPacks, packToRemove.Name)
	result.Success = true

	if options.Verbose {
		fmt.Printf("DRY RUN COMPLETE: Would uninstall pack '%s' successfully\n", packToRemove.Name)
		if len(simulation.DependentPacks) > 0 {
			fmt.Printf("WARNING: %d dependent pack(s) would be affected\n", len(simulation.DependentPacks))
		}
		fmt.Println("No actual changes were made to the server")
	}

	return result, nil
}

// convertToInstallOptions converts UninstallOptions to InstallOptions for showStepResult compatibility
func convertToInstallOptions(uninstallOpts UninstallOptions) InstallOptions {
	return InstallOptions{
		Interactive: uninstallOpts.Interactive,
		Verbose:     uninstallOpts.Verbose,
		DryRun:      uninstallOpts.DryRun,
	}
}
