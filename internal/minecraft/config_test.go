package minecraft

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetWorldNameFromProperties(t *testing.T) {
	tests := []struct {
		name           string
		propertiesData string
		expectedLevel  string
		expectError    bool
	}{
		{
			name: "valid server.properties",
			propertiesData: `# Minecraft server properties
#Mon Jan 01 12:00:00 UTC 2024
gamemode=survival
difficulty=easy
level-name=Bedrock level
server-name=Dedicated Server
server-port=19132
server-portv6=19133
max-players=10
allow-cheats=false
online-mode=true
white-list=false`,
			expectedLevel: "Bedrock level",
			expectError:   false,
		},
		{
			name: "properties with spaces",
			propertiesData: `level-name=My World With Spaces
server-name=Test Server
max-players=20`,
			expectedLevel: "My World With Spaces",
			expectError:   false,
		},
		{
			name: "properties with special characters",
			propertiesData: `level-name=Test_World-2024
server-name=Server & Co.
max-players=5`,
			expectedLevel: "Test_World-2024",
			expectError:   false,
		},
		{
			name: "missing level-name",
			propertiesData: `server-name=Test Server
max-players=10
gamemode=creative`,
			expectedLevel: "",
			expectError:   true,
		},
		{
			name:           "empty properties file",
			propertiesData: ``,
			expectedLevel:  "",
			expectError:    true,
		},
		{
			name: "commented level-name",
			propertiesData: `#level-name=Commented World
server-name=Test Server
level-name=Actual World`,
			expectedLevel: "Actual World",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tempDir, err := os.MkdirTemp("", "blockbench-config-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			propertiesPath := filepath.Join(tempDir, "server.properties")
			err = os.WriteFile(propertiesPath, []byte(tt.propertiesData), 0600)
			if err != nil {
				t.Fatalf("Failed to write properties file: %v", err)
			}

			// Test function - NewServerPaths should succeed if level-name is found
			tempServerDir := filepath.Dir(propertiesPath)
			serverPaths, err := NewServerPaths(tempServerDir)

			// For cases with valid level-name, we expect success and can extract the level name
			// from the world directory path: WorldBehaviorPacks = worlds/LEVELNAME/world_behavior_packs.json
			var levelName string
			if err == nil && serverPaths != nil {
				worldBehaviorPath := serverPaths.WorldBehaviorPacks
				// Extract: /temp/worlds/LEVELNAME/world_behavior_packs.json -> LEVELNAME
				worldDir := filepath.Dir(worldBehaviorPath) // /temp/worlds/LEVELNAME
				levelName = filepath.Base(worldDir)         // LEVELNAME
			}

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if levelName != tt.expectedLevel {
				t.Errorf("Expected level name %q, got %q", tt.expectedLevel, levelName)
			}
		})
	}
}

func TestNewServerPathsNonExistent(t *testing.T) {
	_, err := NewServerPaths("/path/that/does/not/exist")
	if err == nil {
		t.Error("Expected error for non-existent server directory")
	}
}

func TestLoadWorldConfig(t *testing.T) {
	tests := []struct {
		name        string
		configData  string
		expectError bool
		validate    func(*testing.T, WorldConfig)
	}{
		{
			name: "valid behavior pack config",
			configData: `[
				{
					"pack_id": "12345678-1234-1234-1234-123456789abc",
					"version": [1, 0, 0]
				},
				{
					"pack_id": "87654321-4321-4321-4321-fedcba987654",
					"version": [2, 1, 0]
				}
			]`,
			expectError: false,
			validate: func(t *testing.T, config WorldConfig) {
				if len(config) != 2 {
					t.Errorf("Expected 2 pack configs, got %d", len(config))
				}
				if config[0].PackID != "12345678-1234-1234-1234-123456789abc" {
					t.Errorf("Expected first pack ID '12345678-1234-1234-1234-123456789abc', got %q",
						config[0].PackID)
				}
				expectedVersion := [3]int{1, 0, 0}
				if config[0].Version != expectedVersion {
					t.Errorf("Expected first pack version %v, got %v", expectedVersion, config[0].Version)
				}
			},
		},
		{
			name:        "empty config",
			configData:  `[]`,
			expectError: false,
			validate: func(t *testing.T, config WorldConfig) {
				if len(config) != 0 {
					t.Errorf("Expected empty config, got %d entries", len(config))
				}
			},
		},
		{
			name:        "invalid JSON",
			configData:  `[invalid json`,
			expectError: true,
			validate:    nil,
		},
		{
			name: "config with missing fields",
			configData: `[
				{
					"pack_id": "12345678-1234-1234-1234-123456789abc"
				}
			]`,
			expectError: false,
			validate: func(t *testing.T, config WorldConfig) {
				if len(config) != 1 {
					t.Errorf("Expected 1 pack config, got %d", len(config))
				}
				// Version should be zero value
				expectedVersion := [3]int{0, 0, 0}
				if config[0].Version != expectedVersion {
					t.Errorf("Expected zero version %v, got %v", expectedVersion, config[0].Version)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tempDir, err := os.MkdirTemp("", "blockbench-config-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			configPath := filepath.Join(tempDir, "world_behavior_packs.json")
			err = os.WriteFile(configPath, []byte(tt.configData), 0600)
			if err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			// Test function
			config, err := LoadWorldConfig(configPath)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.validate != nil && config != nil {
				tt.validate(t, config)
			}
		})
	}
}

func TestSaveWorldConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "blockbench-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test config
	config := WorldConfig{
		{
			PackID:  "12345678-1234-1234-1234-123456789abc",
			Version: [3]int{1, 0, 0},
		},
		{
			PackID:  "87654321-4321-4321-4321-fedcba987654",
			Version: [3]int{2, 1, 0},
		},
	}

	configPath := filepath.Join(tempDir, "test_config.json")
	err = SaveWorldConfig(configPath, config)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Load and verify content
	loadedConfig, err := LoadWorldConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if len(loadedConfig) != len(config) {
		t.Errorf("Expected %d entries, got %d", len(config), len(loadedConfig))
	}

	for i, entry := range config {
		if loadedConfig[i].PackID != entry.PackID {
			t.Errorf("Entry %d PackID mismatch: expected %q, got %q",
				i, entry.PackID, loadedConfig[i].PackID)
		}
		if loadedConfig[i].Version != entry.Version {
			t.Errorf("Entry %d Version mismatch: expected %v, got %v",
				i, entry.Version, loadedConfig[i].Version)
		}
	}
}

func TestSaveWorldConfigEmptyConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "blockbench-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test empty config
	config := WorldConfig{}
	configPath := filepath.Join(tempDir, "empty_config.json")

	err = SaveWorldConfig(configPath, config)
	if err != nil {
		t.Fatalf("Failed to save empty config: %v", err)
	}

	// Load and verify
	loadedConfig, err := LoadWorldConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load empty config: %v", err)
	}

	if len(loadedConfig) != 0 {
		t.Errorf("Expected empty config, got %d entries", len(loadedConfig))
	}
}

func TestLoadWorldConfigNonExistent(t *testing.T) {
	config, err := LoadWorldConfig("/path/that/does/not/exist/config.json")
	if err != nil {
		t.Errorf("Expected no error for non-existent file (should return empty config), got: %v", err)
	}
	if len(config) != 0 {
		t.Errorf("Expected empty config for non-existent file, got %d entries", len(config))
	}
}

func TestSaveWorldConfigInvalidPath(t *testing.T) {
	config := WorldConfig{
		{PackID: "test", Version: [3]int{1, 0, 0}},
	}

	// Try to save to invalid path (using a file as a directory path)
	// This should fail because /dev/null is a file, not a directory
	err := SaveWorldConfig("/dev/null/config.json", config)
	if err == nil {
		t.Error("Expected error for invalid save path")
	}
}

func TestPackReference(t *testing.T) {
	entry := PackReference{
		PackID:  "12345678-1234-1234-1234-123456789abc",
		Version: [3]int{2, 1, 3},
	}

	if entry.PackID != "12345678-1234-1234-1234-123456789abc" {
		t.Errorf("Expected PackID '12345678-1234-1234-1234-123456789abc', got %q", entry.PackID)
	}

	expectedVersion := [3]int{2, 1, 3}
	if entry.Version != expectedVersion {
		t.Errorf("Expected version %v, got %v", expectedVersion, entry.Version)
	}
}

func BenchmarkLoadWorldConfig(b *testing.B) {
	// Create temporary config file
	tempDir, err := os.MkdirTemp("", "blockbench-config-bench")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configData := `[
		{
			"pack_id": "12345678-1234-1234-1234-123456789abc",
			"version": [1, 0, 0]
		},
		{
			"pack_id": "87654321-4321-4321-4321-fedcba987654",
			"version": [2, 1, 0]
		},
		{
			"pack_id": "11111111-1111-1111-1111-111111111111",
			"version": [1, 2, 3]
		}
	]`

	configPath := filepath.Join(tempDir, "benchmark_config.json")
	err = os.WriteFile(configPath, []byte(configData), 0600)
	if err != nil {
		b.Fatalf("Failed to write config file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadWorldConfig(configPath)
		if err != nil {
			b.Fatalf("Load failed: %v", err)
		}
	}
}
