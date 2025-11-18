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

#### Installation Failures

**"Server validation failed: server.properties not found"**
- **Cause**: The path provided is not a valid Minecraft Bedrock server directory
- **Solution**: Ensure you're pointing to the server root (where `server.properties` is located)
  ```bash
  # Correct
  blockbench install addon.mcaddon /path/to/bedrock-server

  # Incorrect
  blockbench install addon.mcaddon /path/to/bedrock-server/worlds
  ```

**"level-name property not found in server.properties"**
- **Cause**: `server.properties` doesn't contain the `level-name` configuration
- **Solution**: Add the level-name property to your `server.properties`:
  ```properties
  level-name=Bedrock level
  ```

**"World config not found"**
- **Cause**: World directory or configuration JSON files are missing
- **Solution**:
  1. Ensure the world directory exists in `worlds/[level-name]/`
  2. Check for `world_behavior_packs.json` and `world_resource_packs.json`
  3. If missing, create them as empty arrays: `[]`

**"Pack already installed" or UUID conflict**
- **Cause**: A pack with the same UUID is already installed
- **Solution**:
  ```bash
  # List installed packs to see conflicts
  blockbench list /server

  # Force reinstall (replaces existing)
  blockbench install addon.mcaddon /server --force

  # Or uninstall first
  blockbench uninstall <pack-uuid> /server
  ```

**"No manifest found in archive" or "Invalid addon file"**
- **Cause**: The file is not a valid .mcaddon/.mcpack ZIP archive
- **Solution**:
  1. Verify the file is a valid ZIP: `unzip -t addon.mcaddon`
  2. Check that `manifest.json` exists in the pack root
  3. Ensure the manifest is valid JSON with required fields

**"Missing dependencies: Pack requires X"**
- **Cause**: The addon depends on packs that aren't installed
- **Solution**:
  ```bash
  # See what's missing
  blockbench install addon.mcaddon /server --dry-run

  # Install dependencies first, or use --force to skip validation
  blockbench install addon.mcaddon /server --force
  ```

**"Circular dependency detected"**
- **Cause**: Packs have circular dependency relationships (A→B→C→A)
- **Solution**: This is typically a pack design issue. Review pack manifests:
  ```bash
  # Visualize dependencies
  blockbench list /server --tree
  ```

#### Permission Errors

**"Permission denied" when installing**
- **Cause**: Insufficient permissions to write to server directories
- **Solution**:
  ```bash
  # Check directory permissions
  ls -la /server/development_behavior_packs

  # Fix ownership (Linux/macOS)
  sudo chown -R $USER:$USER /server

  # Or run with appropriate permissions
  sudo blockbench install addon.mcaddon /server
  ```

**"Failed to create config directory"**
- **Cause**: Cannot write to world configuration directory
- **Solution**: Ensure write permissions on `worlds/[level-name]/`

#### Uninstallation Issues

**"Cannot uninstall: Pack X depends on this pack"**
- **Cause**: Other packs depend on the one you're trying to remove
- **Solution**:
  ```bash
  # See what depends on it
  blockbench list /server --grouped

  # Uninstall dependents first, or force removal
  blockbench uninstall <pack-uuid> /server --force
  ```

#### Large Pack Issues

**"File too large after decompression"**
- **Cause**: Pack exceeds 100MB default limit (decompression bomb protection)
- **Solution**: Increase the limit for large texture packs:
  ```bash
  # Allow up to 200MB
  export BLOCKBENCH_MAX_FILE_SIZE=209715200
  blockbench install large-pack.mcaddon /server
  ```

### Debug Information

```bash
# Verbose output for detailed troubleshooting
blockbench install addon.mcaddon /server --verbose

# Dry-run to preview operations without making changes
blockbench install addon.mcaddon /server --dry-run --verbose

# Interactive mode for step-by-step confirmation
blockbench install addon.mcaddon /server --interactive

# Check what's currently installed
blockbench list /server

# See dependency relationships
blockbench list /server --tree

# JSON output for scripting/debugging
blockbench list /server --json
```

### Getting Help

If you encounter issues not covered here:

1. **Check verbose output**: Most errors include suggestions in `--verbose` mode
2. **Validate your setup**: Run `blockbench list /server` to ensure basic functionality
3. **Check server structure**: Ensure your server follows the standard Bedrock layout
4. **Review logs**: Check for detailed error messages in the output
5. **Report issues**: [Open an issue](https://github.com/makutaku/blockbench/issues) with:
   - Command you ran
   - Full error message
   - Server directory structure (`tree -L 2 /server`)
   - Blockbench version (`blockbench version`)