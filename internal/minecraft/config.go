package minecraft

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PackReference represents a pack reference in world config files
type PackReference struct {
	PackID  string `json:"pack_id"`
	Version [3]int `json:"version"`
}

// WorldConfig represents the structure of world config files
type WorldConfig []PackReference

// ServerPaths contains paths to important server directories and files
type ServerPaths struct {
	ServerRoot           string
	WorldsDir            string
	BehaviorPacksDir     string
	ResourcePacksDir     string
	WorldBehaviorPacks   string
	WorldResourcePacks   string
	WorldBehaviorHistory string
	WorldResourceHistory string
}

// NewServerPaths creates a ServerPaths struct with standard Bedrock server paths
func NewServerPaths(serverRoot string) (*ServerPaths, error) {
	worldsDir := filepath.Join(serverRoot, "worlds")

	// Get world name from server.properties - no fallbacks
	worldName, err := getWorldNameFromProperties(serverRoot)
	if err != nil {
		return nil, err
	}
	worldDir := filepath.Join(worldsDir, worldName)

	return &ServerPaths{
		ServerRoot:           serverRoot,
		WorldsDir:            worldsDir,
		BehaviorPacksDir:     filepath.Join(serverRoot, "development_behavior_packs"),
		ResourcePacksDir:     filepath.Join(serverRoot, "development_resource_packs"),
		WorldBehaviorPacks:   filepath.Join(worldDir, "world_behavior_packs.json"),
		WorldResourcePacks:   filepath.Join(worldDir, "world_resource_packs.json"),
		WorldBehaviorHistory: filepath.Join(worldDir, "world_behavior_pack_history.json"),
		WorldResourceHistory: filepath.Join(worldDir, "world_resource_pack_history.json"),
	}, nil
}

// getWorldNameFromProperties reads the world name from server.properties
func getWorldNameFromProperties(serverRoot string) (string, error) {
	propertiesPath := filepath.Join(serverRoot, "server.properties")

	file, err := os.Open(propertiesPath)
	if err != nil {
		return "", fmt.Errorf("cannot read server.properties at %s: %w", propertiesPath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// Look for level-name property
		if strings.HasPrefix(line, "level-name=") {
			worldName := strings.TrimPrefix(line, "level-name=")
			worldName = strings.TrimSpace(worldName)
			if worldName == "" {
				return "", fmt.Errorf("level-name property is empty in %s", propertiesPath)
			}
			return worldName, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading server.properties: %w", err)
	}

	return "", fmt.Errorf("level-name property not found in %s", propertiesPath)
}

// ValidateServerStructure checks if the server directory has the expected structure
func (sp *ServerPaths) ValidateServerStructure() error {
	requiredDirs := []string{
		sp.WorldsDir,
		sp.BehaviorPacksDir,
		sp.ResourcePacksDir,
	}

	for _, dir := range requiredDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return fmt.Errorf("required directory does not exist: %s", dir)
		}
	}

	return nil
}

// LoadWorldConfig loads a world config file (behavior or resource packs)
func LoadWorldConfig(filePath string) (WorldConfig, error) {
	// If file doesn't exist, return empty config
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return WorldConfig{}, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}

	var config WorldConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", filePath, err)
	}

	return config, nil
}

// SaveWorldConfig saves a world config file
func SaveWorldConfig(filePath string, config WorldConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", filePath, err)
	}

	return nil
}

// AddPackToConfig adds a pack reference to a config, avoiding duplicates
func AddPackToConfig(config WorldConfig, packID string, version [3]int) WorldConfig {
	// Check if pack already exists
	for i, pack := range config {
		if pack.PackID == packID {
			// Update existing pack version
			config[i].Version = version
			return config
		}
	}

	// Add new pack
	newPack := PackReference{
		PackID:  packID,
		Version: version,
	}

	return append(config, newPack)
}

// RemovePackFromConfig removes a pack reference from a config
func RemovePackFromConfig(config WorldConfig, packID string) WorldConfig {
	var result WorldConfig
	for _, pack := range config {
		if pack.PackID != packID {
			result = append(result, pack)
		}
	}
	return result
}

// HasPack checks if a pack is present in the config
func (wc WorldConfig) HasPack(packID string) bool {
	for _, pack := range wc {
		if pack.PackID == packID {
			return true
		}
	}
	return false
}

// GetPack retrieves a pack reference by ID
func (wc WorldConfig) GetPack(packID string) (*PackReference, bool) {
	for _, pack := range wc {
		if pack.PackID == packID {
			return &pack, true
		}
	}
	return nil, false
}
