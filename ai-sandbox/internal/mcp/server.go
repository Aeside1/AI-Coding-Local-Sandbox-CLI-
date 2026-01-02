package mcp

import (
	"fmt"
	"os"

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
	fmt.Fprintf(os.Stderr, "MCP Server 已启动，等待连接...\n")
	fmt.Fprintf(os.Stderr, "工作目录: %s\n", s.cfg.GetWorkDir())
	fmt.Fprintf(os.Stderr, "可用 Tools:\n")
	fmt.Fprintf(os.Stderr, "  - fs_list_files: 列出文件结构\n")
	fmt.Fprintf(os.Stderr, "  - fs_read_file: 读取文件内容\n")
	fmt.Fprintf(os.Stderr, "  - fs_write_file: 写入文件\n")
	fmt.Fprintf(os.Stderr, "  - sandbox_compose_up: 启动 Docker 环境\n")
	fmt.Fprintf(os.Stderr, "  - sandbox_compose_down: 停止 Docker 环境\n")
	fmt.Fprintf(os.Stderr, "  - sandbox_compose_exec: 在容器中执行命令\n")
	fmt.Fprintf(os.Stderr, "  - sandbox_compose_logs: 获取容器日志\n")
	fmt.Fprintf(os.Stderr, "---\n")

	return server.ServeStdio(s.mcpServer)
}

// GetFileSystem 获取文件系统实例（用于测试）
func (s *Server) GetFileSystem() *filesystem.FileSystem {
	return s.fs
}

// GetCompose 获取 Compose 实例（用于测试）
func (s *Server) GetCompose() *docker.Compose {
	return s.compose
}
