package addon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/makutaku/blockbench/internal/minecraft"
	"github.com/makutaku/blockbench/pkg/filesystem"
)

// ExtractedAddon represents an extracted addon with its components
type ExtractedAddon struct {
	TempDir       string
	BehaviorPacks []*ExtractedPack
	ResourcePacks []*ExtractedPack
}

// ExtractedPack represents a single extracted pack
type ExtractedPack struct {
	Path     string
	Manifest *minecraft.Manifest
	PackType minecraft.PackType
}

// Cleanup removes the temporary directory
func (ea *ExtractedAddon) Cleanup() error {
	if ea.TempDir != "" {
		return os.RemoveAll(ea.TempDir)
	}
	return nil
}

// GetAllPacks returns all packs (behavior and resource) in a single slice
func (ea *ExtractedAddon) GetAllPacks() []*ExtractedPack {
	var allPacks []*ExtractedPack
	allPacks = append(allPacks, ea.BehaviorPacks...)
	allPacks = append(allPacks, ea.ResourcePacks...)
	return allPacks
}

// ExtractAddon extracts a .mcaddon or .mcpack file and analyzes its contents
func ExtractAddon(addonPath string, dryRun bool) (*ExtractedAddon, error) {
	// Validate file extension
	ext := strings.ToLower(filepath.Ext(addonPath))
	if ext != ".mcaddon" && ext != ".mcpack" {
		return nil, fmt.Errorf("unsupported file type: %s (expected .mcaddon or .mcpack)", ext)
	}

	// Validate archive
	if err := filesystem.ValidateArchive(addonPath); err != nil {
		return nil, fmt.Errorf("archive validation failed: %w", err)
	}

	// Get archive info
	archiveInfo, err := filesystem.GetArchiveInfo(addonPath)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze archive: %w", err)
	}

	if !archiveInfo.HasManifest {
		return nil, fmt.Errorf("archive does not contain any manifest.json files")
	}

	if dryRun {
		// For dry run, just analyze without extracting
		return &ExtractedAddon{
			TempDir: "(dry-run)",
		}, nil
	}

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "blockbench_extract_*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	// Extract archive
	if err := filesystem.ExtractArchive(addonPath, tempDir); err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to extract archive: %w", err)
	}

	// Analyze extracted contents
	addon, err := analyzeExtractedAddon(tempDir)
	if err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to analyze extracted addon: %w", err)
	}

	addon.TempDir = tempDir
	return addon, nil
}

// analyzeExtractedAddon analyzes the contents of an extracted addon
func analyzeExtractedAddon(tempDir string) (*ExtractedAddon, error) {
	addon := &ExtractedAddon{
		BehaviorPacks: make([]*ExtractedPack, 0),
		ResourcePacks: make([]*ExtractedPack, 0),
	}

	// Find all manifest.json files
	manifests, err := findManifestFiles(tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to find manifest files: %w", err)
	}

	if len(manifests) == 0 {
		return nil, fmt.Errorf("no manifest.json files found in extracted addon")
	}

	// Process each manifest
	for _, manifestPath := range manifests {
		pack, err := processManifest(manifestPath)
		if err != nil {
			return nil, fmt.Errorf("failed to process manifest %s: %w", manifestPath, err)
		}

		switch pack.PackType {
		case minecraft.PackTypeBehavior:
			addon.BehaviorPacks = append(addon.BehaviorPacks, pack)
		case minecraft.PackTypeResource:
			addon.ResourcePacks = append(addon.ResourcePacks, pack)
		default:
			return nil, fmt.Errorf("unknown pack type in manifest %s", manifestPath)
		}
	}

	return addon, nil
}

// findManifestFiles recursively finds all manifest.json files in a directory
func findManifestFiles(rootDir string) ([]string, error) {
	var manifests []string

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.ToLower(info.Name()) == "manifest.json" {
			manifests = append(manifests, path)
		}

		return nil
	})

	return manifests, err
}

// processManifest loads and validates a manifest file
func processManifest(manifestPath string) (*ExtractedPack, error) {
	manifest, err := minecraft.ParseManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	if err := minecraft.ValidateManifest(manifest); err != nil {
		return nil, fmt.Errorf("manifest validation failed: %w", err)
	}

	packType := manifest.GetPackType()
	if packType == minecraft.PackTypeUnknown {
		return nil, fmt.Errorf("unable to determine pack type from manifest")
	}

	// Get pack directory (parent directory of manifest)
	packDir := filepath.Dir(manifestPath)

	return &ExtractedPack{
		Path:     packDir,
		Manifest: manifest,
		PackType: packType,
	}, nil
}

// ValidateAddonFile performs pre-extraction validation on an addon file
func ValidateAddonFile(addonPath string) error {
	// Check if file exists
	if _, err := os.Stat(addonPath); os.IsNotExist(err) {
		return fmt.Errorf("addon file does not exist: %s", addonPath)
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(addonPath))
	if ext != ".mcaddon" && ext != ".mcpack" {
		return fmt.Errorf("unsupported file type: %s (expected .mcaddon or .mcpack)", ext)
	}

	// Validate archive structure
	if err := filesystem.ValidateArchive(addonPath); err != nil {
		return fmt.Errorf("archive validation failed: %w", err)
	}

	// Get basic archive info
	info, err := filesystem.GetArchiveInfo(addonPath)
	if err != nil {
		return fmt.Errorf("failed to analyze archive: %w", err)
	}

	if !info.HasManifest {
		return fmt.Errorf("archive does not contain any manifest.json files")
	}

	if info.TotalFiles == 0 {
		return fmt.Errorf("archive is empty")
	}

	return nil
}
