package docker

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestValidateVolume(t *testing.T) {
	tmpDir := t.TempDir()
	compose := NewCompose(tmpDir)

	tests := []struct {
		name      string
		volume    string
		wantErr   bool
		errMsg    string
		skipOnWin bool // 某些测试在 Windows 上跳过
	}{
		{
			name:    "允许相对路径",
			volume:  "./src:/app",
			wantErr: false,
		},
		{
			name:    "允许当前目录",
			volume:  ".:/app",
			wantErr: false,
		},
		{
			name:    "允许命名卷",
			volume:  "mydata:/data",
			wantErr: false,
		},
		{
			name:    "允许子目录",
			volume:  "./subdir/data:/app/data",
			wantErr: false,
		},
		{
			name:      "禁止绝对路径 Unix",
			volume:    "/etc:/app",
			wantErr:   true,
			errMsg:    "禁止使用绝对路径挂载",
			skipOnWin: true, // Windows 上 /etc 不被视为绝对路径
		},
		{
			name:      "禁止敏感路径",
			volume:    "/etc/passwd:/app/passwd",
			wantErr:   true,
			errMsg:    "禁止使用绝对路径挂载",
			skipOnWin: true,
		},
		{
			name:    "禁止父目录逃逸",
			volume:  "../:/app",
			wantErr: true,
			errMsg:  "禁止挂载父目录",
		},
		{
			name:    "禁止复杂父目录逃逸",
			volume:  "../../etc:/app",
			wantErr: true,
			errMsg:  "禁止挂载父目录",
		},
		{
			name:    "禁止隐蔽的父目录逃逸",
			volume:  "./foo/../../etc:/app",
			wantErr: true,
			errMsg:  "禁止挂载父目录",
		},
		{
			name:    "禁止 home 目录",
			volume:  "~/:/app",
			wantErr: true,
			errMsg:  "禁止使用 home 目录路径",
		},
		{
			name:    "禁止 home 子目录",
			volume:  "~/Documents:/app",
			wantErr: true,
			errMsg:  "禁止使用 home 目录路径",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnWin && runtime.GOOS == "windows" {
				t.Skip("跳过: Windows 上路径处理不同")
			}

			err := compose.validateVolume(tt.volume)
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

func TestValidateVolumeWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("仅在 Windows 上运行")
	}

	tmpDir := t.TempDir()
	compose := NewCompose(tmpDir)

	tests := []struct {
		name    string
		volume  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "禁止 Windows 绝对路径",
			volume:  "C:\\Users:/app",
			wantErr: true,
			errMsg:  "禁止使用绝对路径挂载",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := compose.validateVolume(tt.volume)
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

func TestValidateComposeFile(t *testing.T) {
	tmpDir := t.TempDir()
	compose := NewCompose(tmpDir)

	t.Run("安全配置应该通过", func(t *testing.T) {
		safeYaml := `
services:
  app:
    build: .
    volumes:
      - .:/app
      - ./data:/data
    ports:
      - "8080:8080"
`
		err := os.WriteFile(filepath.Join(tmpDir, "docker-compose.yml"), []byte(safeYaml), 0644)
		if err != nil {
			t.Fatal(err)
		}

		if err := compose.ValidateComposeFile(); err != nil {
			t.Errorf("安全配置校验失败: %v", err)
		}
	})

	t.Run("父目录逃逸应该被拒绝", func(t *testing.T) {
		escapeYaml := `
services:
  app:
    image: alpine
    volumes:
      - ../secret:/app
`
		err := os.WriteFile(filepath.Join(tmpDir, "docker-compose.yml"), []byte(escapeYaml), 0644)
		if err != nil {
			t.Fatal(err)
		}

		if err := compose.ValidateComposeFile(); err == nil {
			t.Error("父目录逃逸应该被拒绝")
		}
	})

	t.Run("命名卷应该允许", func(t *testing.T) {
		namedVolumeYaml := `
services:
  db:
    image: postgres
    volumes:
      - pgdata:/var/lib/postgresql/data

volumes:
  pgdata:
`
		err := os.WriteFile(filepath.Join(tmpDir, "docker-compose.yml"), []byte(namedVolumeYaml), 0644)
		if err != nil {
			t.Fatal(err)
		}

		if err := compose.ValidateComposeFile(); err != nil {
			t.Errorf("命名卷配置应该通过: %v", err)
		}
	})

	t.Run("空服务应该被拒绝", func(t *testing.T) {
		emptyYaml := `
services:
`
		err := os.WriteFile(filepath.Join(tmpDir, "docker-compose.yml"), []byte(emptyYaml), 0644)
		if err != nil {
			t.Fatal(err)
		}

		if err := compose.ValidateComposeFile(); err == nil {
			t.Error("空服务配置应该被拒绝")
		}
	})

	t.Run("home 目录应该被拒绝", func(t *testing.T) {
		homeYaml := `
services:
  app:
    image: alpine
    volumes:
      - ~/data:/app
`
		err := os.WriteFile(filepath.Join(tmpDir, "docker-compose.yml"), []byte(homeYaml), 0644)
		if err != nil {
			t.Fatal(err)
		}

		if err := compose.ValidateComposeFile(); err == nil {
			t.Error("home 目录应该被拒绝")
		}
	})
}

func TestGetServices(t *testing.T) {
	tmpDir := t.TempDir()
	compose := NewCompose(tmpDir)

	yaml := `
services:
  web:
    image: nginx
  api:
    image: node
  db:
    image: postgres
`
	err := os.WriteFile(filepath.Join(tmpDir, "docker-compose.yml"), []byte(yaml), 0644)
	if err != nil {
		t.Fatal(err)
	}

	services, err := compose.GetServices()
	if err != nil {
		t.Fatalf("GetServices 失败: %v", err)
	}

	if len(services) != 3 {
		t.Errorf("期望 3 个服务，得到 %d", len(services))
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
