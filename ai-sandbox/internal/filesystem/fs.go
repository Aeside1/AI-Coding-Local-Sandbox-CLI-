package filesystem

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileSystem 文件系统操作
type FileSystem struct {
	workDir string
}

// New 创建文件系统实例
func New(workDir string) *FileSystem {
	return &FileSystem{workDir: workDir}
}

// GetWorkDir 获取工作目录
func (fs *FileSystem) GetWorkDir() string {
	return fs.workDir
}

// ListFiles 递归列出文件
func (fs *FileSystem) ListFiles(relativePath string, maxDepth int) (string, error) {
	// 安全校验
	fullPath, err := fs.SafePath(relativePath)
	if err != nil {
		return "", err
	}

	// 检查路径是否存在
	info, err := os.Stat(fullPath)
	if err != nil {
		return "", fmt.Errorf("路径不存在: %s", relativePath)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("不是目录: %s", relativePath)
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("目录: %s\n", relativePath))
	builder.WriteString("```\n")
	err = fs.walkDir(fullPath, "", 0, maxDepth, &builder)
	if err != nil {
		return "", err
	}
	builder.WriteString("```\n")

	return builder.String(), nil
}

func (fs *FileSystem) walkDir(path, prefix string, depth, maxDepth int, builder *strings.Builder) error {
	if depth > maxDepth {
		return nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	// 过滤并排序条目
	var filtered []os.DirEntry
	for _, entry := range entries {
		name := entry.Name()
		// 跳过隐藏文件和常见忽略目录
		if strings.HasPrefix(name, ".") ||
			name == "node_modules" ||
			name == "__pycache__" ||
			name == "vendor" ||
			name == ".git" {
			continue
		}
		filtered = append(filtered, entry)
	}

	for i, entry := range filtered {
		isLast := i == len(filtered)-1
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		builder.WriteString(prefix + connector + entry.Name())
		if entry.IsDir() {
			builder.WriteString("/")
		}
		builder.WriteString("\n")

		if entry.IsDir() && depth < maxDepth {
			newPrefix := prefix + "│   "
			if isLast {
				newPrefix = prefix + "    "
			}
			if err := fs.walkDir(filepath.Join(path, entry.Name()), newPrefix, depth+1, maxDepth, builder); err != nil {
				return err
			}
		}
	}

	return nil
}

// ReadFile 读取文件内容
func (fs *FileSystem) ReadFile(relativePath string, maxLines int) (string, error) {
	fullPath, err := fs.SafePath(relativePath)
	if err != nil {
		return "", err
	}

	// 检查是否是文件
	info, err := os.Stat(fullPath)
	if err != nil {
		return "", fmt.Errorf("无法访问文件: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("不能读取目录，请使用 fs_list_files")
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return "", fmt.Errorf("无法打开文件: %w", err)
	}
	defer file.Close()

	var builder strings.Builder
	scanner := bufio.NewScanner(file)
	// 增加 buffer 大小以处理长行
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lineCount := 0
	for scanner.Scan() && lineCount < maxLines {
		builder.WriteString(scanner.Text())
		builder.WriteString("\n")
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("读取文件错误: %w", err)
	}

	if lineCount >= maxLines {
		builder.WriteString(fmt.Sprintf("\n... (已截断，超过 %d 行)", maxLines))
	}

	return builder.String(), nil
}

// WriteFile 写入文件
func (fs *FileSystem) WriteFile(relativePath, content string) error {
	fullPath, err := fs.SafePath(relativePath)
	if err != nil {
		return err
	}

	// 确保父目录存在
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	return os.WriteFile(fullPath, []byte(content), 0644)
}

// FileExists 检查文件是否存在
func (fs *FileSystem) FileExists(relativePath string) bool {
	fullPath, err := fs.SafePath(relativePath)
	if err != nil {
		return false
	}
	_, err = os.Stat(fullPath)
	return err == nil
}

// DeleteFile 删除文件
func (fs *FileSystem) DeleteFile(relativePath string) error {
	fullPath, err := fs.SafePath(relativePath)
	if err != nil {
		return err
	}
	return os.Remove(fullPath)
}
