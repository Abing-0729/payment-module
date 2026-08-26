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

// 状态转换表（正向允许）：Pending 发起支付，Processing 收网关结果，
// 失败/成功均为终态，重试走新建支付单
var allowedTransitions = map[PaymentStatus][]PaymentStatus{
	PaymentStatusPending:    {PaymentStatusProcessing, PaymentStatusFailed},
	PaymentStatusProcessing: {PaymentStatusSucceeded, PaymentStatusFailed},
	// Succeeded 和 Failed 不允许再变
}

func CanTransitionPayment(from, to PaymentStatus) bool {
	allowed := allowedTransitions[from]
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
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
