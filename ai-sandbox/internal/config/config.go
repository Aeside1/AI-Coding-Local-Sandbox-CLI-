package config

import (
	"path/filepath"
	"sync"
)

// Config 全局配置
type Config struct {
	WorkDir     string // 工作目录绝对路径
	ComposeFile string // docker-compose.yml 路径
	mu          sync.RWMutex
}

// New 创建新配置
func New(workDir string) *Config {
	return &Config{
		WorkDir:     workDir,
		ComposeFile: filepath.Join(workDir, "docker-compose.yml"),
	}
}

// GetWorkDir 获取工作目录（线程安全）
func (c *Config) GetWorkDir() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.WorkDir
}

// GetComposeFile 获取 Compose 文件路径
func (c *Config) GetComposeFile() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ComposeFile
}
