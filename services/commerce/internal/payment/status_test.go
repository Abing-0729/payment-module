package payment

import (
	"errors"
	"testing"

	errorpb "kratos-payment-lab/api/commerce/v1/errors"
	bizerr "kratos-payment-lab/services/commerce/internal/platform/errors"
)

// paymentTransitionExpectation 期望的支付状态转换结果（行=from，列=to），
// 独立于实现硬编码，穷举所有组合
var paymentTransitionExpectation = map[PaymentStatus]map[PaymentStatus]bool{
	PaymentStatusPending: {
		PaymentStatusPending:    false, // 相同状态
		PaymentStatusProcessing: true,
		PaymentStatusSucceeded:  false, // 跳级：必须经网关回调
		PaymentStatusFailed:     true,  // 超时/取消
	},
	PaymentStatusProcessing: {
		PaymentStatusPending:    false, // 回退
		PaymentStatusProcessing: false, // 相同状态
		PaymentStatusSucceeded:  true,  // 网关成功回调
		PaymentStatusFailed:     true,  // 网关失败回调
	},
	PaymentStatusSucceeded: {
		PaymentStatusPending:    false,
		PaymentStatusProcessing: false, // 回退（issue 场景：SUCCEEDED -> PROCESSING）
		PaymentStatusSucceeded:  false, // 相同状态
		PaymentStatusFailed:     false, // 终态：成功不可变失败
	},
	PaymentStatusFailed: {
		PaymentStatusPending:    false,
		PaymentStatusProcessing: false, // 终态：失败重试走新建支付单
		PaymentStatusSucceeded:  false,
		PaymentStatusFailed:     false, // 相同状态
	},
	// 未知状态：任何转换都不允许
	PaymentStatus(99): {},
}

func TestCanTransitionPayment(t *testing.T) {
	states := []PaymentStatus{
		PaymentStatusPending,
		PaymentStatusProcessing,
		PaymentStatusSucceeded,
		PaymentStatusFailed,
		PaymentStatus(99), // 未知状态
	}

	for _, from := range states {
		for _, to := range states {
			want := paymentTransitionExpectation[from][to]
			if got := CanTransitionPayment(from, to); got != want {
				t.Errorf("CanTransitionPayment(%v, %v) = %v, want %v", from, to, got, want)
			}
		}
	}
}

func TestValidatePaymentTransition(t *testing.T) {
	t.Run("合法转换返回 nil", func(t *testing.T) {
		valid := []struct{ from, to PaymentStatus }{
			{PaymentStatusPending, PaymentStatusProcessing},
			{PaymentStatusPending, PaymentStatusFailed},
			{PaymentStatusProcessing, PaymentStatusSucceeded},
			{PaymentStatusProcessing, PaymentStatusFailed},
		}
		for _, tc := range valid {
			if err := ValidatePaymentTransition(tc.from, tc.to); err != nil {
				t.Errorf("ValidatePaymentTransition(%v, %v) = %v, want nil", tc.from, tc.to, err)
			}
		}
	})

	t.Run("状态回退", func(t *testing.T) {
		backward := []struct{ from, to PaymentStatus }{
			{PaymentStatusSucceeded, PaymentStatusProcessing},
			{PaymentStatusSucceeded, PaymentStatusPending},
			{PaymentStatusProcessing, PaymentStatusPending},
			{PaymentStatusFailed, PaymentStatusProcessing},
			{PaymentStatusFailed, PaymentStatusPending},
		}
		for _, tc := range backward {
			assertPaymentTransitionError(t, tc.from, tc.to, bizerr.PaymentInvalidStatusTransition)
		}
	})

	t.Run("跳级", func(t *testing.T) {
		skip := []struct{ from, to PaymentStatus }{
			{PaymentStatusPending, PaymentStatusSucceeded},
			{PaymentStatusFailed, PaymentStatusSucceeded},
		}
		for _, tc := range skip {
			assertPaymentTransitionError(t, tc.from, tc.to, bizerr.PaymentInvalidStatusTransition)
		}
	})

	t.Run("终态不允许再变", func(t *testing.T) {
		terminal := []struct{ from, to PaymentStatus }{
			{PaymentStatusSucceeded, PaymentStatusFailed},
			{PaymentStatusFailed, PaymentStatusSucceeded},
		}
		for _, tc := range terminal {
			assertPaymentTransitionError(t, tc.from, tc.to, bizerr.PaymentInvalidStatusTransition)
		}
	})

	t.Run("相同状态重复事件", func(t *testing.T) {
		same := []PaymentStatus{
			PaymentStatusPending,
			PaymentStatusProcessing,
			PaymentStatusSucceeded,
			PaymentStatusFailed,
		}
		for _, s := range same {
			assertPaymentTransitionError(t, s, s, bizerr.INVALID_ARGUMENT)
		}
	})

	t.Run("未知状态", func(t *testing.T) {
		assertPaymentTransitionError(t, PaymentStatus(99), PaymentStatusPending, bizerr.PaymentInvalidStatusTransition)
		assertPaymentTransitionError(t, PaymentStatusPending, PaymentStatus(100), bizerr.PaymentInvalidStatusTransition)
	})
}

func assertPaymentTransitionError(t *testing.T, from, to PaymentStatus, wantCode errorpb.ErrorCode) {
	t.Helper()
	err := ValidatePaymentTransition(from, to)
	if err == nil {
		t.Fatalf("ValidatePaymentTransition(%v, %v) = nil, want error", from, to)
	}
	var be *bizerr.BusinessError
	if !errors.As(err, &be) {
		t.Fatalf("ValidatePaymentTransition(%v, %v) 返回类型 %T, want *BusinessError", from, to, err)
	}
	if be.Code != wantCode {
		t.Errorf("ValidatePaymentTransition(%v, %v) code = %v, want %v", from, to, be.Code, wantCode)
	}
}
