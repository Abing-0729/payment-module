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
