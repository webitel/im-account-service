package model

import (
	"cmp"
	"context"
	"log/slog"
	"net/netip"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jws"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/webitel/im-account-service/internal/errors"
	"github.com/webitel/im-account-service/internal/service/jwks"
	im_auth "github.com/webitel/im-account-service/proto/gen/im/service/auth/v1"
	"github.com/webitel/webitel-go-kit/pkg/semconv"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"
)

type AppClients struct {
	app *Application
	// prepared
	ua  []*regexp.Regexp
	web []*regexp.Regexp
	net []netip.Prefix
	jwt *JWTAuthentication
}

func (app *Application) Clients() AppClients {

	app.mx.Lock()
	defer app.mx.Unlock()

	if app.clients.app == app {
		// prepared
		return app.clients
	}

	scheme := &app.clients
	scheme.app = app // init

	// build config
	opts := app.opts.GetClients()
	if opts == nil {
		// no configuration / constraints ; OK
		return app.clients
	}

	// constraint: [U]ser-[A]gent
	if n := len(opts.Ua); n > 0 {
		// values := make([]*regexp.Regexp, 0, n)
		// for _, input := range opts.Ua {
		// 	pattern, err := regexp.Compile(input)
		// 	if err != nil {
		// 		// FIXME: just ignore ?
		// 		app.Log(
		// 			slog.LevelWarn, "app.clients.ua",
		// 			"ua", input, "err", err.Error(),
		// 		)
		// 		continue
		// 	}
		// 	values = append(values, pattern)
		// }
		// if len(values) > 0 {
		// 	scheme.ua = values
		// }
	}
	// constraint: net.cidr
	if n := len(opts.GetNet().GetCidr()); n > 0 {
		values := make([]netip.Prefix, 0, n)
		for _, input := range opts.Net.Cidr {
			cidr, err := netip.ParsePrefix(input)
			if err != nil {
				// FIXME: just ignore ?
				app.Log(
					slog.LevelWarn, "app.clients.net.cidr",
					"cidr", input, semconv.ErrorKey, err.Error(),
				)
				continue
			}
			values = append(values, cidr)
		}
		if len(values) > 0 {
			scheme.net = values
		}
	}
	// constraint: web.origin
	if n := len(opts.GetWeb().GetOrigin()); n > 0 {
		// values := make([]*regexp.Regexp, 0, n)
		// for _, input := range opts.Web.Origin {
		// 	pattern, err := regexp.Compile(input)
		// 	if err != nil {
		// 		// FIXME: just ignore ?
		// 		app.Log(
		// 			slog.LevelWarn, "app.clients.web.origin",
		// 			"origin", input, "err", err.Error(),
		// 		)
		// 		continue
		// 	}
		// 	values = append(values, pattern)
		// }
		// if len(values) > 0 {
		// 	scheme.web = values
		// }
	}
	// authentication: JWT
	jwt, err := newJWTAuthentication(
		app,
	)
	if err != nil {
		app.Log(
			slog.LevelWarn, "app.clients.jwt",
			semconv.ErrorKey, err.Error(),
		)
	} else {
		scheme.jwt = jwt
	}
	// done
	return app.clients
}

func (c AppClients) Authorize(client *Device) error {
	// MUST
	_ = c.app.ClientId()

	ok := true // default: NO constraints

	// [User-Agent] constraints ..
	UA := client.App.String // client != nil
	for _, pattern := range c.ua {
		if ok = pattern.MatchString(UA); ok {
			break // ALLOWED ; Match FOUND !
		}
	}

	if !ok {
		return errors.Forbidden(
			errors.Message("client: [User-Agent] not allowed"),
		)
	}

	// [net] constraints ..
	ok = true
	IP, _ := netip.AddrFromSlice([]byte(client.IP()))
	for _, subnet := range c.net {
		if ok = subnet.Contains(IP); ok {
			break // ALLOWED ; Match FOUND !
		}
	}

	if !ok {
		return errors.Forbidden(
			errors.Message("client: [IP] address not allowed"),
		)
	}

	// [web] constraints ..
	ok = true
	Origin := []byte("") // TODO !!!!!!!
	for _, pattern := range c.web {
		if ok = pattern.Match(Origin); ok {
			break // ALLOWED ; Match FOUND !
		}
	}

	if !ok {
		return errors.Forbidden(
			errors.Message("client: Web [Origin] not allowed"),
		)
	}

	// [ OK ]
	return nil
}

// Verifies given [identity] as Contact profile
// is satisfied with [app.oauth.iDp] constraints
func (c AppClients) NewContact(identity *Contact) error {

	// MUST
	_ = c.app.ClientId()

	// Validate [idToken.Sub] subject identifier
	if identity.Sub == "" {
		return errors.BadRequest(
			errors.Status("NO_SUBJECT"),
			errors.Message("identity: subject identifier is missing"),
		)
	}
	// [TODO]: .sub ~= /[A-Za-z0-9\-\.]+/ ; BAD_SUBJECT

	// Validate [idToken.Iss] issuer identifier
	issuer := identity.Iss
	// FIXME: disallow .well-known issuers
	err := isValidIssuer(issuer)
	if err != nil {
		// [iss] not accepted !
		return err
	}

	// app.clients (Authentication) config
	clients := c.app.opts.GetClients()
	// idp := clients.GetJwt()
	// MUST be registered to allow (external: login) usage
	trusted := clients.GetIDp()
	if _, ok := trusted[issuer]; !ok {
		return errors.BadRequest(
			errors.Status("BAD_ISSUER"),
			errors.Message("identity: issuer(%s) has no trusted relationship", issuer),
		)
	}
	// resolve contact (protocol) type for trusted issuer
	contactType := trusted[issuer]
	contactType = cmp.Or(contactType, issuer) // default: issuer

	if identity.Dc < 1 {
		// invalid or not assigned !
		identity.Dc = c.app.GetDc()
	}
	// Ensure App.Dc tenant match !
	if identity.Dc != c.app.GetDc() {
		// given Contact [identity] appertains to other Business (Domain) account
		// and cannot be used with this Application
		return errors.Internal(
			errors.Message("identity: cross business constraint"),
		)
	}

	// Validate [idToken.Sub] subject identifier
	// /[A-Za-z0-9\-\.]+/
	contactName := ContactName{
		CommonName: identity.Name,
		GivenName:  identity.GivenName,
		MiddleName: identity.MiddleName,
		FamilyName: identity.FamilyName,
	}

	if !contactName.IsValid() {
		return errors.BadRequest(
			errors.Message("identity: subject name is missing"),
		)
	}

	// build ; normalize ..

	// ( identity.Dc > 0 ) // +[OK]
	if identity.App == "" {
		// current
		identity.App = c.app.ClientId()
	}

	// ( identity.Iss != "" ) // +ALLOWED
	// ( identity.Sub != "" ) // +ALLOWED
	identity.Type = contactType
	identity.Name = contactName.String()

	// [ OK ]
	return nil
}

func (c AppClients) JWTAuthentication() *JWTAuthentication {
	return c.jwt
}

// JWTAuthentication scheme of the Application (Client) configuration
type JWTAuthentication struct {
	app *Application
	// opts im_admin.JwtIdentity // cached config
	keyset jwk.Set   // prepared JWKs keyset
	claims claimsMap // map [identity.field] = jwt.(payload).claim[ "|" .. ] ;
	err    error     // init: (critical) error
}

func newJWTAuthentication(app *Application) (scheme *JWTAuthentication, err error) {

	opts := app.opts.GetClients().GetJwt()
	if !opts.GetEnabled() { // opts == nil {
		// No configuration | Disabled !
		return nil, nil
	}

	scheme = &JWTAuthentication{
		app:    app, // app reference
		claims: buildClaimsMap(opts.GetClaims()),
	}
	// cache (running) configuration
	// proto.Merge(&scheme.opts, opts)
	return scheme, nil
}

// map [identity.field] = jwt.(payload).claim["|"claim].. ;
type claimsMap map[string][]string

func buildClaimsMap(input map[string]string) claimsMap {

	const fallbackSep = "|"
	config := make(claimsMap, len(input))

	for fd, src := range input {
		// fallback: "claim-A" [ "|claim-B" ]..
		claims := strings.Split(src, fallbackSep)
		// normalize:
		// - drop empty value(s)..
		for e, n := 1, len(claims); e < n; e++ {
			// skip ; no name ..
			if strings.TrimSpace(claims[e]) == "" {
				claims = append(claims[0:e], claims[e+1:]...)
				e--
				n--
				continue
			}
			// skip ; duplicate..
			if slices.Contains(claims[0:e], claims[e]) {
				claims = append(claims[0:e], claims[e+1:]...)
				e--
				n--
				continue
			}
		}

		if len(claims) == 0 {
			if fd == "" {
				// skip ; has no affect
				continue
			}
			// case: { "scope": "" } ; TODO: save metadata["scope"] if claim (value) provided
			claims = append(claims, "")
		}

		config[fd] = claims
	}

	// normalize: split [nokey] claims
	if claims, ok := config[""]; ok {
		delete(config, "")
		for _, att := range claims {
			if _, ok = config[att]; ok {
				continue // already defined
			}
			// define: self
			config[att] = []string{""}
		}
	}

	return config
}

// GetJWKs loads [J]son[W]eb[K]eys used to verify JWT token signature & prove issuer trust.
func (c *JWTAuthentication) GetJWKs(ctx context.Context) (keyset jwk.Set, err error) {

	tx := &c.app.mx

	tx.Lock()
	defer tx.Unlock()

	if c.keyset != nil {
		return c.keyset, nil
	}

	// from: configuration
	opts := c.app.opts.GetClients().GetJwt()

	if uri := opts.GetJwksUri(); uri != "" {
		// [RE]Fetch JWKs resource keys ..
		// keyset, err = jwks.Register(ctx, uri)
		// keyset, err = jwks.Default.Register2(
		// 	ctx, uri, func(reg *jwks.ResourceOptions) {
		// 		reg.OnSyncFailure = func(_ error) time.Duration {
		// 			// [CachedSet] become Unavailable ! free resource
		// 			// Will try to sync again on next [LoadKeySet] call ..
		// 			// _ = jwks.Default.Unregister(
		// 			// 	context.Background(), uri,
		// 			// )
		// 			c.keyset = nil
		// 			return 0 // Unregister
		// 		}
		// 	},
		// )
		keyset, err = jwks.Fetch(
			ctx, uri, func(opts *jwks.ResourceOptions) {
				// options ...
			},
		)
		if err != nil {
			re := err
			type wrapError interface {
				Unwrap() error
			}
			e, _ := re.(wrapError)
			for e != nil {
				re = e.Unwrap()
				e, _ = re.(wrapError)
			}
			// Could not fetch JWKs (keys) resource ...
			err = errors.BadGateway(
				errors.Message("messaging: fetch( app.client.jwt.jwks_uri ); error: %v", re),
			)
		}
	} else if src := opts.GetJwks(); len(src) > 0 {
		keyset, err = jwk.Parse(src)// jwk.WithPEM(false),
		// jwk.WithTypedField("", nil),
		// jwk.WithIgnoreParseError(false),

		if err != nil {
			keyset = nil
			err = errors.BadGateway(
				errors.Message("messaging: parse( app.client.jwt.jwks ); error: %v", err),
			)
		}
	}

	if err != nil {
		// (502) Bad Gateway
		// JWKs resorce error
		return nil, err
	}

	// initialized ?
	if keyset == nil {
		// NO JWKs available ! Consider: NOT secure !
		return nil, errors.BadGateway(
			errors.Message("messaging: fetch( app.clients.jwt.jwks ); error: invalid or missing"),
		)
	}
	// +[ OK ]
	c.keyset = keyset
	return keyset, nil
}

// GetContact verifies & extract Identity info from given [bearer] JWT token string
func (c *JWTAuthentication) GetContact(ctx context.Context, date time.Time, bearer string) (jwt.Token, *Contact, error) {
	// parse & verify & validate & extract JWT token Identity
	token, idtoken, err := c.getIdentity(ctx, date, bearer)
	if err != nil {
		return token, nil, err
	}
	identity, err := c.newIdentity(idtoken)
	if err != nil {
		return token, nil, err
	}
	// check: can Login VIA this Application
	err = c.app.Clients().NewContact(identity)
	if err != nil {
		return token, nil, err
	}
	// +[ OK ]
	return token, identity, nil
}

func (c *JWTAuthentication) getIdentity(ctx context.Context, date time.Time, bearer string) (token jwt.Token, idtoken *im_auth.Identity, err error) {

	keyset, err := c.GetJWKs(ctx)
	if err != nil {
		return nil, nil, err
	}

	// prepare: output
	token = jwt.New()
	token.Options().Enable(jwt.FlattenAudience)

	// parse & verify & validate !
	token, err = jwt.ParseString(
		// input
		bearer,
		// options
		jwt.WithToken(token), // output

		jwt.WithVerify(true),
		jwt.WithKeySet(keyset,

			jws.WithRequireKid(true),
			// jws.WithInferAlgorithmFromKey(v bool) WithKeySetSuboption
			// jws.WithMultipleKeysPerKeyID(v bool) WithKeySetSuboption
			// jws.WithRequireKid(v bool) WithKeySetSuboption
			// jws.WithUseDefault(v bool) WithKeySetSuboption

		),
		// jwt.WithKeyProvider(jws.KeyProviderFunc(
		// 	func(ctx context.Context, sink jws.KeySink, sign *jws.Signature, msg *jws.Message) error {
		// 		panic("not implemented")
		// 		// sink.Key(alg, key)
		// 		// return nil
		// 	},
		// )),
		// jwt.WithCookie(v **http.Cookie) ParseOption
		// jwt.WithCookieKey(v string) ParseOption
		// jwt.WithFormKey(v string) ParseOption
		// jwt.WithHeaderKey(v string) ParseOption
		// jwt.WithKeyProvider(v jws.KeyProvider) ParseOption
		// jwt.WithKeySet(set jwk.Set, options ...any) ParseOption
		// jwt.WithPedantic(v bool) ParseOption
		// jwt.WithToken(v Token) ParseOption
		// jwt.WithTypedClaim(name string, object any) ParseOption
		// jwt.WithValidate(v bool) ParseOption
		// jwt.WithVerify(v bool) ParseOption
		// jwt.WithVerifyAuto(f jwk.Fetcher, options ...jwk.FetchOption) ParseOption

		jwt.WithValidate(true),

		jwt.WithContext(ctx),
		jwt.WithClock(jwt.ClockFunc(
			func() time.Time {
				return date
			},
		)),
		// jwt.WithValidator(jwt.ValidatorFunc(
		// 	func(ctx context.Context, token jwt.Token) error {
		// 		// "iss" MUST be oneof registered app.auth.(jwt).issuers
		// 		iss, ok := token.Issuer()
		// 		if !ok || iss == "" {
		// 			return jwt.InvalidIssuerError()
		// 		}
		// 		trusted := c.opts.GetIss()
		// 		if _, ok = trusted[iss]; !ok {
		// 			// jwt: has no [iss] trusted relationship
		// 			return jwt.InvalidIssuerError()
		// 		}
		// 		// OK
		// 		return nil
		// 	},
		// )),

		// jwt.WithAcceptableSkew(v time.Duration) ValidateOption
		// jwt.WithAudience(s string) ValidateOption
		// jwt.WithClaimValue(name string, v any) ValidateOption
		// jwt.WithClock(v Clock) ValidateOption
		// jwt.WithContext(v context.Context) ValidateOption
		// jwt.WithIssuer(s string) ValidateOption
		// jwt.WithJwtID(s string) ValidateOption
		// jwt.WithMaxDelta(dur time.Duration, c1, c2 string) ValidateOption
		// jwt.WithMinDelta(dur time.Duration, c1, c2 string) ValidateOption
		// jwt.WithRequiredClaim(name string) ValidateOption
		// jwt.WithResetValidators(v bool) ValidateOption
		// jwt.WithSubject(s string) ValidateOption
		// jwt.WithValidator(v Validator) ValidateOption

	)

	if err != nil {
		// Invalid JWT claims !
		return token, nil, err
	}

	// Build (Contact) from Identity
	idtoken, _ = c.claims.JwtIdentity(token)
	return token, idtoken, err
}

// Verifies given [idToken] as Contact profile
// is satisfied with [c.contacts.auth] constraints
func (c *JWTAuthentication) newIdentity(idtoken *im_auth.Identity) (profile *Contact, _ error) {

	// early binding
	_ = idtoken.Iss

	app := c.app
	return &Contact{
		Id:                  "", // unknown yet
		Dc:                  app.GetDc(),
		App:                 app.ClientId(),
		Iss:                 idtoken.Iss,
		Sub:                 idtoken.Sub,
		Type:                "", // unknown yet
		Name:                idtoken.Name,
		GivenName:           idtoken.GivenName,
		MiddleName:          idtoken.MiddleName,
		FamilyName:          idtoken.FamilyName,
		Username:            "", // idtoken.PreferredUsername,
		Birthdate:           idtoken.Birthdate,
		Zoneinfo:            idtoken.Zoneinfo,
		Profile:             idtoken.Profile,
		Picture:             idtoken.Picture,
		Gender:              idtoken.Gender,
		Locale:              idtoken.Locale,
		Email:               idtoken.Email,
		EmailVerified:       idtoken.EmailVerified,
		PhoneNumber:         idtoken.PhoneNumber,
		PhoneNumberVerified: idtoken.PhoneNumberVerified,
		Metadata:            idtoken.Metadata.AsMap(), // map[string]any{},
		// CreatedAt:           time.Time{},
		// UpdatedAt:           &time.Time{},
		// DeletedAt:           &time.Time{},
	}, nil
}

// JwtIdentity builds [auth.Identity] from given [jwt.Token] payload
func (x claimsMap) JwtIdentity(token jwt.Token) (idtoken *im_auth.Identity, _ error) {

	claimValue := func(claim string) (value any, ok bool) {
		_, ok = token.Get(claim, &value), token.Has(claim)
		return // value
	}

	claimString := func(claim string) (value string) {
		_ = token.Get(claim, &value)
		return // value
	}

	claimBoolean := func(claim string) (value bool) {
		_ = token.Get(claim, &value)
		return // value
	}

	// standard claims first ..
	idtoken = &im_auth.Identity{
		Iss:                 claimString(jwt.IssuerKey),
		Sub:                 claimString(jwt.SubjectKey),
		Name:                claimString("name"),
		GivenName:           claimString("given_name"),
		MiddleName:          claimString("middle_name"),
		FamilyName:          claimString("family_name"),
		Birthdate:           claimString("birthdate"),
		Zoneinfo:            claimString("zoneinfo"),
		Profile:             claimString("profile"),
		Picture:             claimString("picture"),
		Gender:              claimString("gender"),
		Locale:              claimString("locale"),
		Email:               claimString("email"),
		EmailVerified:       claimBoolean("email_verified"),
		PhoneNumber:         claimString("phone_number"),
		PhoneNumberVerified: claimBoolean("phone_number_verified"),
		Metadata:            nil, // &structpb.Struct{},
	}

	var (
		fd     protoreflect.FieldDescriptor
		rdst   = idtoken.ProtoReflect()
		fields = rdst.Descriptor().Fields()

		metadata = make(map[string]any)
	)
	for att, claims := range x {
		for _, claim := range claims {
			// case: { "scope": "" } ; TODO: save metadata["scope"] value if provided
			claim = cmp.Or(claim, att)
			value, ok := claimValue(claim)
			if !ok {
				// not found !
				continue
			}
			// found !
			fd, att = nil, cmp.Or(att, claim)
			switch att {
			// Omit <system> field(s) setup !
			// Move such to "metadata" variables
			case "metadata":
			// case "sources":
			// case "created_at":
			// case "updated_at":
			// case "deleted_at":
			default:
				fd = fields.ByTextName(att)
			}
			if fd == nil {
				// set to .metadata value
				if strings.EqualFold(att, "metadata") {
					// force: as [jwt.attribute] name !
					// case: {"metadata":"given_name|name"}
					att = claim
				}
				metadata[att] = value
				break // fallback iteration ; claim value found !
			}
			rdst.Set(
				fd, protoreflect.ValueOf(value),
			)
			// claim value found
			// ignore the rest fallback claims ..
			break
		}
	}

	if len(metadata) > 0 {
		for claim, value := range metadata {
			rv := reflect.ValueOf(value)
			switch rv.Kind() {
			case reflect.Array, reflect.Slice:
				{
					if rv.Type().Elem().Kind() != reflect.Interface {
						n := rv.Len()
						v2 := reflect.MakeSlice(
							reflect.SliceOf(reflect.TypeFor[any]()), n, n,
						)
						for e := range n {
							v2.Index(e).Set(rv.Index(e))
						}
						metadata[claim] = v2.Interface()
					}
				}
			}
		}
		idtoken.Metadata, _ = structpb.NewStruct(metadata)
	}

	return idtoken, nil
}
