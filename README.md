# Blockbench

A comprehensive command-line tool for managing Minecraft Bedrock Edition addons on servers with advanced dependency analysis and safety features.

## ✨ Key Features

### 📦 Advanced File Support
- **`.mcaddon` files** - Multi-pack addons containing behavior and resource packs
- **`.mcpack` files** - Individual behavior or resource packs (can be installed directly)
- **Nested archive handling** - Automatically processes .mcpack files within .mcaddon archives
- **Archive validation** - Comprehensive ZIP file integrity checking

### 🔍 Dependency Analysis & Visualization
- **Smart pack grouping** - Automatically categorizes packs by relationships
- **Dependency tree visualization** - ASCII tree display with emoji indicators 📦🎨
- **Root pack identification** - Find packs that other packs depend on
- **Standalone pack filtering** - Identify self-contained packs
- **Script API module tracking** - Monitor @minecraft/server and other module usage

### 🛡️ Comprehensive Safety Features
- **Detailed dry-run simulation** - See exactly what would happen before committing
- **Automatic backups** - Complete backup before any operation
- **Intelligent rollback** - Auto-restore on installation failures
- **UUID conflict detection** - Prevent pack conflicts before installation
- **Dependency checking** - Safe uninstallation with impact analysis

### 🎯 Interactive & User-Friendly
- **Interactive mode** - Step-by-step confirmation with detailed previews
- **Multiple output formats** - Table, JSON, grouped, and tree views
- **Verbose logging** - Detailed operation information
- **Idempotent operations** - Safe to re-run commands multiple times

## 🚀 Usage Examples

### Basic Operations
```bash
# Install any addon file (automatically detects .mcaddon vs .mcpack)
blockbench install addon.mcaddon /path/to/server
blockbench install pack.mcpack /path/to/server

# Uninstall with dependency checking
blockbench uninstall addon-name /path/to/server

# List installed addons (basic table view)
blockbench list /path/to/server
```

### Advanced Listing & Visualization
```bash
# Dependency-aware grouped view
blockbench list /path/to/server --grouped

# Visual dependency tree with emojis
blockbench list /path/to/server --tree

# Show only standalone packs
blockbench list /path/to/server --standalone

# Show only root packs (others depend on these)
blockbench list /path/to/server --roots

# JSON output with dependency information
blockbench list /path/to/server --grouped --json
```

### Comprehensive Dry-Run Simulation
```bash
# Preview installation with detailed simulation
blockbench install addon.mcaddon /path/to/server --dry-run --verbose

# Interactive installation with step-by-step confirmation
blockbench install addon.mcaddon /path/to/server --interactive

# Simulate uninstallation to see dependency impact
blockbench uninstall addon-name /path/to/server --dry-run --interactive
```

### Advanced Installation Options
```bash
# Force installation despite conflicts
blockbench install addon.mcaddon /path/to/server --force

# Custom backup directory
blockbench install addon.mcaddon /path/to/server --backup-dir /custom/backup/path

# Uninstall by UUID instead of name
blockbench uninstall --uuid 12345678-1234-5678-9012-123456789abc /path/to/server
```

### Version Information
```bash
# Show detailed version information
blockbench version

# JSON format for scripting
blockbench version --json

# Short version only
blockbench version --short
```

## Installation

### From releases (recommended)
Download the latest binary from the [releases page](https://github.com/makutaku/blockbench/releases).

### Using Go
```bash
go install github.com/makutaku/blockbench/cmd/blockbench@latest
```

### Build from source
```bash
git clone https://github.com/makutaku/blockbench.git
cd blockbench
make build
# Binary will be in ./bin/blockbench
```

## 📋 Server Requirements

Blockbench requires a properly configured Minecraft Bedrock server with:

```
server/
├── server.properties              # Must contain 'level-name=your-world'
├── development_behavior_packs/    # Behavior packs directory
├── development_resource_packs/    # Resource packs directory  
└── worlds/
    └── your-world/
        ├── world_behavior_packs.json    # Behavior pack configuration
        └── world_resource_packs.json    # Resource pack configuration
```

**Important:** Blockbench automatically detects the world name from `server.properties` and will fail if this file is missing or improperly configured.

## 🎯 Command Reference

### Global Flags
- `--dry-run` - Preview operations without making changes (comprehensive simulation)
- `--verbose` - Detailed output with step-by-step information
- `--version` - Show version information

### Install Command
```bash
blockbench install [addon-file] [server-path] [options]
```
**Options:**
- `--force` - Install despite UUID conflicts
- `--backup-dir` - Custom backup location
- `--interactive` - Step-by-step confirmation mode

### Uninstall Command  
```bash
blockbench uninstall [addon-name] [server-path] [options]
```
**Options:**
- `--uuid` - Uninstall by UUID instead of name
- `--backup-dir` - Custom backup location
- `--interactive` - Confirmation before each step

### List Command
```bash  
blockbench list [server-path] [options]
```
**Display Modes:**
- `--grouped` - Categorize by dependency relationships
- `--tree` - Visual dependency tree with emojis
- `--standalone` - Only standalone packs (no dependencies)
- `--roots` - Only root packs (that others depend on)
- `--json` - JSON output format

### Version Command
```bash
blockbench version [options]
```
**Options:**
- `--json` - JSON format output
- `--short` - Version number only

## 🏗️ Development

### Quick Start
```bash
git clone https://github.com/makutaku/blockbench.git
cd blockbench

# Development build (no version injection)  
make build-dev

# Production build with version info
make build

# Run all quality checks
make check
```

### Build System
```bash
# Available targets
make build        # Production build with version injection
make build-dev     # Development build
make build-all     # Cross-platform builds (Linux, macOS, Windows)
make install       # Install to GOPATH/bin
make test          # Run tests  
make test-coverage # Generate coverage report
make lint          # Run golangci-lint (auto-installs if missing)
make fmt           # Format code
make vet           # Static analysis
make check         # All quality checks
make tidy          # Update dependencies
make clean         # Clean build artifacts
```

### Release Creation
```bash
./scripts/release.sh v1.0.0    # Create tagged release with binaries
```

## 🔧 Architecture

### Core Components
- **DependencyAnalyzer** - Advanced pack relationship analysis
- **DryRunSimulator** - Comprehensive operation simulation
- **BackupManager** - Automatic backup and restore functionality
- **Installation Pipeline** - Multi-stage validation and rollback
- **Manifest Parser** - Supports modern Minecraft addon formats

### Safety-First Design
- **Pre-flight validation** - Extensive checks before any operation  
- **Atomic operations** - Complete success or complete rollback
- **Backup integration** - Automatic restoration on failures
- **Dependency awareness** - Never break existing pack relationships

## 🐛 Troubleshooting

### Common Issues
- **"Server validation failed"** - Check `server.properties` contains `level-name`
- **"World config not found"** - Ensure world directory and JSON files exist
- **"Pack already installed"** - Use `--force` flag or check for UUID conflicts
- **"No manifest found"** - Verify addon file is a valid ZIP archive

### Debug Information
```bash
# Verbose output for troubleshooting
blockbench install addon.mcaddon /server --verbose

# Dry-run to see what would happen
blockbench install addon.mcaddon /server --dry-run --verbose  

# Interactive mode for step-by-step debugging
blockbench install addon.mcaddon /server --interactive
```