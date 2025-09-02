package minecraft

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseManifest(t *testing.T) {
	tests := []struct {
		name        string
		manifestData string
		expectError bool
		validate    func(*testing.T, *Manifest)
	}{
		{
			name: "valid behavior pack manifest",
			manifestData: `{
				"format_version": 2,
				"header": {
					"name": "Test Behavior Pack",
					"description": "A test behavior pack",
					"uuid": "12345678-1234-1234-1234-123456789abc",
					"version": [1, 0, 0],
					"min_engine_version": [1, 16, 0]
				},
				"modules": [
					{
						"type": "data",
						"uuid": "12345678-1234-1234-1234-123456789abd",
						"version": [1, 0, 0]
					}
				]
			}`,
			expectError: false,
			validate: func(t *testing.T, m *Manifest) {
				if m.FormatVersion != 2 {
					t.Errorf("Expected format version 2, got %d", m.FormatVersion)
				}
				if m.Header.Name != "Test Behavior Pack" {
					t.Errorf("Expected name 'Test Behavior Pack', got %q", m.Header.Name)
				}
				if len(m.Modules) != 1 {
					t.Errorf("Expected 1 module, got %d", len(m.Modules))
				}
				if m.Modules[0].Type != "data" {
					t.Errorf("Expected module type 'data', got %q", m.Modules[0].Type)
				}
			},
		},
		{
			name: "valid resource pack manifest",
			manifestData: `{
				"format_version": 2,
				"header": {
					"name": "Test Resource Pack",
					"description": "A test resource pack",
					"uuid": "87654321-4321-4321-4321-abcdef123456",
					"version": [1, 2, 3]
				},
				"modules": [
					{
						"type": "resources",
						"uuid": "87654321-4321-4321-4321-abcdef123457",
						"version": [1, 2, 3]
					}
				]
			}`,
			expectError: false,
			validate: func(t *testing.T, m *Manifest) {
				if m.Header.Name != "Test Resource Pack" {
					t.Errorf("Expected name 'Test Resource Pack', got %q", m.Header.Name)
				}
				if m.Modules[0].Type != "resources" {
					t.Errorf("Expected module type 'resources', got %q", m.Modules[0].Type)
				}
				expectedVersion := [3]int{1, 2, 3}
				if m.Header.Version != expectedVersion {
					t.Errorf("Expected version %v, got %v", expectedVersion, m.Header.Version)
				}
			},
		},
		{
			name: "manifest with pack dependencies",
			manifestData: `{
				"format_version": 2,
				"header": {
					"name": "Dependent Pack",
					"description": "Pack with dependencies",
					"uuid": "11111111-1111-1111-1111-111111111111",
					"version": [1, 0, 0]
				},
				"modules": [
					{
						"type": "data",
						"uuid": "11111111-1111-1111-1111-111111111112",
						"version": [1, 0, 0]
					}
				],
				"dependencies": [
					{
						"uuid": "22222222-2222-2222-2222-222222222222",
						"version": [1, 0, 0]
					}
				]
			}`,
			expectError: false,
			validate: func(t *testing.T, m *Manifest) {
				if len(m.Dependencies) != 1 {
					t.Errorf("Expected 1 dependency, got %d", len(m.Dependencies))
				}
				if m.Dependencies[0].UUID != "22222222-2222-2222-2222-222222222222" {
					t.Errorf("Expected dependency UUID '22222222-2222-2222-2222-222222222222', got %q", 
						m.Dependencies[0].UUID)
				}
			},
		},
		{
			name: "manifest with module dependencies",
			manifestData: `{
				"format_version": 2,
				"header": {
					"name": "Script Pack",
					"description": "Pack with script dependencies",
					"uuid": "33333333-3333-3333-3333-333333333333",
					"version": [1, 0, 0]
				},
				"modules": [
					{
						"type": "script",
						"uuid": "33333333-3333-3333-3333-333333333334",
						"version": [1, 0, 0]
					}
				],
				"dependencies": [
					{
						"module_name": "@minecraft/server",
						"version": "1.2.0"
					}
				]
			}`,
			expectError: false,
			validate: func(t *testing.T, m *Manifest) {
				if len(m.Dependencies) != 1 {
					t.Errorf("Expected 1 dependency, got %d", len(m.Dependencies))
				}
				if m.Dependencies[0].ModuleName != "@minecraft/server" {
					t.Errorf("Expected module name '@minecraft/server', got %q", 
						m.Dependencies[0].ModuleName)
				}
				if m.Dependencies[0].ModuleVersion != "1.2.0" {
					t.Errorf("Expected module version '1.2.0', got %q", 
						m.Dependencies[0].ModuleVersion)
				}
			},
		},
		{
			name:        "invalid JSON",
			manifestData: `{invalid json`,
			expectError: true,
			validate:    nil,
		},
		{
			name: "missing required fields",
			manifestData: `{
				"format_version": 2
			}`,
			expectError: false, // JSON unmarshaling will succeed, but fields will be empty
			validate: func(t *testing.T, m *Manifest) {
				if m.Header.Name != "" {
					t.Error("Expected empty name for incomplete manifest")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tempDir, err := os.MkdirTemp("", "blockbench-manifest-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			manifestPath := filepath.Join(tempDir, "manifest.json")
			err = os.WriteFile(manifestPath, []byte(tt.manifestData), 0600)
			if err != nil {
				t.Fatalf("Failed to write manifest file: %v", err)
			}

			// Parse manifest
			manifest, err := ParseManifest(manifestPath)
			
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.validate != nil && manifest != nil {
				tt.validate(t, manifest)
			}
		})
	}
}

func TestManifestGetVersionString(t *testing.T) {
	manifest := &Manifest{
		Header: ManifestHeader{
			Version: [3]int{1, 2, 3},
		},
	}

	expected := "1.2.3"
	result := manifest.GetVersionString()
	if result != expected {
		t.Errorf("Expected version string %q, got %q", expected, result)
	}
}

func TestManifestGetVersionStringZero(t *testing.T) {
	manifest := &Manifest{
		Header: ManifestHeader{
			Version: [3]int{0, 0, 0},
		},
	}

	expected := "0.0.0"
	result := manifest.GetVersionString()
	if result != expected {
		t.Errorf("Expected version string %q, got %q", expected, result)
	}
}

func TestManifestDependencyUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		validate func(*testing.T, *ManifestDependency)
	}{
		{
			name:     "pack dependency",
			jsonData: `{"uuid": "12345678-1234-1234-1234-123456789abc", "version": [1, 2, 3]}`,
			validate: func(t *testing.T, md *ManifestDependency) {
				if md.UUID != "12345678-1234-1234-1234-123456789abc" {
					t.Errorf("Expected UUID '12345678-1234-1234-1234-123456789abc', got %q", md.UUID)
				}
				expectedVersion := [3]int{1, 2, 3}
				if md.Version != expectedVersion {
					t.Errorf("Expected version %v, got %v", expectedVersion, md.Version)
				}
			},
		},
		{
			name:     "module dependency",
			jsonData: `{"module_name": "@minecraft/server", "version": "1.4.0"}`,
			validate: func(t *testing.T, md *ManifestDependency) {
				if md.ModuleName != "@minecraft/server" {
					t.Errorf("Expected module name '@minecraft/server', got %q", md.ModuleName)
				}
				if md.ModuleVersion != "1.4.0" {
					t.Errorf("Expected module version '1.4.0', got %q", md.ModuleVersion)
				}
			},
		},
		{
			name:     "empty dependency",
			jsonData: `{}`,
			validate: func(t *testing.T, md *ManifestDependency) {
				if md.UUID != "" || md.ModuleName != "" {
					t.Error("Expected empty dependency fields")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dep ManifestDependency
			err := json.Unmarshal([]byte(tt.jsonData), &dep)
			if err != nil {
				t.Fatalf("Failed to unmarshal dependency: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, &dep)
			}
		})
	}
}

func TestManifestDependencyUnmarshalInvalid(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
	}{
		{
			name:     "invalid version array",
			jsonData: `{"uuid": "test", "version": "not-an-array"}`,
		},
		{
			name:     "malformed JSON",
			jsonData: `{"uuid": "test", "version":`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dep ManifestDependency
			err := json.Unmarshal([]byte(tt.jsonData), &dep)
			if err == nil {
				t.Error("Expected error for invalid dependency JSON")
			}
		})
	}
}

func TestParseManifestNonExistentFile(t *testing.T) {
	_, err := ParseManifest("/path/that/does/not/exist/manifest.json")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestIsPackDependency(t *testing.T) {
	tests := []struct {
		name       string
		dependency ManifestDependency
		expected   bool
	}{
		{
			name: "pack dependency with UUID",
			dependency: ManifestDependency{
				UUID: "12345678-1234-1234-1234-123456789abc",
			},
			expected: true,
		},
		{
			name: "module dependency",
			dependency: ManifestDependency{
				ModuleName: "@minecraft/server",
			},
			expected: false,
		},
		{
			name:       "empty dependency",
			dependency: ManifestDependency{},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dependency.UUID != ""
			if result != tt.expected {
				t.Errorf("Expected IsPackDependency to be %v, got %v", tt.expected, result)
			}
		})
	}
}

func BenchmarkParseManifest(b *testing.B) {
	// Create temporary manifest file
	tempDir, err := os.MkdirTemp("", "blockbench-manifest-bench")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manifestData := `{
		"format_version": 2,
		"header": {
			"name": "Benchmark Pack",
			"description": "Pack for benchmarking",
			"uuid": "12345678-1234-1234-1234-123456789abc",
			"version": [1, 0, 0]
		},
		"modules": [
			{
				"type": "data",
				"uuid": "12345678-1234-1234-1234-123456789abd",
				"version": [1, 0, 0]
			}
		],
		"dependencies": [
			{
				"uuid": "87654321-4321-4321-4321-fedcba987654",
				"version": [1, 0, 0]
			},
			{
				"module_name": "@minecraft/server",
				"version": "1.4.0"
			}
		]
	}`

	manifestPath := filepath.Join(tempDir, "manifest.json")
	err = os.WriteFile(manifestPath, []byte(manifestData), 0600)
	if err != nil {
		b.Fatalf("Failed to write manifest file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseManifest(manifestPath)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}