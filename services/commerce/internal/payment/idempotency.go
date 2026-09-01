package payment

import (
	"context"
	stderrors "errors"

	apperrors "kratos-payment-lab/services/commerce/internal/platform/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type IdempotencyResult struct {
	Payment  *PaymentAttempt
	Replayed bool
}
type PaymentRepository interface {
	Create(ctx context.Context, payment *PaymentAttempt) error
	GetByIdempotencyKey(ctx context.Context, key string) (*PaymentAttempt, error)
}
type IdempotencyService struct {
	repo PaymentRepository
}

func NewIdempotencyService(repo PaymentRepository) *IdempotencyService {
	return &IdempotencyService{
		repo: repo,
	}
}

func (s *IdempotencyService) Create(ctx context.Context, req CreatePaymentCommand) (IdempotencyResult, error) {
	if req.OrderNo == "" || req.AmountCents <= 0 || req.Channel == PaymentChannelUnknown || req.IdempotencyKey == "" {
		return IdempotencyResult{}, apperrors.New(apperrors.INVALID_ARGUMENT, "invalid payment request")
	}
	//1. 校验 order_no、amount_cents、channel、idempotency_key
	//2. 计算 request fingerprint
	//3. 查询 idempotency_key
	//4. 查到记录：
	//   - fingerprint 相同：返回旧记录，Replayed=true
	//   - fingerprint 不同：返回冲突错误
	//5. 查不到记录：
	//   - 尝试 INSERT
	//   - 插入成功：返回新记录，Replayed=false
	//   - 唯一约束冲突：重新查询旧记录，再比较 fingerprint
	current := Fingerprint(req)
	existing, err := s.repo.GetByIdempotencyKey(ctx, req.IdempotencyKey)

	if err == nil {
		if existing.RequestFingerprint != current {
			return IdempotencyResult{}, apperrors.ErrorIdempotencyKeyReused
		}

		return IdempotencyResult{
			Payment:  existing,
			Replayed: true,
		}, nil
	}

	if !stderrors.Is(err, pgx.ErrNoRows) {
		return IdempotencyResult{}, err
	}

	payment := &PaymentAttempt{
		OrderNo:            req.OrderNo,
		AmountCents:        req.AmountCents,
		Channel:            req.Channel,
		IdempotencyKey:     req.IdempotencyKey,
		RequestFingerprint: current,
		Status:             PaymentStatusPending,
	}
	if err := s.repo.Create(ctx, payment); err != nil {
		if pgErr, ok := stderrors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			existing, getErr := s.repo.GetByIdempotencyKey(ctx, req.IdempotencyKey)
			if getErr != nil {
				return IdempotencyResult{}, getErr
			}
			if existing.RequestFingerprint != current {
				return IdempotencyResult{}, apperrors.ErrorIdempotencyKeyReused
			}
			return IdempotencyResult{Payment: existing, Replayed: true}, nil
		}
		return IdempotencyResult{}, err
	}
	return IdempotencyResult{Payment: payment}, nil
}
