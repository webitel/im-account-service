package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	sfmt "github.com/samber/slog-formatter"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	"github.com/webitel/im-account-service/config"
	"github.com/webitel/im-account-service/infra/db/pg"
	"github.com/webitel/im-account-service/infra/pubsub"
	"github.com/webitel/im-account-service/infra/pubsub/factory"
	"github.com/webitel/im-account-service/infra/pubsub/factory/amqp"
	grpc_srv "github.com/webitel/im-account-service/infra/server/grpc"
	infra_tls "github.com/webitel/im-account-service/infra/tls"
	"github.com/webitel/webitel-go-kit/infra/discovery"
	_ "github.com/webitel/webitel-go-kit/infra/discovery/consul"
	otelsdk "github.com/webitel/webitel-go-kit/infra/otel/sdk"
	"github.com/webitel/wlog"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"
	"go.uber.org/fx"

	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/log/otlp"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/log/stdout"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/metric/otlp"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/metric/stdout"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/trace/otlp"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/trace/stdout"
)

func ProvideLogger(cfg *config.Config, lc fx.Lifecycle) (*slog.Logger, error) {
	logSettings := cfg.Log

	if !logSettings.Console && !logSettings.Otel && logSettings.File == "" {
		logSettings.Console = true
	}

	level := parseLevel(logSettings.Level)
	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handlers []slog.Handler

	if logSettings.Console {
		var h slog.Handler
		if logSettings.JSON {
			h = slog.NewJSONHandler(os.Stdout, opts)
		} else {
			// h = slog.NewTextHandler(os.Stdout, opts)
			h = console(os.Stdout, level)
		}
		handlers = append(handlers, h)
	}

	// File Handler
	if logSettings.File != "" {
		f, err := os.OpenFile(logSettings.File, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, err
		}

		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				return f.Close()
			},
		})

		var h slog.Handler
		if logSettings.JSON {
			h = slog.NewJSONHandler(f, opts)
		} else {
			h = slog.NewTextHandler(f, opts)
		}
		handlers = append(handlers, h)
	}

	if logSettings.Otel {
		service := resource.NewSchemaless(
			semconv.ServiceName(ServiceName),
			semconv.ServiceVersion(version),
			semconv.ServiceInstanceID(cfg.Service.Id),
			semconv.ServiceNamespace(ServiceNamespace),
		)
		otelHandler := otelslog.NewHandler("slog")

		shutdown, err := otelsdk.Configure(context.Background(), otelsdk.WithResource(service),
			otelsdk.WithLogBridge(
				func() {
					handlers = append(handlers, otelHandler)
				},
			),
		)
		if err != nil {
			return nil, err
		}

		handlers = append(handlers)
		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				return shutdown(ctx)
			},
		})
	}

	var finalHandler slog.Handler
	if len(handlers) == 0 {
		finalHandler = slog.NewTextHandler(os.Stdout, opts)
	} else if len(handlers) == 1 {
		finalHandler = handlers[0]
	} else {
		finalHandler = MultiHandler(handlers...)
	}

	logger := slog.New(finalHandler)
	slog.SetDefault(logger)

	return logger, nil
}

func parseLevel(input string) (level slog.Level) {
	err := level.UnmarshalText([]byte(input))
	if err != nil {
		// default: info
		level = slog.LevelInfo
	}
	return // level
}

func console(output *os.File, verbose slog.Level) slog.Handler {
	colorize, _ := strconv.ParseBool(
		os.Getenv("WBTL_LOG_COLOR"),
	)
	if colorize {
		colorize = isatty.IsTerminal(
			output.Fd(),
		)
	}
	return sfmt.NewFormatterHandler(
		// sfmt.FormatByType(func(e *myError) slog.Value {
		// 	return slog.GroupValue(
		// 		slog.Int("code", e.code),
		// 		slog.String("message", e.msg),
		// 	)
		// }),
		// sfmt.ErrorFormatter("error_with_generic_formatter"),
		// sfmt.FormatByKey("email", func(v slog.Value) slog.Value {
		// 	return slog.StringValue("***********")
		// }),
		// sfmt.FormatByGroupKey([]string{"a-group"}, "hello", func(v slog.Value) slog.Value {
		// 	return slog.StringValue("eve")
		// }),
		// sfmt.FormatByGroup([]string{"hq"}, func(attrs []slog.Attr) slog.Value {
		// 	return slog.GroupValue(
		// 		slog.Group(
		// 			"address",
		// 			lo.ToAnySlice(attrs)...,
		// 		),
		// 	)
		// }),
		// sfmt.PIIFormatter("hq"),
		sfmt.ErrorFormatter("err"),
		sfmt.ErrorFormatter("error"),
	)(
		// slog.NewJSONHandler(os.Stdout, nil),
		tint.NewHandler(output, &tint.Options{
			AddSource: false,
			Level:     verbose.Level(),
			// ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			// 	return attr
			// },
			TimeFormat: "Jan 02 15:04:05.000", // time.StampMilli,
			NoColor:    !colorize,
		}),
	)
}

type multiHandler struct {
	handlers []slog.Handler
}

func MultiHandler(handlers ...slog.Handler) slog.Handler {
	return &multiHandler{handlers: handlers}
}

func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, hh := range h.handlers {
		if hh.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, hh := range h.handlers {
		if hh.Enabled(ctx, r.Level) {
			_ = hh.Handle(ctx, r)
		}
	}
	return nil
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, hh := range h.handlers {
		newHandlers[i] = hh.WithAttrs(attrs)
	}
	return &multiHandler{handlers: newHandlers}
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, hh := range h.handlers {
		newHandlers[i] = hh.WithGroup(name)
	}
	return &multiHandler{handlers: newHandlers}
}

func ProvideGrpcServer(config *config.Config, logger *slog.Logger, creds *infra_tls.Config, lc fx.Lifecycle) (*grpc_srv.Server, error) {

	var ssl *tls.Config
	if creds != nil {
		ssl = creds.Server
	}

	s, err := grpc_srv.New(config.Service.Address, logger, ssl)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			if err := s.Shutdown(); err != nil {
				logger.Error(err.Error(), wlog.Err(err))
				return err
			}
			return nil
		},
	})

	return s, nil
}

//
//func ProvideCluster(cfg *config.Config, srv *grpc_srv.Server, l *slog.Logger, lc fx.Lifecycle) (*consul.Cluster, error) {
//	c := consul.NewCluster(model.ServiceName, cfg.Consul.Address, l)
//	host := srv.Host()
//
//	lc.Append(fx.Hook{
//		OnStart: func(ctx context.Context) error {
//			return c.Start(cfg.Service.Id, host, srv.Port())
//		},
//		OnStop: func(ctx context.Context) error {
//			c.Stop()
//			return nil
//		},
//	})
//
//	return c, nil
//}

func ProvideSD(cfg *config.Config, log *slog.Logger, lc fx.Lifecycle) (discovery.DiscoveryProvider, error) {
	provider, err := discovery.DefaultFactory.CreateProvider(
		discovery.ProviderConsul,
		log,
		cfg.Consul.Address,
		discovery.WithHeartbeat[discovery.DiscoveryProvider](true),
		discovery.WithTimeout[discovery.DiscoveryProvider](time.Second*30),
	)

	if err != nil {
		return nil, err
	}

	var si = new(discovery.ServiceInstance)
	{
		si.Id = cfg.Service.Id
		si.Name = ServiceName
		si.Version = version
		si.Metadata = map[string]string{
			"commit":         commit,
			"commitDate":     commitDate,
			"branch":         branch,
			"buildTimestamp": buildTimestamp,
		}
		si.Endpoints = []string{(&url.URL{Scheme: "grpc", Host: cfg.Service.Address}).String()}
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := provider.Register(ctx, si); err != nil {
				return err
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if err := provider.Deregister(ctx, si); err != nil {
				return err
			}
			return nil
		},
	})

	return provider, nil
}

func ProvidePubSub(config *config.Config, logger *slog.Logger, runtime fx.Lifecycle) (pubsub.Provider, error) {

	var (
		pubsubConfig  = config.Pubsub
		loggerAdapter = watermill.NewSlogLogger(logger)
		pubsubFactory factory.Factory
		err           error
	)

	driver := strings.ToLower(pubsubConfig.Driver)
	switch driver {
	case "amqp", "rabbitmq":
		pubsubFactory, err = amqp.NewFactory(pubsubConfig.URL, loggerAdapter)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("broker [%s] not supported", driver)
	}

	router, err := message.NewRouter(message.RouterConfig{}, loggerAdapter)
	if err != nil {
		return nil, err
	}

	router.AddMiddleware(middleware.Recoverer)

	runtime.Append(fx.Hook{
		OnStart: func(ctx context.Context) (err error) {
			// return router.Run(ctx)

			// This call is blocking while the router is running.
			go func() {
				err = router.Run(ctx)
			}()
			<-router.Running()
			return // err
		},
		OnStop: func(ctx context.Context) error {
			return router.Close()
		},
	})

	return pubsub.NewDefaultProvider(router, pubsubFactory)
}

func ProvideNewDBConnection(cfg *config.Config, l *slog.Logger, lc fx.Lifecycle) (*pg.DB, error) {
	db, err := pg.New(context.Background(), l, cfg.Postgres.DSN)
	if err != nil {
		return nil, err
	}

	pg.SetDefault(db)

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) (_ error) {
			db.Client().Close()
			return // nil
		},
	})

	return db, err
}
