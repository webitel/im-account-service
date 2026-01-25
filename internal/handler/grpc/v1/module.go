package v1

import (
	"go.uber.org/fx"
)

var Module = fx.Module(
	"grpc",
	// fx.Provide(NewAccountService),
	// fx.Provide(NewApplicationService),
	// fx.Invoke(RegisterAccountService),
	// fx.Invoke(RegisterApplicationService),
	fx.Provide(
		NewAccountService,
		NewApplicationService,
	),
	fx.Invoke(
		RegisterAccountService,
		RegisterApplicationService,
	),
)
