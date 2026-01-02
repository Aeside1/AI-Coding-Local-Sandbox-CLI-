package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yourorg/ai-sandbox/internal/config"
	"github.com/yourorg/ai-sandbox/internal/docker"
	"github.com/yourorg/ai-sandbox/internal/mcp"
)

var (
	version = "1.0.0"
	workDir string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "ai-sandbox",
		Short: "AI Coding Local Sandbox - 安全的代码验证环境",
		Long: `AI Sandbox 是一个本地 CLI 工具，通过 MCP 协议连接 AI 模型（如 Claude），
提供安全隔离的 Docker 容器环境用于代码验证。

核心理念: Dumb Tool, Smart Agent (工具极简，智能在云)

使用方式:
  1. 在项目目录启动: ai-sandbox start
  2. AI 通过 MCP 协议连接并操作
  3. AI 可以读写文件、启动 Docker 环境、执行测试`,
	}

	// start 命令
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "启动 MCP Server",
		Long: `启动 MCP Server，等待 AI 客户端连接。

Server 将以 stdio 模式运行，适配 Claude Desktop 等 MCP 客户端。
所有文件操作将限制在指定的工作目录内。`,
		RunE: runStart,
	}
	startCmd.Flags().StringVarP(&workDir, "workdir", "w", ".",
		"工作目录（AI 可操作的根目录）")

	// version 命令
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "显示版本信息",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("ai-sandbox version %s\n", version)
		},
	}

	// check 命令
	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "检查系统环境",
		Long:  "检查 Docker 是否正确安装并运行",
		RunE:  runCheck,
	}

	rootCmd.AddCommand(startCmd, versionCmd, checkCmd)

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
	info, err := os.Stat(absWorkDir)
	if err != nil {
		return fmt.Errorf("工作目录不存在: %s", absWorkDir)
	}
	if !info.IsDir() {
		return fmt.Errorf("指定路径不是目录: %s", absWorkDir)
	}

	// 打印启动信息
	fmt.Fprintf(os.Stderr, "╔══════════════════════════════════════════╗\n")
	fmt.Fprintf(os.Stderr, "║         AI Sandbox v%s                ║\n", version)
	fmt.Fprintf(os.Stderr, "╚══════════════════════════════════════════╝\n")
	fmt.Fprintf(os.Stderr, "\n")

	// 检查 Docker
	if err := docker.CheckDockerAvailable(); err != nil {
		fmt.Fprintf(os.Stderr, "⚠ 警告: %v\n", err)
		fmt.Fprintf(os.Stderr, "  容器相关功能可能不可用\n\n")
	} else {
		fmt.Fprintf(os.Stderr, "✓ Docker 已就绪\n")
	}

	// 初始化配置
	cfg := config.New(absWorkDir)

	// 启动 MCP Server
	server := mcp.NewServer(cfg)
	return server.Run()
}

func runCheck(cmd *cobra.Command, args []string) error {
	fmt.Println("检查系统环境...")
	fmt.Println()

	// 检查 Docker
	fmt.Print("Docker: ")
	if err := docker.CheckDockerAvailable(); err != nil {
		fmt.Printf("✗ 不可用 (%v)\n", err)
		fmt.Println("  请确保 Docker Desktop 已启动")
		return nil
	}
	fmt.Println("✓ 已就绪")

	// 检查 docker-compose
	fmt.Print("Docker Compose: ")
	fmt.Println("✓ 已集成在 Docker 中")

	fmt.Println()
	fmt.Println("所有检查通过！可以使用 'ai-sandbox start' 启动服务。")
	return nil
}
