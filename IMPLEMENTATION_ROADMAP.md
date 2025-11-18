# Implementation Roadmap - Remaining Recommendations

This document outlines the remaining recommendations from the comprehensive analysis that are planned for future implementation.

## Status Summary

### ✅ Completed (Current PR)

| Priority | ID | Item | Status | Commit |
|----------|-----|------|--------|--------|
| HIGH | H1 | Extract duplicate manifest loading logic | ✅ Complete | fac1bb6 |
| MEDIUM | M1 | Define constants for magic numbers | ✅ Complete | fac1bb6 |
| MEDIUM | M3 | Enhance inline documentation | ✅ Complete | c533c04 |
| LOW | L1 | Add troubleshooting guide to README | ✅ Complete | c533c04 |
| LOW | L2 | Add performance benchmarks | ✅ Complete | c533c04 |

**Total Completed:** 5 items
**Lines Changed:** ~650 lines (additions + modifications)
**Files Modified:** 12 files
**New Files:** 2 files (PERFORMANCE.md, IMPLEMENTATION_ROADMAP.md)

### 🔄 Planned for Future PRs

| Priority | ID | Item | Estimated Effort | Complexity |
|----------|-----|------|-----------------|------------|
| HIGH | H2 | Add integration test suite | 6-8 hours | High |
| HIGH | H3 | Standardize warning collection | 2 hours | Medium |
| MEDIUM | M2 | Add CLI command tests | 3-4 hours | Medium |
| LOW | L3 | Add example server setup guide | 2 hours | Low |

**Total Remaining:** 4 items
**Estimated Total Effort:** 13-16 hours

---

## H2: Add Integration Test Suite (HIGH Priority)

### Objective

Create comprehensive integration tests for end-to-end workflow validation, addressing the 0% coverage gap in the `internal/addon` package.

### Scope

**Test Coverage Goals:**
- Increase `internal/addon` from 0% to ~60-70%
- Add integration tests for critical workflows
- Validate rollback mechanisms
- Test dependency detection in practice

### Implementation Plan

#### Phase 1: Test Infrastructure (2 hours)

**Create test fixtures:**
```
internal/addon/testdata/
├── test-server/                    # Mock server structure
│   ├── server.properties
│   ├── worlds/
│   │   └── test-world/
│   │       ├── world_behavior_packs.json
│   │       └── world_resource_packs.json
│   ├── development_behavior_packs/
│   └── development_resource_packs/
├── valid-addon.mcaddon             # Simple test addon
├── circular-deps/                  # Circular dependency test cases
│   ├── pack-a.mcpack
│   ├── pack-b.mcpack
│   └── pack-c.mcpack
└── missing-deps.mcaddon            # Missing dependency test case
```

**Helper functions:**
```go
// internal/addon/testing_helpers.go
func createTestServer(t *testing.T) *minecraft.Server
func createTestAddon(t *testing.T, name string, deps []string) string
func assertPackInstalled(t *testing.T, server *minecraft.Server, uuid string)
func assertPackNotInstalled(t *testing.T, server *minecraft.Server, uuid string)
```

#### Phase 2: Installation Tests (2 hours)

**File:** `internal/addon/installer_integration_test.go`

```go
func TestFullInstallationWorkflow(t *testing.T) {
    // 1. Create test server structure
    // 2. Install addon
    // 3. Verify files copied correctly
    // 4. Verify world config updated
    // 5. Verify backup created
    // 6. List packs and verify appears
}

func TestInstallWithCircularDependencies(t *testing.T) {
    // 1. Create addon with circular deps (A→B→C→A)
    // 2. Attempt installation
    // 3. Verify circular dependency detected
    // 4. Verify proper error message
    // 5. Verify no partial installation
}

func TestInstallWithMissingDependencies(t *testing.T) {
    // 1. Create addon requiring pack X (not installed)
    // 2. Attempt installation without --force
    // 3. Verify failure with helpful message
    // 4. Retry with --force flag
    // 5. Verify installation succeeds with warning
}

func TestRollbackOnInstallFailure(t *testing.T) {
    // 1. Mock filesystem error mid-install
    // 2. Trigger installation
    // 3. Verify rollback executed
    // 4. Verify server state restored to pre-install
    // 5. Verify no partial installation remains
}

func TestDryRunSimulation(t *testing.T) {
    // 1. Install addon with --dry-run
    // 2. Verify no files actually copied
    // 3. Verify simulation report is accurate
    // 4. Verify conflicts detected without changes
}
```

#### Phase 3: Uninstallation Tests (1.5 hours)

**File:** `internal/addon/uninstaller_integration_test.go`

```go
func TestUninstallWithDependents(t *testing.T) {
    // 1. Install pack A
    // 2. Install pack B that depends on A
    // 3. Attempt to uninstall A without --force
    // 4. Verify prevention with clear error
    // 5. Verify B still installed
}

func TestUninstallProtection(t *testing.T) {
    // 1. Install packs with dependency chain A→B→C
    // 2. Attempt uninstall of B (middle of chain)
    // 3. Verify warning about A depending on B
    // 4. Verify uninstall prevented without --force
}

func TestRollbackOnUninstallFailure(t *testing.T) {
    // 1. Install pack
    // 2. Mock error during uninstall
    // 3. Verify rollback restores pack
    // 4. Verify config not corrupted
}
```

#### Phase 4: Dependency Analysis Tests (1.5 hours)

**File:** `internal/addon/dependencies_integration_test.go`

```go
func TestCircularDependencyDetection(t *testing.T) {
    // 1. Create test server with packs in circular dependency
    // 2. Run dependency analysis
    // 3. Verify all circular chains found
    // 4. Verify packs properly categorized
    // 5. Test with multiple independent circles
}

func TestTransitiveDependencyValidation(t *testing.T) {
    // 1. Create dependency graph: Root→A→B, Root→C
    // 2. Validate transitive deps all exist
    // 3. Test with missing transitive dependency
    // 4. Verify error includes full chain
}

func TestDependencyImpactAnalysis(t *testing.T) {
    // 1. Create dependency graph: Root→A→B, Root→C
    // 2. Analyze impact of removing Root
    // 3. Verify all dependents identified (A, B, C)
    // 4. Verify transitive deps included
}
```

#### Phase 5: Edge Cases (1 hour)

```go
func TestLargePackHandling(t *testing.T)          // 200MB+ pack
func TestCorruptedArchiveRecovery(t *testing.T)   // Malformed ZIP
func TestPartialExtractionFailure(t *testing.T)   // Disk full simulation
func TestBrokenManifestHandling(t *testing.T)     // Invalid JSON
func TestConcurrentOperationPrevention(t *testing.T) // Locking
```

### Success Criteria

- [ ] `internal/addon` coverage increases to 60-70%
- [ ] All critical workflows have integration tests
- [ ] Tests run in < 5 seconds total
- [ ] Tests are deterministic (no flaky tests)
- [ ] Test fixtures are minimal and maintainable
- [ ] Clear test names and documentation

### Future Enhancements

- Add fuzzing tests for manifest parsing
- Add property-based tests for dependency graphs
- Add stress tests with 1000+ packs

---

## H3: Standardize Warning Collection (HIGH Priority)

### Objective

Ensure consistent warning handling across the codebase for programmatic access and better user experience.

### Current Inconsistencies

**Issue 1:** Mixed warning destinations
- `dependencies.go:50-64` → warnings to stderr
- `uninstaller.go:213` → warnings to result.Warnings

**Issue 2:** No unified warning interface

### Implementation Plan

#### Step 1: Define Warning Interface (30 min)

**File:** `internal/addon/warnings.go` (new file)

```go
package addon

import (
    "fmt"
    "os"
)

// Warning represents a non-fatal issue during an operation
type Warning struct {
    Code    string // Machine-readable code (e.g., "MANIFEST_LOAD_FAILED")
    Message string // Human-readable message
    Context string // Additional context (pack ID, file path, etc.)
}

// WarningCollector collects warnings during operations
type WarningCollector struct {
    warnings []Warning
    verbose  bool
}

// NewWarningCollector creates a new warning collector
func NewWarningCollector(verbose bool) *WarningCollector {
    return &WarningCollector{
        warnings: make([]Warning, 0),
        verbose:  verbose,
    }
}

// Add adds a warning and optionally prints to stderr
func (wc *WarningCollector) Add(code, message, context string) {
    warning := Warning{
        Code:    code,
        Message: message,
        Context: context,
    }
    wc.warnings = append(wc.warnings, warning)

    if wc.verbose {
        fmt.Fprintf(os.Stderr, "Warning [%s]: %s\n", code, message)
        if context != "" {
            fmt.Fprintf(os.Stderr, "  Context: %s\n", context)
        }
    }
}

// Warnings returns all collected warnings
func (wc *WarningCollector) Warnings() []Warning {
    return wc.warnings
}

// ToStrings converts warnings to string slice for result structures
func (wc *WarningCollector) ToStrings() []string {
    strs := make([]string, len(wc.warnings))
    for i, w := range wc.warnings {
        strs[i] = fmt.Sprintf("[%s] %s", w.Code, w.Message)
    }
    return strs
}
```

#### Step 2: Update Result Structures (15 min)

```go
// Add to InstallResult, UninstallResult, etc.
type InstallResult struct {
    // ... existing fields ...
    Warnings []Warning // Change from []string to []Warning
}

// Add helper method
func (r *InstallResult) WarningMessages() []string {
    msgs := make([]string, len(r.Warnings))
    for i, w := range r.Warnings {
        msgs[i] = fmt.Sprintf("[%s] %s", w.Code, w.Message)
    }
    return msgs
}
```

#### Step 3: Update Dependencies Package (45 min)

**File:** `internal/addon/dependencies.go`

```go
// Before (stderr only):
fmt.Fprintf(os.Stderr, "Warning: Could not analyze pack %s (%s): %v\n",
    pack.Name, pack.PackID, err)

// After (collector):
warnings.Add(
    "MANIFEST_LOAD_FAILED",
    fmt.Sprintf("Could not analyze pack %s: %v", pack.Name, err),
    fmt.Sprintf("Pack ID: %s, Type: %s", pack.PackID, pack.Type),
)
```

#### Step 4: Update Uninstaller Package (30 min)

**File:** `internal/addon/uninstaller.go`

```go
// Before (result.Warnings append):
result.Warnings = append(result.Warnings, "Incomplete dependency check: "+warning)

// After (collector):
warnings.Add(
    "INCOMPLETE_DEPENDENCY_CHECK",
    warning,
    fmt.Sprintf("Pack: %s (%s)", pack.Name, pack.PackID),
)
```

### Success Criteria

- [ ] All warnings collected in structured format
- [ ] Warnings printed to stderr when --verbose
- [ ] Warnings included in result structures
- [ ] JSON output includes structured warnings
- [ ] Consistent warning codes across codebase
- [ ] Backward compatible with existing code

### Warning Code Catalog

| Code | Description | Location |
|------|-------------|----------|
| `MANIFEST_LOAD_FAILED` | Cannot read/parse manifest | dependencies.go, uninstaller.go |
| `INCOMPLETE_DEPENDENCY_CHECK` | Dependency check partial | uninstaller.go |
| `MISSING_DEPENDENCY` | Required pack not found | installer.go |
| `CIRCULAR_DEPENDENCY` | Circular dep detected | dependencies.go |
| `UUID_CONFLICT` | Pack UUID already exists | installer.go |

---

## M2: Add CLI Command Tests (MEDIUM Priority)

### Objective

Test command parsing, flag handling, and output formatting for all CLI commands.

### Scope

- `internal/cli/install.go`
- `internal/cli/uninstall.go`
- `internal/cli/list.go`
- `internal/cli/version.go`

### Implementation Plan

#### Test Infrastructure

```go
// internal/cli/testing_helpers.go
func executeCommand(t *testing.T, args ...string) (output string, err error) {
    // Execute command and capture output
}

func mockServer(t *testing.T) *minecraft.Server {
    // Create mock server for CLI tests
}
```

#### Test Cases (3-4 hours)

**File:** `internal/cli/install_test.go`
```go
func TestInstallCommandParsing(t *testing.T)
func TestInstallFlagHandling(t *testing.T)
func TestInstallOutputFormatting(t *testing.T)
func TestInstallErrorMessages(t *testing.T)
```

**File:** `internal/cli/uninstall_test.go`
```go
func TestUninstallCommandParsing(t *testing.T)
func TestUninstallUUIDValidation(t *testing.T)
func TestUninstallOutputFormatting(t *testing.T)
```

**File:** `internal/cli/list_test.go`
```go
func TestListCommandFormats(t *testing.T)       // table, JSON, tree
func TestListGroupingFlag(t *testing.T)
func TestListTreeVisualization(t *testing.T)
func TestListFilterFlags(t *testing.T)          // --standalone, --roots
```

**File:** `internal/cli/version_test.go`
```go
func TestVersionOutput(t *testing.T)
func TestVersionJSONFormat(t *testing.T)
```

### Success Criteria

- [ ] All CLI commands have test coverage
- [ ] Flag parsing validated
- [ ] Output formatting tested
- [ ] Error messages validated
- [ ] Help text checked

---

## L3: Add Example Server Setup Guide (LOW Priority)

### Objective

Create step-by-step guide for first-time users to set up Blockbench with their Minecraft Bedrock server.

### Implementation Plan (2 hours)

**File:** `docs/SERVER_SETUP_GUIDE.md`

### Content Outline

1. **Prerequisites**
   - Minecraft Bedrock Dedicated Server installed
   - Basic terminal/command line knowledge
   - Blockbench installed

2. **Server Directory Structure**
   - Explanation of standard Bedrock layout
   - Where to find server.properties
   - World directory structure

3. **Step-by-Step Setup**
   - Verify server.properties has level-name
   - Check world directory exists
   - Create initial world config files if missing
   - Test with `blockbench list /server`

4. **First Addon Installation**
   - Download a simple test addon
   - Install with dry-run first
   - Install for real
   - Verify with list command
   - Check in-game

5. **Common Pitfalls**
   - Server must be stopped during operations
   - World name must match level-name property
   - Config files must be valid JSON

6. **Advanced Topics**
   - Managing dependencies
   - Using backups
   - Troubleshooting with --verbose

### Diagrams

```
Server Directory Structure:
bedrock-server/
├── server.properties          ← Contains level-name
├── development_behavior_packs/ ← Behavior packs go here
├── development_resource_packs/ ← Resource packs go here
└── worlds/
    └── Bedrock level/         ← World name from level-name
        ├── world_behavior_packs.json
        └── world_resource_packs.json
```

---

## Priority Recommendations for Next PR

### Recommended Order

1. **H3: Standardize Warning Collection** (2 hours)
   - Small, focused change
   - Improves API consistency
   - Foundation for better error reporting

2. **L3: Server Setup Guide** (2 hours)
   - Low risk
   - High user value
   - No code changes

3. **M2: CLI Command Tests** (3-4 hours)
   - Moderate complexity
   - Improves reliability
   - Good coverage increase

4. **H2: Integration Test Suite** (6-8 hours)
   - Largest effort
   - Separate PR for easier review
   - Significant coverage improvement

### Alternative: Smaller Increments

If breaking into smaller PRs:

**PR #2: Documentation + Warning Standardization**
- H3: Standardize Warning Collection
- L3: Server Setup Guide
- Estimated: 4 hours
- Low risk, high value

**PR #3: CLI Tests**
- M2: Add CLI Command Tests
- Estimated: 3-4 hours
- Moderate complexity

**PR #4: Integration Tests**
- H2: Add Integration Test Suite
- Estimated: 6-8 hours
- Highest complexity

---

## Summary

### Current PR Achievements

✅ **5 recommendations implemented:**
- H1: Duplicate code elimination
- M1: Magic number constants
- M3: Enhanced documentation
- L1: Comprehensive troubleshooting
- L2: Performance analysis

**Impact:**
- Reduced maintenance burden
- Improved code readability
- Better user experience
- Clear performance expectations

### Remaining Work

🔄 **4 recommendations planned:**
- H2: Integration test suite (high value, high effort)
- H3: Warning standardization (high value, low effort)
- M2: CLI command tests (medium value, medium effort)
- L3: Setup guide (medium value, low effort)

**Estimated Total:** 13-16 hours

### Risk Assessment

All remaining items are **low risk**:
- No breaking changes
- Additive only (tests, docs, refactoring)
- Can be implemented incrementally
- Easy to review in smaller PRs

---

**Document Version:** 1.0
**Last Updated:** 2025-11-18
**Maintained By:** Claude Code Analysis
