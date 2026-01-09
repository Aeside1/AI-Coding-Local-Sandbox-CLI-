# AI Sandbox

AI Coding Local Sandbox - 安全的代码验证环境

## 简介

AI Sandbox 是一个本地 CLI 工具，通过 MCP (Model Context Protocol) 协议连接 AI 模型（如 Claude），提供安全隔离的 Docker 容器环境用于代码验证。

**核心理念**: Dumb Tool, Smart Agent (工具极简，智能在云)

## 特性

- 🔒 **安全沙箱**: 所有文件操作限制在工作目录内
- 🐳 **Docker 隔离**: 代码在容器中运行，与主机隔离
- 🤖 **MCP 协议**: 与 Claude Desktop 等 AI 客户端无缝集成
- 📁 **文件操作**: AI 可读写项目文件
- 🔧 **容器编排**: 支持 docker-compose 环境管理

## 安装

### 从源码构建

```bash
# 克隆项目
git clone https://github.com/yourorg/ai-sandbox.git
cd ai-sandbox

# 安装依赖
go mod tidy

# 编译
make build

# 或者直接安装到 GOPATH
make install
```

### 预编译二进制

从 [Releases](https://github.com/yourorg/ai-sandbox/releases) 下载对应平台的二进制文件。

## 使用

### 1. 检查环境

```bash
ai-sandbox check
```

### 2. 启动服务

```bash
cd /path/to/your/project
ai-sandbox start
```

### 3. 配置 Claude Desktop

编辑 Claude Desktop 配置文件：

**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "ai-sandbox": {
      "command": "ai-sandbox",
      "args": ["start"]
    }
  }
}
```

注意：不需要在配置中指定工作目录，AI Sandbox 会默认使用启动时的当前目录作为工作目录。当在项目目录中启动 Claude Desktop 时，AI 将能够访问该项目的文件。

## MCP Tools

AI 可以使用以下工具：

### 文件系统

| 工具 | 说明 |
|------|------|
| `fs_list_files` | 列出目录结构 |
| `fs_read_file` | 读取文件内容 |
| `fs_write_file` | 写入文件 |

### 容器编排

| 工具 | 说明 |
|------|------|
| `sandbox_compose_up` | 启动 Docker 环境 |
| `sandbox_compose_down` | 停止环境 |
| `sandbox_compose_exec` | 在容器中执行命令 |
| `sandbox_compose_logs` | 获取容器日志 |
| `sandbox_compose_ps` | 查看容器状态 |

## 安全特性

- ✅ 路径校验：禁止访问工作目录外的文件
- ✅ Volume 校验：禁止挂载绝对路径和父目录
- ✅ 命令隔离：使用 exec.Command 而非 shell
- ✅ 超时控制：命令执行有超时限制

## 开发

```bash
# 运行测试
make test

# 生成覆盖率报告
make coverage

# 开发模式运行
make dev
```

## 许可证

MIT License
