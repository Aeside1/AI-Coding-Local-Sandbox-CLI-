package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// getArgs 从请求中获取参数 map
func getArgs(req mcp.CallToolRequest) map[string]interface{} {
	if req.Params.Arguments == nil {
		return make(map[string]interface{})
	}
	if args, ok := req.Params.Arguments.(map[string]interface{}); ok {
		return args
	}
	return make(map[string]interface{})
}

// ========== 文件系统处理器 ==========

func (s *Server) handleFsListFiles(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)

	// 获取参数
	path := "."
	if v, ok := args["path"].(string); ok && v != "" {
		path = v
	}

	maxDepth := 3
	if v, ok := args["max_depth"].(float64); ok {
		maxDepth = int(v)
	}

	// 执行操作
	result, err := s.fs.ListFiles(path, maxDepth)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("列出文件失败: %v", err)), nil
	}

	return mcp.NewToolResultText(result), nil
}

func (s *Server) handleFsReadFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)

	// 获取必填参数
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return mcp.NewToolResultError("错误: path 参数必填"), nil
	}

	// 获取可选参数
	maxLines := 1000
	if v, ok := args["max_lines"].(float64); ok {
		maxLines = int(v)
	}

	// 执行操作
	content, err := s.fs.ReadFile(path, maxLines)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("读取文件失败: %v", err)), nil
	}

	return mcp.NewToolResultText(content), nil
}

func (s *Server) handleFsWriteFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)

	// 获取必填参数
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return mcp.NewToolResultError("错误: path 参数必填"), nil
	}

	content, ok := args["content"].(string)
	if !ok {
		return mcp.NewToolResultError("错误: content 参数必填"), nil
	}

	// 执行操作
	if err := s.fs.WriteFile(path, content); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("写入文件失败: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("✓ 文件已写入: %s (%d 字节)", path, len(content))), nil
}

// ========== 容器编排处理器 ==========

func (s *Server) handleComposeUp(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 检查 compose 文件是否存在
	if !s.compose.ComposeFileExists() {
		return mcp.NewToolResultError("错误: docker-compose.yml 不存在。请先使用 fs_write_file 创建配置文件。"), nil
	}

	args := getArgs(req)

	// 获取参数
	background := true
	if v, ok := args["background"].(bool); ok {
		background = v
	}

	recreate := false
	if v, ok := args["recreate"].(bool); ok {
		recreate = v
	}

	// 执行操作（ValidateComposeFile 在 Up 内部调用）
	output, err := s.compose.Up(background, recreate)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("启动失败: %v\n\n输出:\n%s", err, output)), nil
	}

	result := "✓ Docker Compose 环境启动成功\n"
	if output != "" {
		result += "\n输出:\n" + output
	}

	return mcp.NewToolResultText(result), nil
}

func (s *Server) handleComposeDown(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)

	// 获取参数
	cleanVolumes := false
	if v, ok := args["clean_volumes"].(bool); ok {
		cleanVolumes = v
	}

	// 执行操作
	output, err := s.compose.Down(cleanVolumes)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("停止失败: %v\n\n输出:\n%s", err, output)), nil
	}

	result := "✓ Docker Compose 环境已停止"
	if cleanVolumes {
		result += "（已清理数据卷）"
	}
	if output != "" {
		result += "\n\n输出:\n" + output
	}

	return mcp.NewToolResultText(result), nil
}

func (s *Server) handleComposeExec(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)

	// 获取必填参数
	service, ok := args["service"].(string)
	if !ok || service == "" {
		return mcp.NewToolResultError("错误: service 参数必填"), nil
	}

	command, ok := args["command"].(string)
	if !ok || command == "" {
		return mcp.NewToolResultError("错误: command 参数必填"), nil
	}

	// 执行操作
	output, err := s.compose.Exec(service, command)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("执行命令失败: %v\n\n输出:\n%s", err, output)), nil
	}

	if output == "" {
		output = "(命令执行完成，无输出)"
	}

	return mcp.NewToolResultText(output), nil
}

func (s *Server) handleComposeLogs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)

	// 获取可选参数
	service := ""
	if v, ok := args["service"].(string); ok {
		service = v
	}

	tail := 100
	if v, ok := args["tail"].(float64); ok {
		tail = int(v)
	}

	// 执行操作
	output, err := s.compose.Logs(service, tail)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("获取日志失败: %v", err)), nil
	}

	if output == "" {
		output = "(暂无日志)"
	}

	return mcp.NewToolResultText(output), nil
}

func (s *Server) handleComposePs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	output, err := s.compose.Ps()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("获取容器状态失败: %v", err)), nil
	}

	if output == "" {
		output = "(没有运行中的容器)"
	}

	return mcp.NewToolResultText(output), nil
}
