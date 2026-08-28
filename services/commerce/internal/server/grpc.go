package server

import (
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"

	v1 "kratos-payment-lab/api/commerce/v1"
	"kratos-payment-lab/services/commerce/internal/order"
)

// NewGRPCServer 创建 gRPC 传输并注册 HealthService。
func NewGRPCServer(c *Server, health *HealthService, orders *order.Service, logger log.Logger) *grpc.Server {
	opts := []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			logging.Server(logger),
		),
	}
	if c.GRPC != nil {
		if c.GRPC.Network != "" {
			opts = append(opts, grpc.Network(c.GRPC.Network))
		}
		if c.GRPC.Addr != "" {
			opts = append(opts, grpc.Address(c.GRPC.Addr))
		}
		if time.Duration(c.GRPC.Timeout) > 0 {
			opts = append(opts, grpc.Timeout(time.Duration(c.GRPC.Timeout)))
		}
	}
	srv := grpc.NewServer(opts...)
	v1.RegisterHealthServiceServer(srv, health)
	v1.RegisterOrderServiceServer(srv, orders)
	return srv
}
