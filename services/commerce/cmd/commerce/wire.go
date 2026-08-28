//go:build wireinject
// +build wireinject

package main

import (
	"kratos-payment-lab/services/commerce/internal/order"
	"kratos-payment-lab/services/commerce/internal/server"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// wireApp 组装依赖注入图:main → server(HealthService/http/grpc)→ kratos.App。
func wireApp(conf *server.Bootstrap, logger log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, order.ProviderSet, newApp))
}
