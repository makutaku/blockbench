# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Building
- `make build` - Production build with version injection in `./bin/blockbench`
- `make build-dev` - Development build without version injection as `./blockbench`
- `make build-all` - Cross-platform builds for Linux, macOS, and Windows
- `make install` - Install to GOPATH/bin with version injection

### Testing and Quality
- `make test` - Run all tests
- `make test-coverage` - Run tests with coverage report (generates `coverage.html`)
- `make lint` - Run golangci-lint (auto-installs if missing)
- `make fmt` - Format code with go fmt
- `make vet` - Run go vet
- `make check` - Run all quality checks (fmt, vet, tidy, test, lint)

### Maintenance
- `make tidy` - Tidy module dependencies
- `make clean` - Clean build artifacts
- `./scripts/release.sh v1.0.0` - Create a release with version tagging and cross-platform builds

## Architecture Overview

### Core Components

**Command Structure**: Uses `spf13/cobra` for CLI with main commands:
- `install` - Install `.mcaddon`/`.mcpack` files to server
- `uninstall` - Remove addons with dependency checking  
- `list` - Show installed addons
- `version` - Display version information

**Key Packages**:
- `internal/addon/` - Core addon management with advanced features:
  - `installer.go` - Installation orchestration with comprehensive dry-run simulation
  - `uninstaller.go` - Safe uninstallation with dependency checking and interactive mode
  - `extractor.go` - Advanced archive extraction with nested .mcpack support
  - `dependencies.go` - **NEW** - Dependency analysis and relationship mapping
  - `simulator.go` - **NEW** - Comprehensive dry-run simulation engine
  - `backup.go` - Backup and restore management
  - `rollback.go` - Automatic rollback on failures
- `internal/minecraft/` - Minecraft server interaction and manifest parsing:
  - `server.go` - Enhanced server management with dependency-aware operations
  - `manifest.go` - Advanced manifest parsing with mixed dependency format support
  - `config.go` - World configuration management with proper error handling
- `internal/cli/` - Command implementations:
  - `install.go` - Install command with interactive mode and enhanced help
  - `uninstall.go` - Uninstall command with UUID support and interactive mode
  - `list.go` - **ENHANCED** - Advanced listing with dependency grouping and tree visualization
  - `version.go` - Version command with multiple output formats
- `internal/version/` - Build-time version injection
- `pkg/filesystem/` - Archive handling with comprehensive validation
- `pkg/validation/` - UUID and data validation utilities

### Enhanced Installation Flow

The installation process now includes comprehensive dry-run simulation and interactive mode:

1. **Pre-validation** - Extensive file and server structure validation
   - Archive integrity checking
   - Server directory structure verification
   - World configuration file validation

2. **Archive extraction** - Advanced extraction with nested support
   - Direct `.mcpack` file support
   - Nested `.mcpack` extraction from `.mcaddon` files
   - Content analysis and pack discovery

3. **Content validation** - Deep manifest and pack structure validation
   - Modern manifest.json parsing with mixed dependency formats
   - Pack type detection (behavior vs resource)
   - Script API module dependency tracking

4. **Conflict detection** - Comprehensive UUID and dependency conflict analysis
   - Existing pack UUID conflicts
   - Dependency relationship validation
   - Impact analysis for dependency chains

5. **Dry-run simulation** (when `--dry-run` flag used)
   - Complete operation simulation without file changes
   - Detailed preview of all operations
   - Conflict and dependency impact reporting

6. **Interactive confirmation** (when `--interactive` flag used)
   - Step-by-step confirmation with detailed previews
   - User can abort at any stage
   - Full visibility into each operation

7. **Backup creation** - Automatic backup with rollback preparation
   - Timestamp-based backup IDs
   - Complete server state preservation
   - Metadata tracking for rollback operations

8. **Pack installation** - Atomic file operations with progress tracking
   - Directory creation with proper naming conventions
   - World configuration updates (world_*_packs.json)
   - File permission preservation

9. **Post-validation** - Installation verification and integrity checking
   - Pack registration verification
   - Configuration file integrity
   - Dependency relationship validation

10. **Automatic rollback** - Intelligent failure recovery
    - Rollback on any step failure
    - Complete state restoration
    - Error reporting and user notification

### Server Integration

Works with Minecraft Bedrock server directory structure:
- `development_behavior_packs/` - Behavior packs
- `development_resource_packs/` - Resource packs  
- `worlds/[world]/world_behavior_packs.json` - Behavior pack config
- `worlds/[world]/world_resource_packs.json` - Resource pack config

Where `[world]` is automatically detected from the `level-name` property in `server.properties`.

### Advanced Safety Features

- **Comprehensive dry-run simulation** - Full operation preview with detailed analysis
- **Automatic backups** - Complete state preservation before any operation
- **Intelligent rollback** - Automatic restoration on any failure
- **Interactive mode** - Step-by-step confirmation with detailed previews
- **Dependency analysis** - Advanced relationship mapping and impact assessment
- **UUID conflict detection** - Prevent pack conflicts before installation
- **Idempotent operations** - Safe to re-run commands multiple times
- **Atomic operations** - Complete success or complete rollback
- **Error recovery** - Graceful failure handling with user-friendly messages

### Dependency Analysis & List Features

#### **Command Flags:**
- `blockbench list /server --grouped` - Group by dependency relationships
- `blockbench list /server --tree` - ASCII tree visualization with emojis
- `blockbench list /server --standalone` - Show only standalone packs
- `blockbench list /server --roots` - Show only root packs (others depend on these)
- `blockbench list /server --json` - Enhanced JSON with dependency information

#### **Pack Categories:**
- **Root Packs** - Foundation packs that other packs depend on
- **Dependent Packs** - Packs that require other packs to function
- **Standalone Packs** - Self-contained packs with no relationships
- **Module Dependencies** - Script API modules (@minecraft/server, etc.)

#### **Visualization Features:**
- Emoji indicators: 📦 (behavior packs), 🎨 (resource packs)
- Dependency relationship mapping
- Module usage tracking
- Impact analysis for uninstallation

### Interactive Mode & Dry-Run Enhancements

#### **Interactive Mode Features:**
- Step-by-step operation confirmation
- Detailed preview of each operation
- User can abort at any stage
- Available for both install and uninstall commands
- Enhanced with dry-run simulation data

#### **Advanced Dry-Run Capabilities:**
- **Installation simulation:**
  - Complete pack analysis with extraction
  - Conflict detection and resolution preview
  - Configuration change preview
  - Backup operation simulation
  - Dependency impact analysis

- **Uninstallation simulation:**
  - Dependency impact assessment
  - File removal preview
  - Configuration cleanup preview
  - Backup operation details
  - Breaking change warnings

## Development Notes

- **Version Management:** Build-time version injection via linker flags
- **Go Version:** Requires Go 1.23.4+
- **Dependencies:** Only uses `github.com/spf13/cobra` for CLI framework
- **Architecture:** Clean separation of concerns with comprehensive error handling
- **Cross-platform:** Full support for Linux, macOS, and Windows
- **Safety-first design:** All operations designed for maximum safety and recoverability
- **Modern Minecraft support:** Handles latest addon formats and Script API dependencies
- **No external services:** Completely self-contained tool
- **Comprehensive logging:** Detailed verbose output for troubleshooting
- **JSON API:** Structured data output for automation and integration

### Key Data Structures
- `PackRelationship` - Pack with complete dependency metadata
- `DependencyGroup` - Categorized pack relationships  
- `SimulatedInstallOperation` - Detailed installation preview
- `SimulatedUninstallOperation` - Uninstallation impact analysis
- `ExtractedAddon` - Multi-pack addon representation with dry-run support