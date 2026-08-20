# Payment and Fulfillment System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a portfolio-grade Go payment and fulfillment backend with idempotent ordering, asynchronous callbacks, refunds, reliable fulfillment, reconciliation, and failure evidence.

**Architecture:** A Kratos v2 modular monolith owns order, payment, refund, fulfillment, reconciliation, and admin domains inside one PostgreSQL transaction boundary. An independent MockPay service simulates WeChat and Alipay; Redis is auxiliary, while PostgreSQL constraints and conditional updates preserve correctness.

**Tech Stack:** Go, Kratos v2, Protobuf, HTTP/gRPC, PostgreSQL 16, pgx, sqlc, goose, Redis, Docker Compose, testcontainers-go, GitHub Actions; RabbitMQ is optional V2 work.

**Spec:** `outputs/payment-system-design.md`

## Global Constraints

- Use integer cents (`int64`/`BIGINT`) for every monetary value; never use floating point.
- Do not use real payment credentials, company source code, internal names, production payloads, or proprietary documentation.
- PostgreSQL transactions, unique constraints, and conditional updates are the correctness source of truth.
- Redis may accelerate reads, rate limits, and duplicate detection, but Redis failure must not break payment idempotency.
- V1 remains a modular monolith plus an independent MockPay service; no premature commerce-service split.
- External HTTP/gRPC calls must not execute while holding a PostgreSQL transaction open.
- RabbitMQ, Kubernetes, and a graphical admin frontend do not block the V1 release.
- Every Issue must pass its listed tests and complete its AI collaboration checkpoint before closure.

---

> 建议仓库名：`kratos-payment-lab`
>
> 实施方式：按 Milestone 顺序执行。每个 Issue 都必须独立提交、通过验收命令，并完成“AI协作检查点”。

## Labels

- `type:feature`
- `type:test`
- `type:docs`
- `type:infra`
- `area:order`
- `area:payment`
- `area:refund`
- `area:fulfillment`
- `area:admin`
- `area:mockpay`
- `priority:p0`
- `priority:p1`

## Milestone M0: Foundation

### Issue 1: 重构为 Commerce 模块化单体并保留 MockPay 服务边界

**Labels:** `type:infra`, `priority:p0`

**目标**

建立 `commerce` 和 `mockpay` 两个可独立启动的 Kratos 服务。`commerce`内部预留 order/payment/refund/fulfillment/reconciliation/admin 模块，不实现业务。

**文件**

- 新建 `api/commerce/v1/commerce.proto`
- 新建 `api/mockpay/v1/mockpay.proto`
- 新建 `services/commerce/cmd/commerce/{main.go,wire.go}`
- 新建 `services/commerce/internal/{server,order,payment,refund,fulfillment,reconciliation,admin,platform}`
- 新建 `services/mockpay/cmd/mockpay/{main.go,wire.go}`
- 修改 `Makefile`

**任务**

- [x] 定义两个服务的健康检查API。
- [X] 为两个服务建立独立配置和端口。
- [X] Wire生成依赖组装代码。
- [X] `make build`同时构建两个服务。
- [X] 删除新业务对旧User/Product模块的依赖；旧模块暂不删除。

**测试场景**

- commerce健康检查返回200；
- mockpay健康检查返回200；
- 任一服务停止不会阻止另一个进程启动。

**验收**

```powershell
make api
make wire
make build
go test -race ./services/commerce/... ./services/mockpay/...
```

**AI协作检查点**

- [X] 自己解释 `main.go -> wire -> server` 的依赖创建顺序。
- [X] 暂时移除一个Provider，确认Wire生成会失败，再恢复。

---

### Issue 2: 接入 PostgreSQL、goose 与 sqlc

**Labels:** `type:infra`, `priority:p0`

**Depends on:** Issue 1

**目标**

建立可重复执行的数据库迁移和类型安全查询生成流程。

**文件**

- 新建 `db/migrations/00001_init.sql`
- 新建 `db/queries/health.sql`
- 新建 `db/sqlc.yaml`
- 新建 `services/commerce/internal/platform/postgres/postgres.go`
- 新建 `services/commerce/internal/platform/postgres/tx.go`
- 修改 `deploy/docker-compose/docker-compose.yml`
- 修改 `Makefile`

**接口**

```go
type Transactor interface {
    WithinTransaction(ctx context.Context, fn func(context.Context) error) error
}
```

**任务**

- [ ] Docker Compose增加PostgreSQL 16。
- [ ] goose提供 `make migrate-up` 和 `make migrate-down`。
- [ ] sqlc提供 `make sqlc`，生成代码不允许手改。
- [ ] 配置pgx连接池、最大连接数和连接超时。
- [ ] 使用事务上下文保证同一业务事务复用同一连接。

**测试场景**

- 空数据库可完整迁移；
- down后可重新up；
- 事务回调返回错误时数据回滚；
- context取消时事务终止。

**验收**

```powershell
docker compose -f deploy/docker-compose/docker-compose.yml up -d postgres
make migrate-up
make sqlc
go test -race ./services/commerce/internal/platform/postgres/...
```

**AI协作检查点**

- [ ] 自己用SQL插入一行并在事务中制造错误，观察回滚。
- [ ] 解释为什么外部HTTP调用不能放进数据库事务。

---

### Issue 3: 建立 CI 和统一质量命令

**Labels:** `type:infra`, `type:test`, `priority:p0`

**Depends on:** Issue 2

**目标**

每次提交自动验证生成代码、迁移、测试、竞态和构建。

**文件**

- 新建 `.github/workflows/ci.yml`
- 修改 `Makefile`
- 新建 `scripts/verify-generated.ps1`

**任务**

- [ ] CI启动PostgreSQL和Redis服务容器。
- [ ] 执行 `buf generate`、`sqlc generate`和Wire生成校验。
- [ ] 执行迁移、`go vet`、`go test -race`和构建。
- [ ] 生成代码后工作树有差异时CI失败。

**验收**

```powershell
make verify
```

预期：本地和CI均成功；手动修改生成文件后 `make verify` 失败。

---

## Milestone M1: Payment Core

### Issue 4: 实现订单与支付状态机

**Labels:** `type:feature`, `area:order`, `area:payment`, `priority:p0`

**Depends on:** Issue 2

**目标**

用纯Go领域逻辑定义合法状态转换，非法转换返回稳定业务错误。

**文件**

- 新建 `services/commerce/internal/order/status.go`
- 新建 `services/commerce/internal/order/status_test.go`
- 新建 `services/commerce/internal/payment/status.go`
- 新建 `services/commerce/internal/payment/status_test.go`
- 新建 `services/commerce/internal/platform/errors/errors.go`

**接口**

```go
func CanTransitionOrder(from, to OrderStatus) bool
func CanTransitionPayment(from, to PaymentStatus) bool
func ValidateOrderTransition(from, to OrderStatus) error
func ValidatePaymentTransition(from, to PaymentStatus) error
```

**测试场景**

- 所有允许的正向转换；
- `SUCCEEDED -> PROCESSING`等状态回退；
- 相同状态的重复事件；
- 未知状态。

**验收**

```powershell
go test -race ./services/commerce/internal/order/... ./services/commerce/internal/payment/...
```

**AI协作检查点**

- [ ] 先独立画状态转换表，再让AI生成表驱动测试。
- [ ] 故意允许一条非法回退，确认测试失败。

---

### Issue 5: 实现订单创建与查询

**Labels:** `type:feature`, `area:order`, `priority:p0`

**Depends on:** Issues 2, 4

**目标**

客户端可以创建金额合法的业务订单并按订单号查询。

**文件**

- 修改 `db/migrations/00001_init.sql`
- 新建 `db/queries/orders.sql`
- 新建 `services/commerce/internal/order/model.go`
- 新建 `services/commerce/internal/order/usecase.go`
- 新建 `services/commerce/internal/order/repository.go`
- 新建 `services/commerce/internal/order/service.go`
- 新建对应单元和集成测试

**接口**

```go
type OrderRepository interface {
    Create(ctx context.Context, order *Order) error
    GetByOrderNo(ctx context.Context, orderNo string) (*Order, error)
    Transition(ctx context.Context, orderNo string, from []OrderStatus, to OrderStatus) (bool, error)
}
```

**验收条件**

- 金额使用`int64`分；
- `amount_cents <= 0`被拒绝；
- `order_no`唯一；
- API不接受客户端指定订单状态。

**验收**

```powershell
go test -race ./services/commerce/internal/order/...
```

---

### Issue 6: 实现支付幂等键与请求指纹

**Labels:** `type:feature`, `area:payment`, `priority:p0`

**Depends on:** Issues 4, 5

**目标**

相同请求可以安全重试，同一幂等键不能用于不同请求。

**文件**

- 新建 `db/queries/payment_attempts.sql`
- 新建 `services/commerce/internal/payment/fingerprint.go`
- 新建 `services/commerce/internal/payment/idempotency.go`
- 新建对应单元和PostgreSQL集成测试

**接口**

```go
func Fingerprint(req CreatePaymentCommand) string

type IdempotencyResult struct {
    Payment *PaymentAttempt
    Replayed bool
}
```

**测试场景**

- 相同键相同参数返回原支付单；
- 相同键不同金额、订单或渠道返回冲突；
- 20个并发请求只创建一条记录；
- Redis停止后结果仍正确。

**验收**

```powershell
go test -race ./services/commerce/internal/payment/... -run Idempotency
```

**AI协作检查点**

- [ ] 自己解释唯一约束如何解决“先查后写”竞态。
- [ ] 删除唯一约束并运行并发测试，确认测试能暴露重复数据，再恢复。

---

### Issue 7: 实现统一 PaymentGateway 与双渠道 MockPay

**Labels:** `type:feature`, `area:payment`, `area:mockpay`, `priority:p0`

**Depends on:** Issues 1, 6

**目标**

commerce不依赖具体渠道SDK；mockpay模拟微信和支付宝的渠道交易。

**文件**

- 新建 `services/commerce/internal/payment/gateway.go`
- 新建 `services/commerce/internal/payment/gateway_client.go`
- 新建 `services/mockpay/internal/channel/{gateway.go,wechat.go,alipay.go}`
- 新建 `services/mockpay/internal/store/store.go`
- 修改 `api/mockpay/v1/mockpay.proto`

**接口**

```go
type PaymentGateway interface {
    CreatePayment(context.Context, CreateGatewayPayment) (*GatewayPayment, error)
    QueryPayment(context.Context, string) (*GatewayPayment, error)
    CreateRefund(context.Context, CreateGatewayRefund) (*GatewayRefund, error)
    QueryRefund(context.Context, string) (*GatewayRefund, error)
}
```

**测试场景**

- 微信和支付宝返回不同前缀的渠道流水号；
- 相同`payment_no`重复创建返回同一渠道交易；
- 支持success、failure和response_timeout_after_success；
- commerce可以通过统一接口查询两个渠道。

**验收**

```powershell
go test -race ./services/mockpay/... ./services/commerce/internal/payment/...
```

---

### Issue 8: 实现支付回调事务与重复通知处理

**Labels:** `type:feature`, `area:payment`, `priority:p0`

**Depends on:** Issues 5, 6, 7

**目标**

回调原子更新支付、订单、履约任务和Outbox；重复回调只处理一次。

**文件**

- 新建 `db/queries/payment_callbacks.sql`
- 新建 `db/queries/fulfillment_tasks.sql`
- 新建 `db/queries/outbox_events.sql`
- 新建 `services/commerce/internal/payment/callback.go`
- 新建 `services/commerce/internal/payment/callback_test.go`
- 新建 `services/commerce/internal/payment/callback_integration_test.go`

**事务结果**

```text
callback + payment SUCCEEDED + order PAID + one fulfillment task + outbox event
```

**测试场景**

- 正常回调完成全部本地写入；
- 同一回调并发提交十次，只产生一条履约任务；
- 金额不一致时不更新状态；
- 中间步骤故障时事务全部回滚；
- 已退款支付收到迟到成功回调时状态不回退。

**验收**

```powershell
go test -race ./services/commerce/internal/payment/... -run Callback
```

**AI协作检查点**

- [ ] 给事务中的某一步注入错误，亲自查询五张表确认全部回滚。
- [ ] 故意移除履约任务唯一约束，确认重复回调测试失败，再恢复。

---

## Milestone M2: Reliability

### Issue 9: 实现数据库履约任务 Worker

**Labels:** `type:feature`, `area:fulfillment`, `priority:p0`

**Depends on:** Issue 8

**目标**

多个Worker并发运行时不重复领取任务，失败任务按退避策略重试。

**文件**

- 新建 `services/commerce/internal/fulfillment/worker.go`
- 新建 `services/commerce/internal/fulfillment/backoff.go`
- 新建 `services/commerce/internal/fulfillment/executor.go`
- 新建对应单元和集成测试

**接口**

```go
type Executor interface {
    Fulfill(context.Context, Order) error
}

func NextRetryAt(now time.Time, attempts int) time.Time
```

**测试场景**

- 两个Worker不会领取同一任务；
- 成功后订单进入COMPLETED；
- 前两次失败第三次成功；
- 超过最大次数进入FULFILL_FAILED；
- Worker崩溃后锁释放，任务可以再次领取。

**验收**

```powershell
go test -race ./services/commerce/internal/fulfillment/...
```

---

### Issue 10: 实现退款状态机与幂等退款

**Labels:** `type:feature`, `area:refund`, `priority:p0`

**Depends on:** Issues 6, 7, 8

**目标**

已支付订单可以发起幂等退款，防止重复退款和超额退款。

**文件**

- 新建 `db/queries/refunds.sql`
- 新建 `services/commerce/internal/refund/{model.go,status.go,usecase.go,repository.go,service.go}`
- 新建对应单元和集成测试

**验收条件**

- 只有允许的订单状态可以退款；
- 退款金额必须大于0且不超过可退金额；
- 相同幂等键相同参数返回第一次结果；
- 回调重复时退款状态不重复推进；
- 支持查询渠道退款状态。

**验收**

```powershell
go test -race ./services/commerce/internal/refund/...
```

---

### Issue 11: 实现支付和退款主动对账

**Labels:** `type:feature`, `area:payment`, `area:refund`, `priority:p1`

**Depends on:** Issues 7, 8, 10

**目标**

修复回调丢失、接口响应丢失和长时间处理中状态。

**文件**

- 新建 `services/commerce/internal/reconciliation/scanner.go`
- 新建 `services/commerce/internal/reconciliation/reconciler.go`
- 新建对应测试

**测试场景**

- MockPay支付成功但不发回调，扫描后修正为成功；
- 查询超时不创建新支付单；
- 本地已经终态时不重复修正；
- 退款回调丢失后可修正退款状态；
- 扫描批次有大小和时间范围限制。

**验收**

```powershell
go test -race ./services/commerce/internal/reconciliation/...
```

---

### Issue 12: 实现 Admin 异常处理API

**Labels:** `type:feature`, `area:admin`, `priority:p1`

**Depends on:** Issues 9, 10, 11

**目标**

提供只读查询和受控人工补偿入口，不允许直接任意修改数据库状态。

**文件**

- 修改 `api/commerce/v1/commerce.proto`
- 新建 `services/commerce/internal/admin/service.go`
- 新建 `services/commerce/internal/admin/usecase.go`
- 新建对应API测试

**API**

- 查询异常支付单；
- 查询回调记录；
- 主动同步渠道状态；
- 重试履约；
- 重试退款；
- 关闭超时未支付订单。

**验收条件**

- 补偿接口必须接收`reason`；
- 每次操作记录operator、reason、request_id和结果；
- 不提供“直接把订单改成成功”的接口；
- 重复补偿请求保持幂等。

**验收**

```powershell
go test -race ./services/commerce/internal/admin/...
```

---

## Milestone M3: Production Evidence

### Issue 13: 建立端到端故障场景测试

**Labels:** `type:test`, `priority:p0`

**Depends on:** Issues 8-12

**目标**

使用Docker Compose证明核心闭环和故障恢复，而不是只证明单个函数通过测试。

**文件**

- 新建 `scripts/e2e-payment.ps1`
- 新建 `scripts/e2e-refund.ps1`
- 新建 `scripts/e2e-failures.ps1`
- 修改 `Makefile`

**必须覆盖**

- 下单、支付、回调、履约成功；
- 十次重复回调只发货一次；
- 回调丢失后主动查询恢复；
- 渠道成功但响应超时后复用原支付单；
- 履约失败后重试成功；
- 支付成功后退款；
- Redis停止后幂等仍正确。

**验收**

```powershell
make run-all
make e2e
go test -race ./...
```

**AI协作检查点**

- [ ] 为每个脚本写出预期数据库最终状态。
- [ ] 手动制造一个错误，确认脚本以非零状态退出。

---

### Issue 14: 增加基础指标与结构化日志

**Labels:** `type:feature`, `priority:p1`

**Depends on:** Issues 8-12

**目标**

出现异常时可以通过指标和trace_id定位支付、回调、履约与退款阶段。

**文件**

- 新建 `services/commerce/internal/platform/metrics/metrics.go`
- 修改现有logger和trace_id middleware
- 新建 `docs/observability.md`

**指标**

- 支付创建总数和耗时；
- 回调接收、重复和失败次数；
- 支付状态修正次数；
- 履约成功、失败和重试次数；
- 退款成功和失败次数；
- 幂等重放与冲突次数。

**验收条件**

- 日志不记录密钥、完整回调敏感信息或个人数据；
- HTTP、gRPC和Worker日志可以通过trace_id关联；
- 指标标签不使用order_no等高基数字段。

---

### Issue 15: 编写性能与故障报告

**Labels:** `type:test`, `type:docs`, `priority:p1`

**Depends on:** Issues 13, 14

**目标**

用可复现数据说明系统行为，不追求虚高QPS。

**文件**

- 新建 `scripts/load/payment.js`
- 新建 `docs/benchmark.md`
- 新建 `docs/failure-scenarios.md`

**报告必须包含**

- 测试机器配置；
- 数据库和连接池配置；
- 并发数和测试时长；
- P50、P95、P99、吞吐量和错误率；
- Redis开启/关闭差异；
- 重复回调压力下是否重复履约；
- MockPay超时和回调丢失时的恢复时间。

**验收**

```powershell
k6 run scripts/load/payment.js
```

---

### Issue 16: 完善README、架构文档与v1.0.0演示

**Labels:** `type:docs`, `priority:p0`

**Depends on:** Issues 13-15

**目标**

陌生开发者可以在15分钟内理解项目、启动系统并复现一次故障恢复。

**文件**

- 修改 `README.md`
- 新建 `docs/architecture.md`
- 新建 `docs/state-machines.md`
- 新建 `docs/idempotency.md`
- 新建 `docs/testing.md`

**README必须包含**

- 项目解决的问题，而不是技术栈列表；
- 架构与数据流；
- 状态机；
- 五分钟快速启动；
- 正常支付和故障恢复演示；
- 测试命令和测试分层；
- 性能数据链接；
- 项目与实习代码完全独立的声明；
- 当前限制和V2路线图。

**验收**

- 在干净环境按README命令成功启动；
- 创建`v1.0.0`标签前执行`make verify && make e2e`；
- 仓库不包含密钥、真实业务数据和公司内部命名。

---

## Milestone M4: Async Evolution（V2，可选）

### Issue 17: 使用 Transactional Outbox 发布 RabbitMQ 事件

**Labels:** `type:feature`, `area:fulfillment`, `priority:p1`

**Depends on:** Issue 16

**目标**

将已提交的Outbox事件可靠投递至RabbitMQ，允许重复投递但不允许事件永久丢失。

**任务**

- [ ] 使用 `FOR UPDATE SKIP LOCKED`领取Outbox事件。
- [ ] Publisher Confirm成功后标记已发布。
- [ ] 发布失败按退避策略重试。
- [ ] 模拟“RabbitMQ接收成功但数据库标记失败”，验证会重复投递。
- [ ] 消费者使用数据库唯一约束保证幂等。

**验收**

- RabbitMQ停止后事件保留在PostgreSQL；
- RabbitMQ恢复后积压事件最终发布；
- 重复事件不会重复履约；
- V1数据库Worker仍可通过配置启用，方便比较两种方案。

---

### Issue 18: 将 Fulfillment 拆成独立服务（研究性）

**Labels:** `type:feature`, `area:fulfillment`, `priority:p1`

**Depends on:** Issue 17

**目标**

仅在RabbitMQ事件链路稳定后拆分履约服务，并记录拆分前后的收益与成本。

**验收条件**

- commerce不直接访问fulfillment数据库；
- 事件契约有版本字段；
- 消费者重复执行仍保持幂等；
- 文档比较模块化单体与拆分后的部署、故障和调试复杂度；
- 没有可证明收益时允许关闭该Issue，不为微服务而微服务。

## 推荐执行节奏

```text
第1周：Issues 1-4
第2周：Issues 5-7
第3周：Issues 8-10
第4周：Issues 11-13
第5周：Issues 14-16，发布v1.0.0
后续：Issues 17-18
```

每周只要求真正弄懂一类测试：

```text
第1周：纯函数表驱动测试
第2周：Repository集成测试与唯一约束
第3周：事务和并发测试
第4周：端到端故障测试
第5周：性能测试与结果解释
```
