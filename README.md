# Kratos Payment Model

这是一个使用 Go、Kratos、PostgreSQL、sqlc 和 protobuf 构建的支付与履约练习项目，重点练习订单、支付、退款中的金额精度、事务、状态机、幂等和测试。

## 目录

- `api/`：protobuf 定义和生成的 API 代码。
- `db/migrations/`：goose 数据库迁移。
- `db/queries/`：sqlc 查询定义。
- `services/commerce/internal/order/`：订单领域模型、业务规则和 API service。
- `services/commerce/internal/platform/`：PostgreSQL、事务和错误处理。
- `services/commerce/internal/server/`：HTTP、gRPC 和健康检查。
- `services/mockpay/`：模拟支付渠道。

## 分层职责

调用方向是：客户端 → protobuf API → `order.Service` → `UseCase` → `Repository` → sqlc → PostgreSQL。

`model.go` 定义领域对象，`status.go` 定义状态机，`repository.go` 负责 SQL 和模型转换，`usecase.go` 负责业务规则和事务，`service.go` 实现 API 方法，`server/http.go` 与 `server/grpc.go` 负责注册 API，Wire 负责组装依赖。

### 请求调用链

```mermaid
flowchart LR
    Client[客户端] --> API[HTTP / gRPC API]
    API --> Service[order.Service]
    Service --> UseCase[UseCase]
    UseCase --> Repo[OrderRepository]
    Repo --> SQLC[sqlc 生成代码]
    SQLC --> DB[(PostgreSQL)]
```

HTTP 和 gRPC 是两种不同的传输入口，但最终都会调用同一个 `order.Service`，再进入相同的 usecase 和 repository。

### 依赖组装

```mermaid
flowchart TD
    Config[服务配置] --> Postgres[Postgres 连接池]
    Postgres --> Repository[OrderRepository]
    Repository --> UseCase[Order UseCase]
    UseCase --> OrderService[Order Service]
    OrderService --> HTTP[HTTP Server]
    OrderService --> GRPC[gRPC Server]
```

Wire 的作用就是按照这条依赖关系创建对象。业务代码不需要自己 `new` 数据库、repository 或 service。

## 金额与状态

金额使用分作为单位，并使用 `int64`。例如 19.99 元保存为 1999。不能使用浮点数表示金额。

数据库中的金额约束是 `amount_cents BIGINT NOT NULL CHECK (amount_cents > 0)`。

订单状态在数据库中使用字符串，例如 `PENDING`、`PROCESSING`、`SHIPPED`、`DELIVERED`、`CANCELLED`，不保存 `0、1、2`。Go 内部可以使用枚举判断，repository 读写时负责字符串转换。

## 启动数据库

```powershell
docker compose -f deploy/docker-compose/docker-compose.yml up -d postgres
make migrate-up
```

默认 DSN 是 `postgres://payment:payment@localhost:5432/payment?sslmode=disable`。

## 生成代码

修改 protobuf 后运行 `make api`；修改 SQL 或迁移后运行 `make sqlc`；修改 Wire provider 后运行 `make wire`。不要手动修改 `*_pb.go`、sqlc 输出目录和 `wire_gen.go`。

## 启动服务

```powershell
make run-commerce
```

默认 HTTP 地址是 `localhost:8000`，gRPC 地址是 `localhost:9000`。健康检查接口为 `GET http://localhost:8000/healthz`。

## 订单 API

创建订单时客户端只提供用户和金额，不提供状态：

```json
{"user_id":1001,"amount_cents":1999}
```

```powershell
Invoke-RestMethod -Method Post -Uri http://localhost:8000/v1/orders -ContentType "application/json" -Body '{"user_id":1001,"amount_cents":1999}'
```

查询订单：`GET http://localhost:8000/v1/orders/{order_no}`。

服务端创建订单时始终设置 `PENDING`。金额为零或负数会被拒绝，重复 `order_no` 会被数据库唯一约束拒绝。

### 创建订单时序

```mermaid
sequenceDiagram
    participant C as 客户端
    participant T as order.Service
    participant U as UseCase
    participant R as Repository
    participant D as PostgreSQL

    C->>T: CreateOrder(user_id, amount_cents)
    T->>U: Create(CreateOrderInput)
    U->>U: 校验 amount_cents > 0
    U->>U: 生成 order_no
    U->>U: 设置 status = PENDING
    U->>R: Create(order)
    R->>D: INSERT orders
    D-->>R: 返回订单
    R-->>U: Order
    U-->>T: Order
    T-->>C: CreateOrderResponse
```

如果金额不合法，usecase 会在调用 repository 之前返回错误；如果订单号重复，数据库的唯一约束会拒绝插入。

### 订单状态机

```mermaid
stateDiagram-v2
    [*] --> PENDING: 创建订单
    PENDING --> PROCESSING: 开始处理
    PENDING --> CANCELLED: 取消订单
    PROCESSING --> SHIPPED: 发货
    PROCESSING --> CANCELLED: 取消订单
    SHIPPED --> DELIVERED: 完成配送
    DELIVERED --> [*]
    CANCELLED --> [*]
```

状态更新使用“来源状态 + 目标状态”的条件更新。终态 `DELIVERED` 和 `CANCELLED` 没有出边，不能再次修改。

## 测试

```powershell
go test -race ./services/commerce/internal/order/...
make unit-test
make race-test
make vet
```

订单测试应覆盖非正金额、合法创建、默认状态、订单号唯一、按订单号查询、非法状态转换和并发状态更新。

## 开发顺序

先写测试，再写模型和业务规则；然后写迁移和 SQL，运行 sqlc；接着实现 repository 和 usecase；最后修改 proto、生成 API、实现 service、注册 HTTP/gRPC、更新 Wire，并运行 `go test -race`。

Go 层校验用于尽早返回清晰错误，数据库约束用于防止遗漏和并发绕过，两者都需要保留。
