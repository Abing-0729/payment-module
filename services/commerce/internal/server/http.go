package server

import (
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"

	v1 "kratos-payment-lab/api/commerce/v1"
	"kratos-payment-lab/services/commerce/internal/order"
)

// NewHTTPServer 创建 HTTP 传输并注册 HealthService 路由。
// 中间件仅保留恢复与访问日志;Issue 14 再补充 trace_id 与指标。
func NewHTTPServer(c *Server, health *HealthService, orders *order.Service, logger log.Logger) *http.Server {
	opts := []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			logging.Server(logger),
		),
	}
	if c.HTTP != nil {
		if c.HTTP.Network != "" {
			opts = append(opts, http.Network(c.HTTP.Network))
		}
		if c.HTTP.Addr != "" {
			opts = append(opts, http.Address(c.HTTP.Addr))
		}
		if time.Duration(c.HTTP.Timeout) > 0 {
			opts = append(opts, http.Timeout(time.Duration(c.HTTP.Timeout)))
		}
	}
	srv := http.NewServer(opts...)
	v1.RegisterHealthServiceHTTPServer(srv, health)
	v1.RegisterOrderServiceHTTPServer(srv, orders)
	return srv
}
