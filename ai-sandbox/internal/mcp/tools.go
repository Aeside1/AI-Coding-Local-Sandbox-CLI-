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
			mcp.WithDescription("递归列出工作目录下的文件结构。用于了解项目布局和查找文件。"),
			mcp.WithString("path",
				mcp.Description("相对路径，默认为根目录 '.'"),
			),
			mcp.WithNumber("max_depth",
				mcp.Description("最大递归深度，默认为 3"),
			),
		),
		s.handleFsListFiles,
	)

	// fs_read_file
	s.mcpServer.AddTool(
		mcp.NewTool("fs_read_file",
			mcp.WithDescription("读取指定文件的内容。支持文本文件和代码文件。"),
			mcp.WithString("path",
				mcp.Required(),
				mcp.Description("文件相对路径"),
			),
			mcp.WithNumber("max_lines",
				mcp.Description("最大读取行数，默认为 1000"),
			),
		),
		s.handleFsReadFile,
	)

	// fs_write_file
	s.mcpServer.AddTool(
		mcp.NewTool("fs_write_file",
			mcp.WithDescription("写入或覆盖文件内容。可用于创建代码文件、配置文件、docker-compose.yml 等。"),
			mcp.WithString("path",
				mcp.Required(),
				mcp.Description("文件相对路径"),
			),
			mcp.WithString("content",
				mcp.Required(),
				mcp.Description("要写入的文件内容"),
			),
		),
		s.handleFsWriteFile,
	)

	// ========== 容器编排能力 ==========

	// sandbox_compose_up
	s.mcpServer.AddTool(
		mcp.NewTool("sandbox_compose_up",
			mcp.WithDescription("启动 Docker Compose 环境。会自动构建镜像并启动所有服务。执行前会进行安全校验，确保没有危险的卷挂载。"),
			mcp.WithBoolean("background",
				mcp.Description("是否后台运行，默认为 true"),
			),
			mcp.WithBoolean("recreate",
				mcp.Description("是否强制重建容器"),
			),
		),
		s.handleComposeUp,
	)

	// sandbox_compose_down
	s.mcpServer.AddTool(
		mcp.NewTool("sandbox_compose_down",
			mcp.WithDescription("停止并清理 Docker Compose 环境。"),
			mcp.WithBoolean("clean_volumes",
				mcp.Description("是否同时删除数据卷，用于完全重置环境"),
			),
		),
		s.handleComposeDown,
	)

	// sandbox_compose_exec
	s.mcpServer.AddTool(
		mcp.NewTool("sandbox_compose_exec",
			mcp.WithDescription("在运行中的容器内执行命令。用于运行测试、安装依赖、调试等操作。"),
			mcp.WithString("service",
				mcp.Required(),
				mcp.Description("服务名称（docker-compose.yml 中定义的 service 名）"),
			),
			mcp.WithString("command",
				mcp.Required(),
				mcp.Description("要执行的命令，如 'pytest tests/' 或 'npm test'"),
			),
		),
		s.handleComposeExec,
	)

	// sandbox_compose_logs
	s.mcpServer.AddTool(
		mcp.NewTool("sandbox_compose_logs",
			mcp.WithDescription("获取容器日志。用于调试启动失败或运行时错误。"),
			mcp.WithString("service",
				mcp.Description("服务名称，为空则获取所有服务的日志"),
			),
			mcp.WithNumber("tail",
				mcp.Description("显示最后 N 行日志，默认为 100"),
			),
		),
		s.handleComposeLogs,
	)

	// sandbox_compose_ps
	s.mcpServer.AddTool(
		mcp.NewTool("sandbox_compose_ps",
			mcp.WithDescription("列出当前运行的容器状态。"),
		),
		s.handleComposePs,
	)
}
