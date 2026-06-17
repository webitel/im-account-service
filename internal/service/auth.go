package service

import (
	"github.com/webitel/im-account-service/internal/errors"
)

// Authentication Policy. Scheme
type AuthenticationScheme interface {
	// Authenticate ctx.Contact (User) Identity.
	//
	// [ACR] stands for [A]uthenticated [C]redentials [R]ule.
	// If [acr] was returned, it means that the [Authorization] credentials
	// satisfies the Authentication scheme policy and no further methods will be involved
	//
	// Non-nil [acr] indicates accept of credentials
	// Non-nil [err] indicates failure of verification
	//
	// [Hint] indicates whether verification
	Authenticate(rpc *Context, hint bool) (acr any, err error)

	// String policy name
	String() string
}

// Authenticate Context Control
type Authenticate struct {
	Hint    bool                   // Just identify session & skip verification .. if creds provided ..
	Require bool                   // Require credentials
	Schemes []AuthenticationScheme // Schemes available for authentication ..
}

var ErrAccountUnauthorized = errors.Unauthorized(
	errors.Status("UNAUTHORIZED"),
	errors.Message("messaging: account not authorized"),
)

func (x Authenticate) Do(rpc *Context) error {
	defer func() {
		if len(rpc.Header.Get(XJwtPayloadHeader)) > 0 {
			rpc.UpdateIncomingHeaders()
			return
		}
	}()

	if rpc.Contact != nil {
		// once ; OK
		return nil
	}

	// if vs, ok := rpc.Header[model.H2_X_Access_Token]; ok {
	// 	if bearer := model.CoalesceLast(vs...); bearer != "" {
	for _, scheme := range x.Schemes {
		acr, err := scheme.Authenticate(rpc, x.Hint)
		if err != nil {
			// LOG
		}
		// got credentials ?
		if acr != nil {
			if err == nil {
				err = authorizeEndUser(rpc)
			}
			rpc.Auth = acr
			rpc.Error = err
			return err
		}
	}
	// 	}
	// }
	if x.Require {
		return ErrAccountUnauthorized
	}
	// [ OK ]
	return nil

}

func authorizeEndUser(ctx *Context) error {
	if ctx.Contact == nil {
		return ErrAccountUnauthorized
	}
	// CHECK: ctx.App.AuthContact(ctx.Account.Contact)
	return nil
}
