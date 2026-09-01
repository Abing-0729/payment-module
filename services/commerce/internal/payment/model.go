package payment

import "time"

type PaymentChannel string

const (
	PaymentChannelUnknown PaymentChannel = "UNKNOWN"
	PaymentChannelAlipay  PaymentChannel = "ALIPAY"
	PaymentChannelWechat  PaymentChannel = "WECHAT"
)

type PaymentAttempt struct {
	ID                 int64
	OrderNo            string
	AmountCents        int64
	Channel            PaymentChannel
	IdempotencyKey     string
	RequestFingerprint string
	Status             PaymentStatus
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
