package handler

import (
	"cmp"
	"log/slog"
	"strings"

	"github.com/webitel/im-account-service/infra/log/slogx"
	"github.com/webitel/im-account-service/internal/errors"
	"github.com/webitel/im-account-service/internal/model"
	"github.com/webitel/im-account-service/internal/store"
)

// Session (internal) Authentication scheme
type SessionAuth struct{}

var _ Authentication = SessionAuth{}

// Authenticate ctx.Account (User) Identity.
// [ACR] stands for [A]uthenticated [C]redentials [R]ule.
// If [acr] was returned, it means that the authorization data
// satisfies the authentication scheme policy and no further methods will be involved
//
// Non-nil [acr] indicates accept of credentials
// Non-nil [err] indicates failure of verification
func (SessionAuth) Auth(rpc *Context) (acr any, err error) {
	// [X-Webitel-Access]: [token] ; Authorization
	bearer := model.GetHeaderH2(
		rpc.Header, model.H2_X_Access_Token,
	)
	if bearer == "" {
		// No Authorization !
		return nil, nil
	}

	var ok bool // bearer.(token) accepted ?
	ok, err = SessionAuthentication(rpc, bearer)
	if ok && err != nil {
		// acr = err ; accepted -but- invalid !
		return err, err
	}
	if ok {
		// acr = session.(*AccessToken)
		return rpc.Session.Grant, nil
	}

	return // nil, err
}

// String policy name
func (SessionAuth) String() string {
	return "session"
}

// prefix for quick indication of the service [internal] access token
const SessionTokenPrefix = "im:"

func SessionAuthentication(rpc *Context, token string) (ok bool, err error) {
	if ok = (SessionTokenPrefix != ""); ok {
		if token, ok = strings.CutPrefix(token, SessionTokenPrefix); !ok {
			// This is NOT expected token format
			return // false, nil // ErrTokenInvalid
		}
	}
	// [ok]
	//   ?  [true]   accepted: has prefix defined
	//   :  [false]  unknown -but- we need to check ..
	lookup := store.ListSessionRequest{

		Context: rpc.Context,
		Page:    1,
		Size:    1,

		Dc:       0,
		Token:    token,
		AppId:    "",
		DeviceId: "",
	}

	// if via := ctx.App; via != nil {
	// 	lookup.Dc = via.GetDc()
	// 	lookup.AppId = via.ClientId() // app.[client_id] ; token
	// }
	// if dev := ctx.Device; dev != nil {
	// 	lookup.DeviceId = dev.Id // subscriber.id ; token
	// }

	// FIXME: according to above filters existed session MAY NOT be returned
	// DESIGN: lookup by [token] and than check authorization creds match

	// TODO: lookup session for given token
	sessions := rpc.Service.Options().Sessions
	session, err := model.Get(sessions.Search(lookup))

	if err != nil {
		// storage internal error
		return // ok?, err
	}
	// Ensure [access_token] string matched !
	if session != nil {
		if session.Grant == nil || session.Grant.Token != token {
			session = nil // invalidate ; not found ; MAY: be storage lookup filter(s) apply issue ...
		}
	}

	if ok && session == nil {
		// accepted token format -but- session not found !
		return true, ErrTokenInvalid
	}

	if session == nil {
		// no token hard prefix defined
		// not sure it's our token, but no session found
		// try other authentication policy schemes ..
		return false, nil
	}

	// Authorize [access_token] grant requested !
	if session.Grant == nil || session.Grant.Token != token {
		return true, ErrTokenInvalid
	}
	// Verify token (grant) can be used ; ( !revoked | nbf < date < exp | .. )
	err = session.Grant.Verify(rpc.Date)
	if err != nil {
		// invalid / revoked   access token
		return true, err
	}

	return true, AuthorizeSession(rpc, session)
}

func AuthorizeSession(rpc *Context, session *model.Authorization) error {

	if session == nil {
		return ErrTokenInvalid
	}

	// current Authorization
	var (
		err    error
		app    = rpc.App
		device = rpc.Device
	)
	// Ensure: ( app | device ) match with session Authorization
	if app != nil && app.ClientId() != session.AppId {
		// WARN: Authorization [session] invalid [app.id]
		rpc.Warn( // rpc.Logger.WarnContext(rpc.Context,
			"Authorization [session] invalid [client.id]", // [app.id] ; [X-Webitel-Client]
			"rpc.session.id", slogx.DeferValue(func() slog.Value {
				return slog.StringValue(session.Id)
			}),
		)
		return ErrTokenInvalid
	}
	if device != nil && device.Id != session.Device.Id {
		// WARN: Authorization [session] invalid [device.id]
		rpc.Warn( // rpc.Logger.WarnContext(rpc.Context,
			"Authorization [session] invalid [client.sub]", // [device.id] ; [X-Webitel-Device]
			"rpc.session.id", slogx.DeferValue(func() slog.Value {
				return slog.StringValue(session.Id)
			}),
		)
		// return ErrDeviceAuthorization // ErrTokenInvalid
		return errors.Unauthorized(
			errors.Status("UNAUTHORIZED_CLIENT"),
			errors.Message("messaging: device not authorized"),
		)
	}
	// Load references if not yet ..
	if app == nil && session.AppId != "" {
		// TODO: Authorize Application
		app, err = rpc.Service.GetApplication(rpc.Context, session.AppId)
		if err != nil {
			return err
		}
		if app == nil || app.ClientId() != session.AppId {
			return errors.Unauthorized(
				errors.Status("UNAUTHORIZED_CLIENT"),
				errors.Message("messaging: application not authorized"),
			)
		}
	}
	// expose latest known session device registration
	if device == nil { // && session.Device.Id != "" {
		clone := session.Device
		device = &clone
	}
	device.Id = cmp.Or(device.Id, session.Device.Id)
	device.Push = session.Device.Push

	// resolved
	rpc.App = app
	rpc.Device = device
	rpc.Session = session
	// ctx.Auth = grant // grant.Token // resolved by ..

	// ensure: ( device + app ) match with authorization spec
	// TODO: load references, e.g.: business domain, app, contact etc.

	// emit only reference for now ..
	source := session.Contact
	// ctx.Contact = &model.Contact{
	// 	Dc:  session.Dc,
	// 	Id:  ref.Id,
	// 	Iss: ref.Iss,
	// 	Sub: ref.Sub,
	// }
	lookup := []ContactSearchOption{
		FindContactDc(session.Dc),
	}
	if source.Id != "" {
		lookup = append(lookup, FindContactId(source.Id))
	}
	if source.Iss != "" && source.Sub != "" {
		lookup = append(lookup, FindContactSubject(source.Iss, source.Sub))
	}
	// Resolve Contact profile
	contact, err := rpc.Service.GetContact(rpc.Context, lookup...)

	if err != nil {
		return err
	}

	if contact == nil {
		return errors.New(
			errors.Code(401),
			errors.Status("UNAUTHORIZED"),
			errors.Message("contact( %s@%s ); not found", source.Sub, source.Iss),
		)
	}
	// assign resolved session contact
	rpc.Contact = contact

	// [ OK ]
	return nil
}

var TokenGen = model.GenerateOptions{
	Type:    "bearer",
	Length:  64,
	Expires: 0,
	Refresh: nil,
	GenOpts: []model.GenerateOption{
		model.TokenNoRefresh(),
	},
}
