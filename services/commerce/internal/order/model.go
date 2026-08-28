package order

import "time"

// 定义订单结构体
type Order struct {
	ID          int64
	OrderNo     string
	UserID      int64
	AmountCents int64
	Status      OrderStatus
	Version     int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CreateOrderInput struct {
	UserID      int64
	AmountCents int64
}
