package addon

import (
	"fmt"

	"github.com/makutaku/blockbench/internal/minecraft"
	"github.com/makutaku/blockbench/pkg/filesystem"
)

// InstallOptions contains options for addon installation
type InstallOptions struct {
	DryRun      bool
	Verbose     bool
	BackupDir   string
	ForceUpdate bool
}

// InstallResult contains the result of an installation
type InstallResult struct {
	Success        bool
	InstalledPacks []string
	BackupMetadata *filesystem.BackupMetadata
	Errors         []string
	Warnings       []string
}

// Installer handles addon installation operations
type Installer struct {
	server        *minecraft.Server
	backupManager *BackupManager
}

// NewInstaller creates a new addon installer
func NewInstaller(server *minecraft.Server, backupDir string) *Installer {
	return &Installer{
		server:        server,
		backupManager: NewBackupManager(server, backupDir),
	}
}

// InstallAddon installs an addon with full validation and rollback support
func (i *Installer) InstallAddon(addonPath string, options InstallOptions) (*InstallResult, error) {
	result := &InstallResult{
		InstalledPacks: make([]string, 0),
		Errors:         make([]string, 0),
		Warnings:       make([]string, 0),
	}

	if options.Verbose {
		fmt.Printf("Starting installation of %s\n", addonPath)
	}

	// Step 1: Pre-installation validation
	if err := i.preInstallValidation(addonPath, options.Verbose); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Pre-installation validation failed: %v", err))
		return result, err
	}

	if options.DryRun {
		if options.Verbose {
			fmt.Println("DRY RUN: Validation passed, installation would proceed")
		}
		result.Success = true
		return result, nil
	}

	// Step 2: Extract addon
	extractedAddon, err := ExtractAddon(addonPath, false)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Extraction failed: %v", err))
		return result, err
	}
	defer extractedAddon.Cleanup()

	// Step 3: Validate extracted content
	if err := i.validateExtractedAddon(extractedAddon); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Content validation failed: %v", err))
		return result, err
	}

	// Step 4: Check for conflicts
	conflicts, err := i.checkForConflicts(extractedAddon)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Conflict check failed: %v", err))
		return result, err
	}

	if len(conflicts) > 0 && !options.ForceUpdate {
		for _, conflict := range conflicts {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Conflict detected: %s", conflict))
		}
		return result, fmt.Errorf("conflicts detected, use --force to override")
	}

	// Step 5: Create backup
	var addonName, addonUUID string
	allPacks := extractedAddon.GetAllPacks()
	if len(allPacks) > 0 {
		addonName = allPacks[0].Manifest.GetDisplayName()
		addonUUID = allPacks[0].Manifest.Header.UUID
	}

	if options.Verbose {
		fmt.Println("Creating backup before installation...")
	}

	backup, err := i.backupManager.CreateInstallBackup(addonName, addonUUID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Backup creation failed: %v", err))
		return result, err
	}
	result.BackupMetadata = backup

	// Step 6: Install packs (with rollback on failure)
	if err := i.installPacks(extractedAddon, options.Verbose); err != nil {
		if options.Verbose {
			fmt.Println("Installation failed, rolling back...")
		}

		// Rollback on failure
		if rollbackErr := i.backupManager.RestoreBackup(backup.ID); rollbackErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Rollback failed: %v", rollbackErr))
		} else if options.Verbose {
			fmt.Println("Successfully rolled back changes")
		}

		result.Errors = append(result.Errors, fmt.Sprintf("Installation failed: %v", err))
		return result, err
	}

	// Step 7: Post-installation validation
	if err := i.postInstallValidation(extractedAddon); err != nil {
		if options.Verbose {
			fmt.Println("Post-installation validation failed, rolling back...")
		}

		// Rollback on validation failure
		if rollbackErr := i.backupManager.RestoreBackup(backup.ID); rollbackErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Rollback failed: %v", rollbackErr))
		}

		result.Errors = append(result.Errors, fmt.Sprintf("Post-installation validation failed: %v", err))
		return result, err
	}

	// Success!
	for _, pack := range allPacks {
		result.InstalledPacks = append(result.InstalledPacks, pack.Manifest.GetDisplayName())
	}
	result.Success = true

	if options.Verbose {
		fmt.Printf("Successfully installed %d packs\n", len(result.InstalledPacks))
	}

	return result, nil
}

// preInstallValidation performs validation before installation
func (i *Installer) preInstallValidation(addonPath string, verbose bool) error {
	if verbose {
		fmt.Println("Validating addon file...")
	}

	// Validate addon file
	if err := ValidateAddonFile(addonPath); err != nil {
		return fmt.Errorf("addon file validation failed: %w", err)
	}

	// Validate server structure
	if err := i.server.Paths.ValidateServerStructure(); err != nil {
		return fmt.Errorf("server validation failed: %w", err)
	}

	return nil
}

// validateExtractedAddon validates the extracted addon content
func (i *Installer) validateExtractedAddon(addon *ExtractedAddon) error {
	allPacks := addon.GetAllPacks()
	if len(allPacks) == 0 {
		return fmt.Errorf("no valid packs found in addon")
	}

	// Validate each pack
	for _, pack := range allPacks {
		if err := minecraft.ValidateManifest(pack.Manifest); err != nil {
			return fmt.Errorf("manifest validation failed for pack %s: %w", pack.Manifest.GetDisplayName(), err)
		}
	}

	return nil
}

// checkForConflicts checks if the addon conflicts with existing installations
func (i *Installer) checkForConflicts(addon *ExtractedAddon) ([]string, error) {
	var conflicts []string

	installedPacks, err := i.server.ListInstalledPacks()
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packs: %w", err)
	}

	// Check for UUID conflicts
	for _, newPack := range addon.GetAllPacks() {
		for _, installedPack := range installedPacks {
			if newPack.Manifest.Header.UUID == installedPack.PackID {
				conflicts = append(conflicts, fmt.Sprintf("Pack %s (UUID: %s) is already installed",
					installedPack.Name, installedPack.PackID))
			}
		}
	}

	return conflicts, nil
}

// installPacks installs all packs in the addon
func (i *Installer) installPacks(addon *ExtractedAddon, verbose bool) error {
	allPacks := addon.GetAllPacks()

	for _, pack := range allPacks {
		if verbose {
			fmt.Printf("Installing %s pack: %s\n", pack.PackType, pack.Manifest.GetDisplayName())
		}

		if err := i.server.InstallPack(pack.Manifest, pack.Path); err != nil {
			return fmt.Errorf("failed to install pack %s: %w", pack.Manifest.GetDisplayName(), err)
		}
	}

	return nil
}

// postInstallValidation validates the installation was successful
func (i *Installer) postInstallValidation(addon *ExtractedAddon) error {
	// Verify all packs are now listed as installed
	installedPacks, err := i.server.ListInstalledPacks()
	if err != nil {
		return fmt.Errorf("failed to list installed packs for validation: %w", err)
	}

	installedUUIDs := make(map[string]bool)
	for _, installed := range installedPacks {
		installedUUIDs[installed.PackID] = true
	}

	for _, pack := range addon.GetAllPacks() {
		if !installedUUIDs[pack.Manifest.Header.UUID] {
			return fmt.Errorf("pack %s was not found in installed packs after installation",
				pack.Manifest.GetDisplayName())
		}
	}

	return nil
}
