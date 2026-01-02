.PHONY: build test clean install run dev build-all

# 变量
BINARY_NAME=ai-sandbox
VERSION=1.0.0
BUILD_DIR=build
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# Go 参数
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# 默认目标
all: build

# 编译
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME).exe ./cmd/ai-sandbox

# 跨平台编译
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/ai-sandbox
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/ai-sandbox
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/ai-sandbox
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/ai-sandbox

# 运行测试
test:
	@echo "Running tests..."
	$(GOTEST) -v -race ./...

# 测试覆盖率
coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# 代码检查（需要安装 golangci-lint）
lint:
	@echo "Running linter..."
	golangci-lint run

# 整理依赖
tidy:
	@echo "Tidying modules..."
	$(GOMOD) tidy

# 安装到 GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	cp $(BUILD_DIR)/$(BINARY_NAME).exe $(GOPATH)/bin/

# 清理
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# 开发模式运行
dev:
	$(GOCMD) run ./cmd/ai-sandbox start --workdir .

# 运行检查
check:
	$(GOCMD) run ./cmd/ai-sandbox check

# 显示帮助
help:
	@echo "Available targets:"
	@echo "  build      - Build the binary for current platform"
	@echo "  build-all  - Build for all platforms (darwin, linux, windows)"
	@echo "  test       - Run tests"
	@echo "  coverage   - Run tests with coverage report"
	@echo "  lint       - Run linter (requires golangci-lint)"
	@echo "  tidy       - Tidy go modules"
	@echo "  install    - Install binary to GOPATH/bin"
	@echo "  clean      - Clean build artifacts"
	@echo "  dev        - Run in development mode"
	@echo "  check      - Run environment check"
