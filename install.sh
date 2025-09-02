#!/bin/bash
set -e

# Blockbench Installer Script
# Usage: curl -sSL https://raw.githubusercontent.com/makutaku/blockbench/main/install.sh | bash

REPO="makutaku/blockbench"
BINARY_NAME="blockbench"
INSTALL_DIR="${BLOCKBENCH_INSTALL_DIR:-/usr/local/bin}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

error() {
    echo -e "${RED}❌ $1${NC}"
    exit 1
}

# Detect OS and architecture
detect_platform() {
    local os arch
    
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    arch="$(uname -m)"
    
    case "$os" in
        linux) os="linux" ;;
        darwin) os="darwin" ;;
        *) error "Unsupported operating system: $os" ;;
    esac
    
    case "$arch" in
        x86_64|amd64) arch="amd64" ;;
        arm64|aarch64) arch="arm64" ;;
        *) error "Unsupported architecture: $arch" ;;
    esac
    
    echo "${os}-${arch}"
}

# Get latest release version
get_latest_version() {
    local latest_url="https://api.github.com/repos/$REPO/releases/latest"
    
    if command -v curl >/dev/null 2>&1; then
        curl -s "$latest_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "$latest_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    else
        error "Neither curl nor wget is available. Please install one of them."
    fi
}

# Download and install binary
install_binary() {
    local version="$1"
    local platform="$2"
    local binary_name="${BINARY_NAME}-${platform}"
    local download_url="https://github.com/$REPO/releases/download/$version/$binary_name"
    local checksum_url="${download_url}.sha256"
    local temp_dir
    
    temp_dir="$(mktemp -d)"
    cd "$temp_dir"
    
    info "Downloading $BINARY_NAME $version for $platform..."
    
    # Download binary
    if command -v curl >/dev/null 2>&1; then
        curl -L -o "$binary_name" "$download_url" || error "Failed to download binary"
        curl -L -o "${binary_name}.sha256" "$checksum_url" || error "Failed to download checksum"
    elif command -v wget >/dev/null 2>&1; then
        wget -O "$binary_name" "$download_url" || error "Failed to download binary"
        wget -O "${binary_name}.sha256" "$checksum_url" || error "Failed to download checksum"
    fi
    
    # Verify checksum
    info "Verifying checksum..."
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum -c "${binary_name}.sha256" || error "Checksum verification failed"
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 -c "${binary_name}.sha256" || error "Checksum verification failed"
    else
        warning "Cannot verify checksum: neither sha256sum nor shasum available"
    fi
    
    # Make binary executable
    chmod +x "$binary_name"
    
    # Create install directory if it doesn't exist
    if [[ ! -d "$INSTALL_DIR" ]]; then
        info "Creating install directory: $INSTALL_DIR"
        sudo mkdir -p "$INSTALL_DIR" || mkdir -p "$INSTALL_DIR" 2>/dev/null || error "Failed to create install directory"
    fi
    
    # Install binary
    info "Installing $BINARY_NAME to $INSTALL_DIR..."
    if [[ -w "$INSTALL_DIR" ]]; then
        mv "$binary_name" "$INSTALL_DIR/$BINARY_NAME"
    else
        sudo mv "$binary_name" "$INSTALL_DIR/$BINARY_NAME" || error "Failed to install binary"
    fi
    
    # Cleanup
    cd - >/dev/null
    rm -rf "$temp_dir"
    
    success "$BINARY_NAME $version installed successfully!"
}

# Verify installation
verify_installation() {
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        local version
        version="$($BINARY_NAME version --short 2>/dev/null || echo "unknown")"
        success "Installation verified: $BINARY_NAME $version"
        info "Run '$BINARY_NAME --help' to get started"
    else
        warning "Installation completed, but $BINARY_NAME is not in PATH"
        info "You may need to add $INSTALL_DIR to your PATH or restart your shell"
        info "Run 'export PATH=\"$INSTALL_DIR:\$PATH\"' to add to current session"
    fi
}

# Main installation process
main() {
    info "Installing $BINARY_NAME..."
    
    # Check for existing installation
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        local current_version
        current_version="$($BINARY_NAME version --short 2>/dev/null || echo "unknown")"
        warning "$BINARY_NAME is already installed (version: $current_version)"
        echo -n "Do you want to continue and replace it? [y/N]: "
        read -r response
        case "$response" in
            [yY][eE][sS]|[yY]) ;;
            *) info "Installation cancelled"; exit 0 ;;
        esac
    fi
    
    local platform version
    platform="$(detect_platform)"
    version="$(get_latest_version)"
    
    if [[ -z "$version" ]]; then
        error "Failed to get latest version"
    fi
    
    info "Latest version: $version"
    info "Platform: $platform"
    info "Install directory: $INSTALL_DIR"
    
    install_binary "$version" "$platform"
    verify_installation
    
    echo
    success "🎉 $BINARY_NAME installation complete!"
    echo
    info "Quick start:"
    echo "  $BINARY_NAME --help                    # Show help"
    echo "  $BINARY_NAME install addon.mcaddon /server  # Install addon"
    echo "  $BINARY_NAME list /server --grouped         # List with dependencies"
    echo
    info "Documentation: https://github.com/$REPO"
}

# Check for help flag
if [[ "$1" == "-h" || "$1" == "--help" ]]; then
    echo "Blockbench Installer"
    echo ""
    echo "This script installs the latest version of blockbench."
    echo ""
    echo "Usage:"
    echo "  curl -sSL https://raw.githubusercontent.com/makutaku/blockbench/main/install.sh | bash"
    echo ""
    echo "Environment variables:"
    echo "  BLOCKBENCH_INSTALL_DIR  Directory to install binary (default: /usr/local/bin)"
    echo ""
    echo "Examples:"
    echo "  # Install to default location"
    echo "  curl -sSL https://raw.githubusercontent.com/makutaku/blockbench/main/install.sh | bash"
    echo ""
    echo "  # Install to custom directory"
    echo "  curl -sSL https://raw.githubusercontent.com/makutaku/blockbench/main/install.sh | BLOCKBENCH_INSTALL_DIR=~/.local/bin bash"
    exit 0
fi

# Run main installation
main "$@"