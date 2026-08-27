package order

import (
	"context"

	v1 "kratos-payment-lab/api/commerce/v1"
)

type Service struct {
	v1.UnimplementedOrderServiceServer
	useCase *UseCase
}

func NewService(useCase *UseCase) *Service { return &Service{useCase: useCase} }

func (s *Service) CreateOrder(ctx context.Context, req *v1.CreateOrderRequest) (*v1.CreateOrderResponse, error) {
	created, err := s.useCase.Create(ctx, CreateOrderRequest{UserID: req.GetUserId(), AmountCents: req.GetAmountCents()})
	if err != nil {
		return nil, err
	}
	return &v1.CreateOrderResponse{Order: toProto(created)}, nil
}

func (s *Service) GetOrder(ctx context.Context, req *v1.GetOrderRequest) (*v1.GetOrderResponse, error) {
	found, err := s.useCase.GetByOrderNo(ctx, req.GetOrderNo())
	if err != nil {
		return nil, err
	}
	return &v1.GetOrderResponse{Order: toProto(found)}, nil
}

func toProto(order *Order) *v1.Order {
	return &v1.Order{OrderNo: order.OrderNo, UserId: order.UserID, AmountCents: order.AmountCents, Status: order.Status.String(), Version: order.Version, CreatedAtUnix: order.CreatedAt.Unix(), UpdatedAtUnix: order.UpdatedAt.Unix()}
}
