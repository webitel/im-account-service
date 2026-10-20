package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/webitel/webitel-go-kit/infra/profiler"
	"github.com/webitel/webitel-go-kit/pkg/depenlog"
	"github.com/webitel/webitel-go-kit/pkg/logger"
	wsemconv "github.com/webitel/webitel-go-kit/pkg/semconv"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/webitel/im-account-service/config"
	"github.com/webitel/im-account-service/infra/postgres"
	"github.com/webitel/im-account-service/infra/pubsub"
	"github.com/webitel/im-account-service/infra/pubsub/factory"
	"github.com/webitel/im-account-service/infra/pubsub/factory/amqp"
	grpc_srv "github.com/webitel/im-account-service/infra/server/grpc"
	infra_tls "github.com/webitel/im-account-service/infra/tls"
	"github.com/webitel/im-account-service/infra/x/logx"
	"github.com/webitel/webitel-go-kit/infra/discovery"
	_ "github.com/webitel/webitel-go-kit/infra/discovery/consul"
	otelsdk "github.com/webitel/webitel-go-kit/infra/otel/sdk"
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

// ProvideLogger builds the service's unified logger via depenlog and exposes it
// in two shapes: the *slog.Logger (slog.Default(), for the many consumers that
// take one) and the logger.Logger handle (for fx and the profiler). depenlog.New
// installs the logger process-wide — slog.SetDefault plus grpc-go's global
// logger — so no separate gRPC wiring is needed here.
func ProvideLogger(cfg *config.Config, lc fx.Lifecycle) (*slog.Logger, logger.Logger, error) {
	logSettings := cfg.Log

	if !logSettings.Console && !logSettings.Otel && logSettings.File == "" {
		logSettings.Console = true
	}

	dcfg := depenlog.Config{
		Level:   logSettings.Level,
		JSON:    logSettings.JSON,
		File:    logSettings.File,
		Console: logSettings.Console,
	}

	var opts []depenlog.Option
	if logSettings.Otel {
		service := resource.NewSchemaless(
			semconv.ServiceName(ServiceName),
			semconv.ServiceVersion(version),
			semconv.ServiceInstanceID(discovery.GenerateInstanceID(ServiceName)),
			semconv.ServiceNamespace(ServiceNamespace),
		)

		// When the OTel log bridge is enabled, route the unified logger through
		// it so the OTel LoggerProvider/exporter owns schema and trace
		// correlation (WithHandler bypasses depenlog's console/file handlers).
		var otelHandler slog.Handler
		shutdown, err := otelsdk.Configure(
			context.Background(),
			otelsdk.WithResource(service),
			otelsdk.WithLogBridge(func() {
				otelHandler = otelslog.NewHandler("slog")
			}),
		)
		if err != nil {
			return nil, nil, err
		}

		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				return shutdown(ctx)
			},
		})

		if otelHandler != nil {
			opts = append(opts, depenlog.WithHandler(otelHandler))
		}
	}

	kit := depenlog.New(dcfg, opts...)

	return slog.Default(), kit, nil
}

func ProvideGrpcServer(config *config.Config, logger *slog.Logger, creds *infra_tls.Config, lc fx.Lifecycle) (*grpc_srv.Server, error) {

	var ssl *tls.Config
	if creds != nil {
		ssl = creds.Server
	}

	s, err := grpc_srv.New(config.Service.Addr, logger, ssl)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			if err := s.Shutdown(); err != nil {
				logger.Error(err.Error(), wsemconv.ErrorKey, err)
				return err
			}
			return nil
		},
	})

	return s, nil
}

//
//func ProvideCluster(cfg *config.Config, srv *grpc_srv.Server, l *slog.Logger, lc fx.Lifecycle) (*consul.Cluster, error) {
//	c := consul.NewCluster(model.ServiceName, cfg.Consul.Addr, l)
//	host := srv.Host()
//
//	lc.Append(fx.Hook{
//		OnStart: func(ctx context.Context) error {
//			return c.Start(discovery.GenerateInstanceID(ServiceName), host, srv.Port())
//		},
//		OnStop: func(ctx context.Context) error {
//			c.Stop()
//			return nil
//		},
//	})
//
//	return c, nil
//}

func ProvideSD(cfg *config.Config, log *slog.Logger, srv *grpc_srv.Server, lc fx.Lifecycle) (discovery.DiscoveryProvider, error) {
	provider, err := discovery.DefaultFactory.CreateProvider(
		discovery.ProviderConsul,
		log,
		cfg.Consul.Addr,
		discovery.WithHeartbeat[discovery.DiscoveryProvider](true),
		discovery.WithTimeout[discovery.DiscoveryProvider](time.Second*30),
	)

	if err != nil {
		return nil, err
	}

	var si = new(discovery.ServiceInstance)
	{
		si.Id = discovery.GenerateInstanceID(ServiceName)
		si.Name = ServiceName
		si.Version = version
		si.Metadata = map[string]string{
			"commit":         commit,
			"commitDate":     commitDate,
			"branch":         branch,
			"buildTimestamp": buildTimestamp,
		}
		si.Endpoints = []string{(&url.URL{
			// [input]: cfg.Service.Address,
			// [serve]: srv.Addr ; [::]
			// [advert]: [::] ~ public IP
			Scheme: "grpc", Host: srv.Advertise(),
		}).String()}
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
		err           error
		pubsubConfig  = config.Pubsub
		pubsubFactory factory.Factory
		// loggerAdapter = watermill.NewSlogLogger(logger)
		loggerAdapter watermill.LoggerAdapter
	)

	// WBTL_LOG_DEBUG=broker
	if logx.Debug("broker", "rabbitmq") {
		// enable
		loggerAdapter = watermill.NewSlogLogger(
			logx.ModuleLogger("broker", logger),
		)
	} else {
		// disable ; default
		loggerAdapter = watermill.NewSlogLogger(
			slog.New(slog.DiscardHandler),
		)
	}

	driver := strings.ToLower(pubsubConfig.Driver)
	switch driver {
	case "amqp", "rabbitmq":
		var broker *amqp.Factory
		broker, err = amqp.NewFactory(
			pubsubConfig.URL, // connectionString
			loggerAdapter,    // logger
		)
		if err != nil {
			return nil, err
		}
		broker.ServiceId = discovery.GenerateInstanceID(ServiceName)
		broker.ServiceName = ServiceName
		pubsubFactory = broker
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

func ProvideDB(config *config.Config, logger *slog.Logger, runtime fx.Lifecycle) (*postgres.DB, error) {

	dbo, err := postgres.New(
		postgres.Logger(logger),
		postgres.ConnOptions(
			postgres.FallbackApplicationName(discovery.GenerateInstanceID(ServiceName)),
		),
		postgres.DataSourceName(config.Postgres.DSN),
	)

	if err != nil {
		return nil, err
	}

	postgres.SetDefault(dbo)

	runtime.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// check connection is available ?
			return dbo.Client().Ping(ctx)
		},
		OnStop: func(ctx context.Context) (_ error) {
			// blocking call
			dbo.Client().Close()
			return // nil
		},
	})

	return dbo, err
}

// ProvideProfiler supplies the profiler config. The profiler module consumes
// logger.Logger from the fx graph, which ProvideLogger now provides.
func ProvideProfiler(config *config.Config) profiler.Config {
	return profiler.Config{
		Addr:                 config.Profiler.Addr,
		MutexProfileFraction: config.Profiler.MutexFraction,
		BlockProfileRate:     config.Profiler.BlockRate,
	}
}
