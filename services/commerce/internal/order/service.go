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

// CreateOrder 创建新订单的方法
// 参数:
//
//	ctx - 上下文信息，用于控制请求的超时和取消
//	req - 创建订单的请求参数，包含用户ID和金额信息
//
// 返回值:
//
//	*v1.CreateOrderResponse - 创建订单的响应结果，包含订单信息
//	error - 错误信息，如果创建过程中出现错误则返回
func (s *Service) CreateOrder(ctx context.Context, req *v1.CreateOrderRequest) (*v1.CreateOrderResponse, error) {
	// 调用useCase层的Create方法创建订单
	// 将请求参数转换为useCase需要的格式
	created, err := s.useCase.Create(ctx, CreateOrderRequest{UserID: req.GetUserId(), AmountCents: req.GetAmountCents()})
	// 如果创建过程中出现错误，直接返回错误
	if err != nil {
		return nil, err
	}
	// 将创建的订单转换为协议缓冲区格式并返回响应
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
