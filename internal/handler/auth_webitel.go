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
		App:      "", // none
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
	return bearer, nil
}

// String policy name
func (WebitelAuth) String() string {
	return "webitel"
}
