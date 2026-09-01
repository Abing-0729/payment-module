package payment

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// CreatePaymentCommand 本次请求
type CreatePaymentCommand struct {
	OrderNo        string
	AmountCents    int64
	Channel        PaymentChannel
	IdempotencyKey string
}

// Fingerprint 请求指纹
func Fingerprint(req CreatePaymentCommand) string {
	payload, _ := json.Marshal(struct {
		OrderNo     string `json:"order_no"`
		AmountCents int64  `json:"amount_cents"`
		Channel     string `json:"channel"`
	}{
		req.OrderNo,
		req.AmountCents,
		string(req.Channel),
	})

	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}
