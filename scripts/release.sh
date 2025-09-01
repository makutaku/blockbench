#!/usr/bin/env bash

# Release script for Blockbench
# Usage: ./scripts/release.sh [version]
# Example: ./scripts/release.sh v1.0.0

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if git is available
if ! command -v git &> /dev/null; then
    log_error "git is required but not installed"
    exit 1
fi

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    log_error "Not in a git repository"
    exit 1
fi

# Check for uncommitted changes
if ! git diff-index --quiet HEAD -- 2>/dev/null; then
    log_error "There are uncommitted changes. Please commit or stash them before releasing."
    exit 1
fi

# Get version from argument or auto-generate
if [ $# -eq 1 ]; then
    VERSION="$1"
    # Validate version format (semantic versioning)
    if [[ ! $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$ ]]; then
        log_error "Version must follow semantic versioning (e.g., v1.0.0, v1.0.0-beta.1)"
        exit 1
    fi
else
    log_error "Usage: $0 <version>"
    log_error "Example: $0 v1.0.0"
    exit 1
fi

# Check if tag already exists
if git tag -l | grep -q "^${VERSION}$"; then
    log_error "Tag ${VERSION} already exists"
    exit 1
fi

log_info "Preparing release ${VERSION}..."

# Update version in version.go if needed
log_info "Updating version information..."

# Run quality checks
log_info "Running quality checks..."
if command -v make &> /dev/null; then
    make check
else
    log_warn "make not available, running basic checks..."
    go fmt ./...
    go vet ./...
    go mod tidy
    go test ./...
fi

# Build release binaries
log_info "Building release binaries..."
if command -v make &> /dev/null; then
    make build-all
else
    log_warn "make not available, building for current platform only..."
    mkdir -p bin
    go build -ldflags "\
        -X github.com/makutaku/blockbench/internal/version.Version=${VERSION} \
        -X github.com/makutaku/blockbench/internal/version.GitCommit=$(git rev-parse HEAD) \
        -X github.com/makutaku/blockbench/internal/version.BuildDate=$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
        -o bin/blockbench ./cmd/blockbench
fi

# Create and push tag
log_info "Creating git tag ${VERSION}..."
git tag -a "${VERSION}" -m "Release ${VERSION}"

log_info "Pushing tag to origin..."
git push origin "${VERSION}"

# Generate release notes template
RELEASE_NOTES_FILE="release-notes-${VERSION}.md"
log_info "Generating release notes template: ${RELEASE_NOTES_FILE}"

cat > "${RELEASE_NOTES_FILE}" << EOF
# Release ${VERSION}

## What's Changed

<!-- Add your release notes here -->

## Features
- 

## Bug Fixes
- 

## Breaking Changes
- 

## Installation

### Binary Download
Download the appropriate binary for your platform from the release assets.

### Go Install
\`\`\`bash
go install github.com/makutaku/blockbench/cmd/blockbench@${VERSION}
\`\`\`

### Build from Source
\`\`\`bash
git clone https://github.com/makutaku/blockbench.git
cd blockbench
git checkout ${VERSION}
make build
\`\`\`

## Checksums
<!-- Add checksums here if needed -->

**Full Changelog**: https://github.com/makutaku/blockbench/compare/...${VERSION}
EOF

log_info "✅ Release ${VERSION} preparation complete!"
log_info ""
log_info "Next steps:"
log_info "1. Edit ${RELEASE_NOTES_FILE} with detailed release notes"
log_info "2. Create a GitHub release using the tag ${VERSION}"
log_info "3. Upload the binaries from ./bin/ as release assets"
log_info "4. Publish the release"
log_info ""
log_info "Binaries built:"
if [ -d "./bin" ]; then
    ls -la ./bin/
fi