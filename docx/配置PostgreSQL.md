> 核心流程：选驱动，配链接，设池子
## 驱动
推荐：pgx (github.com/jackc/pgx/v5)
## 连接数据库
使用pgx主要有两种接入方式
1. 导入驱动：需要匿名导入`pgx`的`stdlib`包来注册驱动
2. 打开数据库

## 接入层：配置文件+连接池
```yaml
# config.yaml
data:
  database:
    dsn: postgres://payment:payment@localhost:5432/payment?sslmode=disable
    max_conns: 10
    connect_timeout: 5s
```
- DSN 与 docker-compose 严格对应：用户名 payment、密码 payment、库名 payment，保证“数据库怎么起”和“服务怎么连”是配套的。

- max_conns: 10 就是连接池的上限，对应之前提到的 MaxConns 配置。


## 注意
- pg_isready是PG自带的健康检查工具，确保容器在完全就绪后才被认为是“启动成功”
- enbed.go是关键技巧：将迁移文件嵌入到测试二进制汇总，使得单元测试可以不依靠`goose`命令行工具，直接运行迁移
- sqlc是代码生成器，帮助生成类型安全、0反射的Go代码
> **sql_package: "pgx/v5" 意味着生成的代码直接使用 pgx 的原生接口，而不是 database/sql 标准库。这比标准库性能更好，且能天然支持 PG 的数组、JSONB 等高级类型。**



## 完整流程
```bash
# 1. 启动数据库容器
docker compose up -d
# → 容器启动，PostgreSQL 监听 localhost:5432

# 2. 执行迁移，建表
make migrate-up
# → goose 连接到数据库，执行 db/migrations/00001_init.sql 的 Up 部分

# 3. 生成查询代码
make sqlc
# → sqlc 读取 migrations 了解表结构，读取 queries 中的 SQL，生成 Go 代码到指定目录

# 4. 服务启动（读配置）
# → postgres.go 读取 config.yaml 中的 dsn，调用 pgxpool.New() 创建连接池

# 5. 单元测试
go test
# → 测试代码使用 embed 的迁移文件，在测试开始前自动建表，跑完自动清理
# → 测试中调用 q.Ping(ctx) 验证数据库连通性
```
