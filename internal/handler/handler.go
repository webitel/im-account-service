package handler

import (
	"log/slog"

	"github.com/webitel/webitel-go-kit/infra/discovery"
	"go.uber.org/fx"

	"github.com/webitel/im-account-service/internal/client/contacts"
	webitel "github.com/webitel/im-account-service/internal/client/webitel/auth"
	c1pb "github.com/webitel/im-account-service/proto/gen/im/service/contact/v1"
)

var Module = fx.Module(
	"handler",
	fx.Provide(
		func(logger *slog.Logger, registry discovery.DiscoveryProvider) (*webitel.Client, error) {
			return webitel.NewClient(logger, registry)
		},
		func(logger *slog.Logger, registry discovery.DiscoveryProvider) (c1pb.ContactsClient, error) {
			return contacts.NewClient(logger, registry)
		},
		NewService,
	),
)
