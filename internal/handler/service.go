package handler

import (
	"log/slog"

	// grpc_srv "github.com/webitel/im-account-service/infra/server/grpc"
	"github.com/webitel/im-account-service/infra/pubsub"
	auth "github.com/webitel/im-account-service/internal/client/webitel/auth"
	"github.com/webitel/im-account-service/internal/store"
	cspb "github.com/webitel/im-account-service/proto/gen/im/service/contact/v1"
	"go.uber.org/fx"
)

// Service (Handler) Options
type ServiceOptions struct {
	fx.In // Input Params
	Logs  *slog.Logger
	// Server  *grpc_srv.Server
	Broker pubsub.Provider
	// Catalog struct {
	Apps     store.AppStore
	Sessions store.SessionStore

	Webitel  *auth.Client
	Contacts cspb.ContactsClient
	// }
}

// Service Handler
type Service struct {
	opts ServiceOptions
}

func NewService(opts ServiceOptions) (*Service, error) {
	return &Service{
		opts: opts,
	}, nil
}

func (h *Service) Options() ServiceOptions {
	return h.opts
}
