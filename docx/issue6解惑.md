可以把这段代码理解成“给支付请求加身份证”。

### 1. `CreatePaymentCommand` 是本次请求

```go
type CreatePaymentCommand struct {
    OrderNo        string
    AmountCents    int64
    Channel        PaymentChannel
    IdempotencyKey string
}
```

它代表客户端这一次想做什么：

```text
订单 O100
金额 1000 分
渠道 ALIPAY
幂等键 abc123
```

API 请求会先转换成这个结构，再交给业务层。

### 2. `Fingerprint` 是请求指纹

```go
current := Fingerprint(req)
```

指纹就是把订单号、金额、渠道计算成一个 SHA-256 字符串：

```text
O100 + 1000 + ALIPAY
    -> 8f3a...
```

相同参数一定得到相同指纹：

```text
O100 + 1000 + ALIPAY -> AAA
O100 + 1000 + ALIPAY -> AAA
```

参数有变化，指纹就会变化：

```text
O100 + 2000 + ALIPAY -> BBB
```

幂等键不参与指纹计算，因为幂等键是“用来查旧请求的索引”，不是业务参数。

### 3. 第一次请求怎么处理

第一次使用 `abc123`：

```text
数据库查询 abc123
    -> 没找到
    -> 创建支付记录
    -> 保存 fingerprint=AAA
    -> 返回 Replayed=false
```

代码对应：

```go
payment := &PaymentAttempt{
    OrderNo:            req.OrderNo,
    AmountCents:        req.AmountCents,
    Channel:            req.Channel,
    IdempotencyKey:     req.IdempotencyKey,
    RequestFingerprint: current,
    Status:             PaymentStatusPending,
}
```

这里把本次请求和指纹一起保存到数据库。

### 4. 相同请求重试

再次使用 `abc123`，参数没变：

```text
数据库查到 abc123
已有指纹 AAA
当前指纹 AAA
    -> 说明是同一个请求
    -> 返回原来的支付记录
    -> Replayed=true
```

所以不会重复创建支付单。

### 5. 同一个键但参数变化

再次使用 `abc123`，但金额改成 2000：

```text
数据库查到 abc123
已有指纹 AAA
当前指纹 BBB
    -> 同一个幂等键对应了不同请求
    -> 返回冲突错误
```

对应代码：

```go
if existing.RequestFingerprint != current {
    return IdempotencyResult{}, apperrors.ErrorIdempotencyKeyReused
}
```

这就是防止客户端错误地复用幂等键。

### 6. 为什么还要数据库唯一约束

仅仅这样写是不够的：

```text
请求 A：查询，没有记录
请求 B：查询，没有记录
请求 A：创建
请求 B：创建
```

这样可能创建两条记录。

所以数据库表必须有：

```sql
UNIQUE (idempotency_key)
```

当两个请求同时创建时，数据库保证只有一个成功，另一个会收到 PostgreSQL 错误 `23505`，表示唯一键冲突。

代码再重新查询：

```text
插入失败，原因是唯一键冲突
    -> 重新查询 abc123
    -> 比较指纹
    -> 相同：返回已有支付单，Replayed=true
```

这就是并发安全的关键。

### 7. `PaymentChannel` 为什么不能使用 `iota`

原来是：

```go
type PaymentChannel string

const (
    PaymentChannelUnknown PaymentChannel = iota
)
```

`iota` 生成的是整数：

```text
0、1、2
```

但 `PaymentChannel` 的底层类型是 `string`，所以类型不匹配。

现在改成：

```go
const (
    PaymentChannelUnknown PaymentChannel = "UNKNOWN"
    PaymentChannelAlipay  PaymentChannel = "ALIPAY"
    PaymentChannelWechat  PaymentChannel = "WECHAT"
)
```

这样数据库里也能直接保存：

```text
ALIPAY
WECHAT
```

### 8. 整体流程

```text
客户端请求
    |
    v
CreatePaymentCommand
    |
    v
计算 fingerprint
    |
    v
按 idempotency_key 查询
    |
    +-- 找到 + 指纹相同 --> 返回旧支付单，Replayed=true
    |
    +-- 找到 + 指纹不同 --> 返回幂等键冲突
    |
    +-- 没找到 ----------> 创建支付单，Replayed=false
                              |
                              +-- 并发唯一键冲突
                                      |
                                      v
                                重新查询并重放
```

你现在重点只需要记住一句：

> 幂等键负责找到“之前的请求”，请求指纹负责判断“这次是不是同一个请求”。
