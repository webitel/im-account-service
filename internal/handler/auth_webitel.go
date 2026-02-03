package handler

import (
	"cmp"
	"strconv"

	webitel "github.com/webitel/im-account-service/internal/client/webitel/auth"
	"github.com/webitel/im-account-service/internal/model"
)

// Webitel Authentication scheme
type WebitelAuth struct {
	Client *webitel.Client
}

var _ Authentication = WebitelAuth{}

// Authenticate ctx.Account (User) Identity.
// [ACR] stands for [A]uthenticated [C]redentials [R]ule.
// If [acr] was returned, it means that the authorization data
// satisfies the authentication scheme policy and no further methods will be involved
//
// Non-nil [acr] indicates accept of credentials
// Non-nil [err] indicates failure of verification
func (x WebitelAuth) Auth(rpc *Context) (acr any, err error) {
	// [X-Webitel-Access]: [token] ; Authorization
	bearer := model.GetHeaderH2(
		rpc.Header, model.H2_X_Access_Token,
	)
	if bearer == "" {
		// No Authorization !
		return nil, nil
	}

	debug, err := x.Client.Inspect(
		rpc.Context, bearer,
		webitel.InspectDate(rpc.Date),
	)

	if debug == nil {
		// Not Sure ..
		return nil, err
	}

	if err != nil {
		// Invalid !
		return err, err
	}

	// [X-Webitel-Client]: [app-id] ; OPTIONAL
	err = AppAuthorization(false)(rpc)
	if err != nil {
		// Header specified, but invalid
		return bearer, err
	}

	app := rpc.App
	if app != nil && app.GetDc() != debug.Dc {
		// Cross-DC App (Client) usage attempt !
		return bearer, ErrClientUnauthorized
	}

	const contactIssuer = "webitel"
	const contactProto = "webitel"

	endUser := &model.Contact{
		Dc:       debug.Dc,
		Id:       "", // unknown
		Iss:      contactIssuer,
		Sub:      strconv.FormatInt(debug.UserId, 10),
		App:      "", // none ; default: domain.(app)
		Type:     contactProto,
		Name:     cmp.Or(debug.Name, debug.Username),
		Username: debug.Username,
		// GivenName:           "",
		// MiddleName:          "",
		// FamilyName:          "",
		// Birthdate:           "",
		// Zoneinfo:            "",
		// Profile:             "",
		// Picture:             "",
		// Gender:              "",
		// Locale:              "",
		// Email:               "",
		// EmailVerified:       false,
		// PhoneNumber:         "",
		// PhoneNumberVerified: false,
		// Metadata:            map[string]any{},
		// CreatedAt:           time.Time{},
		// UpdatedAt:           &time.Time{},
		// DeletedAt:           &time.Time{},
	}

	if app == nil {
		endUser.App = "domain" // default
	} else {
		endUser.App = app.ClientId()
	}

	if debug.UpdatedAt > 0 {
		// epoch:milli
		date := model.Timestamp.Date(debug.UpdatedAt)
		endUser.UpdatedAt = &date
	}

	// Save / Update latest Contact profile info
	err = rpc.Service.AddContact(rpc.Context, endUser)
	if err != nil {
		// failed to persist latest contact info
		return bearer, err
	}

	// Authorize Webitel end-User
	rpc.Contact = endUser

	// Find session for ( device + contact )
	err = DeviceAuthorization(false)(rpc)
	if err != nil {
		return bearer, err
	}

	rpc.Session = nil
	session := rpc.Session

	if rpc.Device.Id != "" {
		session, err = rpc.Service.GetSession(
			rpc.Context, func(req *SessionListOptions) error {
				// UNIQUE( device_id, contact_id )
				req.DeviceId = rpc.Device.Id
				req.ContactId = &model.ContactId{
					Dc:  endUser.Dc,
					Id:  endUser.Id,
					Iss: endUser.Iss,
					Sub: endUser.Sub,
				}
				req.Dc = endUser.Dc
				return nil
			},
		)
		if err != nil {
			// Failed lookup session
			return bearer, err
		}
		if session == nil {
			// Not Found ; Init ..
			session = &model.Authorization{
				Id:     "", // Not Found
				Dc:     endUser.Dc,
				IP:     rpc.Device.IP(),
				Date:   rpc.Date,
				Name:   model.SessionName(rpc.Device),
				AppId:  "",            // UUID NULL ; app.(domain)
				Device: (*rpc.Device), // shallowcopy
				Contact: &model.ContactId{
					Dc:  endUser.Dc,
					Id:  endUser.Id,
					Iss: endUser.Iss,
					Sub: endUser.Sub,
				},
				Metadata: make(map[string]any),
				Current:  false,
				//Grant:    nil,
			}

			if app != nil {
				session.AppId = app.ClientId() // UUID
			}
		}
	}
	// Webitel (session) Authorization prepared
	// No (internal) token [grant] assignment
	rpc.Dc = session.Dc
	rpc.Session = session

	return bearer, nil
}

// String policy name
func (WebitelAuth) String() string {
	return "webitel"
}
