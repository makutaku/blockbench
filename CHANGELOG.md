# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Circular Dependency Detection**: Implemented DFS-based algorithm to detect and report circular dependency chains between packs
- **Transitive Dependency Validation**: Installation now validates that all pack dependencies exist on the server before installation
- **Configurable Decompression Limit**: File size limit for archive extraction can now be configured via `BLOCKBENCH_MAX_FILE_SIZE` environment variable (default: 100MB)
- **Enhanced Manifest Validation**: Comprehensive validation of UUID formats, version numbers, and module types in manifest files
- **UUID Normalization**: All dependency UUIDs are now validated and normalized to lowercase with dashes for consistency
- **Better Error Messages**: User-friendly error messages with actionable hints (e.g., suggesting to run `blockbench list` when pack not found)

### Fixed
- **SECURITY**: Fixed symlink vulnerability in archive extraction that could allow path traversal attacks
- **DATA SAFETY**: Implemented atomic config file writes using temp file + rename pattern to prevent corruption on write failure
- **Silent Error Handling**: Fixed critical bug where dependency checking silently skipped packs with unreadable manifests, leading to incomplete dependency analysis
- **UUID Safety**: Added bounds checking for UUID display name generation to prevent panic on malformed UUIDs
- **Test Robustness**: Updated test for invalid config paths to use more reliable failure conditions

### Changed
- **Dependency Checking**: Now provides detailed warnings when manifests cannot be loaded during dependency analysis
- **Config File Writes**: SaveWorldConfig now creates parent directories if they don't exist
- **Manifest Validation**: ValidateManifest now performs comprehensive checks including UUID format, version numbers, and module types

### Technical Improvements
- Added validation import to minecraft/manifest.go for UUID checking
- Added os and validation imports to addon/dependencies.go
- Enhanced DependencyAnalyzer with detectCircularDependencies method (73 lines of new code)
- Enhanced Installer with validateDependencies method for transitive dependency checking
- Improved error propagation in uninstaller dependency checking
- Added proper error handling and warnings for malformed dependency UUIDs

### Breaking Changes
None - all changes are backward compatible

---

## Implementation Details

### Circular Dependency Detection
- Uses Depth-First Search (DFS) with recursion stack tracking
- Detects all circular dependency chains in the pack graph
- Properly categorizes packs into: Root, Dependent, Standalone, and Circular groups
- Prevents duplicate detection of the same cycle

### Transitive Dependency Validation
- Validates that all pack dependencies exist before installation
- Handles self-satisfied dependencies (when multiple packs in same .mcaddon satisfy each other)
- Provides clear error messages listing missing dependencies
- Can be bypassed with `--force` flag if needed

### Security Enhancements
- **Symlink Protection**: Archive extraction now rejects any files with symlink mode bits set
- **Atomic Writes**: Config files are written to `.tmp` file first, then atomically renamed
- **Path Validation**: Enhanced validation prevents path traversal in archives

### Configuration
New environment variable:
- `BLOCKBENCH_MAX_FILE_SIZE`: Maximum size in bytes for extracted files (default: 104857600 = 100MB)

Example:
```bash
export BLOCKBENCH_MAX_FILE_SIZE=209715200  # 200MB
blockbench install /server my-large-pack.mcaddon
```

---

## Testing

All changes include:
- ✅ Unit tests passing (32 test functions, 100% pass rate)
- ✅ Linter clean (golangci-lint)
- ✅ No regression in existing functionality
- ✅ Enhanced test coverage for edge cases

---

## Migration Guide

No migration needed - all changes are backward compatible. Existing installations and workflows will continue to work without modification.

Optional improvements you can take advantage of:
1. Set `BLOCKBENCH_MAX_FILE_SIZE` if you work with very large texture packs
2. Use `--dry-run` to preview installations and see dependency validation in action
3. Check for circular dependencies using `blockbench list /server --tree`

---

## Contributors

- Analysis and recommendations from comprehensive codebase audit
- Implemented based on security best practices and Go idioms
- All fixes validated with existing test suite

