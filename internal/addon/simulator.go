package addon

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/makutaku/blockbench/internal/minecraft"
)

// DryRunSimulator provides simulation of file operations for dry-run mode
type DryRunSimulator struct {
	server *minecraft.Server
}

// NewDryRunSimulator creates a new dry-run simulator
func NewDryRunSimulator(server *minecraft.Server) *DryRunSimulator {
	return &DryRunSimulator{
		server: server,
	}
}

// SimulatedInstallOperation represents a simulated installation operation
type SimulatedInstallOperation struct {
	PackName        string
	PackUUID        string
	PackVersion     [3]int
	PackType        minecraft.PackType
	SourcePath      string
	TargetDirectory string
	ConfigFile      string
	ConfigEntry     minecraft.PackReference
	Conflicts       []string
	Dependencies    []minecraft.ManifestDependency
}

// SimulatedUninstallOperation represents a simulated uninstallation operation
type SimulatedUninstallOperation struct {
	PackName            string
	PackUUID            string
	PackType            minecraft.PackType
	DirectoryToRemove   string
	ConfigFile          string
	ConfigEntryToRemove minecraft.PackReference
	DependentPacks      []string
	FilesToBackup       []string
}

// SimulatePackInstallation simulates the installation of a single pack
func (s *DryRunSimulator) SimulatePackInstallation(pack *ExtractedPack) (*SimulatedInstallOperation, error) {
	manifest := pack.Manifest
	packType := manifest.GetPackType()

	var targetDir string
	var configFile string

	switch packType {
	case minecraft.PackTypeBehavior:
		targetDir = s.server.Paths.BehaviorPacksDir
		configFile = s.server.Paths.WorldBehaviorPacks
	case minecraft.PackTypeResource:
		targetDir = s.server.Paths.ResourcePacksDir
		configFile = s.server.Paths.WorldResourcePacks
	default:
		return nil, fmt.Errorf("unknown pack type for pack %s", manifest.Header.UUID)
	}

	// Create pack directory name (same logic as real installation)
	packDirName := fmt.Sprintf("%s_%s", manifest.GetDisplayName(), manifest.Header.UUID[:8])
	finalPackDir := filepath.Join(targetDir, packDirName)

	// Simulate config entry that would be added
	configEntry := minecraft.PackReference{
		PackID:  manifest.Header.UUID,
		Version: manifest.Header.Version,
	}

	// Check for conflicts with existing packs
	conflicts, err := s.checkInstallationConflicts(manifest.Header.UUID)
	if err != nil {
		return nil, fmt.Errorf("failed to check conflicts: %w", err)
	}

	return &SimulatedInstallOperation{
		PackName:        manifest.GetDisplayName(),
		PackUUID:        manifest.Header.UUID,
		PackVersion:     manifest.Header.Version,
		PackType:        packType,
		SourcePath:      pack.Path,
		TargetDirectory: finalPackDir,
		ConfigFile:      configFile,
		ConfigEntry:     configEntry,
		Conflicts:       conflicts,
		Dependencies:    manifest.Dependencies,
	}, nil
}

// SimulatePackUninstallation simulates the uninstallation of a pack
func (s *DryRunSimulator) SimulatePackUninstallation(packID string) (*SimulatedUninstallOperation, error) {
	// Find the pack in installed packs
	installedPacks, err := s.server.ListInstalledPacks()
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packs: %w", err)
	}

	var targetPack *minecraft.InstalledPack
	for _, pack := range installedPacks {
		if pack.PackID == packID {
			targetPack = &pack
			break
		}
	}

	if targetPack == nil {
		return nil, fmt.Errorf("pack %s is not installed", packID)
	}

	// Determine what would be removed
	var configFile string
	switch targetPack.Type {
	case minecraft.PackTypeBehavior:
		configFile = s.server.Paths.WorldBehaviorPacks
	case minecraft.PackTypeResource:
		configFile = s.server.Paths.WorldResourcePacks
	default:
		return nil, fmt.Errorf("unknown pack type for pack %s", packID)
	}

	// Find the pack directory path
	packPath, err := s.findPackDirectory(targetPack.PackID, targetPack.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to find pack directory: %w", err)
	}

	// Check for dependent packs
	dependents, err := s.checkUninstallationDependencies(packID)
	if err != nil {
		return nil, fmt.Errorf("failed to check dependencies: %w", err)
	}

	// Simulate what files would be backed up
	filesToBackup := []string{
		packPath,
		configFile,
	}

	// Create config entry that would be removed
	configEntry := minecraft.PackReference{
		PackID:  targetPack.PackID,
		Version: targetPack.Version, // Use the version from the installed pack
	}

	return &SimulatedUninstallOperation{
		PackName:            targetPack.Name,
		PackUUID:            targetPack.PackID,
		PackType:            targetPack.Type,
		DirectoryToRemove:   packPath,
		ConfigFile:          configFile,
		ConfigEntryToRemove: configEntry,
		DependentPacks:      dependents,
		FilesToBackup:       filesToBackup,
	}, nil
}

// checkInstallationConflicts checks for UUID conflicts during installation
func (s *DryRunSimulator) checkInstallationConflicts(newPackUUID string) ([]string, error) {
	var conflicts []string

	installedPacks, err := s.server.ListInstalledPacks()
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packs: %w", err)
	}

	for _, installedPack := range installedPacks {
		if installedPack.PackID == newPackUUID {
			conflicts = append(conflicts, fmt.Sprintf("Pack %s (UUID: %s) is already installed",
				installedPack.Name, installedPack.PackID))
		}
	}

	return conflicts, nil
}

// checkUninstallationDependencies checks what packs depend on the pack being removed
func (s *DryRunSimulator) checkUninstallationDependencies(packID string) ([]string, error) {
	var dependents []string

	installedPacks, err := s.server.ListInstalledPacks()
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packs: %w", err)
	}

	for _, installedPack := range installedPacks {
		// Skip the pack being removed
		if installedPack.PackID == packID {
			continue
		}

		// Try to load the pack's manifest to check dependencies
		packPath, err := s.findPackDirectory(installedPack.PackID, installedPack.Type)
		if err != nil {
			// If we can't find the pack directory, skip dependency check for this pack
			continue
		}
		manifestPath := filepath.Join(packPath, "manifest.json")
		manifest, err := minecraft.ParseManifest(manifestPath)
		if err != nil {
			// If we can't load the manifest, we can't check dependencies
			continue
		}

		for _, dep := range manifest.Dependencies {
			if dep.UUID == packID {
				dependents = append(dependents, installedPack.Name)
				break
			}
		}
	}

	return dependents, nil
}

// findPackDirectory finds the directory path for an installed pack by searching pack directories
func (s *DryRunSimulator) findPackDirectory(packID string, packType minecraft.PackType) (string, error) {
	var baseDir string
	switch packType {
	case minecraft.PackTypeBehavior:
		baseDir = s.server.Paths.BehaviorPacksDir
	case minecraft.PackTypeResource:
		baseDir = s.server.Paths.ResourcePacksDir
	default:
		return "", fmt.Errorf("unknown pack type")
	}

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return "", fmt.Errorf("failed to read directory %s: %w", baseDir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if this directory contains the pack by looking for the UUID in manifest
		packPath := filepath.Join(baseDir, entry.Name())
		manifestPath := filepath.Join(packPath, "manifest.json")

		if manifest, err := minecraft.ParseManifest(manifestPath); err == nil {
			if manifest.Header.UUID == packID {
				return packPath, nil
			}
		}
	}

	return "", fmt.Errorf("pack directory not found for pack ID: %s", packID)
}
