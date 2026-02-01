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

	const issuerId = "im.webitel.org"
	const contactType = "webitel"

	contact := &model.Contact{
		Dc:       debug.Dc,
		Id:       "", // unknown
		Iss:      issuerId,
		Sub:      strconv.FormatInt(debug.UserId, 10),
		App:      "", // none ; default: domain.(app)
		Type:     contactType,
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

	if debug.UpdatedAt > 0 {
		// epoch:milli
		date := model.Timestamp.Date(debug.UpdatedAt)
		contact.UpdatedAt = &date
	}

	// Authorize end-User
	rpc.Contact = contact

	// Find session for ( device + contact )
	err = DeviceAuthorization(false)(rpc)
	if err != nil {
		return bearer, nil
	}

	rpc.Session = nil
	session := rpc.Session

	if rpc.Device.Id != "" {
		session, err = rpc.Service.GetSession(
			rpc.Context, func(req *SessionListOptions) error {
				// UNIQUE( device_id, contact_id )
				req.DeviceId = rpc.Device.Id
				req.ContactId = &model.ContactId{
					Dc:  contact.Dc,
					Id:  contact.Id,
					Iss: contact.Iss,
					Sub: contact.Sub,
				}
				req.Dc = contact.Dc
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
				Dc:     contact.Dc,
				Id:     "", // Not Found
				IP:     rpc.Device.IP(),
				Date:   rpc.Date,
				Name:   model.SessionName(rpc.Device),
				AppId:  "",            // app.(domain)
				Device: (*rpc.Device), // shallowcopy
				Contact: &model.ContactId{
					Dc:  contact.Dc,
					Id:  contact.Id,
					Iss: contact.Iss,
					Sub: contact.Sub,
				},
				Metadata: make(map[string]any),
				Current:  false,
				//Grant:    nil,
			}
		}
	}
	// Webitel (session) correlation
	// No token grant assignment
	rpc.Session = session

	return bearer, nil
}

// String policy name
func (WebitelAuth) String() string {
	return "webitel"
}
