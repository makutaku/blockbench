package filesystem

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractArchive extracts a ZIP archive to a destination directory
func ExtractArchive(archivePath, destDir string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer reader.Close()

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Extract files
	for _, file := range reader.File {
		if err := extractFile(file, destDir); err != nil {
			return fmt.Errorf("failed to extract file %s: %w", file.Name, err)
		}
	}

	return nil
}

// extractFile extracts a single file from a ZIP archive
func extractFile(file *zip.File, destDir string) error {
	// Clean the file path to prevent directory traversal
	cleanPath := filepath.Clean(file.Name)
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("invalid file path: %s", file.Name)
	}

	destPath := filepath.Join(destDir, cleanPath)

	// Create directory for file if needed
	if file.FileInfo().IsDir() {
		return os.MkdirAll(destPath, file.FileInfo().Mode())
	}

	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	// Open file in archive
	srcFile, err := file.Open()
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create destination file
	destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.FileInfo().Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy file contents
	_, err = io.Copy(destFile, srcFile)
	return err
}

// ValidateArchive performs basic validation on a ZIP archive
func ValidateArchive(archivePath string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer reader.Close()

	if len(reader.File) == 0 {
		return fmt.Errorf("archive is empty")
	}

	// Check for suspicious files
	for _, file := range reader.File {
		// Check for directory traversal attempts
		if strings.Contains(file.Name, "..") {
			return fmt.Errorf("archive contains suspicious file path: %s", file.Name)
		}

		// Check for absolute paths
		if filepath.IsAbs(file.Name) {
			return fmt.Errorf("archive contains absolute file path: %s", file.Name)
		}
	}

	return nil
}

// GetArchiveInfo returns basic information about a ZIP archive
type ArchiveInfo struct {
	TotalFiles    int
	TotalSize     int64
	HasManifest   bool
	ManifestFiles []string
	TopLevelDirs  []string
}

// GetArchiveInfo analyzes a ZIP archive and returns information about it
func GetArchiveInfo(archivePath string) (*ArchiveInfo, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open archive: %w", err)
	}
	defer reader.Close()

	info := &ArchiveInfo{
		ManifestFiles: make([]string, 0),
		TopLevelDirs:  make([]string, 0),
	}

	topDirs := make(map[string]bool)

	for _, file := range reader.File {
		info.TotalFiles++
		info.TotalSize += int64(file.UncompressedSize64)

		// Check for manifest files
		if strings.HasSuffix(strings.ToLower(file.Name), "manifest.json") {
			info.HasManifest = true
			info.ManifestFiles = append(info.ManifestFiles, file.Name)
		}

		// Track top-level directories
		pathParts := strings.Split(file.Name, "/")
		if len(pathParts) > 1 {
			topDir := pathParts[0]
			if !topDirs[topDir] {
				topDirs[topDir] = true
				info.TopLevelDirs = append(info.TopLevelDirs, topDir)
			}
		}
	}

	return info, nil
}
