# Go 支付与履约系统 — 构建工具入口
# 目标随 Issue 逐步补充：build/wire（Issue 1 服务代码）、migrate-up/sqlc（Issue 2）、verify（Issue 3）。

.PHONY: tools api fmt-check generate-check unit-test race-test

tools: ## 安装 proto 生成工具（buf + 三个 protoc 插件）
	@go install github.com/bufbuild/buf/cmd/buf@latest
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest

api: ## 从 proto 生成 Go 代码
	cd api && buf generate

fmt-check: ## gofmt 检查（生成代码不允许手改）
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './.git/*'))" || (echo "以下文件需要 gofmt 格式化:" && gofmt -l $$(find . -name '*.go' -not -path './.git/*') && exit 1)

generate-check: ## 检查生成代码与 proto 一致
	@make api
	@test -z "$$(git status --porcelain -- api)" || (echo "api 生成代码与 proto 不一致，请重新执行 make api 并提交:" && git status --porcelain -- api && exit 1)

unit-test: ## 单元测试
	go test ./...

race-test: ## 竞态测试
	go test -race ./...
