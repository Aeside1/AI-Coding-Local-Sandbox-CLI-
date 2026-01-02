package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComposeConfig docker-compose.yml 结构
type ComposeConfig struct {
	Version  string                   `yaml:"version,omitempty"`
	Services map[string]ServiceConfig `yaml:"services"`
	Volumes  map[string]interface{}   `yaml:"volumes,omitempty"`
	Networks map[string]interface{}   `yaml:"networks,omitempty"`
}

// ServiceConfig 服务配置
type ServiceConfig struct {
	Image       string            `yaml:"image,omitempty"`
	Build       interface{}       `yaml:"build,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty"`
	Ports       []string          `yaml:"ports,omitempty"`
	Environment interface{}       `yaml:"environment,omitempty"`
	Command     interface{}       `yaml:"command,omitempty"`
	WorkingDir  string            `yaml:"working_dir,omitempty"`
	DependsOn   interface{}       `yaml:"depends_on,omitempty"`
	Networks    interface{}       `yaml:"networks,omitempty"`
	Restart     string            `yaml:"restart,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
}

// ValidateComposeFile 验证 docker-compose.yml 安全性
func (c *Compose) ValidateComposeFile() error {
	// 检查文件是否存在
	if _, err := os.Stat(c.composeFile); os.IsNotExist(err) {
		return fmt.Errorf("docker-compose.yml 不存在: %s", c.composeFile)
	}

	// 读取文件
	data, err := os.ReadFile(c.composeFile)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	// 解析 YAML
	var config ComposeConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("YAML 解析失败: %w", err)
	}

	// 检查是否有服务定义
	if len(config.Services) == 0 {
		return fmt.Errorf("docker-compose.yml 中没有定义任何服务")
	}

	// 校验每个服务的 volumes
	for serviceName, service := range config.Services {
		for _, volume := range service.Volumes {
			if err := c.validateVolume(volume); err != nil {
				return fmt.Errorf("服务 '%s' 的 volume 校验失败: %w", serviceName, err)
			}
		}
	}

	return nil
}

// validateVolume 验证单个 volume 挂载
func (c *Compose) validateVolume(volume string) error {
	// 解析 volume 格式: host:container[:options]
	// 也可能是命名卷: volume_name:/container/path
	parts := strings.Split(volume, ":")

	// 处理 Windows 路径（如 C:\path:container）
	if runtime.GOOS == "windows" && len(parts) >= 3 && len(parts[0]) == 1 {
		// 这是 Windows 盘符路径，如 C:\path:container
		hostPath := parts[0] + ":" + parts[1]
		return fmt.Errorf("禁止使用绝对路径挂载: %s", hostPath)
	}

	if len(parts) < 2 {
		// 可能是只有容器路径的匿名卷，允许通过
		return nil
	}

	hostPath := parts[0]

	// 检查是否是命名卷（不包含路径分隔符且不以 . 开头的简单名称）
	if !strings.Contains(hostPath, "/") &&
		!strings.Contains(hostPath, "\\") &&
		!strings.HasPrefix(hostPath, ".") &&
		!strings.HasPrefix(hostPath, "~") {
		// 这是命名卷，允许通过
		return nil
	}

	// 检查 ~ 开头的路径（home 目录）
	if strings.HasPrefix(hostPath, "~") {
		return fmt.Errorf("禁止使用 home 目录路径: %s", hostPath)
	}

	// 规则1: 禁止绝对路径
	if filepath.IsAbs(hostPath) {
		return fmt.Errorf("禁止使用绝对路径挂载: %s", hostPath)
	}

	// Unix 绝对路径检查
	if strings.HasPrefix(hostPath, "/") {
		return fmt.Errorf("禁止使用绝对路径挂载: %s", hostPath)
	}

	// 规范化路径
	cleanPath := filepath.Clean(hostPath)

	// 规则2: 禁止父目录逃逸
	if strings.HasPrefix(cleanPath, "..") {
		return fmt.Errorf("禁止挂载父目录: %s", hostPath)
	}

	// 检查路径中是否包含 ..
	// 使用 / 和 \ 分割以处理跨平台情况
	normalizedPath := strings.ReplaceAll(cleanPath, "\\", "/")
	pathParts := strings.Split(normalizedPath, "/")
	for _, part := range pathParts {
		if part == ".." {
			return fmt.Errorf("禁止挂载父目录: %s", hostPath)
		}
	}

	// 规则3: 验证路径在工作目录内
	fullPath := filepath.Join(c.workDir, cleanPath)
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return fmt.Errorf("路径解析失败: %w", err)
	}

	absWorkDir, err := filepath.Abs(c.workDir)
	if err != nil {
		return fmt.Errorf("工作目录解析失败: %w", err)
	}

	// 确保路径在工作目录内
	if !strings.HasPrefix(absPath+string(filepath.Separator), absWorkDir+string(filepath.Separator)) &&
		absPath != absWorkDir {
		return fmt.Errorf("路径越界: %s", hostPath)
	}

	return nil
}

// GetServices 获取所有服务名称
func (c *Compose) GetServices() ([]string, error) {
	data, err := os.ReadFile(c.composeFile)
	if err != nil {
		return nil, err
	}

	var config ComposeConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	var services []string
	for name := range config.Services {
		services = append(services, name)
	}
	return services, nil
}
