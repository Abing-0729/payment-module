# commerce / mockpay proto 设计

> 日期：2026-08-18
> 状态：已与用户确认（2026-08-18）
> 关联文档：`docx/构建计划.md`、`docx/issue分配.md`

## 1. 目标

确定 `api/commerce/v1` 与 `api/mockpay/v1` 的 proto 定义方式：文件组织、定义范围（按 Issue 增量）以及跨 Issue 复用的一致性约定。本设计只定义契约骨架，实现交给对应 Issue。

## 2. 关键决策

| 决策 | 结论 | 理由 |
| --- | --- | --- |
| 定义范围 | 按 Issue 增量定义，每个 Issue 补充对应域 | 与 issue分配.md 的节奏一致，diff 干净 |
| 文件组织 | 按域拆多文件，同包 `commerce.v1` | 与 `internal/` 模块结构对应，每个 Issue 只动自己的文件 |
| 健康检查 | 两个 proto 各自定义自定义 `Health` service，`Check` RPC | 计划 Issue 1 明确要求"定义两个服务的健康检查API" |
| 空域文件 | 不提前创建，等对应 Issue 再加 | 避免空文件占位和生成噪音 |
| 回调 | 不进 proto，用原始 HTTP handler（如 `/v1/callback/wechat`） | 需要验签和保留 `raw_payload`（JSONB），protobuf 解析 body 反而碍事 |
| 双协议 | 每个 RPC 都加 `google.api.http` 注解，一份 proto 同时生成 HTTP + gRPC | 计划要求 "HTTP/gRPC API"，kratos 标准做法 |

## 3. 文件规划（V1）

```text
api/commerce/v1/
├── commerce.proto   # Issue 1: Health service（健康检查）
├── order.proto      # Issue 5: Order 消息 + OrderStatus 枚举 + CreateOrder/GetOrder
├── payment.proto    # Issue 6: PaymentAttempt + PaymentStatus + CreatePayment + Channel 枚举
├── refund.proto     # Issue 10: Refund + RefundStatus + CreateRefund
├── admin.proto      # Issue 12: 异常查询/补偿 RPC（带 reason 字段）
└── common.proto     # 需要时再加（如分页），Issue 1 不建
api/mockpay/v1/
└── mockpay.proto    # Issue 1: Health；Issue 7: 渠道交易/查询/退款 RPC + 场景枚举
```

所有文件同包（`commerce.v1` / `mockpay.v1`），引用只需普通 import 语句，无跨包命名和重复定义问题。

## 4. 全局约定（后续所有 Issue 照做）

### 命名

- RPC：动词+名词，如 `CreateOrder`、`GetOrder`、`CreatePayment`、`CreateRefund`、`RetryFulfillment`。
- 请求/响应消息：`CreateOrderRequest` / `CreateOrderResponse`，放在对应域文件。
- 状态枚举：`OrderStatus` / `PaymentStatus` / `RefundStatus`，枚举值带类型前缀（如 `ORDER_STATUS_CREATED`），避免不同域枚举值冲突。

### 类型

- 金额一律 `int64` 分，字段名 `amount_cents`，与数据库字段一致（计划第 5 节）。
- 时间用 `google.protobuf.Timestamp`。
- 幂等键走 `Idempotency-Key` header，**不进** request body（计划第 7 节）。
- user 信息只保留 `user_id` 引用字段：V1 无用户表/服务（计划明确删除旧 User/Product 依赖），不定义带 name/phone 等无从填充字段的 `User` 消息。

### 错误码

- 稳定业务错误（`IDEMPOTENCY_KEY_REUSED`、非法状态转换等）定义在 Go 侧 `services/commerce/internal/platform/errors`（计划 Issue 4），proto 不定义错误枚举，用 kratos `errors.Reason` 字符串。

### Channel 枚举

- `WECHAT` / `ALIPAY` 两个值，commerce 的 payment.proto 与 mockpay.proto **各自定义一份**，互不 import，避免服务间耦合。

### 健康检查分级

- Issue 1（无 DB/Redis）：存活级，进程活着即返回 200。
- Issue 2 接入 PostgreSQL 后：升级为就绪检查（`Check` 内部 ping DB/Redis）。

## 5. Issue 1 落地内容

- `api/commerce/v1/commerce.proto`：包 `commerce.v1`，`Health` service + `Check` RPC（返回服务名和状态）。
- `api/mockpay/v1/mockpay.proto`：同样的 Health 骨架。
- `make api` 生成流水线（buf/protoc），保证两个 proto 可编译生成。
- 不提前创建 order/payment/refund/admin 空文件。

## 6. 边界与待办

- 回调具体路径和验签方案在 Issue 7/8 设计。
- Admin 分页结构（如需）在 Issue 12 定义到 common.proto。
- 后续每个 Issue 新增 proto 内容时，需同步确认本设计的命名与类型约定未被破坏。
