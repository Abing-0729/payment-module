# Go 支付与履约系统 — 构建工具入口
# 目标随 Issue 逐步补充：build/wire（Issue 1 服务代码）、migrate-up/sqlc（Issue 2）、verify（Issue 3）。
# 声明伪目标
POSTGRES_DSN ?= postgres://payment:payment@localhost:5432/payment?sslmode=disable
MIGRATIONS_DIR := db/migrations

.PHONY: tools api fmt-check generate-check vet verify wire build run-commerce run-mockpay unit-test race-test migrate-up migrate-down sqlc fix-eol

.PHONY: tools api fmt-check generate-check wire build run-commerce run-mockpay unit-test race-test migrate-up migrate-down sqlc fix-eol

tools: ## 安装 proto 生成工具（buf + 三个 protoc 插件）
	@go install github.com/bufbuild/buf/cmd/buf@latest
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	@go install github.com/google/wire/cmd/wire@latest

api: ## 从 proto 生成 Go 代码（仅生成 api 模块，third_party 只参与编译）
	buf generate api

fmt-check: ## gofmt 检查（生成代码不允许手改）
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './.git/*'))" || (echo "以下文件需要 gofmt 格式化:" && gofmt -l $$(find . -name '*.go' -not -path './.git/*') && exit 1)

generate-check: ## 检查生成代码与 proto 一致
	@make api
	@test -z "$$(git status --porcelain -- api)" || (echo "api 生成代码与 proto 不一致，请重新执行 make api 并提交:" && git status --porcelain -- api && exit 1)

vet: ## go vet 静态检查
	go vet ./...

wire: ## 生成 Wire 依赖注入代码（wire_gen.go 不允许手改）
	"$(shell go env GOPATH)/bin/wire" ./services/commerce/cmd/commerce
	"$(shell go env GOPATH)/bin/wire" ./services/mockpay/cmd/mockpay

build: ## 同时构建两个服务到 bin/（bin/ 已在 .gitignore）
	@mkdir -p bin
	go build -o bin/commerce ./services/commerce/cmd/commerce
	go build -o bin/mockpay ./services/mockpay/cmd/mockpay

verify: fmt-check generate-check vet build ## CD 前置校验：格式 + 生成代码一致性 + vet + 构建

run-commerce: ## 独立启动 commerce
	go run ./services/commerce/cmd/commerce -conf services/commerce/configs

run-mockpay: ## 独立启动 mockpay
	go run ./services/mockpay/cmd/mockpay -conf services/mockpay/configs

unit-test: ## 单元测试
	go test ./...

race-test: ## 竞态测试
	go test -race ./...

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
