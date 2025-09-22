package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/firebase/genkit/go/ai"
)

// CopyFile copies source file content to target file, creates the file if target doesn't exist
// srcPath: source file path
// dstPath: target file path
func CopyFile(srcPath, dstPath string) error {
	// Open source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", srcPath, err)
	}
	defer srcFile.Close()

	// Get source file info
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get source file info %s: %w", srcPath, err)
	}

	// Ensure target directory exists
	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory %s: %w", dstDir, err)
	}

	// Create or overwrite target file
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create target file %s: %w", dstPath, err)
	}
	defer dstFile.Close()

	// Copy file content
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Set target file permissions same as source file
	if err := os.Chmod(dstPath, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}

func Tools2ToolRef(tools []ai.Tool) (toolRef []ai.ToolRef) {
	for _, tool := range tools {
		toolRef = append(toolRef, tool)
	}
	return toolRef
}
