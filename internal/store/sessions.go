package store

import (
	"context"

	"github.com/webitel/im-account-service/internal/model"
)

type SessionStore interface {
	//
	Search(ListSessionRequest) (*model.SessionList, error)
	Create(ctx context.Context, session *model.Authorization) error
	Update(ctx context.Context, session *model.Authorization) error
	Delete(ctx context.Context, sessionId string) error

	RegisterDevice(RegisterDeviceRequest) error
	UnregisterDevice(UnregisterDeviceRequest) error

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

type RegisterDeviceRequest struct {
	// Context
	context.Context
	// end-User (Contact) Authorization
	model.Authorization
	// Device token subscription
	Token *model.PushToken
	// List of end-User (Contact) identifiers
	// of other users currently using the Device client.
	//
	// Mostly this field will be blank
	// unless the device client (app)
	// does support multi-sessions.
	OtherUids []*model.ContactId
}

type UnregisterDeviceRequest struct {
	// Context
	context.Context
	// Session.(Authorization).Id
	SessionId string
	// Device (current) token subscription
	// .. to prove that session.Device knows [PUSH] token to be unsubscribed
	Token *model.PushToken
	// List of end-User (Contact) identifiers
	// of other users currently using the Device client.
	//
	// Mostly this field will be blank
	// unless the device client (app)
	// does support multi-sessions.
	OtherUids []*model.ContactId
}


