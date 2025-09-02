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
	if err := os.MkdirAll(destDir, 0750); err != nil {
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
	if err := os.MkdirAll(filepath.Dir(destPath), 0750); err != nil {
		return err
	}

	// Open file in archive
	srcFile, err := file.Open()
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create destination file
	// #nosec G304 - destPath is validated by caller and within controlled temp directory
	destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.FileInfo().Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy file contents with size limit to prevent decompression bombs
	const maxFileSize = 100 * 1024 * 1024 // 100MB limit per file
	limitedReader := io.LimitReader(srcFile, maxFileSize)
	written, err := io.Copy(destFile, limitedReader)
	if err != nil {
		return err
	}

	// Check if we hit the limit (potential decompression bomb)
	if written >= maxFileSize {
		return fmt.Errorf("file too large after decompression: %s (exceeded 100MB limit)", file.Name)
	}

	return nil
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
	TotalFiles     int
	TotalSize      int64
	HasManifest    bool
	ManifestFiles  []string
	TopLevelDirs   []string
	HasMcpackFiles bool
	McpackFiles    []string
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
		McpackFiles:   make([]string, 0),
	}

	topDirs := make(map[string]bool)

	for _, file := range reader.File {
		info.TotalFiles++
		// Safely handle uint64 to int64 conversion and addition to prevent overflow
		const maxInt64 = 9223372036854775807
		if file.UncompressedSize64 > maxInt64 {
			return nil, fmt.Errorf("file size too large: %d bytes", file.UncompressedSize64)
		}

		fileSize := int64(file.UncompressedSize64) // #nosec G115 - checked above
		// Check for potential overflow in addition
		if info.TotalSize > maxInt64-fileSize {
			return nil, fmt.Errorf("total archive size too large, would cause overflow")
		}
		info.TotalSize += fileSize

		// Check for manifest files
		if strings.HasSuffix(strings.ToLower(file.Name), "manifest.json") {
			info.HasManifest = true
			info.ManifestFiles = append(info.ManifestFiles, file.Name)
		}

		// Check for .mcpack files
		if strings.HasSuffix(strings.ToLower(file.Name), ".mcpack") {
			info.HasMcpackFiles = true
			info.McpackFiles = append(info.McpackFiles, file.Name)
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
