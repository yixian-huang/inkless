VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

.PHONY: dev dev-backend dev-frontend build-backend build-cli stop help

# ── 版本信息 ─────────────────────────────────────────────
GIT_COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH  := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
BUILD_TIME  := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
VERSION     := $(GIT_COMMIT)
LDFLAGS     := -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT) -X main.GitBranch=$(GIT_BRANCH)

# ── 一键启动 ──────────────────────────────────────────────
dev: ## 启动前后端（后端 :8088 + 前端 :3000）
	@$(MAKE) -j2 dev-backend dev-frontend

dev-backend: ## 启动后端（需先 build-backend）
	@cd backend && \
	export PORT=8088 && \
	export DB_DSN='file:./data/blotting.db?cache=shared&mode=rwc' && \
	export JWT_SECRET=dev_jwt_secret_change_in_production && \
	export JWT_REFRESH_SECRET=dev_jwt_refresh_secret_change_in_production && \
	export ENV=development && \
	export UPLOAD_DIR=./uploads && \
	./server

dev-frontend: ## 启动前端 dev server
	@cd frontend && pnpm dev

# ── 构建 ──────────────────────────────────────────────────
build-backend: ## 编译后端（自动注入版本信息）
	@cd backend && go build -ldflags '$(LDFLAGS)' -o server ./cmd/server/
	@printf '{"version":"%s","buildTime":"%s","gitCommit":"%s","gitBranch":"%s"}\n' \
		"$(VERSION)" "$(BUILD_TIME)" "$(GIT_COMMIT)" "$(GIT_BRANCH)" > backend/version.json
	@echo "Built backend $(VERSION) ($(GIT_BRANCH)@$(GIT_COMMIT)) at $(BUILD_TIME)"

build-cli: ## 编译 CLI 工具
	@cd backend && go build -ldflags '-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)' -o impress ./cmd/impress/
	@echo "Built CLI $(VERSION)"

build: build-backend build-cli ## 编译前后端 + CLI
	@cd frontend && pnpm build

# ── 停止 ──────────────────────────────────────────────────
stop: ## 停止前后端进程
	@-lsof -i :8088 -sTCP:LISTEN -t | xargs kill 2>/dev/null; true
	@-lsof -i :3000 -sTCP:LISTEN -t | xargs kill 2>/dev/null; true
	@echo "stopped"

# ── 检查 ──────────────────────────────────────────────────
check: ## 运行 lint + type-check
	@cd frontend && pnpm lint && pnpm type-check

# ── 帮助 ──────────────────────────────────────────────────
help: ## 显示所有可用命令
	@grep -E '^[a-zA-Z_-]+:.*?##' $(MAKEFILE_LIST) | awk -F ':.*?## ' '{printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'
