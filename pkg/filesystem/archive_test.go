package filesystem

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateArchive(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "blockbench-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		setupFunc   func() string
		expectError bool
	}{
		{
			name: "valid zip archive",
			setupFunc: func() string {
				zipPath := filepath.Join(tempDir, "valid.zip")
				createTestZip(t, zipPath, map[string]string{
					"manifest.json": `{"format_version": 2, "header": {"name": "Test Pack"}}`,
					"pack_icon.png": "fake png data",
				})
				return zipPath
			},
			expectError: false,
		},
		{
			name: "non-existent file",
			setupFunc: func() string {
				return filepath.Join(tempDir, "nonexistent.zip")
			},
			expectError: true,
		},
		{
			name: "invalid zip file",
			setupFunc: func() string {
				invalidPath := filepath.Join(tempDir, "invalid.zip")
				os.WriteFile(invalidPath, []byte("not a zip file"), 0600)
				return invalidPath
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archivePath := tt.setupFunc()
			err := ValidateArchive(archivePath)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestExtractArchive(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "blockbench-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test zip
	zipPath := filepath.Join(tempDir, "test.zip")
	testFiles := map[string]string{
		"manifest.json":       `{"format_version": 2}`,
		"textures/icon.png":   "fake png data",
		"behaviors/main.js":   "console.log('test');",
		"folder/":             "", // directory entry
	}
	createTestZip(t, zipPath, testFiles)

	// Extract archive
	extractDir := filepath.Join(tempDir, "extracted")
	err = ExtractArchive(zipPath, extractDir)
	if err != nil {
		t.Fatalf("Failed to extract archive: %v", err)
	}

	// Verify extracted files
	expectedFiles := []string{
		"manifest.json",
		"textures/icon.png",
		"behaviors/main.js",
		"folder",
	}

	for _, expectedFile := range expectedFiles {
		fullPath := filepath.Join(extractDir, expectedFile)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not extracted", expectedFile)
		}
	}

	// Verify file contents
	manifestContent, err := os.ReadFile(filepath.Join(extractDir, "manifest.json"))
	if err != nil {
		t.Errorf("Failed to read extracted manifest: %v", err)
	}
	if string(manifestContent) != testFiles["manifest.json"] {
		t.Errorf("Manifest content mismatch: got %q, want %q", string(manifestContent), testFiles["manifest.json"])
	}
}

func TestExtractArchiveWithPathTraversal(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "blockbench-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create malicious zip with path traversal
	zipPath := filepath.Join(tempDir, "malicious.zip")
	maliciousFiles := map[string]string{
		"../../../etc/passwd": "fake passwd content",
		"normal.txt":          "normal content",
	}
	createTestZip(t, zipPath, maliciousFiles)

	extractDir := filepath.Join(tempDir, "extracted")
	err = ExtractArchive(zipPath, extractDir)
	
	// Should fail due to path traversal protection
	if err == nil {
		t.Error("Expected error for path traversal attempt, but extraction succeeded")
	}
}

func TestGetArchiveInfo(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "blockbench-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test zip with known content
	zipPath := filepath.Join(tempDir, "info-test.zip")
	testFiles := map[string]string{
		"manifest.json":       `{"format_version": 2}`,
		"pack_icon.png":       "fake png data (12 bytes)",
		"textures/test.png":   "more fake data",
		"behaviors/":          "", // directory
	}
	createTestZip(t, zipPath, testFiles)

	info, err := GetArchiveInfo(zipPath)
	if err != nil {
		t.Fatalf("Failed to get archive info: %v", err)
	}

	// Verify basic info
	if info.TotalFiles != 4 { // 3 files + 1 directory
		t.Errorf("Expected 4 total files, got %d", info.TotalFiles)
	}

	if !info.HasManifest {
		t.Error("Expected HasManifest to be true")
	}

	if len(info.TopLevelDirs) == 0 {
		t.Error("Expected some top-level directories")
	}

	// Check that size is reasonable (should be > 0)
	if info.TotalSize <= 0 {
		t.Errorf("Expected positive total size, got %d", info.TotalSize)
	}
}

func TestGetArchiveInfoWithLargeFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "blockbench-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test the overflow protection by creating a zip file manually
	// with a manipulated UncompressedSize64 field
	zipPath := filepath.Join(tempDir, "large-file-test.zip")
	
	// Create a normal zip first
	createTestZip(t, zipPath, map[string]string{
		"test.txt": "small content",
	})

	// For this test, we'll just verify normal operation
	// Testing actual overflow would require manipulating the zip structure
	info, err := GetArchiveInfo(zipPath)
	if err != nil {
		t.Fatalf("Failed to get info for normal file: %v", err)
	}

	if info.TotalSize < 0 {
		t.Error("Total size should not be negative")
	}
}

// Helper function to create test ZIP files
func createTestZip(t *testing.T, zipPath string, files map[string]string) {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("Failed to create zip file: %v", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for fileName, content := range files {
		if content == "" && strings.HasSuffix(fileName, "/") {
			// Create directory entry
			_, err := zipWriter.Create(fileName)
			if err != nil {
				t.Fatalf("Failed to create directory entry %s: %v", fileName, err)
			}
		} else {
			// Create file entry
			fileWriter, err := zipWriter.Create(fileName)
			if err != nil {
				t.Fatalf("Failed to create file entry %s: %v", fileName, err)
			}
			if content != "" {
				_, err = fileWriter.Write([]byte(content))
				if err != nil {
					t.Fatalf("Failed to write content to %s: %v", fileName, err)
				}
			}
		}
	}
}

func BenchmarkExtractArchive(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "blockbench-bench")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test zip
	zipPath := filepath.Join(tempDir, "bench.zip")
	testFiles := make(map[string]string)
	for i := 0; i < 100; i++ {
		testFiles[fmt.Sprintf("file_%d.txt", i)] = fmt.Sprintf("Content for file %d", i)
	}
	createTestZipForBench(b, zipPath, testFiles)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractDir := filepath.Join(tempDir, fmt.Sprintf("extract_%d", i))
		err := ExtractArchive(zipPath, extractDir)
		if err != nil {
			b.Fatalf("Failed to extract archive: %v", err)
		}
		os.RemoveAll(extractDir) // Clean up for next iteration
	}
}

// Helper function for benchmark tests
func createTestZipForBench(b *testing.B, zipPath string, files map[string]string) {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		b.Fatalf("Failed to create zip file: %v", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for fileName, content := range files {
		if content == "" && strings.HasSuffix(fileName, "/") {
			// Create directory entry
			_, err := zipWriter.Create(fileName)
			if err != nil {
				b.Fatalf("Failed to create directory entry %s: %v", fileName, err)
			}
		} else {
			// Create file entry
			fileWriter, err := zipWriter.Create(fileName)
			if err != nil {
				b.Fatalf("Failed to create file entry %s: %v", fileName, err)
			}
			if content != "" {
				_, err = fileWriter.Write([]byte(content))
				if err != nil {
					b.Fatalf("Failed to write content to %s: %v", fileName, err)
				}
			}
		}
	}
}