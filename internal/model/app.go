package model

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jws"
	"github.com/webitel/im-account-service/internal/errors"
	v1 "github.com/webitel/im-account-service/proto/gen/im/service/admin/v1"
	"google.golang.org/protobuf/proto"
)

type Mutation struct {
	Date   time.Time
	UserId string
}

// Application [external] Configuration
type Application struct {
	// Dc    int64
	// Id    string
	// Name  string
	// About string

	// Clients  *struct{}
	// Account  *struct{}
	// Service  *struct{}
	// Contacts *struct{}

	// Created *Mutation
	// Updated *Mutation
	// Revoked *Mutation

	src *v1.Application
}

type ApplicationList = Dataset[Application]

func NewApplication(input *v1.InputApp) *Application {
	app := &v1.Application{
		Dc:       input.GetDc(),
		Id:       uuid.NewString(),
		Name:     input.GetName(),
		About:    input.GetAbout(),
		Block:    nil, // &impb.Revocation{},
		Client:   input.GetClient(),
		Service:  input.GetService(), // LIMIT, UPDATES, PUSH
		Account:  nil,                // &impb.Account{},
		Contacts: input.GetContacts(),
	}
	return &Application{
		src: app,
	}
}

func (app *Application) GetDc() int64 {
	return app.src.GetDc()
}

// func (c *Application) GetId() UUID {
// 	return c.src.GetDc()
// }

func (app *Application) ClientId() string {
	return app.src.GetId()
}

func (app *Application) Proto() *v1.Application {
	return proto.CloneOf(app.src)
}

func ProtoApplication(src *v1.Application) *Application {
	return &Application{
		src: proto.CloneOf(src),
	}
}

// Verifies given [idToken] as Contact profile
// is satisfied with [c.contacts.auth] constraints
func (app *Application) NewIdentity(idToken *Contact) error {
	
	// Validate [idToken.Sub] subject identifier
	if idToken.Sub == "" {
		return errors.BadRequest(
			errors.Status("NO_SUBJECT"),
			errors.Message("contacts: subject identifier is missing"),
		)
	}
	// [TODO]: .sub ~= /[A-Za-z0-9\-\.]+/ ; BAD_SUBJECT

	
	// Validate [idToken.Iss] issuer identifier
	issuer := idToken.Iss
	// FIXME: disallow .well-known issuers
	reserved := true
	switch strings.ToLower(issuer) {
	case "app", "service":
	case "bot", "script", "scheme":
	case "user", "webitel", "contact":
	case "viber", "signal", "telegram", "whatsapp", "facebook", "instagram":
	default:
		{
			reserved = false // allowed !
		}
	}

	if reserved {
		return errors.BadRequest(
			errors.Status("BAD_ISSUER"),
			errors.Message("contacts: issuer(%s) reserved", issuer),
		)
	}

	// app.contacts (section) config
	contacts := app.src.GetContacts()
	oauth := contacts.GetAuth()
	// MUST be registered to allow (external: login) usage
	trustedIssuers := oauth.GetIssuers()
	if !slices.Contains(trustedIssuers, issuer) {
		return errors.BadRequest(
			errors.Status("BAD_ISSUER"),
			errors.Message("contacts: issuer(%s) has no trusted relationship", issuer),
		)
	}
	// resolve contact (protocol) type for trusted issuer
	contactTypes := oauth.GetProtos()
	contactType, _ := contactTypes[issuer]
	contactType = cmp.Or(contactType, issuer) // default: issuer

	if idToken.Dc < 1 {
		// invalid or not assigned !
		idToken.Dc = app.GetDc()
	}
	// Ensure App.Dc tenant match !
	if idToken.Dc != app.GetDc() {
		return errors.BadRequest(
			errors.Message("contacts: invalid business identifier"),
		)
	}

	// Validate [idToken.Sub] subject identifier
	// /[A-Za-z0-9\-\.]+/
	contactName := ContactName{
		CommonName: idToken.Name,
		GivenName:  idToken.GivenName,
		MiddleName: idToken.MiddleName,
		FamilyName: idToken.FamilyName,
	}

	if !contactName.IsValid() {
		return errors.BadRequest(
			errors.Message("contacts: subject name is missing"),
		)
	}

	// build ; normalize
	if idToken.App == "" {
		// current
		idToken.App = app.ClientId()
	}
	idToken.Name = contactName.String()
	idToken.Type = contactType

	// [ OK ]
	return nil
}

//  1. Verifies given JWT token ( JWS message ) signature
//  2. Build resulting Contact [idToken] identity from JWT payload
//     according to the [app.contacts.auth.jwt-identity] claims mapping
func (app *Application) JwtIdentity(message *jws.Message) (idToken *Contact, err error) {
	// scheme: [app.contacts.auth.jwt-*] config
	scheme := app.src.GetContacts().GetAuth()
	_ = scheme

	var jwks jwk.Set
	if jwks == nil {
		return nil, errors.Unauthorized(
			errors.Status("UNAUTHORIZED_CLIENT"),
			errors.Message("app: authorization [jwt-identity] scheme not allowed"),
		)
	}

	panic("not implemented")
}

func (app *Application) AcceptJWT(ctx context.Context, token *jws.Message) (*Contact, error) {
	return nil, fmt.Errorf("app.AcceptJWT: not implemented yet")
}
