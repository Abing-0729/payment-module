package order

import (
	"errors"
	"testing"

	errorpb "kratos-payment-lab/api/commerce/v1/errors"
	bizerr "kratos-payment-lab/services/commerce/internal/platform/errors"
)

// transitionExpectation 期望的订单状态转换结果（行=from，列=to），
// 独立于实现硬编码，穷举所有组合
var orderTransitionExpectation = map[OrderStatus]map[OrderStatus]bool{
	OrderStatusPending: {
		OrderStatusPending:    false, // 相同状态
		OrderStatusProcessing: true,
		OrderStatusShipped:    false, // 跳级
		OrderStatusDelivered:  false, // 跳级
		OrderStatusCancelled:  true,
	},
	OrderStatusProcessing: {
		OrderStatusPending:    false, // 回退
		OrderStatusProcessing: false, // 相同状态
		OrderStatusShipped:    true,
		OrderStatusDelivered:  false, // 跳级
		OrderStatusCancelled:  true,
	},
	OrderStatusShipped: {
		OrderStatusPending:    false, // 回退
		OrderStatusProcessing: false, // 回退
		OrderStatusShipped:    false, // 相同状态
		OrderStatusDelivered:  true,
		OrderStatusCancelled:  false, // 已发货不允许取消
	},
	OrderStatusDelivered: {
		OrderStatusPending:    false,
		OrderStatusProcessing: false,
		OrderStatusShipped:    false,
		OrderStatusDelivered:  false, // 相同状态
		OrderStatusCancelled:  false, // 终态
	},
	OrderStatusCancelled: {
		OrderStatusPending:    false,
		OrderStatusProcessing: false,
		OrderStatusShipped:    false,
		OrderStatusDelivered:  false,
		OrderStatusCancelled:  false, // 终态
	},
	// 未知状态：任何转换都不允许
	OrderStatus(99): {},
}

func TestCanTransitionOrder(t *testing.T) {
	states := []OrderStatus{
		OrderStatusPending,
		OrderStatusProcessing,
		OrderStatusShipped,
		OrderStatusDelivered,
		OrderStatusCancelled,
		OrderStatus(99), // 未知状态
	}

	for _, from := range states {
		for _, to := range states {
			want := orderTransitionExpectation[from][to]
			if got := CanTransitionOrder(from, to); got != want {
				t.Errorf("CanTransitionOrder(%v, %v) = %v, want %v", from, to, got, want)
			}
		}
	}
}

func TestValidateOrderTransition(t *testing.T) {
	t.Run("合法转换返回 nil", func(t *testing.T) {
		valid := []struct{ from, to OrderStatus }{
			{OrderStatusPending, OrderStatusProcessing},
			{OrderStatusPending, OrderStatusCancelled},
			{OrderStatusProcessing, OrderStatusShipped},
			{OrderStatusProcessing, OrderStatusCancelled},
			{OrderStatusShipped, OrderStatusDelivered},
		}
		for _, tc := range valid {
			if err := ValidateOrderTransition(tc.from, tc.to); err != nil {
				t.Errorf("ValidateOrderTransition(%v, %v) = %v, want nil", tc.from, tc.to, err)
			}
		}
	})

}

func TestValidateOrderTransitionInvalid(t *testing.T) {
	t.Run("状态回退", func(t *testing.T) {
		backward := []struct{ from, to OrderStatus }{
			{OrderStatusProcessing, OrderStatusPending},
			{OrderStatusShipped, OrderStatusProcessing},
			{OrderStatusDelivered, OrderStatusShipped},
			{OrderStatusDelivered, OrderStatusPending},
		}
		for _, tc := range backward {
			assertOrderTransitionError(t, tc.from, tc.to, bizerr.OrderInvalidStatusTransition)
		}
	})

	t.Run("跳级", func(t *testing.T) {
		skip := []struct{ from, to OrderStatus }{
			{OrderStatusPending, OrderStatusShipped},
			{OrderStatusPending, OrderStatusDelivered},
			{OrderStatusProcessing, OrderStatusDelivered},
		}
		for _, tc := range skip {
			assertOrderTransitionError(t, tc.from, tc.to, bizerr.OrderInvalidStatusTransition)
		}
	})

	t.Run("终态不允许再变", func(t *testing.T) {
		terminal := []struct{ from, to OrderStatus }{
			{OrderStatusDelivered, OrderStatusProcessing},
			{OrderStatusDelivered, OrderStatusCancelled},
			{OrderStatusCancelled, OrderStatusPending},
			{OrderStatusCancelled, OrderStatusProcessing},
		}
		for _, tc := range terminal {
			assertOrderTransitionError(t, tc.from, tc.to, bizerr.OrderInvalidStatusTransition)
		}
	})

	t.Run("相同状态重复事件", func(t *testing.T) {
		same := []OrderStatus{
			OrderStatusPending,
			OrderStatusProcessing,
			OrderStatusShipped,
			OrderStatusDelivered,
			OrderStatusCancelled,
		}
		for _, s := range same {
			assertOrderTransitionError(t, s, s, bizerr.INVALID_ARGUMENT)
		}
	})

	t.Run("未知状态", func(t *testing.T) {
		assertOrderTransitionError(t, OrderStatus(99), OrderStatusPending, bizerr.OrderInvalidStatusTransition)
		assertOrderTransitionError(t, OrderStatusPending, OrderStatus(100), bizerr.OrderInvalidStatusTransition)
	})
}

func assertOrderTransitionError(t *testing.T, from, to OrderStatus, wantCode errorpb.ErrorCode) {
	t.Helper()
	err := ValidateOrderTransition(from, to)
	if err == nil {
		t.Fatalf("ValidateOrderTransition(%v, %v) = nil, want error", from, to)
	}
	var be *bizerr.BusinessError
	if !errors.As(err, &be) {
		t.Fatalf("ValidateOrderTransition(%v, %v) 返回类型 %T, want *BusinessError", from, to, err)
	}
	if be.Code != wantCode {
		t.Errorf("ValidateOrderTransition(%v, %v) code = %v, want %v", from, to, be.Code, wantCode)
	}
}
