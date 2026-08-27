# Go 支付与履约系统 — 构建工具入口
# 目标随 Issue 逐步补充：build/wire（Issue 1 服务代码）、migrate-up/sqlc（Issue 2）、verify（Issue 3，已实现）。
# 声明伪目标
POSTGRES_DSN ?= postgres://payment:payment@localhost:5432/payment?sslmode=disable
MIGRATIONS_DIR := db/migrations

.PHONY: tools api fmt-check generate-check verify-generated vet verify build run-commerce run-mockpay unit-test race-test integration-test migrate-up migrate-down sqlc fix-eol

# 跨平台调用生成校验脚本：
#   Windows（Git Bash/MSYS）→ 系统自带 powershell 跑 ps1
#   其他平台 → 优先 pwsh（GitHub Actions 预装），没有 pwsh（如 WSL/自建 Linux）则退化为 bash 跑 sh 版
UNAME_S := $(shell uname -s)
ifeq ($(findstring MINGW,$(UNAME_S)),MINGW)
VERIFY_GENERATED := powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-generated.ps1
else ifeq ($(findstring MSYS,$(UNAME_S)),MSYS)
VERIFY_GENERATED := powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-generated.ps1
else
PWSH := $(shell command -v pwsh 2>/dev/null)
ifeq ($(PWSH),)
VERIFY_GENERATED := bash scripts/verify-generated.sh
else
VERIFY_GENERATED := pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/verify-generated.ps1
endif
endif

tools: ## 安装 proto / sqlc / goose / wire 工具
	@go install github.com/bufbuild/buf/cmd/buf@latest
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	@go install github.com/google/wire/cmd/wire@latest
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@go install github.com/pressly/goose/v3/cmd/goose@latest

api: ## 从 proto 生成 Go 代码（仅生成 api 模块，third_party 只参与编译）
	buf generate api

fmt-check: ## gofmt 检查（生成代码不允许手改）
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './.git/*'))" || (echo "以下文件需要 gofmt 格式化:" && gofmt -l $$(find . -name '*.go' -not -path './.git/*') && exit 1)

generate-check: ## 检查生成代码与源文件一致（委托 verify-generated）
	@make verify-generated

verify-generated: ## 生成代码一致性校验：重新生成 buf/sqlc/wire 后工作树必须无差异
	$(VERIFY_GENERATED)

vet: ## go vet 静态检查
	go vet ./...

wire: ## 生成 Wire 依赖注入代码（wire_gen.go 不允许手改）
	"$(shell go env GOPATH)/bin/wire" ./services/commerce/cmd/commerce
	"$(shell go env GOPATH)/bin/wire" ./services/mockpay/cmd/mockpay

build: ## 同时构建两个服务到 bin/（bin/ 已在 .gitignore）
	@mkdir -p bin
	go build -o bin/commerce ./services/commerce/cmd/commerce
	go build -o bin/mockpay ./services/mockpay/cmd/mockpay

verify: fmt-check verify-generated migrate-up vet race-test build ## CD 前置校验：格式 + 生成一致性 + 迁移 + vet + 竞态测试 + 构建

run-commerce: ## 独立启动 commerce
	go run ./services/commerce/cmd/commerce -conf services/commerce/configs

run-mockpay: ## 独立启动 mockpay
	go run ./services/mockpay/cmd/mockpay -conf services/mockpay/configs

unit-test: ## 单元测试
	go test ./...

race-test: ## 竞态测试
	go test -race ./...

integration-test: ## 集成测试（依赖本地 Postgres，见 deploy/docker-compose）
	go test -tags integration ./...

migrate-up: ## 应用全部迁移（空库可重复执行）
	goose -dir $(MIGRATIONS_DIR) postgres "$(POSTGRES_DSN)" up

migrate-down: ## 回滚最近一个迁移
	goose -dir $(MIGRATIONS_DIR) postgres "$(POSTGRES_DSN)" down

sqlc: ## 生成类型安全查询代码（生成目录不许手改）
	sqlc generate -f db/sqlc.yaml

fix-eol: ## 一次性规范化所有已跟踪文本文件行尾为 LF（.gitattributes 重新生效，幂等可重复执行）
	@git add --renormalize .
	@git ls-files --eol | awk '$$1 ~ /^i\/crlf/ {print $$NF}' | while read -r f; do tr -d '\r' < "$$f" > "$$f.tmp" && mv "$$f.tmp" "$$f" && echo "已转换: $$f"; done
	@echo "行尾已统一为 LF。请 git add 后提交。"
