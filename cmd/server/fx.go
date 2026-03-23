package server

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/webitel/im-account-service/infra/tls"
	"github.com/webitel/webitel-go-kit/infra/profiler"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"github.com/webitel/im-account-service/cmd"
	"github.com/webitel/im-account-service/config"
	grpcsrv "github.com/webitel/im-account-service/infra/server/grpc"
	"github.com/webitel/im-account-service/infra/x/httpx"
	"github.com/webitel/im-account-service/infra/x/logx"

	"github.com/webitel/im-account-service/internal/service"
	apiV1 "github.com/webitel/im-account-service/internal/service/api/grpc/v1"

	// "github.com/webitel/im-account-service/internal/service"
	"github.com/webitel/im-account-service/internal/store/postgres"
)

func NewApp(cfg *config.Config) *fx.App {
	return fx.New(
		fx.Invoke(func() {
			// HTTP Client trafic DUMP !
			if logx.Debug("http", "https") {
				http.DefaultTransport = httpx.TransportDump{
					Transport: http.DefaultTransport,
					WithBody:  true,
				}
			}
		}),
		fx.Supply(cfg),
		fx.Provide(
			cmd.ProvideLogger,
			cmd.ProvideSD,
			cmd.ProvidePubSub,
			cmd.ProvideDB,
			cmd.ProvideProfiler,
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
		grpcsrv.Module,
		service.Module,
		apiV1.Module,
		profiler.Module,
	)
}
