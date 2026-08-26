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
# 注意：路径必须以 ./ 开头 —— PowerShell 5.1 会把裸参数中 / 开头的内容解析为 switch 参数
foreach ($svc in @('./services/commerce/cmd/commerce', './services/mockpay/cmd/mockpay')) {
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
