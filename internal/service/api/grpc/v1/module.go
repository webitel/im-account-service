package v1

import (
	"go.uber.org/fx"
)

var Module = fx.Module(
	"grpc",
	// options
	fx.Provide(
		NewAccountService,
		NewApplicationService,
	),
	fx.Invoke(
		RegisterAccountService,
		RegisterApplicationService,
	),
)
