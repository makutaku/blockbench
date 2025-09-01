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
```

## Installation

```bash
go install github.com/makutaku/blockbench/cmd/blockbench@latest
```

## Development

```bash
git clone https://github.com/makutaku/blockbench.git
cd blockbench
go build ./cmd/blockbench
```