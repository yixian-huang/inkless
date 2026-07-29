VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

.PHONY: dev dev-up dev-backend dev-frontend build-backend build-cli stop help

# Frontend dev server talks to the local API on this URL
DEV_API_URL ?= http://localhost:8088
# Optional live-link theme clones (relative to frontend/). Example:
#   make dev-frontend THEME_PRODUCT_FIRST_PATH=../inkless-theme-product-first
THEME_PRODUCT_FIRST_PATH ?=
THEME_BLOG_FIRST_PATH ?=
THEME_EDITORIAL_FIRM_PATH ?=

# ── 版本信息 ─────────────────────────────────────────────
GIT_COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH  := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
BUILD_TIME  := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
VERSION     := $(GIT_COMMIT)
LDFLAGS     := -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT) -X main.GitBranch=$(GIT_BRANCH)

# ── 开发启动 ──────────────────────────────────────────────
dev-up: ## 安装依赖、编译后端并启动前后端（SQLite，推荐首次/日常）
	@echo "→ pnpm install"
	@pnpm install
	@$(MAKE) build-backend
	@mkdir -p backend/data backend/uploads
	@$(MAKE) dev

dev: ## 启动前后端（需已 build-backend；完整流程用 dev-up）
	@$(MAKE) -j2 dev-backend dev-frontend

dev-backend: ## 启动后端（需先 build-backend）
	@cd backend && \
	export PORT=8088 && \
	export DB_DSN='file:./data/inkless.db?cache=shared&mode=rwc' && \
	export JWT_SECRET=dev_jwt_secret_change_in_production && \
	export JWT_REFRESH_SECRET=dev_jwt_refresh_secret_change_in_production && \
	export ENV=development && \
	export SEED_MODE=demo && \
	export UPLOAD_DIR=./uploads && \
	./inkless-api-latest

dev-frontend: ## 启动前端 dev server（可传 THEME_*_PATH 实时链主题仓库）
	@cd frontend && \
	VITE_API_BASE_URL=$(DEV_API_URL) \
	THEME_PRODUCT_FIRST_PATH="$(THEME_PRODUCT_FIRST_PATH)" \
	THEME_BLOG_FIRST_PATH="$(THEME_BLOG_FIRST_PATH)" \
	THEME_EDITORIAL_FIRM_PATH="$(THEME_EDITORIAL_FIRM_PATH)" \
	pnpm dev

# ── 构建 ──────────────────────────────────────────────────
build-backend: ## 编译后端（自动注入版本信息）
	@cd backend && go build -ldflags '$(LDFLAGS)' -o inkless-api-$(VERSION) ./cmd/server/
	@cd backend && ln -sf inkless-api-$(VERSION) inkless-api-latest
	@printf '{"version":"%s","buildTime":"%s","gitCommit":"%s","gitBranch":"%s"}\n' \
		"$(VERSION)" "$(BUILD_TIME)" "$(GIT_COMMIT)" "$(GIT_BRANCH)" > backend/version.json
	@echo "Built backend $(VERSION) ($(GIT_BRANCH)@$(GIT_COMMIT)) at $(BUILD_TIME)"

seed-blog-samples: ## 写入约 48 篇示例博客（幂等，需已有数据库）
	@bash scripts/seed-blog-samples.sh

build-cli: ## 编译 CLI 工具
	@cd backend && go build -ldflags '-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)' -o inkless ./cmd/inkless/
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
