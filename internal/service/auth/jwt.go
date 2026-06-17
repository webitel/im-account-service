package auth

import (
	// goerror "errors"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/lestrrat-go/jwx/v3"
	"github.com/lestrrat-go/jwx/v3/jwt"
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

func (JWTokenAuthentication) String() string { return "jwt-identity" }

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

	// Extract Contact Identity from JWT payload ..
	token, idtoken, err := scheme.GetContact(
		rpc.Context, rpc.Date, bearer,
	)
	if err != nil {
		return err, err
	}
	// TODO: cache parsed JWToken result
	_ = token

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
	if err = rpc.Init(DeviceAuthentication{Require: false}); err != nil {
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
			Id:       "", // Not Found
			Dc:       app.GetDc(),
			IP:       rpc.Device.IP(),
			Date:     rpc.Date,
			Name:     model.SessionName(rpc.Device),
			AppId:    app.ClientId(), // MUST
			Device:   (*rpc.Device),  // shallowcopy
			Contact:  refUser,
			Metadata: make(map[string]any),
			Current:  false,
			//Grant:    nil,
		}
	}

	// Webitel (session) Authorization prepared
	// No (internal) token [grant] assignment
	rpc.Dc = session.Dc
	rpc.Session = session

	if err = trySetTokenPayloadIntoHeaders(rpc, token); err != nil {
		slog.Error(
			"propagating token payload into rpc headers",
			slog.Any("error", err),
			slog.String("id", idtoken.Id),
			slog.String("iss", idtoken.Iss),
			slog.String("sub", idtoken.Sub),
		)
	}

	return bearer, nil
}

func extractTokenPayload(token jwt.Token) (string, error) {
	if token == nil {
		return "", nil
	}

	tokenPayload, err := json.Marshal(token)
	if err != nil {
		return "", errors.New(errors.Message("parsing jwt token payload for propagation"))
	}

	encodedPayload := base64.RawStdEncoding.EncodeToString(tokenPayload)

	return encodedPayload, nil
}

func tryPrepareRPCContext(rpc *service.Context) error {
	if rpc == nil {
		return errors.BadRequest(errors.Message("received nil pointer rpc call"))
	}

	if rpc.Header == nil {
		rpc.Header = make(map[string][]string)
	}

	return nil
}

func trySetTokenPayloadIntoHeaders(rpc *service.Context, token jwt.Token) error {
	tokenPayload, err := extractTokenPayload(token)
	if err != nil {
		return err
	}

	if tokenPayload == "" {
		return errors.BadRequest(errors.Message("empty payload from jwt token"))
	}

	if err = tryPrepareRPCContext(rpc); err != nil {
		return err
	}

	rpc.Header.Set(service.XJwtPayloadHeader, tokenPayload)

	return nil
}
