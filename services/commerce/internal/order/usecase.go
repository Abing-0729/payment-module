package order

import "kratos-payment-lab/services/commerce/internal/platform/postgres"

// 负责业务规则和事务边界

type CreateOrderRequest struct {
	UserID      int64
	AmountCents int64
}

type UseCase struct {
	repo       Repository
	transactor postgres.Transactor
}
