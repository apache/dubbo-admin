package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyFile 复制源文件内容到目标文件，如果目标文件不存在，则会创建该文件
// srcPath: 源文件路径
// dstPath: 目标文件路径
func CopyFile(srcPath, dstPath string) error {
	// 打开源文件
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", srcPath, err)
	}
	defer srcFile.Close()

	// 获取源文件信息
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get source file info %s: %w", srcPath, err)
	}

	// 确保目标目录存在
	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory %s: %w", dstDir, err)
	}

	// 创建或覆盖目标文件
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create target file %s: %w", dstPath, err)
	}
	defer dstFile.Close()

	// 复制文件内容
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// 设置目标文件权限与源文件相同
	if err := os.Chmod(dstPath, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}
