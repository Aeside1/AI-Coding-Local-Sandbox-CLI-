package docker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Compose docker-compose 操作封装
type Compose struct {
	workDir     string
	composeFile string
	timeout     time.Duration
}

// NewCompose 创建 Compose 实例
func NewCompose(workDir string) *Compose {
	return &Compose{
		workDir:     workDir,
		composeFile: filepath.Join(workDir, "docker-compose.yml"),
		timeout:     5 * time.Minute,
	}
}

// SetTimeout 设置命令超时时间
func (c *Compose) SetTimeout(d time.Duration) {
	c.timeout = d
}

// GetComposeFile 获取 compose 文件路径
func (c *Compose) GetComposeFile() string {
	return c.composeFile
}

// ComposeFileExists 检查 compose 文件是否存在
func (c *Compose) ComposeFileExists() bool {
	_, err := os.Stat(c.composeFile)
	return err == nil
}

// Up 启动环境
func (c *Compose) Up(background, recreate bool) (string, error) {
	// 先进行安全校验
	if err := c.ValidateComposeFile(); err != nil {
		return "", fmt.Errorf("安全校验失败: %w", err)
	}

	args := []string{"-f", c.composeFile, "up", "--build"}

	if background {
		args = append(args, "-d")
	}
	if recreate {
		args = append(args, "--force-recreate")
	}

	return c.execute(args...)
}

// Down 停止环境
func (c *Compose) Down(cleanVolumes bool) (string, error) {
	args := []string{"-f", c.composeFile, "down"}

	if cleanVolumes {
		args = append(args, "-v")
	}

	return c.execute(args...)
}

// Exec 在容器中执行命令
func (c *Compose) Exec(service, command string) (string, error) {
	// 使用 sh -c 来执行复杂命令
	args := []string{
		"-f", c.composeFile,
		"exec", "-T", service,
		"sh", "-c", command,
	}

	return c.execute(args...)
}

// Logs 获取日志
func (c *Compose) Logs(service string, tail int) (string, error) {
	args := []string{
		"-f", c.composeFile,
		"logs",
		"--tail", fmt.Sprintf("%d", tail),
		"--no-color",
	}

	if service != "" {
		args = append(args, service)
	}

	return c.execute(args...)
}

// Ps 列出容器状态
func (c *Compose) Ps() (string, error) {
	args := []string{
		"-f", c.composeFile,
		"ps",
	}
	return c.execute(args...)
}

// Restart 重启服务
func (c *Compose) Restart(service string) (string, error) {
	args := []string{"-f", c.composeFile, "restart"}
	if service != "" {
		args = append(args, service)
	}
	return c.execute(args...)
}

// execute 执行 docker-compose 命令
func (c *Compose) execute(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	// 尝试使用 docker compose（新版）或 docker-compose（旧版）
	var cmd *exec.Cmd
	var output string
	var err error

	// 首先尝试 docker compose（Docker Compose V2）
	cmd = exec.CommandContext(ctx, "docker", append([]string{"compose"}, args...)...)
	cmd.Dir = c.workDir
	output, err = c.runCommand(cmd)

	// 如果失败，尝试 docker-compose（Docker Compose V1）
	if err != nil && strings.Contains(err.Error(), "unknown command") {
		cmd = exec.CommandContext(ctx, "docker-compose", args...)
		cmd.Dir = c.workDir
		output, err = c.runCommand(cmd)
	}

	return output, err
}

func (c *Compose) runCommand(cmd *exec.Cmd) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	if err != nil {
		// 检查是否是超时
		if cmd.ProcessState == nil {
			return output, fmt.Errorf("命令执行超时")
		}
		return output, fmt.Errorf("命令执行失败 (exit %d): %s", cmd.ProcessState.ExitCode(), strings.TrimSpace(output))
	}

	return strings.TrimSpace(output), nil
}

// CheckDockerAvailable 检查 Docker 是否可用
func CheckDockerAvailable() error {
	cmd := exec.Command("docker", "info")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Docker 不可用: %s", strings.TrimSpace(stderr.String()))
	}
	return nil
}
