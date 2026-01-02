package filesystem

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// SafePath 验证并返回安全的绝对路径
func (fs *FileSystem) SafePath(relativePath string) (string, error) {
	// 处理空路径
	if relativePath == "" {
		relativePath = "."
	}

	// 规范化路径分隔符（Windows 兼容）
	relativePath = filepath.FromSlash(relativePath)

	// 禁止绝对路径
	if filepath.IsAbs(relativePath) {
		return "", fmt.Errorf("禁止使用绝对路径: %s", relativePath)
	}

	// Windows 特殊检查：禁止盘符路径
	if runtime.GOOS == "windows" && len(relativePath) >= 2 && relativePath[1] == ':' {
		return "", fmt.Errorf("禁止使用绝对路径: %s", relativePath)
	}

	// 规范化路径
	cleanPath := filepath.Clean(relativePath)

	// 检查路径逃逸
	if strings.HasPrefix(cleanPath, "..") {
		return "", fmt.Errorf("禁止访问父目录: %s", relativePath)
	}

	// 检查路径中是否包含 ..（更严格的检查）
	parts := strings.Split(cleanPath, string(filepath.Separator))
	for _, part := range parts {
		if part == ".." {
			return "", fmt.Errorf("禁止访问父目录: %s", relativePath)
		}
	}

	// 构建完整路径
	fullPath := filepath.Join(fs.workDir, cleanPath)

	// 获取绝对路径进行最终验证
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("路径解析失败: %w", err)
	}

	absWorkDir, err := filepath.Abs(fs.workDir)
	if err != nil {
		return "", fmt.Errorf("工作目录解析失败: %w", err)
	}

	// 确保路径在工作目录内
	// 需要添加分隔符以避免前缀匹配问题（如 /app 和 /app2）
	if !strings.HasPrefix(absPath+string(filepath.Separator), absWorkDir+string(filepath.Separator)) &&
		absPath != absWorkDir {
		return "", fmt.Errorf("路径越界: %s", relativePath)
	}

	return absPath, nil
}

// IsPathSafe 检查路径是否安全（不抛出错误）
func (fs *FileSystem) IsPathSafe(relativePath string) bool {
	_, err := fs.SafePath(relativePath)
	return err == nil
}
