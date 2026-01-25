package store

import (
	"context"

	"github.com/webitel/im-account-service/internal/model"
)

type SessionStore interface {
	//
	Search(ListSessionRequest) (*model.SessionList, error)
	Create(ctx context.Context, session *model.Authorization) error
	Delete(ctx context.Context, sessionId string) error

	RegisterDevice(ctx context.Context, sessionId string, pushToken *model.PushToken) error
	UnregisterDevice(ctx context.Context, sessionId string, pushToken *model.PushToken) error
}

type ListSessionRequest struct {
	// Context
	context.Context
	// Filter(s)
	Dc        int64
	Id        string
	AppId     string // [X-Webitel-Client] ; App.ID
	Token     string // [X-Webitel-Access]
	DeviceId  string // [X-Webitel-Device] ; Sub.ID
	ContactId *model.ContactId
	PushToken *bool // filter sessions with/without push token
	// Pagination
	Page, Size int
}

type CreateSessionRequest struct {
	// Context
	context.Context
	// Filter(s)
	Dc       int64
	UserId   int64
	DeviceId string // [X-Webitel-Device] ; Sub.ID
	ClientId string // [X-Webitel-Client] ; App.ID
	Token    string // [X-Webitel-Access]
}
