package order

import (
	"context"
	"strings"

	apperrors "kratos-payment-lab/services/commerce/internal/platform/errors"
	"kratos-payment-lab/services/commerce/internal/platform/postgres"

	"github.com/google/uuid"
)

// 负责业务规则和事务边界

type CreateOrderRequest struct {
	UserID      int64
	AmountCents int64
}

type UseCase struct {
	repo       OrderRepository
	transactor postgres.Transactor
}

func NewUseCase(repo OrderRepository, transactor postgres.Transactor) *UseCase {
	return &UseCase{repo: repo, transactor: transactor}
}

func (u *UseCase) Create(ctx context.Context, input CreateOrderRequest) (*Order, error) {
	if input.AmountCents <= 0 {
		return nil, apperrors.New(apperrors.INVALID_ARGUMENT, "amount_cents must be positive")
	}
	if input.UserID <= 0 {
		return nil, apperrors.New(apperrors.INVALID_ARGUMENT, "user_id must be positive")
	}
	created := &Order{OrderNo: strings.ReplaceAll(uuid.NewString(), "-", ""), UserID: input.UserID, AmountCents: input.AmountCents, Status: OrderStatusPending}
	err := u.transactor.WithinTransaction(ctx, func(txCtx context.Context) error { return u.repo.Create(txCtx, created) })
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (u *UseCase) GetByOrderNo(ctx context.Context, orderNo string) (*Order, error) {
	if strings.TrimSpace(orderNo) == "" {
		return nil, apperrors.New(apperrors.INVALID_ARGUMENT, "order_no is required")
	}
	return u.repo.GetByOrderNo(ctx, orderNo)
}

func (u *UseCase) Transition(ctx context.Context, orderNo string, from []OrderStatus, to OrderStatus) error {
	if len(from) == 0 {
		return apperrors.New(apperrors.INVALID_ARGUMENT, "source status is required")
	}
	valid := false
	for _, source := range from {
		if CanTransitionOrder(source, to) {
			valid = true
			break
		}
	}
	if !valid {
		return apperrors.ErrorOrderInvaildTransaction
	}
	updated, err := u.repo.Transition(ctx, orderNo, from, to)
	if err != nil {
		return err
	}
	if !updated {
		return apperrors.New(apperrors.ORDER_CANNOT_BE_MODIFIED, "order cannot be modified")
	}
	return nil
}
