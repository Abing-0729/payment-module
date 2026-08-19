# kratos-payment-lab Makefile
#
# 当前处于 M0 起步阶段（尚无 proto / 服务代码 / 迁移文件）：
# 依赖文件未就位的目标会打印提示并跳过（守卫），文件就位后执行真实命令。
# CI 引用的目标：tools、fmt-check、generate-check、unit-test、race-test、
# migrate-up、integration-test（.github/workflows/ci.yml）

.PHONY: tools generate generate-check fmt-check vet unit-test race-test \
        integration-test build docker-build verify migrate-up migrate-down

# 工具版本暂用 @latest，后续引入 tools/tools.go 后在 go.mod 中固定。
tools:
	go install github.com/bufbuild/buf/cmd/buf@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/google/wire/cmd/wire@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest

fmt-check:
	gofmt -l . | tee /tmp/gofmt.out
	test ! -s /tmp/gofmt.out

generate:
	@if [ ! -f buf.yaml ]; then \
		echo "no buf.yaml yet, skipping buf generate"; \
	else \
		buf generate; \
	fi
	@if [ ! -f db/sqlc.yaml ]; then \
		echo "no db/sqlc.yaml yet, skipping sqlc generate"; \
	else \
		sqlc generate; \
	fi
	@if [ ! -f services/commerce/cmd/commerce/wire.go ]; then \
		echo "no wire.go yet, skipping wire"; \
	else \
		wire ./services/commerce/cmd/commerce; \
	fi
	@if [ ! -f services/mockpay/cmd/mockpay/wire.go ]; then \
		echo "no wire.go yet, skipping wire"; \
	else \
		wire ./services/mockpay/cmd/mockpay; \
	fi

generate-check:
	$(MAKE) generate
	git diff --exit-code

vet:
	@if [ -z "$$(find . -name '*.go' -not -path './.git/*' 2>/dev/null)" ]; then \
		echo "no Go files yet, skipping"; \
	else \
		go vet ./...; \
	fi

TEST_PKGS := $(shell for d in services pkg; do [ -d $$d ] && printf "./%s/... " $$d; done)

unit-test:
	@if [ -n "$(TEST_PKGS)" ]; then \
		go test $(TEST_PKGS); \
	else \
		echo "no services/ or pkg/ yet, skipping"; \
	fi

race-test:
	@if [ -n "$(TEST_PKGS)" ]; then \
		go test -race $(TEST_PKGS); \
	else \
		echo "no services/ or pkg/ yet, skipping"; \
	fi

integration-test:
	@if [ ! -d tests/integration ]; then \
		echo "no tests/integration yet, skipping"; \
	else \
		go test -race -tags=integration ./tests/integration/...; \
	fi

build:
	@if [ -d services/commerce/cmd/commerce ]; then \
		go build ./services/commerce/cmd/commerce; \
	fi
	@if [ -d services/mockpay/cmd/mockpay ]; then \
		go build ./services/mockpay/cmd/mockpay; \
	fi

DATABASE_URL ?= postgres://app:app@localhost:5432/commerce_test?sslmode=disable
MIGRATIONS_DIR ?= db/migrations

migrate-up:
	@if [ -z "$$(ls $(MIGRATIONS_DIR)/*.sql 2>/dev/null)" ]; then \
		echo "no migrations yet, skipping"; \
	else \
		goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up; \
	fi

migrate-down:
	@if [ -z "$$(ls $(MIGRATIONS_DIR)/*.sql 2>/dev/null)" ]; then \
		echo "no migrations yet, skipping"; \
	else \
		goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down; \
	fi

docker-build:
	docker build -f services/commerce/Dockerfile -t commerce:ci .
	docker build -f services/mockpay/Dockerfile -t mockpay:ci .

verify: fmt-check generate-check unit-test race-test build