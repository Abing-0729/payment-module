package order

import (
	errorpb "kratos-payment-lab/services/commerce/internal/platform/errors"
)

type OrderStatus int32

// 状态定义，每个状态整型自增
const (
	OrderStatusPending OrderStatus = iota
	OrderStatusProcessing
	OrderStatusShipped
	OrderStatusDelivered
	OrderStatusCancelled
)

func (s OrderStatus) String() string {
	switch s {
	case OrderStatusPending:
		return "PENDING"
	case OrderStatusProcessing:
		return "PROCESSING"
	case OrderStatusShipped:
		return "SHIPPED"
	case OrderStatusDelivered:
		return "DELIVERED"
	case OrderStatusCancelled:
		return "CANCELLED"
	default:
		return "UNKNOWN"
	}
}
func ParseOrderStatus(value string) (OrderStatus, error) {
	switch value {
	case "PENDING":
		return OrderStatusPending, nil
	case "PROCESSING":
		return OrderStatusProcessing, nil
	case "SHIPPED":
		return OrderStatusShipped, nil
	case "DELIVERED":
		return OrderStatusDelivered, nil
	case "CANCELLED":
		return OrderStatusCancelled, nil
	default:
		return 0, errorpb.New(errorpb.INVALID_ARGUMENT, "unknown order status")
	}
}

// CanTransitionOrder 判断状态转换是否合法。
// 每个状态一个 case 显式声明允许去向；终态（已交付/已取消）无出边，结构上不可变。
func CanTransitionOrder(from, to OrderStatus) bool {
	switch from {
	case OrderStatusPending:
		return to == OrderStatusProcessing || to == OrderStatusCancelled
	case OrderStatusProcessing:
		return to == OrderStatusShipped || to == OrderStatusCancelled
	case OrderStatusShipped:
		return to == OrderStatusDelivered
	case OrderStatusDelivered, OrderStatusCancelled:
		return false // 终态不可变
	default:
		return false // 未知状态
	}
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
