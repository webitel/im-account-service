package handler

import (
	"context"

	"github.com/lestrrat-go/jwx/v3"
	"github.com/lestrrat-go/jwx/v3/jws"
	"github.com/webitel/im-account-service/internal/errors"
	"github.com/webitel/im-account-service/internal/model"
)

type JwtIdentityAuth struct{}

var _ Authentication = JwtIdentityAuth{}

// Authenticate ctx.Account (User) Identity.
// [ACR] stands for [A]uthenticated [C]redentials [R]ule.
// If [acr] was returned, it means that the authorization data
// satisfies the authentication scheme policy and no further methods will be involved
//
// Non-nil [acr] indicates accept of credentials
// Non-nil [err] indicates failure of verification
func (JwtIdentityAuth) Auth(rpc *Context) (acr any, err error) {
	//
	// Authorization:
	//
	//  [X-Webitel-Client]: [client_id]
	//  [X-Webitel-Access]: [JWT]
	//
	bearer := model.GetHeaderH2(
		rpc.Header, model.H2_X_Access_Token,
	)
	if bearer == "" {
		// No Authorization !
		return nil, nil
	}

	// Accept: JWT compact !
	// Format;JWS:  base64:{protected;header}.base64:{payload;jwt}.base64:signature
	compact := []byte(bearer)

	// JWTs are almost always JWS signed
	ok := (jwx.GuessFormat(compact) == jwx.JWS)
	if !ok {
		// Supposed to be NOT a JWT compact token form !
		return // false, nil
	}

	jws_message, err := jws.Parse(
		compact,
		jws.WithCompact(),
	)

	if err != nil {
		// if errors.Is(err, jws.ParseError()) {}
		return false, err
	}

	var scheme interface { // scheme, _ := rpc.App.(interface {
		// 1. Validate JWT signature
		// 2. Form & validate (Contact) Identity from token.(Payload)
		// 3. Upsert Contact List [re]source with the latest data received
		AcceptJWT(ctx context.Context, token *jws.Message) (*model.Contact, error)
	} // )

	if scheme == nil {
		// App does NOT support OAuth authentication scheme
		return true, ErrClientUnauthorized
	}

	profile, err := scheme.AcceptJWT(rpc.Context, jws_message)
	if err != nil {
		return true, err
	}

	if profile == nil {
		return true, ErrTokenInvalid
	}

	return nil, errors.Errorf("TODO")
}

// String policy name
func (JwtIdentityAuth) String() string {
	return "jwt-identity"
}
