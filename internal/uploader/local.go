package uploader

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

type LocalUploadResult struct {
	Uploaded  bool
	TotalSize int64
}

type LocalUploadProgress struct {
	OnProgress func(completed, total int64)
}

// MoveFileLocal moves a single file to local storage path.
// Uses fileId as directory name (matching vdohide media path convention).
func MoveFileLocal(
	storagePath string,
	fileId string,
	filePath string,
	fileName string,
	progress *LocalUploadProgress,
) (*LocalUploadResult, error) {
	result := &LocalUploadResult{}
	destDir := filepath.Join(storagePath, fileId)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", destDir, err)
	}

	var totalSize int64 = 0
	if filePath != "" {
		info, err := os.Stat(filePath)
		if err == nil {
			totalSize = info.Size()
		}
	}

	dest := filepath.Join(destDir, fileName)
	// Ensure subdirectories exist (e.g. sprite/sprite-1.jpg)
	os.MkdirAll(filepath.Dir(dest), 0755)
	size, err := copyFile(filePath, dest)
	if err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	os.Remove(filePath)

	if progress != nil && progress.OnProgress != nil {
		progress.OnProgress(size, totalSize)
	}

	result.Uploaded = true
	result.TotalSize = size
	log.Printf("✅ Moved %s to %s", fileName, destDir)

	return result, nil
}

// MoveMultipleFilesLocal moves multiple files to local storage.
func MoveMultipleFilesLocal(
	storagePath string,
	fileId string,
	files map[string]string, // fileName -> localPath
	progress *LocalUploadProgress,
) (*LocalUploadResult, error) {
	result := &LocalUploadResult{}
	destDir := filepath.Join(storagePath, fileId)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", destDir, err)
	}

	var totalSize int64
	for _, localPath := range files {
		if info, err := os.Stat(localPath); err == nil {
			totalSize += info.Size()
		}
	}

	var completedBytes int64
	for fileName, localPath := range files {
		dest := filepath.Join(destDir, fileName)
		// Ensure subdirectories exist (e.g. sprite/sprite-1.jpg)
		os.MkdirAll(filepath.Dir(dest), 0755)
		size, err := copyFile(localPath, dest)
		if err != nil {
			return nil, fmt.Errorf("failed to copy %s: %w", fileName, err)
		}

		os.Remove(localPath)
		completedBytes += size
		result.TotalSize += size

		if progress != nil && progress.OnProgress != nil {
			progress.OnProgress(completedBytes, totalSize)
		}

		log.Printf("✅ Moved %s", fileName)
	}

	result.Uploaded = true
	log.Printf("✅ All files moved to %s (%.2f MB total)",
		destDir, float64(result.TotalSize)/1024/1024)

	return result, nil
}

func copyFile(src, dst string) (int64, error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer dstFile.Close()

	size, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return 0, err
	}

	if err := dstFile.Close(); err != nil {
		return size, fmt.Errorf("failed to close/flush: %w", err)
	}
	return size, nil
}
