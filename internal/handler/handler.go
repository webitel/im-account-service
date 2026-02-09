package handler

import (
	"log/slog"

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
	"handler",
	fx.Provide(
		func(logger *slog.Logger, registry discovery.DiscoveryProvider, broker pubsub.Provider) (*webitel.Client, error) {
			logger = logx.ModuleLogger("go-webitel-client", logger)
			return webitel.NewClient(logger, registry, broker) //, opts...)
		},
		func(logger *slog.Logger, registry discovery.DiscoveryProvider, secure *infra_tls.Config) (c1pb.ContactsClient, error) {
			logger = logx.ModuleLogger("im-contact-client", logger)
			return contacts.NewClient(logger, registry, secure.Client) // , opts...)
		},
		NewService,
	),
)
