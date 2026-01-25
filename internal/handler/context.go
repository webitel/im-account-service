package handler

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/webitel/im-account-service/internal/errors"
	"github.com/webitel/im-account-service/internal/model"
	"google.golang.org/grpc/metadata"
)

// Authorization Context
type Context struct {
	// Operation ID
	Id string
	// Operation Date
	Date time.Time
	// Logger for this Context
	Logger *slog.Logger
	// Request Header (shorthand)
	Header metadata.MD
	// Operation Context boundary
	Context context.Context
	// Service Handler
	Service *Service

	// Authentication

	// App Application
	// Client *ad.Device

	Dc     int64
	App    *model.Application
	Device *model.Device // Client ( endpoint / instance ) FROM

	// Client *ad.Application // Third-Party Application VIA
	// Native *NativeService  // Native ( internal ) service ( client ) authentication

	Auth    any                  // [A]uthenticated [C]redentials [R]ule ; JWT | session | external ..
	Contact *model.Contact       // Authenticated end-User
	Session *model.Authorization // Authorized session
	// Business ds.Business // Business (domain) Account

	// // Operation (generated) Updates
	// Changes ad.UpdateList // ad.Updates

	// Status
	Error error

	// beforeEnd []func(*Context) error
	// afterEnd  []func(*Context) error

}

// Context Option
type ContextFunc func(rpc *Context) error

type contextKey struct{}

func FromContext(ctx context.Context) (rpc *Context, ok bool) {
	rpc, _ = ctx.Value(contextKey{}).(*Context)
	return rpc, (rpc != nil)
}

func WithContext(ctx context.Context, rpc *Context) context.Context {
	if rpc != nil {
		ctx = context.WithValue(ctx, contextKey{}, rpc)
		rpc.Context = ctx // [re]bind !
	}
	return ctx
}

func NewContext(ctx context.Context, opts ...ContextFunc) (rpc *Context, err error) {

	rpc = &Context{
		Id:   fmt.Sprintf("%p", ctx),
		Date: model.LocalTime.Now(),
		// Business: nil, // &Service,
		// Service:  Service,
		Context: ctx,
	}
	rpc.Header, _ = metadata.FromIncomingContext(rpc.Context)
	return rpc, rpc.Init(opts...)
}

func (ctx *Context) Init(opts ...ContextFunc) error {

	err := ctx.Error
	if err != nil {
		return err
	}

	for _, option := range opts {
		// setup
		err = option(ctx)
		if err != nil {
			// temporary
			return err
		}
		if ctx.Error != nil {
			// critical
			return ctx.Error
		}
	}
	// OK
	return nil
}

func GetContext(ctx context.Context, opts ...ContextFunc) (rpc *Context, err error) {
	if rpc, _ = FromContext(ctx); rpc == nil {
		rpc, _ = NewContext(ctx)
	}
	return rpc, rpc.Init(opts...)
}

func (srv *Service) GetContext(ctx context.Context, opts ...ContextFunc) (rpc *Context, err error) {
	rpc, err = GetContext(ctx, func(rpc *Context) error {
		// init
		if rpc.Service == nil {
			rpc.Service = srv
			rpc.Logger = cmp.Or(rpc.Logger, srv.opts.Logs)
			// rpc.Logger = rpc.Logger.With(
			// 	ContextLog(rpc).Group("rpc"), // freezes current state attributes ; not dynamic !
			// )
		}
		// [re]check
		if rpc.Service != srv {
			return errors.Errorf("messaging: ambiguous [service] authorization")
		}
		// ok
		return nil
	})

	if err == nil && len(opts) > 0 {
		err = rpc.Init(opts...)
	}

	return rpc, err
}
