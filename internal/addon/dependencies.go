package addon

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/makutaku/blockbench/internal/minecraft"
	"github.com/makutaku/blockbench/pkg/validation"
)

// PackRelationship represents a pack and its complete dependency graph
//
// This structure captures both forward dependencies (what this pack needs)
// and reverse dependencies (what needs this pack), enabling impact analysis
// for operations like uninstallation.
type PackRelationship struct {
	Pack         minecraft.InstalledPack // The pack itself (UUID, name, type, version)
	Dependencies []string                // UUIDs of packs this pack depends on (forward edges)
	Dependents   []string                // UUIDs of packs that depend on this pack (reverse edges)
	Modules      []string                // Script API modules used (e.g., "@minecraft/server")
	Manifest     *minecraft.Manifest     // Full manifest data for detailed analysis
}

// DependencyGroup represents packs categorized by their relationship patterns
//
// Packs are classified into four mutually exclusive categories based on their
// dependency graph topology. This enables sophisticated listing and visualization.
//
// Example classification:
//   - Pack A (no dependencies, Pack B depends on it) → RootPack
//   - Pack B (depends on Pack A) → DependentPack
//   - Pack C (no dependencies, nothing depends on it) → StandalonePack
//   - Packs D→E→F→D → CircularGroup
type DependencyGroup struct {
	RootPacks       []PackRelationship   // Foundation packs others depend on (in-degree > 0, out-degree = 0)
	DependentPacks  []PackRelationship   // Packs requiring other packs (out-degree > 0, not in circular group)
	StandalonePacks []PackRelationship   // Self-contained packs (in-degree = 0, out-degree = 0)
	CircularGroups  [][]PackRelationship // Circular dependency chains detected by DFS
}

// DependencyAnalyzer analyzes pack dependencies and relationship graphs
//
// Provides methods to:
// - Build complete dependency graphs from installed packs
// - Detect circular dependencies using DFS
// - Group packs by relationship patterns
// - Validate dependency satisfaction before operations
type DependencyAnalyzer struct {
	server *minecraft.Server // Server instance for pack discovery and manifest loading
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
			// This can happen if the manifest is corrupted or missing
			fmt.Fprintf(os.Stderr, "Warning: Could not analyze pack %s (%s): %v\n", pack.Name, pack.PackID, err)
			fmt.Fprintf(os.Stderr, "  Treating pack as standalone (no dependencies)\n")
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
			// Pack dependency - validate and normalize UUID
			if !validation.ValidateUUID(dep.UUID) {
				// Log warning but don't fail - manifest is already installed
				fmt.Fprintf(os.Stderr, "Warning: Invalid dependency UUID format '%s' in pack %s\n",
					dep.UUID, pack.PackID)
				fmt.Fprintf(os.Stderr, "  Skipping this dependency in analysis\n")
				continue
			}
			normalizedUUID := validation.NormalizeUUID(dep.UUID)
			rel.Dependencies = append(rel.Dependencies, normalizedUUID)
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

// detectCircularDependencies finds circular dependency chains using Depth-First Search (DFS)
//
// Algorithm:
// 1. Maintain a "visited" set to track all nodes seen during the search
// 2. Maintain a "recursionStack" to track nodes in the current DFS path
// 3. For each unvisited pack, perform DFS traversal
// 4. If we encounter a node that's in the recursion stack, we've found a cycle
// 5. Extract the cycle from the path and add to results (avoiding duplicates)
//
// Time Complexity: O(V + E) where V = number of packs, E = number of dependencies
// Space Complexity: O(V) for visited/recursion stack tracking
//
// Example:
//   If Pack A → Pack B → Pack C → Pack A, this will detect the cycle [A, B, C]
func (da *DependencyAnalyzer) detectCircularDependencies(relationships map[string]*PackRelationship) [][]PackRelationship {
	// visited: all packs we've seen during any DFS traversal
	visited := make(map[string]bool)
	// recursionStack: packs currently in the DFS path (detects back edges)
	recursionStack := make(map[string]bool)
	// allCycles: all detected circular dependency chains
	var allCycles [][]PackRelationship
	// cycleFound: deduplicates cycles (same cycle from different starting points)
	cycleFound := make(map[string]bool)

	// DFS closure that traverses the dependency graph
	// Returns the cycle path if one is found, nil otherwise
	var dfs func(packID string, path []string) []string
	dfs = func(packID string, path []string) []string {
		// Mark current pack as visited and add to recursion stack
		visited[packID] = true
		recursionStack[packID] = true
		path = append(path, packID)

		rel, exists := relationships[packID]
		if !exists {
			// Pack has no dependencies, remove from recursion stack and return
			recursionStack[packID] = false
			return nil
		}

		// Explore all dependencies
		for _, depID := range rel.Dependencies {
			if !visited[depID] {
				// Unvisited dependency: recurse deeper
				if cyclePath := dfs(depID, path); cyclePath != nil {
					return cyclePath
				}
			} else if recursionStack[depID] {
				// Back edge detected! This dependency is in our current path = circular dependency
				// Extract the cycle portion from the path
				cycleStart := -1
				for i, id := range path {
					if id == depID {
						cycleStart = i
						break
					}
				}
				if cycleStart != -1 {
					// Return path from cycle start to current (forms the cycle)
					return path[cycleStart:]
				}
			}
			// If visited but not in recursion stack, it's a cross edge (already explored)
		}

		// Done exploring this pack, remove from recursion stack
		recursionStack[packID] = false
		return nil
	}

	// Find all cycles by starting DFS from each unvisited pack
	for packID := range relationships {
		if !visited[packID] {
			if cyclePath := dfs(packID, []string{}); cyclePath != nil {
				// Check if we've already found this cycle (avoid duplicates)
				// Create unique key from sorted pack IDs in cycle
				cycleKey := ""
				for _, id := range cyclePath {
					cycleKey += id + ","
				}
				if !cycleFound[cycleKey] {
					cycleFound[cycleKey] = true

					// Convert UUID path to PackRelationship objects
					cyclePacks := make([]PackRelationship, 0, len(cyclePath))
					for _, id := range cyclePath {
						if rel, ok := relationships[id]; ok {
							cyclePacks = append(cyclePacks, *rel)
						}
					}
					if len(cyclePacks) > 0 {
						allCycles = append(allCycles, cyclePacks)
					}
				}
			}
		}
	}

	return allCycles
}

// groupPacksByRelationships categorizes packs based on their dependency patterns
func (da *DependencyAnalyzer) groupPacksByRelationships(relationships map[string]*PackRelationship) *DependencyGroup {
	group := &DependencyGroup{
		RootPacks:       make([]PackRelationship, 0),
		DependentPacks:  make([]PackRelationship, 0),
		StandalonePacks: make([]PackRelationship, 0),
		CircularGroups:  make([][]PackRelationship, 0),
	}

	// Detect circular dependencies first
	group.CircularGroups = da.detectCircularDependencies(relationships)

	// Track packs in circular dependencies
	inCircularGroup := make(map[string]bool)
	for _, cycle := range group.CircularGroups {
		for _, rel := range cycle {
			inCircularGroup[rel.Pack.PackID] = true
		}
	}

	// Track processed packs
	processed := make(map[string]bool)

	for packID, rel := range relationships {
		if processed[packID] {
			continue // Already processed
		}

		// Skip packs that are in circular groups (they're already categorized)
		if inCircularGroup[packID] {
			processed[packID] = true
			continue
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
			// Pack has both dependencies and dependents but not circular
			// Classify as dependent pack (it's part of a dependency chain)
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
