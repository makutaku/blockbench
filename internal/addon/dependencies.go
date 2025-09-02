package addon

import (
	"fmt"
	"path/filepath"

	"github.com/makutaku/blockbench/internal/minecraft"
)

// PackRelationship represents a pack and its dependency relationships
type PackRelationship struct {
	Pack         minecraft.InstalledPack
	Dependencies []string // UUIDs this pack depends on
	Dependents   []string // UUIDs that depend on this pack
	Modules      []string // Script API modules used
	Manifest     *minecraft.Manifest
}

// DependencyGroup represents logically grouped packs by their relationships
type DependencyGroup struct {
	RootPacks       []PackRelationship   // Packs that others depend on
	DependentPacks  []PackRelationship   // Packs that depend on others
	StandalonePacks []PackRelationship   // Packs with no dependencies/dependents
	CircularGroups  [][]PackRelationship // Circular dependency chains
}

// DependencyAnalyzer analyzes pack dependencies and relationships
type DependencyAnalyzer struct {
	server *minecraft.Server
}

// NewDependencyAnalyzer creates a new dependency analyzer
func NewDependencyAnalyzer(server *minecraft.Server) *DependencyAnalyzer {
	return &DependencyAnalyzer{
		server: server,
	}
}

// AnalyzeDependencies builds a complete dependency graph for all installed packs
func (da *DependencyAnalyzer) AnalyzeDependencies() (*DependencyGroup, error) {
	// Get all installed packs
	installedPacks, err := da.server.ListInstalledPacks()
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packs: %w", err)
	}

	// Build relationships for each pack
	relationships := make(map[string]*PackRelationship)
	for _, pack := range installedPacks {
		rel, err := da.buildPackRelationship(pack)
		if err != nil {
			// If we can't analyze a pack, treat it as standalone
			rel = &PackRelationship{
				Pack:         pack,
				Dependencies: []string{},
				Dependents:   []string{},
				Modules:      []string{},
			}
		}
		relationships[pack.PackID] = rel
	}

	// Calculate dependents (reverse relationships)
	da.calculateDependents(relationships)

	// Group packs by their relationship patterns
	return da.groupPacksByRelationships(relationships), nil
}

// buildPackRelationship analyzes a single pack's dependencies
func (da *DependencyAnalyzer) buildPackRelationship(pack minecraft.InstalledPack) (*PackRelationship, error) {
	rel := &PackRelationship{
		Pack:         pack,
		Dependencies: make([]string, 0),
		Dependents:   make([]string, 0),
		Modules:      make([]string, 0),
	}

	// Load the pack's manifest to get dependency information
	manifest, err := da.loadPackManifest(pack.PackID, pack.Type)
	if err != nil {
		return rel, fmt.Errorf("failed to load manifest for pack %s: %w", pack.PackID, err)
	}

	rel.Manifest = manifest

	// Extract dependencies
	for _, dep := range manifest.Dependencies {
		if dep.UUID != "" {
			// Pack dependency
			rel.Dependencies = append(rel.Dependencies, dep.UUID)
		}
		if dep.ModuleName != "" {
			// Module dependency (Script API)
			rel.Modules = append(rel.Modules, dep.ModuleName)
		}
	}

	return rel, nil
}

// loadPackManifest loads a manifest for an installed pack
func (da *DependencyAnalyzer) loadPackManifest(packID string, packType minecraft.PackType) (*minecraft.Manifest, error) {
	var baseDir string
	switch packType {
	case minecraft.PackTypeBehavior:
		baseDir = da.server.Paths.BehaviorPacksDir
	case minecraft.PackTypeResource:
		baseDir = da.server.Paths.ResourcePacksDir
	default:
		return nil, fmt.Errorf("unknown pack type: %s", packType)
	}

	// Find the pack directory by looking for the UUID in manifest files
	packPath, err := da.findPackDirectory(baseDir, packID)
	if err != nil {
		return nil, fmt.Errorf("failed to find pack directory: %w", err)
	}

	manifestPath := filepath.Join(packPath, "manifest.json")
	return minecraft.ParseManifest(manifestPath)
}

// findPackDirectory finds the directory path for a pack by UUID
func (da *DependencyAnalyzer) findPackDirectory(baseDir, packID string) (string, error) {
	// Use the existing findPackDirectory logic from simulator
	simulator := NewDryRunSimulator(da.server)

	// Try behavior pack directory first
	if path, err := simulator.findPackDirectory(packID, minecraft.PackTypeBehavior); err == nil {
		return path, nil
	}

	// Then try resource pack directory
	if path, err := simulator.findPackDirectory(packID, minecraft.PackTypeResource); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("pack directory not found for UUID: %s", packID)
}

// calculateDependents builds reverse dependency relationships
func (da *DependencyAnalyzer) calculateDependents(relationships map[string]*PackRelationship) {
	for packID, rel := range relationships {
		for _, depID := range rel.Dependencies {
			if depRel, exists := relationships[depID]; exists {
				depRel.Dependents = append(depRel.Dependents, packID)
			}
		}
	}
}

// groupPacksByRelationships categorizes packs based on their dependency patterns
func (da *DependencyAnalyzer) groupPacksByRelationships(relationships map[string]*PackRelationship) *DependencyGroup {
	group := &DependencyGroup{
		RootPacks:       make([]PackRelationship, 0),
		DependentPacks:  make([]PackRelationship, 0),
		StandalonePacks: make([]PackRelationship, 0),
		CircularGroups:  make([][]PackRelationship, 0),
	}

	// Track processed packs to handle circular dependencies
	processed := make(map[string]bool)

	for packID, rel := range relationships {
		if processed[packID] {
			continue // Already processed as part of a circular group
		}

		hasDependencies := len(rel.Dependencies) > 0
		hasDependents := len(rel.Dependents) > 0

		if !hasDependencies && !hasDependents {
			// Standalone pack
			group.StandalonePacks = append(group.StandalonePacks, *rel)
		} else if !hasDependencies && hasDependents {
			// Root pack (others depend on it, but it doesn't depend on anything)
			group.RootPacks = append(group.RootPacks, *rel)
		} else if hasDependencies && !hasDependents {
			// Dependent pack (depends on others, but nothing depends on it)
			group.DependentPacks = append(group.DependentPacks, *rel)
		} else {
			// Pack has both dependencies and dependents - could be circular
			// For now, classify as dependent pack
			// TODO: Implement proper circular dependency detection
			group.DependentPacks = append(group.DependentPacks, *rel)
		}

		processed[packID] = true
	}

	return group
}

// GetDependencyTree builds a tree structure for visualization
func (da *DependencyAnalyzer) GetDependencyTree(group *DependencyGroup) map[string][]PackRelationship {
	tree := make(map[string][]PackRelationship)

	// Build tree starting from root packs
	for _, root := range group.RootPacks {
		children := make([]PackRelationship, 0)
		for _, dependent := range group.DependentPacks {
			// Check if this dependent pack depends on the root
			for _, depID := range dependent.Dependencies {
				if depID == root.Pack.PackID {
					children = append(children, dependent)
					break
				}
			}
		}
		tree[root.Pack.PackID] = children
	}

	// Add standalone packs as their own "roots"
	for _, standalone := range group.StandalonePacks {
		tree[standalone.Pack.PackID] = []PackRelationship{}
	}

	return tree
}

// FindPacksByName searches for packs by name (for uninstall command)
func (da *DependencyAnalyzer) FindPacksByName(searchTerm string) ([]PackRelationship, error) {
	group, err := da.AnalyzeDependencies()
	if err != nil {
		return nil, fmt.Errorf("failed to analyze dependencies: %w", err)
	}

	var matches []PackRelationship
	allPacks := append(append(group.RootPacks, group.DependentPacks...), group.StandalonePacks...)

	for _, rel := range allPacks {
		if containsIgnoreCase(rel.Pack.Name, searchTerm) || rel.Pack.PackID == searchTerm {
			matches = append(matches, rel)
		}
	}

	return matches, nil
}
