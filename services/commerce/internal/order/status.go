package order

import errorpb "kratos-payment-lab/services/commerce/internal/platform/errors"

type OrderStatus int32

// 状态定义，每个状态整型自增
const (
	OrderStatusPending OrderStatus = iota
	OrderStatusProcessing
	OrderStatusShipped
	OrderStatusDelivered
	OrderStatusCancelled
)

// 状态转换表（正向允许）
var allowedTransitions = map[OrderStatus][]OrderStatus{
	OrderStatusPending:    {OrderStatusProcessing, OrderStatusCancelled},
	OrderStatusProcessing: {OrderStatusShipped, OrderStatusCancelled},
	OrderStatusShipped:    {OrderStatusDelivered},
	// 已交付和已取消不允许再变
}

func CanTransitionOrder(from, to OrderStatus) bool {
	allowed := allowedTransitions[from]
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

func ValidateOrderTransition(from, to OrderStatus) error {
	if from == to {
		return errorpb.New(errorpb.INVALID_ARGUMENT, "same status transition is not allowed")
	}
	if !CanTransitionOrder(from, to) {
		return errorpb.ErrorOrderInvaildTransaction
	}
	return nil
}
