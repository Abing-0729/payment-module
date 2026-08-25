# Issue 3 实施指导：CI 验证流程（make verify）

> **Labels:** `type:infra`, `type:test`, `priority:p0` · **Depends on:** Issue 2
>
> **目标：** 每次提交自动验证生成代码、迁移、测试、竞态和构建；本地与 CI 跑同一条命令 `make verify`；手动修改生成文件后 `make verify` 必须失败。

---

## 一、现状盘点与缺口

| 已有 | 缺口 |
|---|---|
| `make verify`（fmt-check + generate-check + vet + build） | 只查 `api` 生成目录，**不含 sqlc / wire** |
| `make tools` | 装了 buf/wire/protoc 插件，**缺 sqlc 和 goose**（CI 上没有这俩，`sqlc generate` 和 `migrate-up` 会直接失败） |
| `ci.yml` 的 quality/unit job | integration job 被注释，且草稿凭据 `app/app` 与本地 docker-compose 的 `payment/payment` **不一致** |
| 集成测试默认 DSN `postgres://payment:payment@localhost:5432/payment` | 与 docker-compose 一致 ✓ |

## 二、设计思路

- **CI 与本地跑同一条命令 `make verify`**，不做两套逻辑，杜绝"CI 过了本地挂 / 本地过了 CI 挂"。
- 代码生成一致性校验抽成 `scripts/verify-generated.ps1`（Windows 用）与 `scripts/verify-generated.sh`（无 pwsh 的 Linux 用），Makefile 负责跨平台调起：
  - Windows（Git Bash/MSYS）→ 系统自带 `powershell` 跑 ps1
  - Linux/CI → 优先 `pwsh`（GitHub Actions 的 ubuntu-latest **预装 pwsh**）；**没有 pwsh 的环境（WSL、自建 Linux）自动退化为 `bash` 跑 sh 版**，逻辑完全一致
- 校验逻辑是 **"生成前 vs 生成后工作树对比"**，而不是"生成后必须干净"。
  区别：如果手改了生成文件但没提交（工作树本来就是脏的），"生成后必须干净"会漏判；快照对比能捕获"生成器把文件改回去了"这一变化。
- CI 的 postgres 服务容器凭据**与本地 docker-compose 对齐**（`payment/payment/payment`），这样 Makefile 默认 `POSTGRES_DSN` 两边都不用改。

---

## 三、第一步：修改 `Makefile`

共 5 处修改，修改后的完整文件如下（可直接整体替换）：

```make
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
```

### 改动点清单

| # | 位置 | 改动 | 原因 |
|---|---|---|---|
| 1 | `tools` | 新增 `sqlc`、`goose` 两行安装 | CI 上 `make tools` 是唯一装工具途径，没有这俩 `sqlc generate` 和 `migrate-up` 直接失败 |
| 2 | `.PHONY` | 合并原来重复的两行声明，新增 `verify-generated` | 原有第 7、9 行重复 |
| 3 | 新增 | `UNAME_S`/`PWSH` 平台检测 + `verify-generated` 目标 | 跨平台调起 ps1 脚本 |
| 4 | `generate-check` | 改为委托 `verify-generated` | 保留旧目标可用，逻辑统一到一处 |
| 5 | `verify` | 改为 `fmt-check verify-generated migrate-up vet race-test build` | 对齐 Issue 3 验收 |

设计点：

- **`migrate-up` 放进 verify**：本身幂等（空库/已迁移库都能重复跑），本地与 CI 的 DSN 默认值都无需改。
- **`race-test` 已覆盖全部单测**（`go test -race ./...`），无需再单独跑 `unit-test`。
- `api` / `sqlc` / `wire` 目标保留——出错时可以单独重跑修复。

---

## 四、第二步：新建 `scripts/verify-generated.ps1`

```powershell
# scripts/verify-generated.ps1
# 生成代码一致性校验：重新执行全部代码生成（buf / sqlc / wire），
# 然后对比生成前后的工作树 —— 有任何差异即失败。
# 生成代码不允许手改：本脚本同时兜住"手动改了生成文件"与"生成器输出漂移"两种情况。
#
# 调用：make verify 会自动选择本机 PowerShell 并传入本脚本；
# 也可手动执行：powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-generated.ps1

$ErrorActionPreference = 'Stop'

function Fail([string]$message) {
    Write-Host ''
    Write-Host "FAIL: $message" -ForegroundColor Red
    Write-Host '生成代码与源文件不一致。请执行 make api && make sqlc && make wire 后提交结果，且不得手改生成文件。'
    exit 1
}

# 前置检查：生成器必须在 PATH 上（make tools 会安装到 GOPATH/bin）
foreach ($cmd in @('buf', 'sqlc', 'wire')) {
    if (-not (Get-Command $cmd -ErrorAction SilentlyContinue)) {
        Fail "未找到 $cmd 命令，请先执行 make tools"
    }
}

# 生成前的工作树快照（git status --porcelain 逐行输出，干净工作树为 0 行）
$before = @(git status --porcelain)

# 1) buf：proto -> api/*.pb.go
Write-Host '==> buf generate api'
buf generate api
if ($LASTEXITCODE -ne 0) { Fail 'buf generate 失败' }

# 2) sqlc：db/queries/*.sql -> 类型安全 Go 代码
Write-Host '==> sqlc generate'
sqlc generate -f db/sqlc.yaml
if ($LASTEXITCODE -ne 0) { Fail 'sqlc generate 失败' }

# 3) wire：两个服务的依赖注入代码（wire_gen.go 不允许手改）
foreach ($svc in @('services/commerce/cmd/commerce', 'services/mockpay/cmd/mockpay')) {
    Write-Host "==> wire $svc"
    wire $svc
    if ($LASTEXITCODE -ne 0) { Fail "wire $svc 失败" }
}

# 4) 对比生成前后的工作树
$after = @(git status --porcelain)
if (@(Compare-Object $before $after).Count -gt 0) {
    Write-Host '生成后相对生成前的工作树差异：' -ForegroundColor Yellow
    git status --porcelain
    git diff --stat
    Fail '生成代码与源文件不一致'
}

Write-Host 'OK: 生成代码与源文件一致。' -ForegroundColor Green
```

### 一个必须注意的坑：UTF-8 BOM

脚本是中文内容，Windows 自带的 PowerShell 5.1 只认 **UTF-8 with BOM**，无 BOM 会把中文注释和字符串读成乱码（CI 的 pwsh 没有这个问题）。保存文件后补一个 BOM（Git Bash 下执行）：

```bash
printf '\xef\xbb\xbf' > /tmp/bom && cat scripts/verify-generated.ps1 >> /tmp/bom && mv /tmp/bom scripts/verify-generated.ps1
```

`git add` 时 `.gitattributes` 会把 `*.ps1` 规范化为 CRLF（仓库里已有 `*.ps1 text eol=crlf`），行尾不用操心。

### 配套：`scripts/verify-generated.sh`（无 pwsh 环境的 Linux/macOS 版）

> 场景：WSL、自建 Linux 机器等没有 pwsh 的环境。GitHub Actions 的 ubuntu-latest 预装 pwsh 会走 ps1 分支，但如果所在 CI 没有 pwsh，`make verify-generated` 会报 `pwsh: command not found`。此时 Makefile 的 `command -v pwsh` 检测会兜底，改跑 bash 版脚本，逻辑与 ps1 完全一致。

```bash
#!/usr/bin/env bash
# scripts/verify-generated.sh
# 生成代码一致性校验（Linux/macOS 版）：重新执行 buf / sqlc / wire 后对比生成前后工作树，
# 有任何差异即失败。由 make 在无 pwsh 的环境（WSL、自建 Linux 等）自动调用。
# Windows 对应实现见 scripts/verify-generated.ps1。
set -euo pipefail

fail() {
    echo ""
    echo "FAIL: $1" >&2
    echo "生成代码与源文件不一致。请执行 make api && make sqlc && make wire 后提交结果，且不得手改生成文件。" >&2
    exit 1
}

# 前置检查：生成器必须在 PATH 上（make tools 会安装到 GOPATH/bin）
for cmd in buf sqlc wire; do
    command -v "$cmd" >/dev/null 2>&1 || fail "未找到 $cmd 命令，请先执行 make tools"
done

# 生成前的工作树快照（git status --porcelain 逐行输出，干净工作树为空）
before="$(git status --porcelain)"

# 1) buf：proto -> api/*.pb.go
echo "==> buf generate api"
buf generate api

# 2) sqlc：db/queries/*.sql -> 类型安全 Go 代码
echo "==> sqlc generate"
sqlc generate -f db/sqlc.yaml

# 3) wire：两个服务的依赖注入代码（wire_gen.go 不允许手改）
for svc in services/commerce/cmd/commerce services/mockpay/cmd/mockpay; do
    echo "==> wire $svc"
    wire "$svc"
done

# 4) 对比生成前后的工作树
after="$(git status --porcelain)"
if [ "$before" != "$after" ]; then
    echo "生成后相对生成前的工作树差异：" >&2
    git status --porcelain
    git diff --stat
    fail "生成代码与源文件不一致"
fi

echo "OK: 生成代码与源文件一致。"
```

bash 脚本用 `set -euo pipefail` 让任意一条命令失败都直接退出，无需逐个检查退出码。`.gitattributes` 已有 `*.sh text eol=lf`，无需处理行尾。

---

## 五、第三步：重写 `.github/workflows/ci.yml`

把原来的 quality + unit 两个 job 合并成**单个 `verify` job**——CI 上跑的就是验收命令本身：

```yaml
name: CI

on:
  # 任意分支的 PR 到 dev/main 都触发质量检查
  pull_request:
    branches: [dev, main]
  # push 到 main/dev（PR 合并成功）后也跑一轮检查
  push:
    branches: [main, dev]

concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true

jobs:
  verify:
    runs-on: ubuntu-latest
    # 服务容器：凭据与本地 deploy/docker-compose/docker-compose.yml 保持一致（payment/payment/payment），
    # 这样 Makefile 默认的 POSTGRES_DSN 在本地和 CI 都无需改动
    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_USER: payment
          POSTGRES_PASSWORD: payment
          POSTGRES_DB: payment
        ports:
          - 5432:5432
        options: >-
          --health-cmd "pg_isready -U payment -d payment"
          --health-interval 5s
          --health-timeout 5s
          --health-retries 20
      redis:
        image: redis:7
        ports:
          - 6379:6379
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 5s
          --health-timeout 5s
          --health-retries 20
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true

      # 安装 buf / protoc 插件 / wire / sqlc / goose 到 GOPATH/bin（setup-go 已将其加入 PATH）
      - name: Install tools
        run: make tools

      # 与本地验收命令完全一致：格式 + 生成一致性 + 迁移 + vet + 竞态测试 + 构建
      - name: Verify
        run: make verify
```

改动要点（对照原有文件）：

- **凭据从 `app/app` 改为 `payment/payment`**——与本地 docker-compose 对齐，这是原注释草稿里的一个隐藏 bug：不改的话 CI 的 `migrate-up` 会连不上库。
- 合并成单 job，删除注释掉的 integration/docker 草稿（`cd-image.yml` 已负责镜像构建）。

---

## 六、第四步：验收

### 本地

```bash
# 0. 前置：本地 PostgreSQL 必须先起来（docker-compose 里只有 postgres，redis 暂未用到）
docker compose -f deploy/docker-compose/docker-compose.yml up -d postgres

# 1. 装齐工具（sqlc、goose 是这次新加的）
make tools

# 2. 一键验收
make verify
```

预期：fmt-check 通过 → ps1 打印 `OK: 生成代码与源文件一致。` → migrate-up 应用迁移（或"无操作"）→ vet 无输出 → race 测试 PASS → `bin/` 下产出两个二进制。

### 失败场景（验收必测）

```bash
# 故意改一个生成文件，比如给 .pb.go 追加一行
echo "// 手改测试" >> api/commerce/v1/commerce.pb.go
make verify   # 必须失败：verify-generated 阶段报"生成代码与源文件不一致"，退出码非 0
git checkout -- api/commerce/v1/commerce.pb.go   # 还原
```

### CI

推送（push 到 `dev`/`main`）或开 PR 到这两个分支即自动触发，等待 `verify` job 变绿。

---

## 七、常见坑

1. **工具版本漂移**：`make tools` 全用 `@latest`，本地和 CI 装上不同版本的 buf/sqlc/wire 可能导致生成结果不同、`make verify` 偶发失败。稳妥做法是把 `@latest` 换成钉死的版本（如 `sqlc/cmd/sqlc@v1.30.0`）。可选优化。
2. **ps1 中文乱码**：Windows PowerShell 5.1 需要 UTF-8 BOM，见第四节末尾的补 BOM 命令。
3. **`migrate-up` 失败**：先确认本地 PG 起来了（`docker compose up -d postgres`）。这是预期行为，不是 bug。
4. **WSL / 自建 Linux 里跑 make**：没有 pwsh 时 Makefile 的 `command -v pwsh` 检测会自动退化为 `bash scripts/verify-generated.sh`，无需装 pwsh；Git Bash 则自动走 `powershell` 分支。
5. **verify 不拦"与生成无关的未提交改动"**：如 `M Dockerfile`、`AM docx/...` 不会导致 verify 失败——before/after 对比只关注生成器造成的变化，这是特性不是缺陷。
