package payment

import errorpb "kratos-payment-lab/services/commerce/internal/platform/errors"

type PaymentStatus int32

// 状态定义，每个状态整型自增
const (
	PaymentStatusPending PaymentStatus = iota
	PaymentStatusProcessing
	PaymentStatusSucceeded
	PaymentStatusFailed
)

// CanTransitionPayment 判断状态转换是否合法。
// 每个状态一个 case 显式声明允许去向；终态（成功/失败）无出边，结构上不可变。
// Pending 发起支付，Processing 收网关结果；重试走新建支付单。
func CanTransitionPayment(from, to PaymentStatus) bool {
	switch from {
	case PaymentStatusPending:
		return to == PaymentStatusProcessing || to == PaymentStatusFailed
	case PaymentStatusProcessing:
		return to == PaymentStatusSucceeded || to == PaymentStatusFailed
	case PaymentStatusSucceeded, PaymentStatusFailed:
		return false // 终态不可变
	default:
		return false // 未知状态
	}
}

func ValidatePaymentTransition(from, to PaymentStatus) error {
	if from == to {
		return errorpb.New(errorpb.INVALID_ARGUMENT, "same status transition is not allowed")
	}
	if !CanTransitionPayment(from, to) {
		return errorpb.ErrorPaymentInvalidTransaction
	}
	return nil
}
