package service

import (
	"log/slog"

	// "github.com/lestrrat-go/httprc/v3"
	// "github.com/lestrrat-go/httprc/v3/errsink"
	// "github.com/lestrrat-go/httprc/v3/tracesink"
	// "github.com/webitel/im-account-service/internal/service/jwks"

	"github.com/webitel/webitel-go-kit/infra/discovery"
	"go.uber.org/fx"

	"github.com/webitel/im-account-service/infra/pubsub"
	infra_tls "github.com/webitel/im-account-service/infra/tls"
	"github.com/webitel/im-account-service/infra/x/logx"
	"github.com/webitel/im-account-service/internal/client/contacts"
	webitel "github.com/webitel/im-account-service/internal/client/webitel/auth"
	c1pb "github.com/webitel/im-account-service/proto/gen/im/service/contact/v1"
)

var Module = fx.Module(
	"service",
	fx.Provide(
		func(logger *slog.Logger, registry discovery.DiscoveryProvider, broker pubsub.Provider) (*webitel.Client, error) {
			logger = logx.ModuleLogger("go-webitel-client", logger)
			return webitel.NewClient(logger, registry, broker) //, opts...)
		},
		func(logger *slog.Logger, registry discovery.DiscoveryProvider, secure *infra_tls.Config) (c1pb.ContactsClient, error) {
			logger = logx.ModuleLogger("im-contact-client", logger)
			return contacts.NewClient(logger, registry, secure.Client) // , opts...)
		},
		New,
	),
	/* fx.Module(
		"jwks", fx.Invoke(func(logger *slog.Logger, module fx.Lifecycle) (_ struct{}){
			logger = logx.ModuleLogger("JWKs", logger)
			module.Append(fx.Hook{
				OnStart: func(ctx context.Context) (_ error) {
					// jwks.Init(

					// 	httprc.WithWorkers(runtime.NumCPU()),
					// 	// httprc.WithErrorSink(errsink.NewSlog()),
					// 	httprc.WithErrorSink(errsink.NewFunc(func(ctx context.Context, err error) {
					// 		// logger.ErrorContext(ctx, "", "err", err)
					// 		logger.ErrorContext(ctx, err.Error())
					// 	})),
					// 	// httprc.WithTraceSink(tracesink.NewSlog())
					// 	httprc.WithTraceSink(tracesink.Func(func(ctx context.Context, msg string) {
					// 		logger.DebugContext(ctx, msg)
					// 	})),
					// 	// httprc.WithHTTPClient(cl HTTPClient) NewClientResourceOption
					// 	// httprc.WithErrorSink(sink httprc.ErrorSink) NewClientOption
					// 	// httprc.WithTraceSink(sink httprc.TraceSink) NewClientOption
					// 	// httprc.WithWhitelist(wl httprc.Whitelist) NewClientOption
					// 	// httprc.WithWorkers(n int) NewClientOption // default: 5
					// 	// httprc.WithWorkers(
					// 	//   runtime.NumCPU(),
					// 	// ),
					// )
					jwks.Init(func(opts *jwks.Options) {
						opts.AddResource = jwks.ResourceOptions{
							// ParseOptions:       []jwk.ParseOption{},
							// RefreshInterval:    0,
							RefreshMinInterval: (time.Second * 5),
							RefreshMaxInterval: (time.Second * 15),
						}
					})
					return jwks.Start(
						ctx,
						// httprc.WithWorkers(runtime.NumCPU()),
						// httprc.WithErrorSink(errsink.NewSlog()),
						httprc.WithErrorSink(errsink.NewFunc(func(ctx context.Context, err error) {
							// logger.ErrorContext(ctx, "", "err", err)
							logger.ErrorContext(ctx, err.Error())
						})),
						// httprc.WithTraceSink(tracesink.NewSlog())
						httprc.WithTraceSink(tracesink.Func(func(ctx context.Context, msg string) {
							logger.DebugContext(ctx, msg)
						})),
						// httprc.WithHTTPClient(cl HTTPClient) NewClientResourceOption
						// httprc.WithErrorSink(sink httprc.ErrorSink) NewClientOption
						// httprc.WithTraceSink(sink httprc.TraceSink) NewClientOption
						// httprc.WithWhitelist(wl httprc.Whitelist) NewClientOption
						// httprc.WithWorkers(n int) NewClientOption // default: 5
						// httprc.WithWorkers(
						//   runtime.NumCPU(),
						// ),
					)
				},
				OnStop: func(ctx context.Context) error {
					// return jwks.Default.Close(ctx)
					return jwks.Stop(ctx)
				},
			})
			return
		}),
	), */
)
