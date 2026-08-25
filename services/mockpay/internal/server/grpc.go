package server

import (
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"

	v1 "kratos-payment-lab/api/mockpay/v1"
)

// NewGRPCServer 创建 gRPC 传输并注册 HealthService。
func NewGRPCServer(c *Server, health *HealthService, logger log.Logger) *grpc.Server {
	opts := []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			logging.Server(logger),
		),
	}
	// 构建gRPC服务器
	// 配置gRPC服务器选项
	// 配置gRPC服务器网络选项
	if c.GRPC != nil {
		if c.GRPC.Network != "" {
			opts = append(opts, grpc.Network(c.GRPC.Network))
		}
		// 配置gRPC服务器地址选项
		if c.GRPC.Addr != "" {
			opts = append(opts, grpc.Address(c.GRPC.Addr))
		}
		// 配置gRPC服务器超时选项
		if time.Duration(c.GRPC.Timeout) > 0 {
			opts = append(opts, grpc.Timeout(time.Duration(c.GRPC.Timeout)))
		}
	}

	// 创建gRPC服务器
	srv := grpc.NewServer(opts...)
	// 注册HealthService
	v1.RegisterHealthServiceServer(srv, health)
	return srv
}
