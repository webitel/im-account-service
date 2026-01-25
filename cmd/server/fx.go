package server

import (
	"go.uber.org/fx"

	"github.com/webitel/im-account-service/cmd"
	"github.com/webitel/im-account-service/config"
	grpcsrv "github.com/webitel/im-account-service/infra/server/grpc"
	"github.com/webitel/im-account-service/internal/handler"
	apiV1 "github.com/webitel/im-account-service/internal/handler/grpc/v1"

	// "github.com/webitel/im-account-service/internal/service"
	"github.com/webitel/im-account-service/internal/store/postgres"
)

func NewApp(cfg *config.Config) *fx.App {
	return fx.New(
		// fx.WithLogger(func(stdlog *slog.Logger) fxevent.Logger {
		// 	return &fxevent.SlogLogger{Logger: stdlog}
		// }),
		fx.Provide(
			func() *config.Config { return cfg },
			cmd.ProvideLogger,
			cmd.ProvideSD,
			cmd.ProvidePubSub,
			cmd.ProvideNewDBConnection,
		),
		postgres.Module,
		// service.Module,
		grpcsrv.Module,
		handler.Module,
		apiV1.Module,
	)
}
