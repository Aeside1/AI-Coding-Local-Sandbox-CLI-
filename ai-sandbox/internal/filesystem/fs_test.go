package filesystem

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSafePath(t *testing.T) {
	// 创建临时目录作为工作目录
	tmpDir := t.TempDir()
	fs := New(tmpDir)

	tests := []struct {
		name      string
		path      string
		wantErr   bool
		errMsg    string
		skipOnWin bool
	}{
		{
			name:    "允许当前目录",
			path:    ".",
			wantErr: false,
		},
		{
			name:    "允许相对路径",
			path:    "src/main.go",
			wantErr: false,
		},
		{
			name:    "允许子目录",
			path:    "./subdir/file.txt",
			wantErr: false,
		},
		{
			name:    "禁止父目录逃逸",
			path:    "../",
			wantErr: true,
			errMsg:  "禁止访问父目录",
		},
		{
			name:    "禁止复杂父目录逃逸",
			path:    "../../etc/passwd",
			wantErr: true,
			errMsg:  "禁止访问父目录",
		},
		{
			name:    "禁止隐蔽的父目录逃逸",
			path:    "foo/../../etc",
			wantErr: true,
			errMsg:  "禁止访问父目录",
		},
		{
			name:      "禁止绝对路径 Unix",
			path:      "/etc/passwd",
			wantErr:   true,
			errMsg:    "禁止使用绝对路径",
			skipOnWin: true, // Windows 上 /etc 不是绝对路径
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnWin && runtime.GOOS == "windows" {
				t.Skip("跳过: Windows 上路径处理不同")
			}

			_, err := fs.SafePath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("期望错误但没有返回")
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("错误消息不匹配: got %v, want contains %v", err, tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("意外错误: %v", err)
			}
		})
	}
}

func TestSafePathWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("仅在 Windows 上运行")
	}

	tmpDir := t.TempDir()
	fs := New(tmpDir)

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "禁止 Windows 绝对路径",
			path:    "C:\\Windows\\System32",
			wantErr: true,
			errMsg:  "禁止使用绝对路径",
		},
		{
			name:    "禁止盘符路径",
			path:    "D:\\data",
			wantErr: true,
			errMsg:  "禁止使用绝对路径",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := fs.SafePath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("期望错误但没有返回")
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("错误消息不匹配: got %v, want contains %v", err, tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("意外错误: %v", err)
			}
		})
	}
}

func TestListFiles(t *testing.T) {
	// 创建临时目录结构
	tmpDir := t.TempDir()
	fs := New(tmpDir)

	// 创建测试文件结构
	os.MkdirAll(filepath.Join(tmpDir, "src"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "src", "app.go"), []byte("package src"), 0644)

	result, err := fs.ListFiles(".", 3)
	if err != nil {
		t.Fatalf("ListFiles 失败: %v", err)
	}

	if !contains(result, "main.go") {
		t.Error("结果应包含 main.go")
	}
	if !contains(result, "src") {
		t.Error("结果应包含 src 目录")
	}
}

func TestReadWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir)

	content := "Hello, World!\nLine 2\nLine 3"
	path := "test/file.txt"

	// 测试写入
	if err := fs.WriteFile(path, content); err != nil {
		t.Fatalf("WriteFile 失败: %v", err)
	}

	// 测试读取
	result, err := fs.ReadFile(path, 100)
	if err != nil {
		t.Fatalf("ReadFile 失败: %v", err)
	}

	if !contains(result, "Hello, World!") {
		t.Errorf("读取内容不正确: %s", result)
	}
}

func TestReadFileTruncation(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir)

	// 创建多行文件
	var content string
	for i := 0; i < 100; i++ {
		content += "Line content\n"
	}
	path := "large.txt"
	fs.WriteFile(path, content)

	// 限制读取 10 行
	result, err := fs.ReadFile(path, 10)
	if err != nil {
		t.Fatalf("ReadFile 失败: %v", err)
	}

	if !contains(result, "已截断") {
		t.Error("应该显示截断提示")
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir)

	// 创建文件
	fs.WriteFile("exists.txt", "content")

	if !fs.FileExists("exists.txt") {
		t.Error("文件应该存在")
	}

	if fs.FileExists("not-exists.txt") {
		t.Error("文件不应该存在")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsHelper(s, substr)
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
