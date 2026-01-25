package handler

// Authentication scheme
type Authentication interface {
	// Authenticate ctx.Contact (User) Identity.
	//
	// [ACR] stands for [A]uthenticated [C]redentials [R]ule.
	// If [acr] was returned, it means that the authorization data
	// satisfies the authentication scheme policy and no further methods will be involved
	//
	// Non-nil [acr] indicates accept of credentials
	// Non-nil [err] indicates failure of verification
	Auth(ctx *Context) (acr any, err error)

	// String policy name
	String() string
}

// [X-Webitel-Access] token authorization
func EndUserAuthorization(require bool) ContextFunc {
	return func(rpc *Context) error {

		if rpc.Contact != nil {
			// once ; OK
			return nil
		}

		// if vs, ok := rpc.Header[model.H2_X_Access_Token]; ok {
		// 	if bearer := model.CoalesceLast(vs...); bearer != "" {
		for _, scheme := range []Authentication{
			SessionAuth{}, JwtIdentityAuth{},
			WebitelAuth{rpc.Service.Options().Webitel},
		} {
			acr, err := scheme.Auth(rpc)
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
		if require {
			return ErrAccountUnauthorized
		}
		// [ OK ]
		return nil
	}
}

func authorizeEndUser(ctx *Context) error {
	if ctx.Contact == nil {
		return ErrAccountUnauthorized
	}
	// CHECK: ctx.App.AuthContact(ctx.Account.Contact)
	return nil
}
