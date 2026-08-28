package order

import (
	"context"
	"testing"
)

type fakeRepository struct{ created *Order }

func (f *fakeRepository) Create(_ context.Context, order *Order) error         { f.created = order; return nil }
func (f *fakeRepository) GetByOrderNo(context.Context, string) (*Order, error) { return nil, nil }
func (f *fakeRepository) Transition(context.Context, string, []OrderStatus, OrderStatus) (bool, error) {
	return true, nil
}

type fakeTransactor struct{}

func (fakeTransactor) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestCreateRejectsNonPositiveAmount(t *testing.T) {
	repo := &fakeRepository{}
	uc := NewUseCase(repo, fakeTransactor{})
	for _, amount := range []int64{0, -1} {
		if _, err := uc.Create(context.Background(), CreateOrderRequest{UserID: 1, AmountCents: amount}); err == nil {
			t.Fatalf("amount %d: expected error", amount)
		}
	}
	if repo.created != nil {
		t.Fatal("repository should not be called")
	}
}

func TestCreateForcesPendingStatus(t *testing.T) {
	repo := &fakeRepository{}
	uc := NewUseCase(repo, fakeTransactor{})
	created, err := uc.Create(context.Background(), CreateOrderRequest{UserID: 7, AmountCents: 1999})
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != OrderStatusPending {
		t.Fatalf("status = %v, want pending", created.Status)
	}
	if created.AmountCents != 1999 || created.UserID != 7 || created.OrderNo == "" {
		t.Fatalf("unexpected order: %+v", created)
	}
}
