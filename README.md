# Blockbench

A command-line tool for managing Minecraft Bedrock Edition addons on servers.

## Features

- Install `.mcaddon` and `.mcpack` files to Minecraft Bedrock servers
- Uninstall addons safely with dependency checking
- List currently installed addons
- Dry-run mode for testing operations
- Automatic backup and rollback on failures
- Idempotent operations (safe to re-run)

## Usage

```bash
# Install an addon
blockbench install addon.mcaddon /path/to/server

# Uninstall an addon
blockbench uninstall addon-name /path/to/server

# List installed addons
blockbench list /path/to/server

# Test operations with dry-run
blockbench install addon.mcaddon /path/to/server --dry-run

# Show version information
blockbench version
blockbench version --json
blockbench --version
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

## Development

```bash
git clone https://github.com/makutaku/blockbench.git
cd blockbench

# Development build (no version injection)
make build-dev

# Production build with version info
make build

# Run tests
make test

# Run all quality checks
make check

# Cross-platform build
make build-all
```