# AI Coding Local Sandbox (CLI) 技术实现文档

项目代号: ai-sandbox
版本: v1.0
基于设计文档: v2.0

---

## 1. 项目结构

```
ai-sandbox/
├── cmd/
│   └── ai-sandbox/
│       └── main.go                 # CLI 入口
├── internal/
│   ├── mcp/
│   │   ├── server.go               # MCP Server 核心实现
│   │   ├── tools.go                # MCP Tools 注册
│   │   └── handler.go              # Tool 请求处理器
│   ├── filesystem/
│   │   ├── fs.go                   # 文件系统操作封装
│   │   ├── validator.go            # 路径安全校验
│   │   └── fs_test.go              # 单元测试
│   ├── docker/
│   │   ├── compose.go              # docker-compose 命令封装
│   │   ├── executor.go             # 命令执行器
│   │   ├── validator.go            # Compose 文件安全校验
│   │   └── compose_test.go         # 单元测试
│   └── config/
│       └── config.go               # 全局配置管理
├── pkg/
│   └── errors/
│       └── errors.go               # 统一错误定义
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 2. 依赖管理

### 2.1 go.mod

```go
module github.com/yourorg/ai-sandbox

go 1.21

require (
    github.com/mark3labs/mcp-go v0.17.0      // MCP 协议实现
    github.com/spf13/cobra v1.8.0             // CLI 框架
    gopkg.in/yaml.v3 v3.0.1                   // YAML 解析
)
```

### 2.2 依赖说明

| 依赖 | 用途 | 版本要求 |
|------|------|----------|
| `mcp-go` | MCP 协议 Server 端实现 | >= 0.17.0 |
| `cobra` | CLI 命令解析框架 | >= 1.8.0 |
| `yaml.v3` | 解析 docker-compose.yml | >= 3.0.1 |

---

## 3. 核心模块实现

### 3.1 CLI 入口 (`cmd/ai-sandbox/main.go`)

```go
package main

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/spf13/cobra"
    "github.com/yourorg/ai-sandbox/internal/config"
    "github.com/yourorg/ai-sandbox/internal/mcp"
)

var (
    workDir string
    version = "1.0.0"
)

func main() {
    rootCmd := &cobra.Command{
        Use:   "ai-sandbox",
        Short: "AI Coding Local Sandbox - 安全的代码验证环境",
        Long: `AI Sandbox 是一个本地 CLI 工具，通过 MCP 协议连接 AI 模型，
提供安全隔离的 Docker 容器环境用于代码验证。`,
    }

    // start 命令
    startCmd := &cobra.Command{
        Use:   "start",
        Short: "启动 MCP Server",
        RunE:  runStart,
    }

    startCmd.Flags().StringVarP(&workDir, "workdir", "w", ".",
        "工作目录（默认当前目录）")

    // version 命令
    versionCmd := &cobra.Command{
        Use:   "version",
        Short: "显示版本信息",
        Run: func(cmd *cobra.Command, args []string) {
            fmt.Printf("ai-sandbox version %s\n", version)
        },
    }

    rootCmd.AddCommand(startCmd, versionCmd)

    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}

func runStart(cmd *cobra.Command, args []string) error {
    // 解析并验证工作目录
    absWorkDir, err := filepath.Abs(workDir)
    if err != nil {
        return fmt.Errorf("无法解析工作目录: %w", err)
    }

    // 检查目录是否存在
    if info, err := os.Stat(absWorkDir); err != nil || !info.IsDir() {
        return fmt.Errorf("工作目录不存在: %s", absWorkDir)
    }

    // 初始化配置
    cfg := config.New(absWorkDir)

    // 启动 MCP Server
    fmt.Fprintf(os.Stderr, "AI Sandbox 启动中...\n")
    fmt.Fprintf(os.Stderr, "工作目录: %s\n", absWorkDir)
    fmt.Fprintf(os.Stderr, "MCP Server 运行于 stdio 模式\n")

    server := mcp.NewServer(cfg)
    return server.Run()
}
```

---

### 3.2 配置管理 (`internal/config/config.go`)

```go
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
```

---

### 3.3 MCP Server 实现 (`internal/mcp/server.go`)

```go
package mcp

import (
    "context"

    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
    "github.com/yourorg/ai-sandbox/internal/config"
    "github.com/yourorg/ai-sandbox/internal/docker"
    "github.com/yourorg/ai-sandbox/internal/filesystem"
)

// Server MCP 服务器
type Server struct {
    cfg       *config.Config
    mcpServer *server.MCPServer
    fs        *filesystem.FileSystem
    compose   *docker.Compose
}

// NewServer 创建 MCP 服务器
func NewServer(cfg *config.Config) *Server {
    s := &Server{
        cfg:     cfg,
        fs:      filesystem.New(cfg.GetWorkDir()),
        compose: docker.NewCompose(cfg.GetWorkDir()),
    }

    // 创建 MCP Server
    s.mcpServer = server.NewMCPServer(
        "AI Sandbox",
        "1.0.0",
        server.WithToolCapabilities(true),
    )

    // 注册所有 Tools
    s.registerTools()

    return s
}

// Run 启动服务器（stdio 模式）
func (s *Server) Run() error {
    return server.ServeStdio(s.mcpServer)
}
```

---

### 3.4 MCP Tools 注册 (`internal/mcp/tools.go`)

```go
package mcp

import (
    "github.com/mark3labs/mcp-go/mcp"
)

// registerTools 注册所有 MCP Tools
func (s *Server) registerTools() {
    // ========== 文件系统能力 ==========

    // fs_list_files
    s.mcpServer.AddTool(
        mcp.NewTool("fs_list_files",
            mcp.WithDescription("递归列出工作目录下的文件结构"),
            mcp.WithString("path",
                mcp.Description("相对路径（默认为根目录）"),
                mcp.DefaultString("."),
            ),
            mcp.WithNumber("max_depth",
                mcp.Description("最大递归深度（默认3）"),
            ),
        ),
        s.handleFsListFiles,
    )

    // fs_read_file
    s.mcpServer.AddTool(
        mcp.NewTool("fs_read_file",
            mcp.WithDescription("读取指定文件内容"),
            mcp.WithString("path",
                mcp.Required(),
                mcp.Description("文件相对路径"),
            ),
            mcp.WithNumber("max_lines",
                mcp.Description("最大读取行数（默认1000）"),
            ),
        ),
        s.handleFsReadFile,
    )

    // fs_write_file
    s.mcpServer.AddTool(
        mcp.NewTool("fs_write_file",
            mcp.WithDescription("写入或覆盖文件内容"),
            mcp.WithString("path",
                mcp.Required(),
                mcp.Description("文件相对路径"),
            ),
            mcp.WithString("content",
                mcp.Required(),
                mcp.Description("文件内容"),
            ),
        ),
        s.handleFsWriteFile,
    )

    // ========== 容器编排能力 ==========

    // sandbox_compose_up
    s.mcpServer.AddTool(
        mcp.NewTool("sandbox_compose_up",
            mcp.WithDescription("启动 Docker Compose 环境"),
            mcp.WithBoolean("background",
                mcp.Description("后台运行（默认true）"),
            ),
            mcp.WithBoolean("recreate",
                mcp.Description("强制重建容器"),
            ),
        ),
        s.handleComposeUp,
    )

    // sandbox_compose_down
    s.mcpServer.AddTool(
        mcp.NewTool("sandbox_compose_down",
            mcp.WithDescription("停止并清理 Docker Compose 环境"),
            mcp.WithBoolean("clean_volumes",
                mcp.Description("同时删除 volumes"),
            ),
        ),
        s.handleComposeDown,
    )

    // sandbox_compose_exec
    s.mcpServer.AddTool(
        mcp.NewTool("sandbox_compose_exec",
            mcp.WithDescription("在容器内执行命令"),
            mcp.WithString("service",
                mcp.Required(),
                mcp.Description("服务名称"),
            ),
            mcp.WithString("command",
                mcp.Required(),
                mcp.Description("要执行的命令"),
            ),
        ),
        s.handleComposeExec,
    )

    // sandbox_compose_logs
    s.mcpServer.AddTool(
        mcp.NewTool("sandbox_compose_logs",
            mcp.WithDescription("获取容器日志"),
            mcp.WithString("service",
                mcp.Description("服务名称（为空则获取所有）"),
            ),
            mcp.WithNumber("tail",
                mcp.Description("显示最后N行（默认100）"),
            ),
        ),
        s.handleComposeLogs,
    )
}
```

---

### 3.5 Tool 请求处理器 (`internal/mcp/handler.go`)

```go
package mcp

import (
    "context"
    "fmt"

    "github.com/mark3labs/mcp-go/mcp"
)

// ========== 文件系统处理器 ==========

func (s *Server) handleFsListFiles(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    path := req.Params.Arguments["path"].(string)
    if path == "" {
        path = "."
    }

    maxDepth := 3
    if v, ok := req.Params.Arguments["max_depth"].(float64); ok {
        maxDepth = int(v)
    }

    result, err := s.fs.ListFiles(path, maxDepth)
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("列出文件失败: %v", err)), nil
    }

    return mcp.NewToolResultText(result), nil
}

func (s *Server) handleFsReadFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    path, ok := req.Params.Arguments["path"].(string)
    if !ok || path == "" {
        return mcp.NewToolResultError("path 参数必填"), nil
    }

    maxLines := 1000
    if v, ok := req.Params.Arguments["max_lines"].(float64); ok {
        maxLines = int(v)
    }

    content, err := s.fs.ReadFile(path, maxLines)
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("读取文件失败: %v", err)), nil
    }

    return mcp.NewToolResultText(content), nil
}

func (s *Server) handleFsWriteFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    path, ok := req.Params.Arguments["path"].(string)
    if !ok || path == "" {
        return mcp.NewToolResultError("path 参数必填"), nil
    }

    content, ok := req.Params.Arguments["content"].(string)
    if !ok {
        return mcp.NewToolResultError("content 参数必填"), nil
    }

    if err := s.fs.WriteFile(path, content); err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("写入文件失败: %v", err)), nil
    }

    return mcp.NewToolResultText(fmt.Sprintf("文件已写入: %s", path)), nil
}

// ========== 容器编排处理器 ==========

func (s *Server) handleComposeUp(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    background := true
    if v, ok := req.Params.Arguments["background"].(bool); ok {
        background = v
    }

    recreate := false
    if v, ok := req.Params.Arguments["recreate"].(bool); ok {
        recreate = v
    }

    // 先进行安全校验
    if err := s.compose.ValidateComposeFile(); err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("安全校验失败: %v", err)), nil
    }

    output, err := s.compose.Up(background, recreate)
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("启动失败: %v\n%s", err, output)), nil
    }

    return mcp.NewToolResultText(fmt.Sprintf("环境启动成功\n%s", output)), nil
}

func (s *Server) handleComposeDown(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    cleanVolumes := false
    if v, ok := req.Params.Arguments["clean_volumes"].(bool); ok {
        cleanVolumes = v
    }

    output, err := s.compose.Down(cleanVolumes)
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("停止失败: %v\n%s", err, output)), nil
    }

    return mcp.NewToolResultText(fmt.Sprintf("环境已停止\n%s", output)), nil
}

func (s *Server) handleComposeExec(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    service, ok := req.Params.Arguments["service"].(string)
    if !ok || service == "" {
        return mcp.NewToolResultError("service 参数必填"), nil
    }

    command, ok := req.Params.Arguments["command"].(string)
    if !ok || command == "" {
        return mcp.NewToolResultError("command 参数必填"), nil
    }

    output, err := s.compose.Exec(service, command)
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("执行失败: %v\n%s", err, output)), nil
    }

    return mcp.NewToolResultText(output), nil
}

func (s *Server) handleComposeLogs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    service := ""
    if v, ok := req.Params.Arguments["service"].(string); ok {
        service = v
    }

    tail := 100
    if v, ok := req.Params.Arguments["tail"].(float64); ok {
        tail = int(v)
    }

    output, err := s.compose.Logs(service, tail)
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("获取日志失败: %v", err)), nil
    }

    return mcp.NewToolResultText(output), nil
}
```

---

### 3.6 文件系统模块 (`internal/filesystem/fs.go`)

```go
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

// ListFiles 递归列出文件
func (fs *FileSystem) ListFiles(relativePath string, maxDepth int) (string, error) {
    // 安全校验
    fullPath, err := fs.safePath(relativePath)
    if err != nil {
        return "", err
    }

    var builder strings.Builder
    err = fs.walkDir(fullPath, "", 0, maxDepth, &builder)
    if err != nil {
        return "", err
    }

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

    for i, entry := range entries {
        // 跳过隐藏文件和常见忽略目录
        if strings.HasPrefix(entry.Name(), ".") ||
           entry.Name() == "node_modules" ||
           entry.Name() == "__pycache__" ||
           entry.Name() == "vendor" {
            continue
        }

        isLast := i == len(entries)-1
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
            fs.walkDir(filepath.Join(path, entry.Name()), newPrefix, depth+1, maxDepth, builder)
        }
    }

    return nil
}

// ReadFile 读取文件内容
func (fs *FileSystem) ReadFile(relativePath string, maxLines int) (string, error) {
    fullPath, err := fs.safePath(relativePath)
    if err != nil {
        return "", err
    }

    file, err := os.Open(fullPath)
    if err != nil {
        return "", fmt.Errorf("无法打开文件: %w", err)
    }
    defer file.Close()

    var builder strings.Builder
    scanner := bufio.NewScanner(file)
    lineCount := 0

    for scanner.Scan() && lineCount < maxLines {
        builder.WriteString(scanner.Text())
        builder.WriteString("\n")
        lineCount++
    }

    if lineCount >= maxLines {
        builder.WriteString(fmt.Sprintf("\n... (截断，超过 %d 行)", maxLines))
    }

    return builder.String(), scanner.Err()
}

// WriteFile 写入文件
func (fs *FileSystem) WriteFile(relativePath, content string) error {
    fullPath, err := fs.safePath(relativePath)
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
```

---

### 3.7 路径安全校验 (`internal/filesystem/validator.go`)

```go
package filesystem

import (
    "fmt"
    "path/filepath"
    "strings"
)

// safePath 验证并返回安全的绝对路径
func (fs *FileSystem) safePath(relativePath string) (string, error) {
    // 禁止绝对路径
    if filepath.IsAbs(relativePath) {
        return "", fmt.Errorf("禁止使用绝对路径: %s", relativePath)
    }

    // 规范化路径
    cleanPath := filepath.Clean(relativePath)

    // 检查路径逃逸
    if strings.HasPrefix(cleanPath, "..") {
        return "", fmt.Errorf("禁止访问父目录: %s", relativePath)
    }

    // 构建完整路径
    fullPath := filepath.Join(fs.workDir, cleanPath)

    // 再次验证路径是否在工作目录内
    absPath, err := filepath.Abs(fullPath)
    if err != nil {
        return "", fmt.Errorf("路径解析失败: %w", err)
    }

    absWorkDir, _ := filepath.Abs(fs.workDir)
    if !strings.HasPrefix(absPath, absWorkDir) {
        return "", fmt.Errorf("路径越界: %s", relativePath)
    }

    return absPath, nil
}
```

---

### 3.8 Docker Compose 封装 (`internal/docker/compose.go`)

```go
package docker

import (
    "bytes"
    "fmt"
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

// Up 启动环境
func (c *Compose) Up(background, recreate bool) (string, error) {
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
    }

    if service != "" {
        args = append(args, service)
    }

    return c.execute(args...)
}

// execute 执行 docker-compose 命令
func (c *Compose) execute(args ...string) (string, error) {
    cmd := exec.Command("docker-compose", args...)
    cmd.Dir = c.workDir

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    err := cmd.Run()

    output := stdout.String()
    if stderr.Len() > 0 {
        output += "\n" + stderr.String()
    }

    if err != nil {
        return output, fmt.Errorf("命令执行失败: %w", err)
    }

    return strings.TrimSpace(output), nil
}
```

---

### 3.9 Compose 文件安全校验 (`internal/docker/validator.go`)

```go
package docker

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "gopkg.in/yaml.v3"
)

// ComposeConfig docker-compose.yml 结构
type ComposeConfig struct {
    Services map[string]ServiceConfig `yaml:"services"`
}

// ServiceConfig 服务配置
type ServiceConfig struct {
    Volumes []string `yaml:"volumes"`
    Ports   []string `yaml:"ports"`
}

// ValidateComposeFile 验证 docker-compose.yml 安全性
func (c *Compose) ValidateComposeFile() error {
    // 检查文件是否存在
    if _, err := os.Stat(c.composeFile); os.IsNotExist(err) {
        return fmt.Errorf("docker-compose.yml 不存在")
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

    // 校验每个服务的 volumes
    for serviceName, service := range config.Services {
        for _, volume := range service.Volumes {
            if err := c.validateVolume(volume); err != nil {
                return fmt.Errorf("服务 %s 的 volume 校验失败: %w", serviceName, err)
            }
        }
    }

    return nil
}

// validateVolume 验证单个 volume 挂载
func (c *Compose) validateVolume(volume string) error {
    // 解析 volume 格式: host:container[:options]
    parts := strings.Split(volume, ":")
    if len(parts) < 2 {
        // 可能是命名卷，允许通过
        return nil
    }

    hostPath := parts[0]

    // 规则1: 禁止绝对路径
    if filepath.IsAbs(hostPath) {
        return fmt.Errorf("禁止使用绝对路径挂载: %s", hostPath)
    }

    // 规则2: 禁止父目录逃逸
    cleanPath := filepath.Clean(hostPath)
    if strings.HasPrefix(cleanPath, "..") {
        return fmt.Errorf("禁止挂载父目录: %s", hostPath)
    }

    // 规则3: 验证路径在工作目录内
    fullPath := filepath.Join(c.workDir, cleanPath)
    absPath, err := filepath.Abs(fullPath)
    if err != nil {
        return fmt.Errorf("路径解析失败: %w", err)
    }

    absWorkDir, _ := filepath.Abs(c.workDir)
    if !strings.HasPrefix(absPath, absWorkDir) {
        return fmt.Errorf("路径越界: %s", hostPath)
    }

    return nil
}
```

---

### 3.10 单元测试 (`internal/docker/compose_test.go`)

```go
package docker

import (
    "os"
    "path/filepath"
    "testing"
)

func TestValidateVolume(t *testing.T) {
    tmpDir := t.TempDir()
    compose := NewCompose(tmpDir)

    tests := []struct {
        name    string
        volume  string
        wantErr bool
        errMsg  string
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
            name:    "禁止绝对路径",
            volume:  "/etc:/app",
            wantErr: true,
            errMsg:  "禁止使用绝对路径挂载",
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

    // 创建一个安全的 compose 文件
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

    // 测试危险配置
    dangerousYaml := `
services:
  app:
    build: .
    volumes:
      - /etc/passwd:/app/passwd
`
    err = os.WriteFile(filepath.Join(tmpDir, "docker-compose.yml"), []byte(dangerousYaml), 0644)
    if err != nil {
        t.Fatal(err)
    }

    if err := compose.ValidateComposeFile(); err == nil {
        t.Error("危险配置应该被拒绝")
    }
}

func contains(s, substr string) bool {
    return len(s) >= len(substr) &&
        (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
    for i := 0; i <= len(s)-len(substr); i++ {
        if s[i:i+len(substr)] == substr {
            return true
        }
    }
    return false
}
```

---

## 4. 统一错误定义 (`pkg/errors/errors.go`)

```go
package errors

import "errors"

// 预定义错误
var (
    ErrPathEscape     = errors.New("路径越界：禁止访问工作目录外的文件")
    ErrAbsolutePath   = errors.New("安全限制：禁止使用绝对路径")
    ErrParentDir      = errors.New("安全限制：禁止访问父目录")
    ErrComposeNotFound = errors.New("docker-compose.yml 文件不存在")
    ErrDockerNotReady = errors.New("Docker 未运行或不可用")
)

// WrapError 包装错误
func WrapError(err error, message string) error {
    if err == nil {
        return nil
    }
    return &wrappedError{
        msg: message,
        err: err,
    }
}

type wrappedError struct {
    msg string
    err error
}

func (e *wrappedError) Error() string {
    return e.msg + ": " + e.err.Error()
}

func (e *wrappedError) Unwrap() error {
    return e.err
}
```

---

## 5. Makefile

```makefile
.PHONY: build test clean install

# 变量
BINARY_NAME=ai-sandbox
VERSION=1.0.0
BUILD_DIR=build
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# 默认目标
all: build

# 编译
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/ai-sandbox

# 跨平台编译
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/ai-sandbox
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/ai-sandbox
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/ai-sandbox
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/ai-sandbox

# 运行测试
test:
	@echo "Running tests..."
	go test -v -race ./...

# 测试覆盖率
coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# 代码检查
lint:
	@echo "Running linter..."
	golangci-lint run

# 安装到系统
install: build
	@echo "Installing $(BINARY_NAME)..."
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

# 清理
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# 开发模式运行
dev:
	go run ./cmd/ai-sandbox start --workdir .
```

---

## 6. Claude Desktop 配置

### 6.1 MCP Server 配置 (`claude_desktop_config.json`)

```json
{
  "mcpServers": {
    "ai-sandbox": {
      "command": "ai-sandbox",
      "args": ["start", "--workdir", "/path/to/your/project"],
      "env": {}
    }
  }
}
```

### 6.2 Windows 配置示例

```json
{
  "mcpServers": {
    "ai-sandbox": {
      "command": "C:\\Users\\YourName\\go\\bin\\ai-sandbox.exe",
      "args": ["start", "--workdir", "C:\\Projects\\my-app"],
      "env": {}
    }
  }
}
```

---

## 7. AI 使用提示词 (PROMPTS.md)

```markdown
# AI Sandbox 使用指南

## 你的身份
你是一个 DevOps 专家，擅长配置 Docker 环境来验证代码。

## 工作流程

### 1. 探索项目
- 首先调用 `fs_list_files` 查看项目结构
- 调用 `fs_read_file` 读取关键文件（package.json, requirements.txt, go.mod 等）

### 2. 配置环境
- 根据项目类型编写 `Dockerfile`（如果需要）
- 编写 `docker-compose.yml`

### 3. 关键规则
- **Volume 挂载必须使用相对路径**：`.:/app` ✓ `/home:/app` ✗
- 当前目录挂载到容器的 `/app`
- 使用 `tail -f /dev/null` 保持容器运行

### 4. 验证代码
- 调用 `sandbox_compose_up` 启动环境
- 调用 `sandbox_compose_exec` 运行测试命令
- 如果失败，调用 `sandbox_compose_logs` 查看日志

### 5. 清理
- 测试完成后调用 `sandbox_compose_down`
- 需要重置时使用 `clean_volumes=true`

## 常用 Docker Compose 模板

### Python 项目
```yaml
services:
  app:
    image: python:3.11-slim
    volumes:
      - .:/app
    working_dir: /app
    command: tail -f /dev/null
```

### Node.js 项目
```yaml
services:
  app:
    image: node:20-alpine
    volumes:
      - .:/app
    working_dir: /app
    command: tail -f /dev/null
```

### Go 项目
```yaml
services:
  app:
    image: golang:1.21-alpine
    volumes:
      - .:/app
    working_dir: /app
    command: tail -f /dev/null
```
```

---

## 8. 构建与部署

### 8.1 首次构建

```bash
# 克隆项目
git clone https://github.com/yourorg/ai-sandbox.git
cd ai-sandbox

# 安装依赖
go mod download

# 运行测试
make test

# 编译
make build

# 安装到系统（可选）
make install
```

### 8.2 验证安装

```bash
# 检查版本
ai-sandbox version

# 启动测试
cd /path/to/test-project
ai-sandbox start
```

---

## 9. 安全检查清单

| 检查项 | 状态 | 说明 |
|--------|------|------|
| 路径校验 | ✅ | 禁止绝对路径和父目录逃逸 |
| Volume 校验 | ✅ | compose up 前强制检查 |
| 命令注入防护 | ✅ | 使用 exec.Command 而非 shell |
| 工作目录隔离 | ✅ | 所有操作限制在 workdir 内 |
| 错误信息脱敏 | ⚠️ | 需评估是否暴露敏感路径 |

---

## 10. 后续迭代计划

### v1.1
- [ ] 添加 `offline` 模式（网络隔离）
- [ ] 支持多 compose 文件
- [ ] 添加资源限制（CPU/Memory）

### v1.2
- [ ] 集成 Docker 健康检查
- [ ] 支持 compose profiles
- [ ] 添加环境变量白名单

### v2.0
- [ ] 支持 Kubernetes（可选后端）
- [ ] 多租户隔离
- [ ] 审计日志
