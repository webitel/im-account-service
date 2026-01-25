package postgres

import (
	"github.com/webitel/im-account-service/internal/store"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"store", fx.Provide(
		fx.Annotate(NewAppStore, fx.As(new(store.AppStore))),
		fx.Annotate(NewSessionStore, fx.As(new(store.SessionStore))),
	),
)
