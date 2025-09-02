# Contributing to Blockbench

Thank you for considering contributing to Blockbench! This document provides guidelines and information for contributors.

## 🚀 Getting Started

### Prerequisites
- Go 1.23.4 or later
- Git
- Make (for build tasks)

### Setting Up Development Environment

1. **Fork and Clone**
   ```bash
   git clone https://github.com/YOUR_USERNAME/blockbench.git
   cd blockbench
   ```

2. **Install Dependencies**
   ```bash
   go mod download
   ```

3. **Build and Test**
   ```bash
   make build-dev    # Development build
   make check        # Run all quality checks
   ```

4. **Run Locally**
   ```bash
   ./bin/blockbench --help
   ```

## 🛠️ Development Workflow

### Code Style
- Follow standard Go conventions
- Use `make fmt` to format code
- Run `make lint` to check for issues
- Ensure `make check` passes before committing

### Commit Messages
Use conventional commit format:
```
type(scope): description

[optional body]

[optional footer]
```

Examples:
- `feat(list): add dependency tree visualization`
- `fix(installer): resolve UUID conflict detection`
- `docs(readme): update installation instructions`

### Testing
- Add tests for new features
- Ensure existing tests pass: `make test`
- Check test coverage: `make test-coverage`
- Test on multiple platforms when possible

### Pull Request Process

1. **Create Feature Branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make Changes**
   - Write code following project conventions
   - Add/update tests as needed
   - Update documentation

3. **Quality Checks**
   ```bash
   make check        # Runs fmt, vet, tidy, test, lint
   make build        # Ensure it builds successfully
   ```

4. **Commit and Push**
   ```bash
   git add .
   git commit -m "feat(scope): your change description"
   git push origin feature/your-feature-name
   ```

5. **Create Pull Request**
   - Use the provided PR template
   - Link related issues
   - Provide clear description of changes
   - Include test cases and examples

## 📋 Types of Contributions

### 🐛 Bug Reports
- Use the bug report template
- Include reproduction steps
- Provide system information
- Include command output with `--verbose` flag

### ✨ Feature Requests
- Use the feature request template
- Explain the use case
- Provide examples of desired behavior
- Consider backward compatibility

### 📚 Documentation
- README improvements
- Code comments
- Usage examples
- Developer documentation (CLAUDE.md)

### 🧪 Testing
- Unit tests for new features
- Integration tests
- Cross-platform testing
- Edge case coverage

## 🏗️ Project Structure

```
blockbench/
├── cmd/blockbench/         # Main application entry point
├── internal/
│   ├── addon/             # Core addon management
│   │   ├── installer.go   # Installation logic
│   │   ├── uninstaller.go # Uninstallation logic
│   │   ├── dependencies.go # Dependency analysis
│   │   └── simulator.go   # Dry-run simulation
│   ├── cli/               # Command implementations
│   ├── minecraft/         # Minecraft server integration
│   └── version/           # Version information
├── pkg/                   # Public packages
├── .github/              # GitHub workflows and templates
├── scripts/              # Build and release scripts
└── docs/                 # Documentation
```

### Key Components

- **DependencyAnalyzer**: Analyzes pack relationships and dependencies
- **DryRunSimulator**: Simulates operations without making changes
- **BackupManager**: Handles backup creation and restoration
- **Installation Pipeline**: Multi-stage validation and rollback system

## 🔧 Build System

### Make Targets
```bash
make build        # Production build with version injection
make build-dev    # Development build
make build-all    # Cross-platform builds
make test         # Run tests
make lint         # Run linter
make fmt          # Format code
make check        # All quality checks
make clean        # Clean build artifacts
```

### Release Process
1. Create and push version tag: `git tag v1.0.0 && git push origin v1.0.0`
2. GitHub Actions automatically builds and publishes release
3. Binaries are available for Linux, macOS, and Windows

## 📖 Documentation Guidelines

### Code Documentation
- Document public APIs
- Explain complex algorithms
- Include usage examples
- Keep comments up to date

### README Updates
- Update feature lists for new capabilities
- Add usage examples for new commands
- Keep installation instructions current
- Update troubleshooting section

### CLAUDE.md Updates
- Document new components and architecture changes
- Update development commands
- Explain design decisions
- Keep technical details current

## 🧪 Testing Guidelines

### Unit Tests
- Test individual functions and methods
- Mock external dependencies
- Cover edge cases and error conditions
- Aim for meaningful test coverage

### Integration Tests
- Test command-line interface
- Test file system operations
- Test cross-platform compatibility
- Test real-world scenarios

### Manual Testing
- Test with various addon files
- Test error conditions
- Test interactive modes
- Test on different operating systems

## 🚦 Code Review Process

### For Contributors
- Respond to feedback promptly
- Make requested changes
- Update tests and documentation
- Rebase if needed

### For Reviewers
- Be constructive and helpful
- Focus on code quality and maintainability
- Check for security issues
- Verify documentation updates

## 🔒 Security Considerations

- Never commit sensitive information (keys, passwords)
- Validate all user inputs
- Handle file system operations securely
- Follow secure coding practices for archive extraction

## 📞 Getting Help

- 💬 [GitHub Discussions](https://github.com/makutaku/blockbench/discussions)
- 🐛 [Issues](https://github.com/makutaku/blockbench/issues)
- 📚 [Documentation](https://github.com/makutaku/blockbench/blob/main/README.md)

## 📄 License

By contributing to Blockbench, you agree that your contributions will be licensed under the same license as the project.

---

Thank you for helping make Blockbench better! 🎉