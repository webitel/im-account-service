package service

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
	// Service (Domain) Manager
	Service *Manager
	// Operation Context boundary
	Context context.Context

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

// Control Option
type Control interface {
	Do(rpc *Context) error
}

// Context Option
type ControlFunc func(rpc *Context) error

var _ Control = ControlFunc(nil)
func (fn ControlFunc) Do(rpc *Context) error {
	if fn != nil {
		return fn(rpc)
	}
	// ignore
	return nil
}

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

func NewContext(ctx context.Context, ctl ...Control) (rpc *Context, err error) {

	rpc = &Context{
		Id:   fmt.Sprintf("%p", ctx),
		Date: model.LocalTime.Now(),
		// Business: nil, // &Service,
		// Service:  Service,
		Context: ctx,
	}
	rpc.Header, _ = metadata.FromIncomingContext(rpc.Context)
	return rpc, rpc.Init(ctl...)
}

func (ctx *Context) Init(ctls ...Control) error {

	err := ctx.Error
	if err != nil {
		return err
	}

	for _, ctrl := range ctls {
		// setup
		err = ctrl.Do(ctx)
		if err != nil {
			// [ temporary ]
			return err
		}
		if ctx.Error != nil {
			// [ critical ]
			return ctx.Error
		}
	}
	// [ OK ]
	return nil
}

func GetContext(ctx context.Context, ctrl ...Control) (rpc *Context, err error) {
	if rpc, _ = FromContext(ctx); rpc == nil {
		rpc, _ = NewContext(ctx)
	}
	return rpc, rpc.Init(ctrl...)
}

func (c *Manager) bind(rpc *Context) error {
	// init
	if rpc.Service == nil {
		rpc.Service = c
		rpc.Logger = cmp.Or(rpc.Logger, c.opts.Logger)
		// rpc.Logger = rpc.Logger.With(
		// 	ContextLog(rpc).Group("rpc"), // freezes current state attributes ; not dynamic !
		// )
	}
	// [re]check
	if rpc.Service != c {
		return errors.Errorf("messaging: ambiguous [service] authorization")
	}
	// ok
	return nil
}

func (c *Manager) GetContext(ctx context.Context, ctrl ...Control) (rpc *Context, err error) {
	
	rpc, err = GetContext(ctx, ControlFunc(c.bind))

	if err == nil && len(ctrl) > 0 {
		err = rpc.Init(ctrl...)
	}

	return rpc, err
}