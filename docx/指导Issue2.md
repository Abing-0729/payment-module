Issue 2 学习指南：PostgreSQL + goose + sqlc

一、先建立心智模型（为什么是这三件套）

这是一个典型的 Go 后端数据层三板斧，各管一件事：

┌───────┬────────────────────────────────────────┬───────────────────────────────┐
│ 工具  │                 管什么                 │             类比              │
├───────┼────────────────────────────────────────┼───────────────────────────────┤
│ goose │ 数据库结构的版本控制（建表/改表/回滚） │ 像 Git 管理代码一样管理 SQL   │
├───────┼────────────────────────────────────────┼───────────────────────────────┤
│ sqlc  │ 从 SQL 自动生成类型安全的 Go 查询代码  │ 让编译器帮你检查 SQL 的正确性 │
├───────┼────────────────────────────────────────┼───────────────────────────────┤
│ pgx   │ PostgreSQL 驱动 + 连接池 + 事务原语    │ 数据库访问的基础设施          │
└───────┴────────────────────────────────────────┴───────────────────────────────┘

三者配合关系：

db/migrations/*.sql  ←── goose 负责执行（make migrate-up/down）
│
└──── sqlc 读取它推导表结构（schema）
│
db/queries/*.sql（你写查询）
│
sqlc generate（make sqlc）
│
生成类型安全 Go 代码（不许手改！）
│
pgx 连接池（pgxpool）执行它们
│
事务上下文（Transactor）保证同事务同连接

两个核心理念，贯穿整个 Issue：

1. SQL 也是代码，需要版本管理 —— 数据库结构变了，代码跟不上就崩；反过来代码升级了数据库没升也崩。goose 解决"两边版本对齐"。
2. 手写 SQL 字符串拼接 = 事故高发区 —— 列名拼错、类型不匹配，都是运行时才炸。sqlc 让这些错误变成编译错误。

  ---
二、现状盘点：3 处必须修正的偏差

你（或上一轮对话）已经创建了空文件，但路径和 Issue 不一致：

┌───────────────────────────────────────────────┬──────────────────────────┬────────────────────────────┐
│                  Issue 要求                   │         仓库现状         │            问题            │
├───────────────────────────────────────────────┼──────────────────────────┼────────────────────────────┤
│ db/queries/health.sql                         │ db/migrations/health.sql │ queries 目录不存在         │
├───────────────────────────────────────────────┼──────────────────────────┼────────────────────────────┤
│ db/sqlc.yaml                                  │ db/migrations/sqlc.yaml  │ 配置文件不该塞在迁移目录里 │
├───────────────────────────────────────────────┼──────────────────────────┼────────────────────────────┤
│ services/commerce/internal/platform/postgres/ │ internal/paltform/       │ 拼写错误（paltform）       │
└───────────────────────────────────────────────┴──────────────────────────┴────────────────────────────┘

▎ 💡 学习点：文件已 git add 不代表"定型了"，随时可以用 git mv 修正再重新 add。写代码的目录名拼错了，比代码本身拼错更隐蔽、更坑人，因为 IDE 不会帮你查。

第一步先修这三处：

mkdir db/queries
git mv services/commerce/internal/paltform services/commerce/internal/platform
git mv db/migrations/health.sql db/queries/health.sql
git mv db/migrations/sqlc.yaml db/sqlc.yaml
git status   # 确认变更符合预期

  ---
三、学习路线图（8 步，每步验证通过再进下一步）

Step 0：安装工具 + 添加依赖

go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go get github.com/jackc/pgx/v5@latest        # 含 pgxpool
go get github.com/pressly/goose/v3@latest    # 测试里要用它的库 API

验证：goose -version、sqlc version、go build ./...

▎ 💡 学习点：goose 和 sqlc 都既提供 CLI 又提供库 API。Makefile 里用 CLI；测试里可以用库 API 在代码内直接跑迁移（后面会用到，这是本 Issue 的精髓之一）。

  ---
Step 1：Docker Compose 增加 PostgreSQL 16

deploy/docker-compose/ 目前不存在，需要新建 docker-compose.yml：

name: payment-lab

services:
postgres:
image: postgres:16-alpine
container_name: payment-postgres
environment:
POSTGRES_USER: payment
POSTGRES_PASSWORD: payment
POSTGRES_DB: payment
ports:
- "5432:5432"
volumes:
- postgres_data:/var/lib/postgresql/data
healthcheck:
test: ["CMD-SHELL", "pg_isready -U payment -d payment"]
interval: 5s
timeout: 3s
retries: 10

volumes:
postgres_data:

验证：

docker compose -f deploy/docker-compose/docker-compose.yml up -d postgres
docker compose -f deploy/docker-compose/docker-compose.yml ps   # 状态 healthy

▎ 💡 学习点（三选一背下）：
▎ - volumes 的 named volume：数据落在宿主机持久化，容器删了数据还在（重试迁移实验的前提）
▎ - healthcheck：让 compose 能告诉你"数据库真正就绪"而不是"容器起来了"
▎ - 端口 5432:5432 左边是宿主机端口，右边是容器端口，和右边冲突会启动失败

  ---
Step 2：写第一个迁移文件（先搞懂 goose 的文件格式）

db/migrations/00001_init.sql，内容自己设计一张最简单的表（建议一张带余额的表，后面测试要用）：

-- +goose Up
CREATE TABLE IF NOT EXISTS accounts (
id         BIGSERIAL PRIMARY KEY,
balance    BIGINT NOT NULL DEFAULT 0 CHECK (balance >= 0),
created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS accounts;

goose 的语法就一句话：-- +goose Up 到 -- +goose Down 之间是"前进"，之后是"回滚"。goose 在数据库里维护一张 goose_db_version 表，记录当前执行到哪个版本。

验证：

make migrate-up    # 待 Step 4 建好 Makefile 目标后

▎ 💡 学习点：迁移文件写完之后永远不许改（像 Git 历史一样）。改 schema 的唯一方式是新增 00002_xxx.sql。想改 00001？删库重来（开发期合法）。

  ---
Step 3：写第一个查询 + sqlc 配置

db/queries/health.sql（sqlc 的查询文件，跟 goose 无关）：

-- name: Ping :one
SELECT 1;

▎ 💡 学习点：-- name: Ping :one 是 sqlc 的注解语法——Ping 是生成函数的函数名，:one 表示"返回一行"（还有 :many、:exec、:execrows）。

db/sqlc.yaml：

version: "2"
sql:
- engine: "postgresql"
schema: "db/migrations"
queries: "db/queries"
gen:
go:
package: "query"
out: "services/commerce/internal/platform/postgres/query"
sql_package: "pgx/v5"
emit_empty_slices: true

▎ ⚠️ 这里埋了一个探索点：schema 指向 goose 迁移目录。sqlc 官方支持 goose 格式（会跳过 -- +goose Down 段），但不同版本行为有差异——你跑一次 make sqlc
▎ 就知道。如果报"表已存在"之类的错，就回来找我，我们换"独立 schema 目录"方案。这正是真实工程里会遇到的分叉，学会调试它比背结论值钱。

验证：sqlc generate -f db/sqlc.yaml 后看到生成目录里出现 query/db.go 和 query/health.sql.go，打开看看 Ping 函数的签名。

▎ 💡 学习点（重要）：sqlc 的 out 目录是托管区——每次 generate 会先清空再写入。所以手写代码永远不许放进生成目录。这就是为什么 postgres.go（手写）和
▎ query/（生成）要分开目录。我们的 fmt-check 检查所有 .go 文件，恰好也验证了"生成代码也要合规"。

  ---
Step 4：Makefile 三件套

在 Makefile 里追加（注意 DSN 和 compose 里的一致）：

POSTGRES_DSN ?= postgres://payment:payment@localhost:5432/payment?sslmode=disable
MIGRATIONS_DIR := db/migrations

# .PHONY 行同步加上 migrate-up migrate-down sqlc

migrate-up: ## 应用全部迁移（空库可重复执行）
goose -dir $(MIGRATIONS_DIR) postgres "$(POSTGRES_DSN)" up

migrate-down: ## 回滚最近一个迁移
goose -dir $(MIGRATIONS_DIR) postgres "$(POSTGRES_DSN)" down

sqlc: ## 生成类型安全查询代码（生成目录不许手改）
sqlc generate -f db/sqlc.yaml

▎ 💡 学习点：?= 是"没设环境变量才用默认值"——支持 POSTGRES_DSN=... make migrate-up 覆盖。

  ---
Step 5：连接池（postgres.go）——配置化参数

services/commerce/internal/platform/postgres/postgres.go。要点：连接池参数（最大连接数、连接超时）必须是可配置的，而不是写死常量——这是"配置连接池"的验收点：

package postgres

import (
"context"
"fmt"
"time"

        "github.com/jackc/pgx/v5/pgxpool"
)

// Postgres 封装 pgx 连接池。
type Postgres struct {
pool *pgxpool.Pool
}

// New 创建连接池并预检连通性。
// maxConns: 池内最大连接数（防止连接风暴打垮数据库）；
// connectTimeout: 单次建连超时（数据库挂了要快速失败，而不是无限等）。
func New(ctx context.Context, dsn string, maxConns int32, connectTimeout time.Duration) (*Postgres, error) {
cfg, err := pgxpool.ParseConfig(dsn)
if err != nil {
return nil, fmt.Errorf("解析 DSN: %w", err)
}
cfg.MaxConns = maxConns
cfg.ConnConfig.ConnectTimeout = connectTimeout

        pool, err := pgxpool.NewWithConfig(ctx, cfg)
        if err != nil {
                return nil, fmt.Errorf("创建连接池: %w", err)
        }
        // 预检：启动即验证数据库可达，而不是第一个请求才暴露问题
        if err := pool.Ping(ctx); err != nil {
                pool.Close()
                return nil, fmt.Errorf("ping 数据库: %w", err)
        }
        return &Postgres{pool: pool}, nil
}

// Close 释放连接池。服务优雅退出时调用。
func (p *Postgres) Close() { p.pool.Close() }

// Pool 暴露底层池。sqlc 生成的 DAO 绑定它执行普通查询。
func (p *Postgres) Pool() *pgxpool.Pool { return p.pool }

  ---
Step 6：事务上下文（tx.go）——本 Issue 的设计核心

services/commerce/internal/platform/postgres/tx.go。Issue 给了接口，你要实现它并想清楚为什么要用 context 传事务：

package postgres

import (
"context"
"fmt"

        "github.com/jackc/pgx/v5"
        "github.com/jackc/pgx/v5/pgxpool"
)

// Transactor 定义事务边界抽象：业务层只依赖这个接口，不依赖具体实现。
type Transactor interface {
WithinTransaction(ctx context.Context, fn func(context.Context) error) error
}

type txKey struct{}

// WithinTransaction 在事务中执行 fn。
//
// 关键设计：*pgx.Tx 通过 context.Context 传递。
// 业务代码无论调用链多深，只要从 ctx 取事务绑定对象，
// 整条链就复用同一个连接（同一个事务）。
// 这保证：事务内两次查询看到彼此未提交的修改——这正是"事务"的意义。
func (p *Postgres) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
// 已在事务中则直接复用（支持嵌套调用，如 service 内再调一个事务方法）
if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
return fn(ctx)
}

        tx, err := p.pool.Begin(ctx)
        if err != nil {
                return fmt.Errorf("开启事务: %w", err)
        }
        // defer 回滚：fn 成功且 Commit 成功后，这里的 Rollback 返回 ErrTxClosed，无害；
        // fn 返回错误时，这里就是真正的回滚。
        defer tx.Rollback(context.WithoutCancel(ctx))

        txCtx := context.WithValue(ctx, txKey{}, tx)
        if err := fn(txCtx); err != nil {
                return err // 不包装，保留原始错误给上层
        }
        if err := tx.Commit(ctx); err != nil {
                return fmt.Errorf("提交事务: %w", err)
        }
        return nil
}

// FromContext 从上下文取出事务。sqlc 生成的 DAO 用它在事务内执行查询。
func FromContext(ctx context.Context) (pgx.Tx, bool) {
tx, ok := ctx.Value(txKey{}).(pgx.Tx)
return tx, ok
}

为什么必须用 context 传？ 因为事务是连接级状态：BEGIN 之后的所有语句必须发到同一个物理连接上才有意义。如果业务层中间调了个
DAO，它自己从连接池拿一条新连接执行查询——那条新连接上根本没有事务，查询的结果也不可见。把 *pgx.Tx（内部持有一个专用连接）放进
context，让"事务内的执行器"顺着调用链流动，是 Go 生态的标准解法。

和 sqlc 怎么接上？ 看生成的 query/db.go 里这个接口：

type DBTX interface {
Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
Query(context.Context, string, ...any) (pgx.Rows, error)
QueryRow(context.Context, string, ...any) pgx.Row
}

pgxpool.Pool 和 pgx.Tx 都满足这个接口——这是整个设计的支点：

q := query.New(p.Pool())   // 普通查询：池里任取连接
tx, _ := FromContext(ctx)  // 事务内查询：
q := query.New(tx)         // 绑定到事务连接，复用同一连接！

▎ 这一步想通，整个 Issue 的"事务上下文保证同一业务事务复用同一连接"就通了。

  ---
Step 7：测试（4 个验收场景全覆盖）

测试文件 services/commerce/internal/platform/postgres/postgres_test.go。关键技巧：用 goose 的库 API 在测试里直接跑迁移，这样"空库迁移""down 后重 up"都能自动化：

先在 db/migrations/embed.go 里嵌入迁移文件（embed 不能跨目录，所以文件放这）：

package migrations

import "embed"

//go:embed *.sql
var FS embed.FS

然后测试骨架（自己动手写完整版）：

package postgres

import (
"context"
"database/sql"
"os"
"testing"
"time"

        "github.com/jackc/pgx/v5/pgxpool"
        "github.com/jackc/pgx/v5/stdlib"
        "github.com/pressly/goose/v3"

        "kratos-payment-lab/db/migrations"
)

var (
dsn  = os.Getenv("TEST_DATABASE_URL") // 默认 localhost 那一套
pool *pgxpool.Pool
)

func TestMain(m *testing.M) {
// 1. 连接数据库
// 2. goose.SetBaseFS(migrations.FS) + goose.SetDialect("postgres")
//    打开 database/sql 连接（stdlib 驱动）执行 goose.Up(db, ".")
// 3. 建 pgxpool
// 4. m.Run()
}

四个测试，每个对应一个验收场景：

┌──────────────────────────────┬───────────────────────────────┬──────────────────────────────────────────────────────────────────┐
│             测试             │             场景              │                               思路                               │
├──────────────────────────────┼───────────────────────────────┼──────────────────────────────────────────────────────────────────┤
│ TestMigrationsUpDown         │ 空库完整迁移 / down 后重新 up │ goose.Down 到底 → 查 accounts 表不存在 → goose.Up → 表存在       │
├──────────────────────────────┼───────────────────────────────┼──────────────────────────────────────────────────────────────────┤
│ TestRollbackOnError          │ 事务回调返回错误时回滚        │ 事务内插入一行 → fn 返回 error → 事务外查不到该行                │
├──────────────────────────────┼───────────────────────────────┼──────────────────────────────────────────────────────────────────┤
│ TestTransactionContextCancel │ context 取消时事务终止        │ 在 fn 里 cancel() 然后执行 SQL → 报 context canceled，数据未落库 │
├──────────────────────────────┼───────────────────────────────┼──────────────────────────────────────────────────────────────────┤
│ TestPing                     │ 生成代码可用                  │ query.New(pool).Ping(ctx) 返回 1                                 │
└──────────────────────────────┴───────────────────────────────┴──────────────────────────────────────────────────────────────────┘

▎ 💡 学习点：测试里的"自己动手验证回滚"，最简单的是在 fn 里 tx.Exec("INSERT INTO accounts (balance) VALUES ($1)", 100) 后返回错误，再在事务外用 sqlc 的 New(pool) 查
▎ SELECT COUNT(*)，断言为 0。

  ---
Step 8：配置接线（可选进阶）

config.yaml 的注释已经预告了"Issue 2 接入后在此追加 data 段"：

data:
database:
dsn: postgres://payment:payment@localhost:5432/payment?sslmode=disable
max_conns: 10
connect_timeout: 5s

然后由 Wire provider 读取并调用 postgres.New(...)。这一步可以等你把 1-7 跑通后再做，避免一次学太多。

  ---
四、验收清单（跑通后自查）

docker compose -f deploy/docker-compose/docker-compose.yml up -d postgres
make migrate-up                                  # 输出 goose: OK
make sqlc                                        # 生成代码出现
go test -race ./services/commerce/internal/platform/postgres/...

额外自查：

- [ ] make migrate-up 再跑一次 → "no new migrations"（幂等，空库可完整迁移 ✓）
- [ ] make migrate-down 后 psql -c '\dt' 看不到 accounts 表；再 make migrate-up 恢复
- [ ] 生成目录里没有你手写的任何字符
- [ ] git status 里没有 bin/、.idea/ 等不该提交的东西

  ---
五、AI 协作检查点（先自己想，再看我的参考）

① 用 SQL 手动验证回滚（打开 psql 或 Goland 的 database 工具连上 payment 库）：

BEGIN;
INSERT INTO accounts (balance) VALUES (100);
SELECT * FROM accounts;          -- 自己能看到这行（当前事务内）
ROLLBACK;                        -- 手动回滚
SELECT * FROM accounts;          -- 行没了

这就是数据库事务的本体。WithinTransaction 里 fn 返回错误 → 走 tx.Rollback，和 ROLLBACK 命令是同一件事。Go 测试只是把它自动化了。

② 为什么外部 HTTP 调用不能放进数据库事务？ 我理解的答案：

1. 锁与连接被占用：事务从 BEGIN 到 COMMIT/ROLLBACK 全程独占一个连接。HTTP 调用可能是秒级甚至十秒级，池子被占满后其他请求全部排队 → 整个服务瘫痪。
2. 事务保护不了"外部"：数据库能回滚它自己的数据，但没法回滚第三方支付网关——钱已经扣了，数据库 ROLLBACK 不会让支付网关退钱。原子性的边界只到数据库为止。
3. 两阶段提交不可行：分布式事务理论上要双方协作，第三方系统不会陪你实现 2PC。
4. 故障放大器：事务开着时进程崩溃 → 未提交事务被回滚，但外部调用已经生效 → 数据库和外部状态永久不一致。

正确姿势（支付系统的经典模式）：本地事务只负责写"支付单 + 状态 = PENDING"→ 提交 → 再调外部网关 → 收结果后再开新事务更新状态为
SUCCESS/FAILED；失败走补偿/对账（回调、定时核对）。一个请求的生命周期里通常有多个短事务，绝不把一个外部调用包在事务里。这是 Issue 后续"支付状态机"的地基。

  ---
建议的执行节奏

1. 先修 3 处路径偏差 → git status 确认
2. 从 Step 0 做到 Step 4，每步跑通验证再继续
3. Step 5-6 写代码时重点想通"为什么用 context 传事务"
4. Step 7 测试写好后，跑检查点①的 psql 实验，亲手感受回滚
5. 全部通过后，你自己决定何时提交（不代劳）

遇到任何一步卡住（比如 sqlc 解析 goose 迁移报错、连接超时、Windows 上 goose
二进制路径问题），直接把报错贴给我，我们一步步调。现在开始吧——先从修正路径偏差开始，完成后来告诉我结果 👍
