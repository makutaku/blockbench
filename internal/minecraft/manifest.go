package minecraft

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// ManifestHeader represents the header section of a manifest.json file
type ManifestHeader struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	UUID        string `json:"uuid"`
	Version     [3]int `json:"version"`
	MinVersion  [3]int `json:"min_engine_version,omitempty"`
}

// ManifestModule represents a module in the manifest
type ManifestModule struct {
	Type        string `json:"type"`
	UUID        string `json:"uuid"`
	Version     [3]int `json:"version"`
	Description string `json:"description,omitempty"`
}

// ManifestDependency represents a dependency on another pack or module
type ManifestDependency struct {
	// Pack dependency format
	UUID    string `json:"uuid,omitempty"`
	Version [3]int `json:"-"` // Custom handling due to version field conflict

	// Module dependency format
	ModuleName    string `json:"module_name,omitempty"`
	ModuleVersion string `json:"-"` // Custom handling due to version field conflict

	// Raw version field for custom parsing
	RawVersion json.RawMessage `json:"version,omitempty"`
}

// UnmarshalJSON custom unmarshaling to handle both pack and module dependency formats
func (md *ManifestDependency) UnmarshalJSON(data []byte) error {
	// Define a temporary struct that matches the JSON structure
	type TempDependency struct {
		UUID       string          `json:"uuid,omitempty"`
		ModuleName string          `json:"module_name,omitempty"`
		RawVersion json.RawMessage `json:"version,omitempty"`
	}

	var temp TempDependency
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	md.UUID = temp.UUID
	md.ModuleName = temp.ModuleName
	md.RawVersion = temp.RawVersion

	// Parse version based on format
	if len(temp.RawVersion) > 0 {
		// Try to parse as array first (pack dependency format)
		var versionArray [3]int
		if err := json.Unmarshal(temp.RawVersion, &versionArray); err == nil {
			md.Version = versionArray
		} else {
			// Parse as string (module dependency format)
			var versionString string
			if err := json.Unmarshal(temp.RawVersion, &versionString); err == nil {
				md.ModuleVersion = versionString
			} else {
				return fmt.Errorf("failed to parse version field: %w", err)
			}
		}
	}

	return nil
}

// Manifest represents a complete manifest.json file
type Manifest struct {
	FormatVersion int                  `json:"format_version"`
	Header        ManifestHeader       `json:"header"`
	Modules       []ManifestModule     `json:"modules"`
	Dependencies  []ManifestDependency `json:"dependencies,omitempty"`
}

// PackType represents the type of a Minecraft pack
type PackType string

const (
	PackTypeBehavior PackType = "behavior"
	PackTypeResource PackType = "resource"
	PackTypeUnknown  PackType = "unknown"
)

// GetPackType determines if this manifest is for a behavior pack or resource pack
func (m *Manifest) GetPackType() PackType {
	for _, module := range m.Modules {
		switch module.Type {
		case "data":
			return PackTypeBehavior
		case "resources":
			return PackTypeResource
		}
	}
	return PackTypeUnknown
}

// GetDisplayName returns a human-readable name for the pack
func (m *Manifest) GetDisplayName() string {
	if m.Header.Name != "" {
		return m.Header.Name
	}
	return fmt.Sprintf("Pack-%s", m.Header.UUID[:8])
}

// GetVersionString returns the version as a string
func (m *Manifest) GetVersionString() string {
	return fmt.Sprintf("%d.%d.%d", m.Header.Version[0], m.Header.Version[1], m.Header.Version[2])
}

// ParseManifest reads and parses a manifest.json file
func ParseManifest(filePath string) (*Manifest, error) {
	// #nosec G304 - filePath is validated manifest.json within controlled extraction directory
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open manifest file: %w", err)
	}
	defer file.Close()

	return ParseManifestFromReader(file)
}

// ParseManifestFromReader parses a manifest from an io.Reader
func ParseManifestFromReader(reader io.Reader) (*Manifest, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest data: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest JSON: %w", err)
	}

	// Validate required fields
	if manifest.Header.UUID == "" {
		return nil, fmt.Errorf("manifest missing required UUID in header")
	}

	if len(manifest.Modules) == 0 {
		return nil, fmt.Errorf("manifest missing required modules")
	}

	return &manifest, nil
}

// ValidateManifest performs additional validation on a manifest
func ValidateManifest(manifest *Manifest) error {
	if manifest.FormatVersion < 1 || manifest.FormatVersion > 2 {
		return fmt.Errorf("unsupported format version: %d", manifest.FormatVersion)
	}

	// Check for duplicate module UUIDs
	moduleUUIDs := make(map[string]bool)
	for _, module := range manifest.Modules {
		if moduleUUIDs[module.UUID] {
			return fmt.Errorf("duplicate module UUID: %s", module.UUID)
		}
		moduleUUIDs[module.UUID] = true
	}

	return nil
}
