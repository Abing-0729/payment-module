package payment

import (
	"context"

	"kratos-payment-lab/services/commerce/internal/platform/postgres"
	"kratos-payment-lab/services/commerce/internal/platform/postgres/query"
)

type Repository struct {
	db *postgres.Postgres
}

func NewRepository(db *postgres.Postgres) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, payment *PaymentAttempt) error {
	row, err := query.New(r.db.Pool()).CreatePaymentAttempt(ctx, query.CreatePaymentAttemptParams{
		OrderNo: payment.OrderNo, AmountCents: payment.AmountCents,
		Channel: string(payment.Channel), IdempotencyKey: payment.IdempotencyKey,
		RequestFingerprint: payment.RequestFingerprint,
	})
	if err != nil {
		return err
	}
	*payment = fromQuery(row)
	return nil
}

func (r *Repository) GetByIdempotencyKey(ctx context.Context, key string) (*PaymentAttempt, error) {
	row, err := query.New(r.db.Pool()).GetPaymentAttemptByIdempotencyKey(ctx, key)
	if err != nil {
		return nil, err
	}
	result := fromQuery(row)
	return &result, nil
}

func fromQuery(row query.PaymentAttempt) PaymentAttempt {
	return PaymentAttempt{ID: row.ID, OrderNo: row.OrderNo, AmountCents: row.AmountCents,
		Channel: PaymentChannel(row.Channel), IdempotencyKey: row.IdempotencyKey,
		RequestFingerprint: row.RequestFingerprint, Status: parseStatus(row.Status),
		CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time}
}

func parseStatus(value string) PaymentStatus {
	switch value {
	case "PROCESSING":
		return PaymentStatusProcessing
	case "SUCCEEDED":
		return PaymentStatusSucceeded
	case "FAILED":
		return PaymentStatusFailed
	default:
		return PaymentStatusPending
	}
}
