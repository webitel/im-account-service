package server

import (
	"context"
	"log/slog"

	"github.com/webitel/im-account-service/infra/tls"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"github.com/webitel/im-account-service/cmd"
	"github.com/webitel/im-account-service/config"
	grpcsrv "github.com/webitel/im-account-service/infra/server/grpc"
	"github.com/webitel/im-account-service/infra/x/logx"
	"github.com/webitel/im-account-service/internal/handler"
	apiV1 "github.com/webitel/im-account-service/internal/handler/grpc/v1"

	// "github.com/webitel/im-account-service/internal/service"
	"github.com/webitel/im-account-service/internal/store/postgres"
)

func NewApp(cfg *config.Config) *fx.App {
	return fx.New(
		fx.Supply(cfg),
		fx.Provide(
			cmd.ProvideLogger,
			cmd.ProvideSD,
			cmd.ProvidePubSub,
			cmd.ProvideNewDBConnection,
		),
		fx.WithLogger(func(stdlog *slog.Logger) fxevent.Logger {
			const debugLog = slog.LevelDebug
			if !(logx.Debug("fx") && stdlog.Enabled(context.TODO(), debugLog)) {
				return fxevent.NopLogger
			}
			fxlog := &fxevent.SlogLogger{
				Logger: logx.ModuleLogger("fx", stdlog), // stdlog,
			}
			// fxlog.UseLogLevel(slog.LevelInfo) // default
			// fxlog.UseErrorLevel(slog.LevelError) // default
			fxlog.UseLogLevel(debugLog)
			return fxlog
		}),
		tls.Module,
		postgres.Module,
		// service.Module,
		grpcsrv.Module,
		handler.Module,
		apiV1.Module,
	)
}
