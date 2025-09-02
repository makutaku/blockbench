package minecraft

import (
	"fmt"
	"os"
	"path/filepath"
)

// Server represents a Minecraft Bedrock server instance
type Server struct {
	Paths *ServerPaths
}

// NewServer creates a new Server instance
func NewServer(serverRoot string) (*Server, error) {
	paths, err := NewServerPaths(serverRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to configure server paths: %w", err)
	}

	if err := paths.ValidateServerStructure(); err != nil {
		return nil, fmt.Errorf("invalid server structure: %w", err)
	}

	return &Server{
		Paths: paths,
	}, nil
}

// InstallPack installs a pack to the server
func (s *Server) InstallPack(manifest *Manifest, packDir string) error {
	packType := manifest.GetPackType()

	var targetDir string
	var configFile string

	switch packType {
	case PackTypeBehavior:
		targetDir = s.Paths.BehaviorPacksDir
		configFile = s.Paths.WorldBehaviorPacks
	case PackTypeResource:
		targetDir = s.Paths.ResourcePacksDir
		configFile = s.Paths.WorldResourcePacks
	default:
		return fmt.Errorf("unknown pack type for pack %s", manifest.Header.UUID)
	}

	// Create pack directory name
	packDirName := fmt.Sprintf("%s_%s", manifest.GetDisplayName(), manifest.Header.UUID[:8])
	finalPackDir := filepath.Join(targetDir, packDirName)

	// Copy pack files
	if err := copyDir(packDir, finalPackDir); err != nil {
		return fmt.Errorf("failed to copy pack files: %w", err)
	}

	// Update world config
	config, err := LoadWorldConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	config = AddPackToConfig(config, manifest.Header.UUID, manifest.Header.Version)

	if err := SaveWorldConfig(configFile, config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// UninstallPack removes a pack from the server
func (s *Server) UninstallPack(packID string) error {
	// Try to find and remove from behavior packs
	behaviorConfig, err := LoadWorldConfig(s.Paths.WorldBehaviorPacks)
	if err != nil {
		return fmt.Errorf("failed to load behavior config: %w", err)
	}

	if behaviorConfig.HasPack(packID) {
		// Remove from behavior packs directory
		if err := s.removePackDir(s.Paths.BehaviorPacksDir, packID); err != nil {
			return fmt.Errorf("failed to remove behavior pack directory: %w", err)
		}

		// Update config
		behaviorConfig = RemovePackFromConfig(behaviorConfig, packID)
		if err := SaveWorldConfig(s.Paths.WorldBehaviorPacks, behaviorConfig); err != nil {
			return fmt.Errorf("failed to save behavior config: %w", err)
		}

		return nil
	}

	// Try to find and remove from resource packs
	resourceConfig, err := LoadWorldConfig(s.Paths.WorldResourcePacks)
	if err != nil {
		return fmt.Errorf("failed to load resource config: %w", err)
	}

	if resourceConfig.HasPack(packID) {
		// Remove from resource packs directory
		if err := s.removePackDir(s.Paths.ResourcePacksDir, packID); err != nil {
			return fmt.Errorf("failed to remove resource pack directory: %w", err)
		}

		// Update config
		resourceConfig = RemovePackFromConfig(resourceConfig, packID)
		if err := SaveWorldConfig(s.Paths.WorldResourcePacks, resourceConfig); err != nil {
			return fmt.Errorf("failed to save resource config: %w", err)
		}

		return nil
	}

	return fmt.Errorf("pack with ID %s not found", packID)
}

// ListInstalledPacks returns a list of all installed packs
func (s *Server) ListInstalledPacks() ([]InstalledPack, error) {
	var packs []InstalledPack

	// Load behavior packs
	behaviorConfig, err := LoadWorldConfig(s.Paths.WorldBehaviorPacks)
	if err != nil {
		return nil, fmt.Errorf("failed to load behavior config: %w", err)
	}

	for _, pack := range behaviorConfig {
		installedPack := InstalledPack{
			PackID:  pack.PackID,
			Version: pack.Version,
			Type:    PackTypeBehavior,
		}

		// Try to load manifest for more details
		if manifest, err := s.loadPackManifest(s.Paths.BehaviorPacksDir, pack.PackID); err == nil {
			installedPack.Name = manifest.GetDisplayName()
			installedPack.Description = manifest.Header.Description
		}

		packs = append(packs, installedPack)
	}

	// Load resource packs
	resourceConfig, err := LoadWorldConfig(s.Paths.WorldResourcePacks)
	if err != nil {
		return nil, fmt.Errorf("failed to load resource config: %w", err)
	}

	for _, pack := range resourceConfig {
		installedPack := InstalledPack{
			PackID:  pack.PackID,
			Version: pack.Version,
			Type:    PackTypeResource,
		}

		// Try to load manifest for more details
		if manifest, err := s.loadPackManifest(s.Paths.ResourcePacksDir, pack.PackID); err == nil {
			installedPack.Name = manifest.GetDisplayName()
			installedPack.Description = manifest.Header.Description
		}

		packs = append(packs, installedPack)
	}

	return packs, nil
}

// ListInstalledPacksWithDependencies returns installed packs with their dependency information
func (s *Server) ListInstalledPacksWithDependencies() ([]InstalledPackWithDependencies, error) {
	packs, err := s.ListInstalledPacks()
	if err != nil {
		return nil, fmt.Errorf("failed to list packs: %w", err)
	}

	var enrichedPacks []InstalledPackWithDependencies
	for _, pack := range packs {
		enriched := InstalledPackWithDependencies{
			InstalledPack: pack,
			Dependencies:  make([]string, 0),
			Modules:       make([]string, 0),
		}

		// Try to load manifest for dependency information
		if manifest, err := s.loadPackManifestByType(pack.PackID, pack.Type); err == nil {
			for _, dep := range manifest.Dependencies {
				if dep.UUID != "" {
					enriched.Dependencies = append(enriched.Dependencies, dep.UUID)
				}
				if dep.ModuleName != "" {
					enriched.Modules = append(enriched.Modules, dep.ModuleName)
				}
			}
		}

		enrichedPacks = append(enrichedPacks, enriched)
	}

	return enrichedPacks, nil
}

// loadPackManifestByType loads a pack manifest given its ID and type
func (s *Server) loadPackManifestByType(packID string, packType PackType) (*Manifest, error) {
	var baseDir string
	switch packType {
	case PackTypeBehavior:
		baseDir = s.Paths.BehaviorPacksDir
	case PackTypeResource:
		baseDir = s.Paths.ResourcePacksDir
	default:
		return nil, fmt.Errorf("unknown pack type: %s", packType)
	}

	return s.loadPackManifest(baseDir, packID)
}

// InstalledPack represents an installed pack
type InstalledPack struct {
	PackID      string   `json:"pack_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     [3]int   `json:"version"`
	Type        PackType `json:"type"`
}

// InstalledPackWithDependencies extends InstalledPack with dependency information
type InstalledPackWithDependencies struct {
	InstalledPack
	Dependencies []string `json:"dependencies"` // Pack UUIDs this pack depends on
	Modules      []string `json:"modules"`      // Script API modules used
}

// removePackDir removes a pack directory by searching for directories containing the pack ID
func (s *Server) removePackDir(baseDir, packID string) error {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", baseDir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		packPath := filepath.Join(baseDir, entry.Name())
		manifestPath := filepath.Join(packPath, "manifest.json")

		manifest, err := ParseManifest(manifestPath)
		if err != nil {
			continue // Skip if can't read manifest
		}

		if manifest.Header.UUID == packID {
			return os.RemoveAll(packPath)
		}
	}

	return fmt.Errorf("pack directory not found for pack ID %s", packID)
}

// loadPackManifest loads a manifest for an installed pack
func (s *Server) loadPackManifest(baseDir, packID string) (*Manifest, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", baseDir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		manifestPath := filepath.Join(baseDir, entry.Name(), "manifest.json")
		manifest, err := ParseManifest(manifestPath)
		if err != nil {
			continue // Skip if can't read manifest
		}

		if manifest.Header.UUID == packID {
			return manifest, nil
		}
	}

	return nil, fmt.Errorf("manifest not found for pack ID %s", packID)
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

		// Copy file
		// #nosec G304 - path is within controlled extraction directory
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		// #nosec G304 - dstPath is within validated server directory structure
		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		if _, err := srcFile.WriteTo(dstFile); err != nil {
			return err
		}

		return os.Chmod(dstPath, info.Mode())
	})
}
