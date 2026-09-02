package model

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	v1 "github.com/webitel/im-account-service/proto/gen/im/service/admin/v1"
	"google.golang.org/protobuf/proto"
)

type Mutation struct {
	Date   time.Time
	UserId string
}

// Application [external] Configuration
type Application struct {
	// Dc    int64
	// Id    string
	// Name  string
	// About string

	// Clients  *struct{}
	// Account  *struct{}
	// Service  *struct{}
	// Contacts *struct{}

	// Created *Mutation
	// Updated *Mutation
	// Revoked *Mutation

	opts *v1.Application // configuration

	mx      sync.Mutex
	clients AppClients
}

type ApplicationList = Dataset[Application]

func NewApplication(input *v1.InputApp) *Application {
	config := &v1.Application{
		Dc:                  input.GetDc(),
		Id:                  uuid.NewString(),
		Name:                input.GetName(),
		About:               input.GetAbout(),
		Block:               nil, // &impb.Revocation{},
		Clients:             input.GetClients(),
		Service:             input.GetService(), // LIMIT, UPDATES, PUSH
		Account:             nil,                // &impb.Account{},
		Contacts:            input.GetContacts(),
		AllowSystemMessages: input.GetAllowSystemMessages(),
	}
	return &Application{
		opts: config,
	}
}

func (app *Application) Log(level slog.Level, msg string, args ...any) {
	slog.Default().Log(
		context.Background(), level,
		msg, append([]any{slog.String("app.id", app.ClientId())}, args...)...,
	)
}

func (app *Application) GetDc() int64 {
	return app.opts.GetDc()
}

// func (c *Application) GetId() UUID {
// 	return c.src.GetDc()
// }

func (app *Application) ClientId() string {
	return app.opts.GetId()
}

func (app *Application) Proto() *v1.Application {
	return proto.CloneOf(app.opts)
}

func ProtoApplication(src *v1.Application) *Application {
	return &Application{
		opts: proto.CloneOf(src),
	}
}
