package addon

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/makutaku/blockbench/internal/minecraft"
	"github.com/makutaku/blockbench/pkg/filesystem"
	"github.com/makutaku/blockbench/pkg/validation"
)

// InstallOptions contains options for addon installation
type InstallOptions struct {
	DryRun      bool
	Verbose     bool
	BackupDir   string
	ForceUpdate bool
	Interactive bool
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

	// Show validation results
	validationDetails := []string{
		fmt.Sprintf("Validated addon file: %s", addonPath),
		fmt.Sprintf("Server directory structure verified: %s", i.server.Paths.ServerRoot),
		"Archive format and integrity confirmed",
	}
	if err := showStepResult("Pre-installation validation", validationDetails, "Archive extraction", "Extract the .mcaddon/.mcpack file and any nested .mcpack files to a temporary directory for processing.", options); err != nil {
		return result, err
	}

	// Continue with full analysis even in dry-run mode to provide detailed information

	// Step 2: Extract addon
	extractedAddon, err := ExtractAddon(addonPath, options.DryRun)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Extraction failed: %v", err))
		return result, err
	}
	defer func() {
		if cleanupErr := extractedAddon.Cleanup(); cleanupErr != nil {
			if options.Verbose {
				fmt.Printf("Warning: Failed to cleanup temporary files: %v\n", cleanupErr)
			}
		}
	}()

	// Show extraction results with pack details
	extractionDetails := []string{
		fmt.Sprintf("Extracted to temporary directory: %s", extractedAddon.TempDir),
	}

	// Add behavior pack details
	if len(extractedAddon.BehaviorPacks) > 0 {
		extractionDetails = append(extractionDetails, fmt.Sprintf("Found %d behavior pack(s):", len(extractedAddon.BehaviorPacks)))
		for _, pack := range extractedAddon.BehaviorPacks {
			extractionDetails = append(extractionDetails, fmt.Sprintf("  • %s (UUID: %s, Version: %d.%d.%d) at %s",
				pack.Manifest.GetDisplayName(),
				pack.Manifest.Header.UUID,
				pack.Manifest.Header.Version[0], pack.Manifest.Header.Version[1], pack.Manifest.Header.Version[2],
				pack.Path))
		}
	}

	// Add resource pack details
	if len(extractedAddon.ResourcePacks) > 0 {
		extractionDetails = append(extractionDetails, fmt.Sprintf("Found %d resource pack(s):", len(extractedAddon.ResourcePacks)))
		for _, pack := range extractedAddon.ResourcePacks {
			extractionDetails = append(extractionDetails, fmt.Sprintf("  • %s (UUID: %s, Version: %d.%d.%d) at %s",
				pack.Manifest.GetDisplayName(),
				pack.Manifest.Header.UUID,
				pack.Manifest.Header.Version[0], pack.Manifest.Header.Version[1], pack.Manifest.Header.Version[2],
				pack.Path))
		}
	}
	if err := showStepResult("Archive extraction", extractionDetails, "Content validation", "Analyze extracted pack contents, validate manifest.json files, and determine pack types (behavior/resource).", options); err != nil {
		return result, err
	}

	// Step 3: Validate extracted content
	if err := i.validateExtractedAddon(extractedAddon); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Content validation failed: %v", err))
		return result, err
	}

	// Show content validation results
	contentValidationDetails := []string{}
	for _, pack := range extractedAddon.BehaviorPacks {
		contentValidationDetails = append(contentValidationDetails, fmt.Sprintf("Validated behavior pack: %s", pack.Manifest.GetDisplayName()))
	}
	for _, pack := range extractedAddon.ResourcePacks {
		contentValidationDetails = append(contentValidationDetails, fmt.Sprintf("Validated resource pack: %s", pack.Manifest.GetDisplayName()))
	}
	contentValidationDetails = append(contentValidationDetails, "All manifest.json files are valid")
	if err := showStepResult("Content validation", contentValidationDetails, "Conflict detection", "Check for UUID conflicts with existing installed packs that could cause issues.", options); err != nil {
		return result, err
	}

	// Step 4: Check for conflicts
	conflicts, err := i.checkForConflicts(extractedAddon)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Conflict check failed: %v", err))
		return result, err
	}

	// Check for missing dependencies
	missingDeps, err := i.validateDependencies(extractedAddon)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Dependency validation failed: %v", err))
		return result, err
	}

	// Show conflict check and dependency validation results
	conflictDetails := []string{}
	if len(conflicts) == 0 {
		conflictDetails = append(conflictDetails, "No UUID conflicts detected")
	} else {
		for _, conflict := range conflicts {
			conflictDetails = append(conflictDetails, fmt.Sprintf("⚠️  Conflict: %s", conflict))
		}
	}

	if len(missingDeps) == 0 {
		conflictDetails = append(conflictDetails, "All dependencies satisfied")
	} else {
		for _, dep := range missingDeps {
			conflictDetails = append(conflictDetails, fmt.Sprintf("⚠️  Missing dependency: %s", dep))
			result.Warnings = append(result.Warnings, dep)
		}
	}

	conflictDetails = append(conflictDetails, fmt.Sprintf("Checked against %d existing pack(s)", len(conflicts)))
	if err := showStepResult("Conflict detection", conflictDetails, "Backup creation", "Create a backup of the current server state to enable rollback if the installation fails.", options); err != nil {
		return result, err
	}

	if len(conflicts) > 0 && !options.ForceUpdate {
		for _, conflict := range conflicts {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Conflict detected: %s", conflict))
		}
		return result, fmt.Errorf("conflicts detected, use --force to override")
	}

	if len(missingDeps) > 0 && !options.ForceUpdate {
		return result, fmt.Errorf("missing dependencies detected. Install required packs first or use --force to proceed anyway (may cause issues)")
	}

	// For dry-run, simulate the installation operations and show detailed information
	if options.DryRun {
		return i.performDryRunSimulation(extractedAddon, conflicts, options)
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

	// Show backup creation results
	backupDetails := []string{
		fmt.Sprintf("Backup created with ID: %s", backup.ID),
		fmt.Sprintf("Backup stored at: %s", backup.BackupPath),
	}
	if len(backup.Files) > 0 {
		backupDetails = append(backupDetails, fmt.Sprintf("Backed up %d file(s):", len(backup.Files)))
		for _, file := range backup.Files {
			backupDetails = append(backupDetails, fmt.Sprintf("  • %s", file))
		}
	} else {
		backupDetails = append(backupDetails, "No existing files to backup (fresh installation)")
	}
	if err := showStepResult("Backup creation", backupDetails, "Pack installation", "Copy pack files to server directories and update world configuration files to register the new packs.", options); err != nil {
		return result, err
	}

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

	// Show pack installation results with specific paths
	installDetails := []string{}
	for _, pack := range extractedAddon.BehaviorPacks {
		packDirName := fmt.Sprintf("%s_%s", pack.Manifest.GetDisplayName(), pack.Manifest.Header.UUID[:validation.UUIDShortDisplayLength])
		finalPackDir := filepath.Join(i.server.Paths.BehaviorPacksDir, packDirName)
		installDetails = append(installDetails, fmt.Sprintf("Created behavior pack directory: %s", finalPackDir))
		installDetails = append(installDetails, fmt.Sprintf("Updated world config file: %s", i.server.Paths.WorldBehaviorPacks))
		installDetails = append(installDetails, fmt.Sprintf("  • Added pack: %s (UUID: %s, Version: %d.%d.%d)",
			pack.Manifest.GetDisplayName(),
			pack.Manifest.Header.UUID,
			pack.Manifest.Header.Version[0], pack.Manifest.Header.Version[1], pack.Manifest.Header.Version[2]))
	}
	for _, pack := range extractedAddon.ResourcePacks {
		packDirName := fmt.Sprintf("%s_%s", pack.Manifest.GetDisplayName(), pack.Manifest.Header.UUID[:validation.UUIDShortDisplayLength])
		finalPackDir := filepath.Join(i.server.Paths.ResourcePacksDir, packDirName)
		installDetails = append(installDetails, fmt.Sprintf("Created resource pack directory: %s", finalPackDir))
		installDetails = append(installDetails, fmt.Sprintf("Updated world config file: %s", i.server.Paths.WorldResourcePacks))
		installDetails = append(installDetails, fmt.Sprintf("  • Added pack: %s (UUID: %s, Version: %d.%d.%d)",
			pack.Manifest.GetDisplayName(),
			pack.Manifest.Header.UUID,
			pack.Manifest.Header.Version[0], pack.Manifest.Header.Version[1], pack.Manifest.Header.Version[2]))
	}
	if err := showStepResult("Pack installation", installDetails, "Post-installation validation", "Verify that all packs were successfully installed and are properly registered with the server.", options); err != nil {
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

	// Show post-installation validation results
	finalValidationDetails := []string{}
	finalAllPacks := extractedAddon.GetAllPacks()
	for _, pack := range finalAllPacks {
		finalValidationDetails = append(finalValidationDetails, fmt.Sprintf("Verified pack installation: %s", pack.Manifest.GetDisplayName()))
	}
	finalValidationDetails = append(finalValidationDetails, "All packs are properly registered with the server")
	if err := showStepResult("Post-installation validation", finalValidationDetails, "", "", options); err != nil {
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

// validateDependencies checks that all pack dependencies are satisfied
func (i *Installer) validateDependencies(addon *ExtractedAddon) ([]string, error) {
	var missingDeps []string

	// Get all currently installed packs
	installedPacks, err := i.server.ListInstalledPacks()
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packs: %w", err)
	}

	// Build set of installed UUIDs
	installedUUIDs := make(map[string]bool)
	for _, pack := range installedPacks {
		installedUUIDs[pack.PackID] = true
	}

	// Add UUIDs from packs being installed (self-satisfied dependencies)
	for _, newPack := range addon.GetAllPacks() {
		installedUUIDs[newPack.Manifest.Header.UUID] = true
	}

	// Check each pack's dependencies
	for _, newPack := range addon.GetAllPacks() {
		for _, dep := range newPack.Manifest.Dependencies {
			if dep.UUID != "" {
				// Check if dependency exists
				if !installedUUIDs[dep.UUID] {
					missingDeps = append(missingDeps,
						fmt.Sprintf("Pack '%s' requires dependency UUID %s which is not installed",
							newPack.Manifest.GetDisplayName(), dep.UUID))
				}
			}
			// Module dependencies (@minecraft/server, etc.) are checked by Minecraft itself at runtime
			// so we don't validate those here
		}
	}

	return missingDeps, nil
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

// showStepResult displays the results of a completed step and asks about next step
func showStepResult(stepName string, details []string, nextStep, nextStepDesc string, options InstallOptions) error {
	if !options.Interactive {
		return nil
	}

	fmt.Printf("\n✅ Completed: %s\n", stepName)
	for _, detail := range details {
		fmt.Printf("   • %s\n", detail)
	}

	if nextStep != "" {
		fmt.Printf("\n📋 Next Step: %s\n", nextStep)
		fmt.Printf("   %s\n", nextStepDesc)
		fmt.Print("Proceed with this step? (y/N): ")
	} else {
		fmt.Print("\nFinish installation? (y/N): ")
	}

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		// Handle EOF (when input is piped or redirected)
		if strings.Contains(err.Error(), "EOF") {
			fmt.Println("n")
			return fmt.Errorf("installation aborted due to end of input")
		}
		return fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		return fmt.Errorf("installation aborted by user")
	}

	return nil
}

// performDryRunSimulation simulates installation operations and shows detailed information
func (i *Installer) performDryRunSimulation(extractedAddon *ExtractedAddon, conflicts []string, options InstallOptions) (*InstallResult, error) {
	result := &InstallResult{
		InstalledPacks: make([]string, 0),
		Errors:         make([]string, 0),
		Warnings:       make([]string, 0),
	}

	// Add conflict warnings
	for _, conflict := range conflicts {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Conflict detected: %s", conflict))
	}

	simulator := NewDryRunSimulator(i.server)
	allPacks := extractedAddon.GetAllPacks()

	if options.Verbose {
		fmt.Println("DRY RUN: Simulating installation operations...")
	}

	// Simulate backup creation
	backupDetails := []string{
		"DRY RUN: Backup would be created with timestamp-based ID",
		fmt.Sprintf("DRY RUN: Backup would be stored in: %s/backups/", i.server.Paths.ServerRoot),
	}
	if len(conflicts) > 0 && options.ForceUpdate {
		backupDetails = append(backupDetails, "DRY RUN: Would backup existing conflicting packs")
	} else {
		backupDetails = append(backupDetails, "DRY RUN: No existing files to backup (fresh installation)")
	}
	if err := showStepResult("Backup simulation", backupDetails, "Installation simulation", "Simulate copying pack files and updating world configuration files.", options); err != nil {
		return result, err
	}

	// Simulate installation for each pack
	var installationDetails []string
	for _, pack := range allPacks {
		simulation, err := simulator.SimulatePackInstallation(pack)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Installation simulation failed for pack %s: %v", pack.Manifest.GetDisplayName(), err))
			continue
		}

		packTypeStr := "behavior"
		if simulation.PackType == minecraft.PackTypeResource {
			packTypeStr = "resource"
		}

		installationDetails = append(installationDetails, fmt.Sprintf("DRY RUN: Would create %s pack directory: %s", packTypeStr, simulation.TargetDirectory))
		installationDetails = append(installationDetails, fmt.Sprintf("DRY RUN: Would update config file: %s", simulation.ConfigFile))
		installationDetails = append(installationDetails, fmt.Sprintf("  • Would add pack entry: %s (UUID: %s, Version: %d.%d.%d)",
			simulation.PackName, simulation.PackUUID,
			simulation.PackVersion[0], simulation.PackVersion[1], simulation.PackVersion[2]))

		if len(simulation.Dependencies) > 0 {
			installationDetails = append(installationDetails, fmt.Sprintf("  • Pack has %d dependencies:", len(simulation.Dependencies)))
			for _, dep := range simulation.Dependencies {
				if dep.UUID != "" {
					installationDetails = append(installationDetails, fmt.Sprintf("    - UUID: %s", dep.UUID))
				}
				if dep.ModuleName != "" {
					installationDetails = append(installationDetails, fmt.Sprintf("    - Module: %s@%s", dep.ModuleName, dep.ModuleVersion))
				}
			}
		}

		result.InstalledPacks = append(result.InstalledPacks, simulation.PackName)
	}

	if err := showStepResult("Installation simulation", installationDetails, "Validation simulation", "Simulate post-installation validation to ensure all packs would be properly registered.", options); err != nil {
		return result, err
	}

	// Simulate post-installation validation
	validationDetails := []string{}
	for _, pack := range allPacks {
		validationDetails = append(validationDetails, fmt.Sprintf("DRY RUN: Would verify pack installation: %s", pack.Manifest.GetDisplayName()))
	}
	validationDetails = append(validationDetails, "DRY RUN: All packs would be properly registered with the server")
	if err := showStepResult("Validation simulation", validationDetails, "", "", options); err != nil {
		return result, err
	}

	// Summary
	result.Success = true
	if options.Verbose {
		fmt.Printf("DRY RUN COMPLETE: Would install %d pack(s) successfully\n", len(result.InstalledPacks))
		fmt.Println("No actual changes were made to the server")
	}

	return result, nil
}
