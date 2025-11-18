# Critical Fixes Required - Action Plan

This document provides **immediately actionable fixes** for the critical issues identified in the comprehensive analysis.

---

## Fix #1: Silent Error Handling in Dependency Checking ⚠️ CRITICAL

**File:** `internal/addon/uninstaller.go`
**Lines:** 203-207

### Current Code (BROKEN):
```go
// Try to load the pack's manifest to check dependencies
manifest, err := u.loadPackManifest(pack.PackID, pack.Type)
if err != nil {
    // If we can't load the manifest, we can't check dependencies
    continue  // ⚠️ SILENTLY CONTINUES - MISSES DEPENDENCIES!
}
```

### Fixed Code:
```go
// Try to load the pack's manifest to check dependencies
manifest, err := u.loadPackManifest(pack.PackID, pack.Type)
if err != nil {
    // If we can't load the manifest, warn but continue
    // (manifest may not exist if pack is broken)
    if u.verbose {
        fmt.Printf("Warning: Could not load manifest for pack %s (%s): %v\n",
            pack.Name, pack.PackID, err)
        fmt.Printf("  Dependency check for this pack will be incomplete\n")
    }
    continue
}
```

**Also fix in:** `internal/addon/dependencies.go:50-59` (same issue)

---

## Fix #2: Circular Dependency Detection 🔄 HIGH PRIORITY

**File:** `internal/addon/dependencies.go`
**Function:** `groupPacksByRelationships`

### Add this new method to DependencyAnalyzer:

```go
// detectCircularDependencies finds circular dependency chains using DFS
func (da *DependencyAnalyzer) detectCircularDependencies(
    relationships map[string]*PackRelationship,
) [][]PackRelationship {
    visited := make(map[string]bool)
    recursionStack := make(map[string]bool)
    var cycles [][]PackRelationship

    var dfs func(packID string, path []string) []string
    dfs = func(packID string, path []string) []string {
        visited[packID] = true
        recursionStack[packID] = true
        path = append(path, packID)

        rel, exists := relationships[packID]
        if !exists {
            recursionStack[packID] = false
            return nil
        }

        for _, depID := range rel.Dependencies {
            if !visited[depID] {
                if cyclePath := dfs(depID, path); cyclePath != nil {
                    return cyclePath
                }
            } else if recursionStack[depID] {
                // Found cycle - extract it from path
                cycleStart := -1
                for i, id := range path {
                    if id == depID {
                        cycleStart = i
                        break
                    }
                }
                if cycleStart != -1 {
                    return path[cycleStart:]
                }
            }
        }

        recursionStack[packID] = false
        return nil
    }

    for packID := range relationships {
        if !visited[packID] {
            if cyclePath := dfs(packID, []string{}); cyclePath != nil {
                // Convert path to PackRelationships
                cyclePacks := make([]PackRelationship, 0, len(cyclePath))
                for _, id := range cyclePath {
                    if rel, ok := relationships[id]; ok {
                        cyclePacks = append(cyclePacks, *rel)
                    }
                }
                cycles = append(cycles, cyclePacks)
            }
        }
    }

    return cycles
}
```

### Update groupPacksByRelationships (line 185):

```go
} else {
    // Pack has both dependencies and dependents - could be circular
    // Check if it's part of a circular dependency
    // For now, classify as dependent pack
    group.DependentPacks = append(group.DependentPacks, *rel)
}

processed[packID] = true
```

**Add before returning the group:**

```go
// Detect circular dependencies
group.CircularGroups = da.detectCircularDependencies(relationships)

return group
```

---

## Fix #3: Transitive Dependency Validation 🔗 HIGH PRIORITY

**File:** `internal/addon/installer.go`
**After:** Line 140 (after conflict detection)

### Add this new method to Installer:

```go
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

    // Add UUIDs from packs being installed
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
        }
    }

    return missingDeps, nil
}
```

### Update InstallAddon function (after line 160):

```go
// Show conflict check results
conflictDetails := []string{}
if len(conflicts) == 0 {
    conflictDetails = append(conflictDetails, "No UUID conflicts detected")
} else {
    for _, conflict := range conflicts {
        conflictDetails = append(conflictDetails, fmt.Sprintf("Conflict found: %s", conflict))
    }
}

// ADD DEPENDENCY VALIDATION HERE:
missingDeps, err := i.validateDependencies(extractedAddon)
if err != nil {
    result.Errors = append(result.Errors, fmt.Sprintf("Dependency validation failed: %v", err))
    return result, err
}

if len(missingDeps) > 0 {
    for _, dep := range missingDeps {
        result.Warnings = append(result.Warnings, dep)
        conflictDetails = append(conflictDetails, fmt.Sprintf("⚠️  Missing dependency: %s", dep))
    }
    if !options.ForceUpdate {
        return result, fmt.Errorf("missing dependencies detected, install required packs first or use --force to proceed anyway")
    }
}

conflictDetails = append(conflictDetails, fmt.Sprintf("Checked against %d existing pack(s)", len(conflicts)))
if err := showStepResult("Conflict detection", conflictDetails, "Backup creation", ...
```

---

## Fix #4: Atomic Config File Writes 💾 HIGH PRIORITY

**File:** `internal/minecraft/config.go`
**Function:** `SaveWorldConfig`
**Lines:** 132-144

### Replace entire function:

```go
// SaveWorldConfig saves a world config file using atomic write
func SaveWorldConfig(filePath string, config WorldConfig) error {
    data, err := json.MarshalIndent(config, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal config: %w", err)
    }

    // Create directory if it doesn't exist
    dir := filepath.Dir(filePath)
    if err := os.MkdirAll(dir, 0750); err != nil {
        return fmt.Errorf("failed to create config directory: %w", err)
    }

    // Write to temporary file first
    tmpFile := filePath + ".tmp"
    if err := os.WriteFile(tmpFile, data, 0600); err != nil {
        return fmt.Errorf("failed to write temp config file: %w", err)
    }

    // Atomic rename (on same filesystem, this is atomic)
    if err := os.Rename(tmpFile, filePath); err != nil {
        os.Remove(tmpFile) // Clean up temp file on error
        return fmt.Errorf("failed to save config file: %w", err)
    }

    return nil
}
```

---

## Fix #5: Symlink Protection 🔒 SECURITY

**File:** `pkg/filesystem/archive.go`
**Function:** `extractFile`
**After:** Line 45

### Add symlink check:

```go
// Create directory for file if needed
if file.FileInfo().IsDir() {
    return os.MkdirAll(destPath, file.FileInfo().Mode())
}

// ADD THIS CHECK:
// Prevent symlink attacks
if file.Mode()&os.ModeSymlink != 0 {
    return fmt.Errorf("symlinks are not allowed in archives (security risk): %s", file.Name)
}

// Create parent directories
if err := os.MkdirAll(filepath.Dir(destPath), 0750); err != nil {
    return err
}
```

---

## Fix #6: UUID Validation in Dependencies 🆔 MEDIUM PRIORITY

**File:** `internal/addon/dependencies.go`
**Function:** `buildPackRelationship`
**Lines:** 87-97

### Update the dependency extraction:

```go
// Extract dependencies
for _, dep := range manifest.Dependencies {
    if dep.UUID != "" {
        // Validate and normalize UUID
        normalizedUUID, err := validation.NormalizeUUID(dep.UUID)
        if err != nil {
            // Log warning but don't fail - manifest is already installed
            if da.verbose {
                fmt.Printf("Warning: Invalid dependency UUID '%s' in pack %s: %v\n",
                    dep.UUID, pack.PackID, err)
            }
            continue
        }
        rel.Dependencies = append(rel.Dependencies, normalizedUUID)
    }
    if dep.ModuleName != "" {
        // Module dependency (Script API)
        rel.Modules = append(rel.Modules, dep.ModuleName)
    }
}
```

**Note:** This requires adding a `verbose` field to DependencyAnalyzer or passing it as a parameter.

---

## Fix #7: Improve Error Messages 📝 MEDIUM PRIORITY

**File:** `internal/minecraft/server.go`
**Line:** 116

### Replace:
```go
return fmt.Errorf("pack with ID %s not found", packID)
```

### With:
```go
return fmt.Errorf("pack with UUID %s is not installed on this server. Use 'blockbench list <server-path>' to see all installed packs", packID)
```

**File:** `internal/minecraft/config.go`
**Line:** 91

### Replace:
```go
return "", fmt.Errorf("level-name property not found in %s", propertiesPath)
```

### With:
```go
return "", fmt.Errorf("level-name property not found in %s. Ensure your server.properties file contains a valid 'level-name=' entry (e.g., 'level-name=Bedrock level')", propertiesPath)
```

---

## Fix #8: Display Name Safety 🛡️ LOW PRIORITY

**File:** `internal/minecraft/manifest.go`
**Function:** `GetDisplayName`
**Line:** 114

### Replace:
```go
return fmt.Sprintf("Pack-%s", m.Header.UUID[:8])
```

### With:
```go
// Ensure UUID is long enough before slicing
if len(m.Header.UUID) >= 8 {
    return fmt.Sprintf("Pack-%s", m.Header.UUID[:8])
}
return fmt.Sprintf("Pack-%s", m.Header.UUID)
```

---

## Testing These Fixes

### Test #1: Silent Error Handling
```bash
# Create a pack with broken manifest
# Verify warnings are shown when checking dependencies
```

### Test #2: Circular Dependencies
```bash
# Create pack A depending on B, B depending on A
# Verify circular dependency is detected
blockbench list /server --tree --verbose
```

### Test #3: Transitive Dependencies
```bash
# Try to install pack requiring non-existent dependency
# Verify it fails with clear error message
blockbench install /server pack-with-missing-deps.mcaddon --dry-run
```

### Test #4: Atomic Writes
```bash
# Simulate disk full during config write
# Verify original config is not corrupted
```

### Test #5: Symlink Attack
```bash
# Create malicious ZIP with symlink
# Verify extraction fails with security error
```

---

## Implementation Order

1. **Fix #5** (Symlink Protection) - 5 minutes, security critical
2. **Fix #4** (Atomic Writes) - 10 minutes, data safety
3. **Fix #1** (Error Handling) - 15 minutes, critical bug
4. **Fix #7** (Error Messages) - 10 minutes, user experience
5. **Fix #8** (Display Name Safety) - 5 minutes, defensive programming
6. **Fix #6** (UUID Validation) - 20 minutes, requires small refactor
7. **Fix #3** (Dependency Validation) - 30 minutes, new feature
8. **Fix #2** (Circular Detection) - 45 minutes, complex algorithm

**Total Time: ~2.5 hours for all fixes**

---

## Validation Checklist

After implementing fixes:

- [ ] All existing tests still pass
- [ ] New tests added for each fix
- [ ] Linter passes (`make lint`)
- [ ] Manual testing with real .mcaddon files
- [ ] Error messages are user-friendly
- [ ] Documentation updated (CLAUDE.md if needed)
- [ ] CHANGELOG.md updated with fixes

---

## Next Steps After Fixes

1. Add integration tests for full workflows
2. Add CLI command tests
3. Implement progress indication for long operations
4. Add configuration file support
5. Consider adding structured logging

---

**Priority:** Implement Fixes #1-#5 immediately (high security/correctness impact)
**Timeline:** All critical fixes can be completed in one afternoon
