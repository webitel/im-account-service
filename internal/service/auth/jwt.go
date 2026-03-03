package auth

import (
	// goerror "errors"
	"strings"

	"github.com/lestrrat-go/jwx/v3"
	"github.com/webitel/im-account-service/internal/errors"
	"github.com/webitel/im-account-service/internal/model"
	"github.com/webitel/im-account-service/internal/service"
)

// func init() {
// 	jwt.Settings(
// 		jwt.WithFlattenAudience(true),
// 	)
// }

type JWTokenAuthentication struct {
	// App (Client) used to aprove JWT issuer
	// [X-Webitel-Client] credentials used if not specified
	App *model.Application
}

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
func (x JWTokenAuthentication) Authenticate(rpc *service.Context, hint bool) (acr any, err error) {

	//
	// Authorization:
	//
	//  [X-Webitel-Device]: [device_id]
	//  [X-Webitel-Client]: [client_id]
	//  [X-Webitel-Access]: [JWT]
	//
	bearer := model.GetHeaderH2(
		rpc.Header, model.H2_X_Access_Token,
	)

	bearer = strings.TrimSpace(bearer)
	if bearer == "" {
		// No Authorization !
		return nil, nil
	}

	// Accept: JWT compact !
	// Format;JWS:  base64:{protected;header}.base64:{payload;jwt}.base64:signature
	compact := []byte(bearer)

	// JWTs are almost always JWS signed
	ok := (compact[0] != '{') // NOT in JSON form !
	ok = (ok && jwx.GuessFormat(compact) == jwx.JWS)
	if !ok {
		// Supposed to be NOT a JWT compact token form !
		return nil, nil
	}

	// TODO: try to find cached result for given [bearer] token input !

	// [REQUIRE]:
	// - [X-Webitel-Client] Application credentials to prove JWS signature(s)
	if x.App == nil {
		// err = ClientAuthorization{Require: true}.Do(rpc)
		err = rpc.Init(
			// [X-Webitel-Client] REQUIRED
			ClientAuthentication{Require: true},
		)

		if err != nil {
			// Guess this is JWT token compact form
			// but no App (Client) valid credentials provided
			return true, err
		}

		// authorized App
		x.App = rpc.App
	}

	// MUST: assertion
	app := x.App
	_ = app.ClientId()

	// JWT Authorization scheme enabled ? configured ?
	scheme := app.Clients().JWTAuthentication()
	if scheme == nil {
		// Not configured ; Not allowed !
		return true, errors.Unauthorized(
			errors.Message("messaging: JWT authorization scheme not allowed"),
		)
	}

	// Verify & Parse & Validate JWT token

	// // prepare output
	// token := jwt.New()
	// token.Options().Enable(jwt.FlattenAudience)
	// // prove is well signed & valid JWT (compact) token given !
	// token, err = jwt.ParseString(
	// 	// input
	// 	bearer,
	// 	// options
	// 	jwt.WithToken(token), // output

	// 	jwt.WithVerify(false), // no App to prove signature yet
	// 	// jwt.WithVerify(true),
	// 	// jwt.WithKeySet(scheme.JWKs),

	// 	jwt.WithValidate(true),
	// 	jwt.WithContext(rpc.Context),
	// 	jwt.WithClock(jwt.ClockFunc(
	// 		// MUST be valid at the request date !
	// 		func() time.Time { return rpc.Date },
	// 	)),
	// 	// jwt.WithValidator(jwt.ValidatorFunc(
	// 	// 	func(ctx context.Context, token jwt.Token) error {
	// 	// 		// "iss" MUST be oneof registered app.auth.(jwt).issuers
	// 	// 		iss, ok := token.Issuer()
	// 	// 		if !ok || iss == "" {
	// 	// 			return jwt.InvalidIssuerError()
	// 	// 		}
	// 	// 		ok = false
	// 	// 		knownIss := c.conf.GetIssuers()
	// 	// 		for _, allow := range knownIss {
	// 	// 			if ok = (allow == iss); ok {
	// 	// 				break
	// 	// 			}
	// 	// 		}
	// 	// 		if !ok {
	// 	// 			return jwt.InvalidIssuerError()
	// 	// 		}
	// 	// 		// OK
	// 	// 		return nil
	// 	// 	},
	// 	// )),

  //   // func WithCookie(v **http.Cookie) ParseOption
  //   // func WithCookieKey(v string) ParseOption
  //   // func WithFormKey(v string) ParseOption
  //   // func WithHeaderKey(v string) ParseOption
  //   // func WithKeyProvider(v jws.KeyProvider) ParseOption
  //   // func WithKeySet(set jwk.Set, options ...any) ParseOption
  //   // func WithPedantic(v bool) ParseOption
  //   // func WithToken(v Token) ParseOption
  //   // func WithTypedClaim(name string, object any) ParseOption
  //   // func WithValidate(v bool) ParseOption
  //   // func WithVerify(v bool) ParseOption
  //   // func WithVerifyAuto(f jwk.Fetcher, options ...jwk.FetchOption) ParseOption

	// 	// jwt.WithAcceptableSkew(v time.Duration) ValidateOption
  //   // jwt.WithAudience(s string) ValidateOption
  //   // jwt.WithClaimValue(name string, v any) ValidateOption
  //   // jwt.WithClock(v Clock) ValidateOption
  //   // jwt.WithContext(v context.Context) ValidateOption
  //   // jwt.WithIssuer(s string) ValidateOption
  //   // jwt.WithJwtID(s string) ValidateOption
  //   // jwt.WithMaxDelta(dur time.Duration, c1, c2 string) ValidateOption
  //   // jwt.WithMinDelta(dur time.Duration, c1, c2 string) ValidateOption
  //   // jwt.WithRequiredClaim(name string) ValidateOption
  //   // jwt.WithResetValidators(v bool) ValidateOption
  //   // jwt.WithSubject(s string) ValidateOption
  //   // jwt.WithValidator(v Validator) ValidateOption

	// )

	// // jws_message, err := jws.Parse(
	// // 	compact,
	// // 	jws.WithCompact(),
	// // )
	// // _ = jws_message

	// if err != nil {
	// 	// if goerror.Is(err, jws.ParseError()) {}
	// 	return false, err
	// }

	// Extract Contact Identity from JWT payload ..
	token, idtoken, err := scheme.GetContact(
		rpc.Context, rpc.Date, bearer,
	)
	if err != nil {
		return err, err
	}
	// TODO: cache parsed JWToken result
	_ = token

	// jwtoken, idtoken, err := scheme.GetIdentity(rpc.Context, rpc.Date, bearer)
	// if err != nil {
	// 	return err, err
	// }
	// _ = jwtoken 
	// // Build Contact profile ..
	// profile, err := scheme.NewIdentity(idtoken)
	// if err != nil {
	// 	return err, err
	// }

	// Save / Update latest Contact profile info
	err = rpc.Service.AddContact(rpc.Context, idtoken)
	if err != nil {
		// failed to persist latest contact info
		return bearer, err
	}

	// Authorize (external) Contact
	rpc.Contact = idtoken

	// Find session for ( device + contact )
	// err = DeviceAuthorization(false)(rpc)
	err = rpc.Init(
		DeviceAuthentication{Require: false},
	)
	
	if err != nil {
		return bearer, err
	}

	rpc.Session = nil
	session := rpc.Session
	refUser := &model.ContactId{
		Dc:  idtoken.Dc,
		Id:  idtoken.Id,
		Iss: idtoken.Iss,
		Sub: idtoken.Sub,
	}

	if rpc.Device.Id != "" {
		session, err = rpc.Service.GetSession(
			rpc.Context, func(req *service.SessionListOptions) error {
				// UNIQUE( device_id, contact_id )
				req.Dc = app.GetDc()
				req.DeviceId = rpc.Device.Id
				req.ContactId = refUser
				return nil
			},
		)
		if err != nil {
			// Failed lookup session
			return bearer, err
		}
	}

	if session == nil {
		// Not Found ; Init ..
		session = &model.Authorization{
			Id:     "", // Not Found
			Dc:     app.GetDc(),
			IP:     rpc.Device.IP(),
			Date:   rpc.Date,
			Name:   model.SessionName(rpc.Device),
			AppId:  app.ClientId(), // MUST
			Device: (*rpc.Device), // shallowcopy
			Contact: refUser,
			Metadata: make(map[string]any),
			Current:  false,
			//Grant:    nil,
		}
	}

	// Webitel (session) Authorization prepared
	// No (internal) token [grant] assignment
	rpc.Dc = session.Dc
	rpc.Session = session

	// [ OK ]
	return bearer, nil
}

func (JWTokenAuthentication) String() string {
	return "jwt-identity"
}
