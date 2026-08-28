package order

import "github.com/google/wire"

// ProviderSet assembles the order repository, use case, and transport service.
var ProviderSet = wire.NewSet(
	NewRepository,
	NewUseCase,
	NewService,
	wire.Bind(new(OrderRepository), new(*Repository)),
)
