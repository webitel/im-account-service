package v1

import (
	"cmp"
	"context"
	"log/slog"
	"net"

	"github.com/google/uuid"
	// v1 "github.com/webitel/im-account-service/gen/auth/v1"
	"github.com/webitel/im-account-service/infra/log/slogx"
	grpcsrv "github.com/webitel/im-account-service/infra/server/grpc"
	"github.com/webitel/im-account-service/internal/errors"
	"github.com/webitel/im-account-service/internal/handler"
	"github.com/webitel/im-account-service/internal/model"
	"github.com/webitel/im-account-service/internal/store"
	v1 "github.com/webitel/im-account-service/proto/gen/im/service/auth/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

var _ v1.AccountServer = &AccountService{}

type AccountService struct {
	v1.UnimplementedAccountServer

	srv *handler.Service
	// logger *slog.Logger
	// // storage  store.AppStore
}

// func NewAccountService(storage store.AppStore, logger *slog.Logger) *AccountService {
// 	return &AccountService{logger: logger}
// }

// type AccountServiceOptions struct {
// 	fx.In   // composite Params
// 	Logger  *slog.Logger
// 	Catalog struct {
// 		Sessions interface{}
// 	}
// }

// func NewAccountService(logger *slog.Logger) *AccountService {
// 	return &AccountService{logger: logger}
// }

func NewAccountService(handler *handler.Service) *AccountService {
	return &AccountService{srv: handler}
}

func RegisterAccountService(server *grpcsrv.Server, handler *AccountService) {
	v1.RegisterAccountServer(server.Server, handler)
}

// func (c *AccountService) mustEmbedUnimplementedAccountServer() {}

// ------------------------------- [API] v1 ---------------------------------------- //

// Access Token Request
func (api *AccountService) Token(ctx context.Context, req *v1.TokenRequest) (*v1.Authorization, error) {

	switch req.GetGrantType().(type) {
	// case *v1.TokenRequest_RefreshToken:
	// 	{
	// 		_ = creds.RefreshToken
	// 	}
	case *v1.TokenRequest_Identity:
		{
			// External (Identity) Contact Login
			rpc, err := api.GrantTokenForUserIdentity(ctx, req)
			if err != nil {
				return nil, err
			}
			// granted := session.Grant
			// return &v1.AccessToken{
			// 	TokenType:    session.Grant.Type,
			// 	AccessToken:  session.Grant.Token,
			// 	RefreshToken: session.Grant.Refresh,
			// 	ExpiresIn:    0,
			// 	Scope:        session.Grant.Scope,
			// 	State:        req.GetState(),
			// }, nil
			session := rpc.Session
			contact := rpc.Contact
			return &v1.Authorization{
				Dc:    session.Dc,
				Id:    session.Id,
				Date:  model.Timestamp.Time(session.Date),
				Name:  session.Name,
				AppId: session.AppId,
				Device: &v1.Device{
					Id: session.Device.Id,
					// Ip: session.Device.IP().String(),
					Ip: session.IP.String(),
					App: &v1.UserAgent{
						Name:      session.Device.App.Name,
						Version:   session.Device.App.Version,
						Os:        session.Device.App.OS,
						OsVersion: session.Device.App.OSVersion,
						Device:    session.Device.App.Device,
						Mobile:    session.Device.App.Mobile,
						Tablet:    session.Device.App.Tablet,
						Desktop:   session.Device.App.Desktop,
						Bot:       session.Device.App.Bot,
						String_:   session.Device.App.String,
					},
					Push: session.Device.Push, // session.Device.Push.GetToken() != nil,
				},
				// Contact: &v1.Identity{
				// 	Iss:                 contact.Iss,
				// 	Sub:                 contact.Sub,
				// 	Name:                contact.Name,
				// 	GivenName:           contact.GivenName,
				// 	MiddleName:          contact.MiddleName,
				// 	FamilyName:          contact.FamilyName,
				// 	Birthdate:           contact.Birthdate,
				// 	Zoneinfo:            contact.Zoneinfo,
				// 	Profile:             contact.Profile,
				// 	Picture:             contact.Picture,
				// 	Gender:              contact.Gender,
				// 	Locale:              contact.Locale,
				// 	Email:               contact.Email,
				// 	EmailVerified:       contact.EmailVerified,
				// 	PhoneNumber:         contact.PhoneNumber,
				// 	PhoneNumberVerified: contact.PhoneNumberVerified,
				// 	Metadata:            nil, // &structpb.Struct{},
				// 	CreatedAt:           model.Timestamp.Time(contact.CreatedAt),
				// 	UpdatedAt:           0,
				// 	DeletedAt:           0,
				// },
				Contact: &v1.Contact{
					Dc:                  contact.Dc,
					Id:                  contact.Id,
					Iss:                 contact.Iss,
					Sub:                 contact.Sub,
					App:                 contact.App,
					Type:                contact.Type,
					Name:                contact.Name,
					GivenName:           contact.GivenName,
					MiddleName:          contact.MiddleName,
					FamilyName:          contact.FamilyName,
					Username:            contact.Username,
					Birthdate:           contact.Birthdate,
					Zoneinfo:            contact.Zoneinfo,
					Profile:             contact.Profile,
					Picture:             contact.Picture,
					Gender:              contact.Gender,
					Locale:              contact.Locale,
					Email:               contact.Email,
					EmailVerified:       contact.EmailVerified,
					PhoneNumber:         contact.PhoneNumber,
					PhoneNumberVerified: contact.PhoneNumberVerified,
					Metadata:            nil, // &structpb.Struct{},
					CreatedAt:           model.Timestamp.Time(contact.CreatedAt),
					UpdatedAt:           0,
					DeletedAt:           0,
				},
				Token: &v1.AccessToken{
					TokenType:    session.Grant.Type,
					AccessToken:  (handler.SessionTokenPrefix + session.Grant.Token),
					RefreshToken: session.Grant.Refresh,
					ExpiresIn:    0, // session.Grant.Expires,
					Scope:        session.Grant.Scope,
					State:        req.GetState(),
				},
				Current: true,
			}, nil
		}
	// case *v1.TokenRequest_Code:
	// 	{
	// 		_ = creds.Code
	// 	}
	default:
		{
			return nil, errors.BadRequest(
				errors.Status("BAD_REQUEST"),
				errors.Message("messaging: invalid [grant_type] request option"),
			)
		}
	}

	return api.UnimplementedAccountServer.Token(ctx, req)
}

// Logout Device Request
func (api *AccountService) Logout(ctx context.Context, req *v1.LogoutRequest) (*v1.LogoutResponse, error) {

	// Authorization
	rpc, err := api.srv.GetContext(
		// RPC Operation Context
		ctx,
		// [X-Webitel-Client] ; REQUIRED
		handler.AppAuthorization(true),
		// [X-Webitel-Device] ; REQUIRED
		handler.DeviceAuthorization(true),
		// [X-Webitel-Access] ; REQUIRED
		handler.EndUserAuthorization(true),
	)

	// Authorized ?
	if err = cmp.Or(err, rpc.Error); err != nil {
		return nil, err
	}

	// Session Authorized ?
	session := rpc.Session
	if session == nil || session.Id == "" {
		// Not (internal) session authorization ! Not supported !
	}

	switch authN := rpc.Auth.(type) {
	// IM (internal) session !
	case *model.AccessToken:
		{
			sessions := api.srv.Options().Sessions
			err := sessions.Delete(rpc.Context, authN.Id.String())
			if err != nil {
				// something went wrong
				return nil, err
			}
			// [ OK ]
			return &v1.LogoutResponse{}, nil
		}
	}

	// default:
	return nil, handler.ErrTokenInvalid
	// return api.UnimplementedAccountServer.Logout(ctx, req)
}

// Inspect [Authorization] Request
func (api *AccountService) Inspect(ctx context.Context, req *v1.InspectRequest) (*v1.Authorization, error) {

	// region: Authentication
	rpc, err := api.srv.GetContext(
		// RPC Operation Context
		ctx,
		// [X-Webitel-Client] ; Dc | App
		handler.AppAuthorization(false),
		// [X-Webitel-Device] ; Client
		handler.DeviceAuthorization(false),
		// [X-Webitel-Access] ; Contact
		handler.EndUserAuthorization(true),
	)

	if err = cmp.Or(err, rpc.Error); err != nil {
		return nil, err
	}

	// if rpc.Auth == nil {
	// 	// UNAUTHORIZED
	// 	return nil, handler.ErrAccountUnauthorized
	// }

	rpc.Debug("Inspect Authorization")

	// encode: v1
	return currentAuthorizationProtoV1(rpc)

	// var (
	// 	// err  error
	// 	auth = Authorization{
	// 		Context: ctx,
	// 		Date:    model.LocalTime.Now(),
	// 	}
	// 	// HTTP/2.0 Header
	// 	header, _ = metadata.FromIncomingContext(ctx)
	// )
	// // [X-Webitel-Device]
	// auth.Client, _ = model.GetDeviceAuthorization(ctx)
	// // [X-Webitel-Client]
	// auth.App, err = api.srv.GetAppAuthorization(ctx, false)
	// if err != nil {
	// 	// UNAUTHORIZED_CLIENT
	// 	return nil, err
	// }

	// err = handler.EndUserAuthorization(rpc, false)
	// if err != nil {
	// 	return nil, err
	// }

	// // if vs, ok := header[model.H2_X_Client_ID]; ok {
	// // 	clientId := model.CoalesceLast(vs...)
	// // 	app, err := api.srv.GetApplication(ctx, clientId)
	// // 	if err != nil {
	// // 		// failed get client (app) config
	// // 		return nil, err
	// // 	}
	// // 	if app == nil {
	// // 		// Not Found ; invalid [client_id] spec
	// // 		return nil, errors.Unauthorized(
	// // 			errors.Status("UNAUTHORIZED_CLIENT"),
	// // 			errors.Message("messaging: invalid [client_id] identifier"),
	// // 		)
	// // 	}
	// // 	auth.App = app
	// // 	api.srv.AuthorizeAppClient(auth.Context, auth.App, &auth.Client)
	// // }
	// // [X-Webitel-Access]
	// if vs, ok := header[model.H2_X_Access_Token]; ok {
	// 	bearer := model.CoalesceLast(vs...)
	// 	// Accept:
	// 	// - JWT
	// 	// - token ; go.webitel.app session
	// 	// - token ; im-account-service session
	// 	_ = bearer
	// }

	// _ = auth
	// endregion: Authentication

	return api.UnimplementedAccountServer.Inspect(ctx, req)
}

// Register device to receive PUSH notifications
func (api *AccountService) RegisterDevice(ctx context.Context, req *v1.RegisterDeviceRequest) (*v1.RegisterDeviceResponse, error) {

	// region: Request Validation
	var ok bool
	switch regtoken := req.Push.Token.(type) {
	case *v1.PUSHSubscription_Fcm:
		ok = (regtoken.Fcm != "")
	case *v1.PUSHSubscription_Apn:
		ok = (regtoken.Apn != "")
	case *v1.PUSHSubscription_Web:
		ok = (regtoken.Web.GetEndpoint() != "")
	}

	if !ok { // if req.GetPush().GetToken() == nil {
		return nil, errors.BadRequest(
			errors.Message("register: PUSH token required"),
		)
	}
	// endregion: Request Validation

	// JWT      ; (external: App)             ; NO (internal) session to attach PUSH token  =((
	// Webitel  ; Contact{ type: "webitel" }  ; NO (internal) session to attach PUSH token  =((
	// Session                                ; OK

	// region: Authentication
	rpc, err := api.srv.GetContext(
		// RPC Operation Context
		ctx,
		// [X-Webitel-Client] ; Dc | App
		handler.AppAuthorization(false),
		// [X-Webitel-Device] ; Client
		handler.DeviceAuthorization(true),
		// [X-Webitel-Access] ; Contact
		handler.EndUserAuthorization(true),
	)

	if err = cmp.Or(err, rpc.Error); err != nil {
		return nil, err
	}

	// if rpc.Auth == nil {
	// 	// UNAUTHORIZED
	// 	return nil, handler.ErrAccountUnauthorized
	// }

	app := rpc.App
	service := app.Proto().GetService().GetPushService()
	switch req.Push.Token.(type) {
	case *v1.PUSHSubscription_Fcm:
		{
			if service.GetFcm() == nil {
				// not supported
			}
		}
	case *v1.PUSHSubscription_Apn:
		{
			if service.GetApn() == nil {
				// not supported
			}
		}
	case *v1.PUSHSubscription_Web:
		{
			// TODO
		}
	default:
	}

	// PERFORM: register for current session
	repo := api.srv.Options().Sessions
	err = repo.RegisterDevice(
		rpc.Context, rpc.Session.Id, req.Push,
	)

	if err != nil {
		return nil, err
	}

	return &v1.RegisterDeviceResponse{}, nil
	// return api.UnimplementedAccountServer.RegisterDevice(ctx, req)
}

// Deletes a device by its token, stops sending PUSH-notifications to it.
func (api *AccountService) UnregisterDevice(ctx context.Context, req *v1.UnregisterDeviceRequest) (*v1.UnregisterDeviceResponse, error) {

	// region: Request Validation
	var ok bool
	// REQUIRE: current PUSH token to deregister
	switch regtoken := req.Push.Token.(type) {
	case *v1.PUSHSubscription_Fcm:
		ok = (regtoken.Fcm != "")
	case *v1.PUSHSubscription_Apn:
		ok = (regtoken.Apn != "")
	case *v1.PUSHSubscription_Web:
		ok = (regtoken.Web.GetEndpoint() != "")
	}

	if !ok { // if req.GetPush().GetToken() == nil {
		return nil, errors.BadRequest(
			errors.Message("unregister: PUSH token required"),
		)
	}
	// endregion: Request Validation

	// JWT      ; (external: App)             ; NO (internal) session to attach PUSH token  =((
	// Webitel  ; Contact{ type: "webitel" }  ; NO (internal) session to attach PUSH token  =((
	// Session                                ; OK

	// region: Authentication
	rpc, err := api.srv.GetContext(
		// RPC Operation Context
		ctx,
		// [X-Webitel-Client] ; Dc | App
		handler.AppAuthorization(false),
		// [X-Webitel-Device] ; Client
		handler.DeviceAuthorization(true),
		// [X-Webitel-Access] ; Contact
		handler.EndUserAuthorization(true),
	)

	if err = cmp.Or(err, rpc.Error); err != nil {
		return nil, err
	}

	// if rpc.Auth == nil {
	// 	// UNAUTHORIZED
	// 	return nil, handler.ErrAccountUnauthorized
	// }

	app := rpc.App
	service := app.Proto().GetService().GetPushService()
	switch req.Push.Token.(type) {
	case *v1.PUSHSubscription_Fcm:
		{
			if service.GetFcm() == nil {
				// not supported
			}
		}
	case *v1.PUSHSubscription_Apn:
		{
			if service.GetApn() == nil {
				// not supported
			}
		}
	case *v1.PUSHSubscription_Web:
		{
			// TODO
		}
	default:
	}

	// PERFORM: deregister for current session
	repo := api.srv.Options().Sessions
	err = repo.UnregisterDevice(
		rpc.Context, rpc.Session.Id, req.Push,
	)

	if err != nil {
		return nil, err
	}

	return &v1.UnregisterDeviceResponse{}, nil
	// return api.UnimplementedAccountServer.UnregisterDevice(ctx, req)
}

// // Get logged-in session(s)
// // https://core.telegram.org/method/account.getAuthorizations
// func (api *AccountService) GetSessions(ctx context.Context, req *v1.GetSessionRequest) (*v1.SessionList, error) {
// 	return api.UnimplementedAccountServer.GetSessions(ctx, req)
// }

// Get logged-in session(s)
// https://core.telegram.org/method/account.getAuthorizations
func (api *AccountService) GetAuthorizations(ctx context.Context, req *v1.GetAuthorizationRequest) (*v1.AuthorizationList, error) {

	// TODO: OPTIONAL Authorization
	rpc, err := api.srv.GetContext(
		// RPC Operation Context
		ctx,
		// [X-Webitel-Client] ; Dc | App
		handler.AppAuthorization(false),
		// [X-Webitel-Device] ; Client
		handler.DeviceAuthorization(false),
		// [X-Webitel-Access] ; Contact
		handler.EndUserAuthorization(false),
	)

	lookup := store.ListSessionRequest{
		Context: ctx,
		Page:    max(int(req.GetPage()), 0),
		Size:    max(int(req.GetSize()), 0),

		Dc:        max(req.GetDc(), 0),
		Id:        req.GetId(),
		AppId:     req.GetAppId(),
		Token:     "", // req.GetToken(),
		DeviceId:  req.GetDeviceId(),
		ContactId: nil,
	}

	if input := req.GetPush(); input != nil {
		value := input.Value
		lookup.PushToken = &value
	}

	if req.GetContact().GetInput() != nil {
		switch input := req.GetContact().GetInput().(type) {
		case *v1.InputContactId_Id:
			{
				lookup.ContactId = &model.ContactId{
					Id: input.Id,
				}
			}
		case *v1.InputContactId_Source:
			{
				lookup.ContactId = &model.ContactId{
					Iss: input.Source.GetIss(),
					Sub: input.Source.GetSub(),
				}
			}
		default:
			{
				return nil, errors.BadRequest(
					errors.Message("authorization: invalid [contact] request option"),
				)
			}
		}
	}

	repo := api.srv.Options().Sessions
	list, err := repo.Search(lookup)
	if err != nil {
		return nil, err
	}

	size := len(list.Data)
	res := &v1.AuthorizationList{
		Data: make([]*v1.Authorization, 0, size),
		Page: max(req.GetPage(), 1),
		Next: (list.Next != nil),
	}

	var currentId string
	if rpc.Session != nil {
		currentId = rpc.Session.Id
	}

	for _, session := range list.Data {
		row := authorizationFormProtoV1(session)
		row.Current = (currentId != "" && row.Id == currentId)
		res.Data = append(res.Data, row)
	}

	return res, nil
	// return api.UnimplementedAccountServer.GetAuthorizations(ctx, req)
}

// ------------------------------- Authentication ---------------------------------------- //

func netIPstring(ip net.IP) string {
	if len(ip) == 0 {
		return ""
	}
	return ip.String()
}

func currentAuthorizationProtoV1(rpc *handler.Context) (*v1.Authorization, error) {

	authN := &v1.Authorization{
		Current: true,
	}

	session := rpc.Session
	if session != nil {

		authN.Dc = session.Dc
		authN.Id = session.Id
		authN.Date = model.Timestamp.Time(session.Date)
		authN.Name = session.Name
		authN.AppId = session.AppId

		device := &session.Device
		authN.Device = &v1.Device{
			Id:   device.Id,
			Ip:   netIPstring(device.IP()),
			Push: device.Push,
		}

		if agent := &device.App; agent.String != "" {
			authN.Device.App = &v1.UserAgent{
				Name:      agent.Name,
				Version:   agent.Version,
				Os:        agent.OS,
				OsVersion: agent.OSVersion,
				Device:    agent.Device,
				Mobile:    agent.Mobile,
				Tablet:    agent.Mobile,
				Desktop:   agent.Desktop,
				Bot:       agent.Bot,
				String_:   agent.String,
			}
		}

		if contact := session.Contact; contact != nil {
			authN.Contact = &v1.Contact{
				Dc:  contact.Dc, // == session.Dc
				Id:  contact.Id,
				Iss: contact.Iss,
				Sub: contact.Sub,
			}
			// authN.Contact = &v1.Identity{
			// 	Iss: contact.Iss,
			// 	Sub: contact.Sub,
			// }
		}

	}

	// current (latest) device from request
	if device := rpc.Device; device != nil {

		authN.Device = &v1.Device{
			Id:   cmp.Or(device.Id, authN.Device.GetId()),
			Ip:   netIPstring(device.IP()),
			Push: cmp.Or(device.Push, authN.Device.GetPush()),
		}

		if agent := &device.App; agent.String != "" {
			authN.Device.App = &v1.UserAgent{
				Name:      agent.Name,
				Version:   agent.Version,
				Os:        agent.OS,
				OsVersion: agent.OSVersion,
				Device:    agent.Device,
				Mobile:    agent.Mobile,
				Tablet:    agent.Mobile,
				Desktop:   agent.Desktop,
				Bot:       agent.Bot,
				String_:   agent.String,
			}
		}

	}

	// current (latest) contact info
	if contact := rpc.Contact; contact != nil {
		metadata, _ := structpb.NewStruct(contact.Metadata)
		authN.Contact = &v1.Contact{
			Dc:                  contact.Dc,
			Id:                  contact.Id,
			Iss:                 contact.Iss,
			Sub:                 contact.Sub,
			App:                 contact.App,
			Type:                contact.Type,
			Name:                contact.Name,
			GivenName:           contact.GivenName,
			MiddleName:          contact.MiddleName,
			FamilyName:          contact.FamilyName,
			Username:            contact.Username,
			Birthdate:           contact.Birthdate,
			Zoneinfo:            contact.Zoneinfo,
			Profile:             contact.Profile,
			Picture:             contact.Picture,
			Gender:              contact.Gender,
			Locale:              contact.Locale,
			Email:               contact.Email,
			EmailVerified:       contact.EmailVerified,
			PhoneNumber:         contact.PhoneNumber,
			PhoneNumberVerified: contact.PhoneNumberVerified,
			Metadata:            metadata,
			CreatedAt:           model.Timestamp.Time(contact.CreatedAt),
			UpdatedAt:           0,
			DeletedAt:           0,
		}
		// authN.Contact = &v1.Identity{
		// 	Iss:                 contact.Iss,
		// 	Sub:                 contact.Sub,
		// 	Name:                contact.Name,
		// 	GivenName:           contact.GivenName,
		// 	MiddleName:          contact.MiddleName,
		// 	FamilyName:          contact.FamilyName,
		// 	Birthdate:           contact.Birthdate,
		// 	Zoneinfo:            contact.Zoneinfo,
		// 	Profile:             contact.Profile,
		// 	Picture:             contact.Picture,
		// 	Gender:              contact.Gender,
		// 	Locale:              contact.Locale,
		// 	Email:               contact.Email,
		// 	EmailVerified:       contact.EmailVerified,
		// 	PhoneNumber:         contact.PhoneNumber,
		// 	PhoneNumberVerified: contact.PhoneNumberVerified,
		// 	Metadata:            metadata,
		// 	CreatedAt:           model.Timestamp.Time(contact.CreatedAt),
		// 	UpdatedAt:           0,
		// 	DeletedAt:           0,
		// }
	}

	return authN, nil
}

func authorizationFormProtoV1(src *model.Authorization) *v1.Authorization {

	if src == nil {
		return nil
	}

	dst := &v1.Authorization{
		Dc:      src.Dc,
		Id:      src.Id,
		Date:    model.Timestamp.Time(src.Date),
		Name:    src.Name,
		AppId:   src.AppId,
		Device:  nil, // &v1.Device{},
		Contact: nil, // &v1.Contact{},
		Token:   nil, // &v1.AccessToken{},
		Current: false,
	}

	// dst.Dc = src.Dc
	// dst.Id = src.Id
	// dst.Date = model.Timestamp.Time(src.Date)
	// dst.Name = src.Name
	// dst.AppId = src.AppId

	device := &src.Device
	dst.Device = &v1.Device{
		Id:   device.Id,
		Ip:   netIPstring(device.IP()),
		Push: device.Push,
	}

	if agent := &device.App; agent.String != "" {
		dst.Device.App = &v1.UserAgent{
			Name:      agent.Name,
			Version:   agent.Version,
			Os:        agent.OS,
			OsVersion: agent.OSVersion,
			Device:    agent.Device,
			Mobile:    agent.Mobile,
			Tablet:    agent.Mobile,
			Desktop:   agent.Desktop,
			Bot:       agent.Bot,
			String_:   agent.String,
		}
	}

	if contact := src.Contact; contact != nil {
		dst.Contact = &v1.Contact{
			Dc:  contact.Dc, // == session.Dc
			Id:  contact.Id,
			Iss: contact.Iss,
			Sub: contact.Sub,
		}
		// authN.Contact = &v1.Identity{
		// 	Iss: contact.Iss,
		// 	Sub: contact.Sub,
		// }
	}

	// // current (latest) device from request
	// if device := rpc.Device; device != nil {

	// 	dst.Device = &v1.Device{
	// 		Id:   cmp.Or(device.Id, dst.Device.GetId()),
	// 		Ip:   netIPstring(device.IP()),
	// 		Push: cmp.Or(device.Push, dst.Device.GetPush()),
	// 	}

	// 	if agent := &device.App; agent.String != "" {
	// 		dst.Device.App = &v1.UserAgent{
	// 			Name:      agent.Name,
	// 			Version:   agent.Version,
	// 			Os:        agent.OS,
	// 			OsVersion: agent.OSVersion,
	// 			Device:    agent.Device,
	// 			Mobile:    agent.Mobile,
	// 			Tablet:    agent.Mobile,
	// 			Desktop:   agent.Desktop,
	// 			Bot:       agent.Bot,
	// 			String_:   agent.String,
	// 		}
	// 	}

	// }

	// // current (latest) contact info
	// if contact := rpc.Contact; contact != nil {
	// 	metadata, _ := structpb.NewStruct(contact.Metadata)
	// 	dst.Contact = &v1.Contact{
	// 		Dc:                  contact.Dc,
	// 		Id:                  contact.Id,
	// 		Iss:                 contact.Iss,
	// 		Sub:                 contact.Sub,
	// 		App:                 contact.App,
	// 		Type:                contact.Type,
	// 		Name:                contact.Name,
	// 		GivenName:           contact.GivenName,
	// 		MiddleName:          contact.MiddleName,
	// 		FamilyName:          contact.FamilyName,
	// 		Username:            contact.Username,
	// 		Birthdate:           contact.Birthdate,
	// 		Zoneinfo:            contact.Zoneinfo,
	// 		Profile:             contact.Profile,
	// 		Picture:             contact.Picture,
	// 		Gender:              contact.Gender,
	// 		Locale:              contact.Locale,
	// 		Email:               contact.Email,
	// 		EmailVerified:       contact.EmailVerified,
	// 		PhoneNumber:         contact.PhoneNumber,
	// 		PhoneNumberVerified: contact.PhoneNumberVerified,
	// 		Metadata:            metadata,
	// 		CreatedAt:           model.Timestamp.Time(contact.CreatedAt),
	// 		UpdatedAt:           0,
	// 		DeletedAt:           0,
	// 	}
	// 	// authN.Contact = &v1.Identity{
	// 	// 	Iss:                 contact.Iss,
	// 	// 	Sub:                 contact.Sub,
	// 	// 	Name:                contact.Name,
	// 	// 	GivenName:           contact.GivenName,
	// 	// 	MiddleName:          contact.MiddleName,
	// 	// 	FamilyName:          contact.FamilyName,
	// 	// 	Birthdate:           contact.Birthdate,
	// 	// 	Zoneinfo:            contact.Zoneinfo,
	// 	// 	Profile:             contact.Profile,
	// 	// 	Picture:             contact.Picture,
	// 	// 	Gender:              contact.Gender,
	// 	// 	Locale:              contact.Locale,
	// 	// 	Email:               contact.Email,
	// 	// 	EmailVerified:       contact.EmailVerified,
	// 	// 	PhoneNumber:         contact.PhoneNumber,
	// 	// 	PhoneNumberVerified: contact.PhoneNumberVerified,
	// 	// 	Metadata:            metadata,
	// 	// 	CreatedAt:           model.Timestamp.Time(contact.CreatedAt),
	// 	// 	UpdatedAt:           0,
	// 	// 	DeletedAt:           0,
	// 	// }
	// }

	return dst
}

func (api *AccountService) GrantTokenForUserIdentity(ctx context.Context, req *v1.TokenRequest) (*handler.Context, error) {

	idToken := req.GetIdentity()
	// [Verify]:
	// ! REQUIRE: iss, sub, name
	// ? app.Contacts.Issuer == idToken.Iss
	contact := &model.Contact{
		// Dc:                  app.GetDc(),
		Iss:                 idToken.Iss,
		Sub:                 idToken.Sub,
		Type:                "",
		Name:                idToken.Name,
		GivenName:           idToken.GivenName,
		MiddleName:          idToken.MiddleName,
		FamilyName:          idToken.FamilyName,
		Birthdate:           idToken.Birthdate,
		Zoneinfo:            idToken.Zoneinfo,
		Profile:             idToken.Profile,
		Picture:             idToken.Picture,
		Gender:              idToken.Gender,
		Locale:              idToken.Locale,
		Email:               idToken.Email,
		EmailVerified:       idToken.EmailVerified,
		PhoneNumber:         idToken.PhoneNumber,
		PhoneNumberVerified: idToken.PhoneNumberVerified,
		Metadata:            idToken.Metadata.AsMap(),
		// CreatedAt:           time.Time{},
		// UpdatedAt:           &time.Time{},
		// DeletedAt:           &time.Time{},
	}

	rpc, err := api.srv.GetContext(
		// RPC Operation Context
		ctx,
		// [X-Webitel-Client] ; REQUIRED
		handler.AppAuthorization(true),
		// [X-Webitel-Device] ; REQUIRED
		handler.DeviceAuthorization(true),
		// [X-Webitel-Access] ; OPTIONAL
		// Used as a [hint] to resolve previously assigned session
		handler.EndUserAuthorization(false),
	)

	if err = cmp.Or(err, rpc.Error); err != nil {
		return nil, err
	}

	// Verifies given Contact profile
	// meets relative App constraints
	err = rpc.App.NewIdentity(contact)
	if err != nil {
		return rpc, err
	}

	// contact.Dc = rpc.App.GetDc()
	// Validate Contact.Iss VIA client App used to login ...
	// err = rpc.App.NewContact(contact)

	// Save ( Update | Create ) given Contact profile as latest known source
	err = api.srv.AddContact(rpc.Context, contact)
	if err != nil {
		// Failed to save Contact latest source
		return rpc, err
	}

	// Authorize Contact for Login
	rpc.Contact = contact

	// previous session (port) resolved ?
	hint := rpc.Session

	// FindSession(!)
	if hint == nil {
		// FIXME: lookup session( app_id, device_id, contact_id );
		sessions := api.srv.Options().Sessions
		hint, err = model.Get(sessions.Search(
			store.ListSessionRequest{
				Context:   rpc.Context,
				Dc:        rpc.App.GetDc(),
				ContactId: nil,
				DeviceId:  rpc.Device.Id,
				AppId:     rpc.App.ClientId(),
				Token:     "",
				Page:      1,
				Size:      1,
			},
		))
		if err != nil {
			api.srv.Options().Logs.Warn(
				"Failed lookup session",
				"error", err,
			)
			hint, err = nil, nil
		}
	}

	if hint != nil {
		// CHECK: has [access_token] grant been already assigned & active ?
		if err := hint.Grant.Verify(rpc.Date); err == nil {
			// Authorize WITH an active [access_token] granted !
			rpc.Session = hint
			rpc.Logger.Log(
				rpc.Context, (slog.LevelInfo + 1),
				"FOUND Authorization Token",
				"session", slogx.DeferValue(func() slog.Value {
					return slog.GroupValue(
						slog.Int64("dc", hint.Dc),
						slog.String("id", hint.Id),
						slog.String("name", hint.Name),
						slog.String("app.id", hint.AppId),
						slog.Group("device",
							"id", hint.Device.Id,
							"push", (hint.Device.Push.GetToken() != nil),
						),
						slog.Group("contact",
							"iss", hint.Contact.Iss,
							"sub", hint.Contact.Sub,
						),
					)
				}),
			)
			return rpc, nil
		}
	}

	session := hint // current
	// const (
	// 	active uint8 = iota
	// 	update
	// 	create
	// )
	// todo := uint8(active)

	// NewSession(!)
	if session == nil {
		session = &model.Authorization{
			Dc:   rpc.App.GetDc(),
			Id:   uuid.NewString(),
			IP:   rpc.Device.IP(),
			Date: rpc.Date,
			Name: model.SessionName(rpc.Device),
			// Grant:  &grant,
			AppId:  rpc.App.ClientId(),
			Device: (*rpc.Device),
			Contact: &model.ContactId{
				Dc:  contact.Dc,
				Iss: contact.Iss,
				Sub: contact.Sub,
			},
			Metadata: make(map[string]any),
			// Current:  true,
		}

		rpc.Logger.Log(
			rpc.Context, (slog.LevelInfo + 1),
			"NEW Authorization Session",
			"session", slogx.DeferValue(func() slog.Value {
				return slog.GroupValue(
					slog.Int64("dc", session.Dc),
					slog.String("id", session.Id),
					slog.String("name", session.Name),
					slog.String("app.id", session.AppId),
					slog.Group("device",
						"id", session.Device.Id,
						"push", (session.Device.Push.GetToken() != nil),
					),
					slog.Group("contact",
						"iss", session.Contact.Iss,
						"sub", session.Contact.Sub,
					),
				)
			}),
		)
	}

	if session.Grant == nil {
		// todo = max(todo, update)
		// Generate NEW [access_token] for session Authorization !
		grant, err := handler.TokenGen.Generate(
			model.TokenNoRefresh(),
			model.TokenNotBefore(rpc.Date),
			model.TokenScope(req.GetScope()),
		)

		if err != nil {
			return nil, err
		}

		// if grant.Token != "" && !strings.HasPrefix(grant.Token, handler.SessionTokenPrefix) {
		// 	grant.Token = handler.SessionTokenPrefix + grant.Token
		// }

		// assign !
		session.Grant = &grant

		rpc.Logger.Log(
			rpc.Context, (slog.LevelInfo + 1),
			"NEW Token [RE]Generation",
			"session", slogx.DeferValue(func() slog.Value {
				return slog.GroupValue(
					slog.Int64("dc", session.Dc),
					slog.String("id", session.Id),
					slog.String("name", session.Name),
					slog.String("app.id", session.AppId),
					slog.Group("device",
						"id", session.Device.Id,
						"push", (session.Device.Push.GetToken() != nil),
					),
					slog.Group("contact",
						"iss", session.Contact.Iss,
						"sub", session.Contact.Sub,
					),
				)
			}),
		)
	}

	if hint == nil {
		// CREATE
		// TODO: save session last known state !
		err = rpc.Service.Options().Sessions.Create(
			rpc.Context, session,
		)
	} else {
		// UPDATE ; rotate session_token grant
	}

	if err != nil {
		return rpc, err
	}

	// Authorize session grant
	rpc.Session = session

	return rpc, nil
}

// // Authorization. credentials
// type Authorization struct {
// 	context.Context
// 	Date time.Time
// 	// // Credentials
// 	// DeviceId string
// 	// ClientId string
// 	// Authorization
// 	App     *model.Application // Client (VIA) App
// 	Client  model.Device       // Client (Device) app endpoint
// 	Session *model.Session     // IM service (internal) session
// 	// Authenticated credentials, e.g.: Token, JWT
// 	Creds any
// }
