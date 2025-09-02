package filesystem

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewBackupManager(t *testing.T) {
	backupRoot := "/test/backup/root"
	bm := NewBackupManager(backupRoot)

	if bm.BackupRoot != backupRoot {
		t.Errorf("Expected BackupRoot to be %q, got %q", backupRoot, bm.BackupRoot)
	}

	if bm.metadata == nil {
		t.Error("Expected metadata slice to be initialized")
	}
}

func TestCreateBackup(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "blockbench-backup-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files to backup
	testFile1 := filepath.Join(tempDir, "test1.txt")
	testFile2 := filepath.Join(tempDir, "test2.txt")
	testContent1 := "test content 1"
	testContent2 := "test content 2"

	err = os.WriteFile(testFile1, []byte(testContent1), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file 1: %v", err)
	}

	err = os.WriteFile(testFile2, []byte(testContent2), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file 2: %v", err)
	}

	// Create backup manager
	backupRoot := filepath.Join(tempDir, "backups")
	bm := NewBackupManager(backupRoot)

	// Create backup
	filesToBackup := []string{testFile1, testFile2}
	metadata, err := bm.CreateBackup("install", "Test backup", filesToBackup)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Verify metadata
	if metadata.Operation != "install" {
		t.Errorf("Expected operation 'install', got %q", metadata.Operation)
	}

	if metadata.Description != "Test backup" {
		t.Errorf("Expected description 'Test backup', got %q", metadata.Description)
	}

	if len(metadata.Files) != 2 {
		t.Errorf("Expected 2 files in metadata, got %d", len(metadata.Files))
	}

	// Verify backup directory exists
	if _, err := os.Stat(metadata.BackupPath); os.IsNotExist(err) {
		t.Errorf("Backup directory does not exist: %s", metadata.BackupPath)
	}

	// Verify backed up files exist
	backupFile1 := filepath.Join(metadata.BackupPath, filepath.Base(testFile1))
	backupFile2 := filepath.Join(metadata.BackupPath, filepath.Base(testFile2))

	if content, err := os.ReadFile(backupFile1); err != nil {
		t.Errorf("Failed to read backed up file 1: %v", err)
	} else if string(content) != testContent1 {
		t.Errorf("Backup file 1 content mismatch: got %q, want %q", string(content), testContent1)
	}

	if content, err := os.ReadFile(backupFile2); err != nil {
		t.Errorf("Failed to read backed up file 2: %v", err)
	} else if string(content) != testContent2 {
		t.Errorf("Backup file 2 content mismatch: got %q, want %q", string(content), testContent2)
	}
}

func TestCreateBackupNonExistentFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "blockbench-backup-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	backupRoot := filepath.Join(tempDir, "backups")
	bm := NewBackupManager(backupRoot)

	// Try to backup non-existent file
	nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")
	metadata, err := bm.CreateBackup("test", "Test missing file", []string{nonExistentFile})
	if err != nil {
		t.Fatalf("Backup should handle missing files: %v", err)
	}

	// Verify marker file was created
	markerFile := filepath.Join(metadata.BackupPath, "nonexistent.txt.missing")
	if _, err := os.Stat(markerFile); os.IsNotExist(err) {
		t.Error("Expected marker file for missing original file")
	}
}

func TestRestoreBackup(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "blockbench-backup-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create original files
	testFile := filepath.Join(tempDir, "original.txt")
	originalContent := "original content"
	err = os.WriteFile(testFile, []byte(originalContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create original file: %v", err)
	}

	// Create backup
	backupRoot := filepath.Join(tempDir, "backups")
	bm := NewBackupManager(backupRoot)
	metadata, err := bm.CreateBackup("test", "Test restore", []string{testFile})
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Modify original file
	modifiedContent := "modified content"
	err = os.WriteFile(testFile, []byte(modifiedContent), 0600)
	if err != nil {
		t.Fatalf("Failed to modify original file: %v", err)
	}

	// Verify file was modified
	if content, err := os.ReadFile(testFile); err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	} else if string(content) != modifiedContent {
		t.Fatalf("File was not modified correctly")
	}

	// Restore backup
	err = bm.RestoreBackup(metadata.ID)
	if err != nil {
		t.Fatalf("Failed to restore backup: %v", err)
	}

	// Verify file was restored
	if content, err := os.ReadFile(testFile); err != nil {
		t.Errorf("Failed to read restored file: %v", err)
	} else if string(content) != originalContent {
		t.Errorf("File was not restored correctly: got %q, want %q", string(content), originalContent)
	}
}

func TestListBackups(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "blockbench-backup-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	backupRoot := filepath.Join(tempDir, "backups")
	bm := NewBackupManager(backupRoot)

	// Initially should be empty
	backups, err := bm.ListBackups()
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}
	if len(backups) != 0 {
		t.Errorf("Expected 0 backups initially, got %d", len(backups))
	}

	// Create test file and backup
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = bm.CreateBackup("install", "First backup", []string{testFile})
	if err != nil {
		t.Fatalf("Failed to create first backup: %v", err)
	}

	time.Sleep(1001 * time.Millisecond) // Ensure different Unix seconds
	_, err = bm.CreateBackup("uninstall", "Second backup", []string{testFile})
	if err != nil {
		t.Fatalf("Failed to create second backup: %v", err)
	}

	// List backups
	backups, err = bm.ListBackups()
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}

	if len(backups) != 2 {
		t.Errorf("Expected 2 backups, got %d", len(backups))
	}

	// Verify backup operations
	operations := make(map[string]bool)
	for _, backup := range backups {
		operations[backup.Operation] = true
	}

	if !operations["install"] {
		t.Error("Expected to find 'install' operation in backups")
	}
	if !operations["uninstall"] {
		t.Error("Expected to find 'uninstall' operation in backups")
	}
}

func TestDeleteBackup(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "blockbench-backup-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test file and backup
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	backupRoot := filepath.Join(tempDir, "backups")
	bm := NewBackupManager(backupRoot)
	metadata, err := bm.CreateBackup("test", "Test delete", []string{testFile})
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Verify backup exists
	if _, err := os.Stat(metadata.BackupPath); os.IsNotExist(err) {
		t.Fatalf("Backup directory should exist before deletion")
	}

	// Delete backup
	err = bm.DeleteBackup(metadata.ID)
	if err != nil {
		t.Fatalf("Failed to delete backup: %v", err)
	}

	// Verify backup is gone
	if _, err := os.Stat(metadata.BackupPath); !os.IsNotExist(err) {
		t.Error("Backup directory should not exist after deletion")
	}

	// Verify metadata file is gone
	metadataFile := filepath.Join(backupRoot, metadata.ID+".json")
	if _, err := os.Stat(metadataFile); !os.IsNotExist(err) {
		t.Error("Metadata file should not exist after deletion")
	}
}

func TestBackupMetadataJSON(t *testing.T) {
	// Test metadata serialization/deserialization
	metadata := BackupMetadata{
		ID:          "test-backup-123",
		Timestamp:   time.Now().Truncate(time.Second), // Truncate for comparison
		Operation:   "install",
		AddonName:   "Test Addon",
		AddonUUID:   "12345678-1234-1234-1234-123456789abc",
		ServerPath:  "/server/path",
		BackupPath:  "/backup/path",
		Files:       []string{"/file1.txt", "/file2.txt"},
		Description: "Test backup description",
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal metadata: %v", err)
	}

	// Unmarshal from JSON
	var restored BackupMetadata
	err = json.Unmarshal(jsonData, &restored)
	if err != nil {
		t.Fatalf("Failed to unmarshal metadata: %v", err)
	}

	// Compare fields
	if restored.ID != metadata.ID {
		t.Errorf("ID mismatch: got %q, want %q", restored.ID, metadata.ID)
	}
	if !restored.Timestamp.Equal(metadata.Timestamp) {
		t.Errorf("Timestamp mismatch: got %v, want %v", restored.Timestamp, metadata.Timestamp)
	}
	if restored.Operation != metadata.Operation {
		t.Errorf("Operation mismatch: got %q, want %q", restored.Operation, metadata.Operation)
	}
	if len(restored.Files) != len(metadata.Files) {
		t.Errorf("Files length mismatch: got %d, want %d", len(restored.Files), len(metadata.Files))
	}
}

func TestGenerateBackupID(t *testing.T) {
	// Test that generateBackupID produces unique IDs
	id1 := generateBackupID()
	time.Sleep(1001 * time.Millisecond) // Ensure different Unix seconds
	id2 := generateBackupID()

	if id1 == id2 {
		t.Error("generateBackupID should produce unique IDs")
	}

	// Test that IDs follow expected format
	expectedPrefix := "backup_"
	if !contains(id1, expectedPrefix) {
		t.Errorf("Expected ID to contain %q, got %q", expectedPrefix, id1)
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
