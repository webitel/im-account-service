package service

import (
	"log/slog"

	// grpc_srv "github.com/webitel/im-account-service/infra/server/grpc"
	broker "github.com/webitel/im-account-service/infra/pubsub"
	webitel "github.com/webitel/im-account-service/internal/client/webitel/auth"
	"github.com/webitel/im-account-service/internal/store"
	cspb "github.com/webitel/im-account-service/proto/gen/im/service/contact/v1"
	"go.uber.org/fx"
)

type Options struct {

	fx.In // FX: Params.(input)

	Logger *slog.Logger
	// Server  *grpc_srv.Server
	Broker broker.Provider
	// Catalog struct {
	Apps     store.AppStore
	Sessions store.SessionStore

	Webitel  *webitel.Client
	Contacts cspb.ContactsClient

}
