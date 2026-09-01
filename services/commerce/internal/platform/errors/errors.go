package errors

import (
	"fmt"
	errorpb "kratos-payment-lab/api/commerce/v1/errors"
)

//使用errorpb.ErrorCode，当前直接导出常量

const (
	SUCCESS                        = errorpb.ErrorCode_SUCCESS
	INTERNAL_ERROR                 = errorpb.ErrorCode_INTERNAL_ERROR
	INVALID_ARGUMENT               = errorpb.ErrorCode_INVALID_ARGUMENT
	NOT_FOUND                      = errorpb.ErrorCode_NOT_FOUND
	ALREADY_EXISTS                 = errorpb.ErrorCode_ALREADY_EXISTS
	PERMISSION_DENIED              = errorpb.ErrorCode_PERMISSION_DENIED
	UNAUTHENTICATED                = errorpb.ErrorCode_UNAUTHENTICATED
	OrderInvalidStatusTransition   = errorpb.ErrorCode_ORDER_INVALID_STATUS_TRANSITION
	ORDER_NOT_FOUND                = errorpb.ErrorCode_ORDER_NOT_FOUND
	ORDER_CANNOT_BE_MODIFIED       = errorpb.ErrorCode_ORDER_CANNOT_BE_MODIFIED
	PaymentInvalidStatusTransition = errorpb.ErrorCode_PAYMENT_INVALID_STATUS_TRANSITION
	PAYMENT_FAILED                 = errorpb.ErrorCode_PAYMENT_FAILED
	PAYMENT_ALREADY_PROCESSED      = errorpb.ErrorCode_PAYMENT_ALREADY_PROCESSED
	IDEMPOTENCY_KEY_REUSED         = errorpb.ErrorCode_ALREADY_EXISTS
)

// BusinessError 实现error接口，携带错误码信息
type BusinessError struct {
	Code     errorpb.ErrorCode
	Message  string
	Metadata map[string]string
}

func (e *BusinessError) Error() string {
	return fmt.Sprintf("[%s]%s", e.Code, e.Message)
}

// New 快捷构造业务错误
func New(code errorpb.ErrorCode, msg string) *BusinessError {
	return &BusinessError{Code: code, Message: msg}
}

// Wrap 包装已有错误
func Wrap(code errorpb.ErrorCode, err error) *BusinessError {
	return &BusinessError{Code: code, Message: err.Error()}
}

// 预定义常用错误
var (
	ErrorOrderInvaildTransaction   = New(OrderInvalidStatusTransition, "invalid order status transaction")
	ErrorPaymentInvalidTransaction = New(PaymentInvalidStatusTransition, "invalid payment status transaction")
	ErrorIdempotencyKeyReused      = New(IDEMPOTENCY_KEY_REUSED, "idempotency key reused with different request")
)
