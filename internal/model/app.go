package model

import (
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
	// app.contacts (section) config
	appContacts := app.src.GetContacts()
	// MUST be registered to allow (external: login) usage
	knownIssuers := appContacts.GetAuth().GetIssuers()
	if !slices.Contains(knownIssuers, idToken.Iss) {
		return errors.BadRequest(
			errors.Message("identity: invalid issuer identifier"),
		)
	}
	// FIXME: disallow .well-known issuers
	reserved := true
	switch strings.ToLower(idToken.Iss) {
	case "bot", "script":
	case "app", "service":
	case "user", "webitel", "contact":
	case "viber":
	case "telegram":
	case "facebook", "instagram", "whatsapp":
	default:
		{
			reserved = false // allowed !
		}
	}

	if reserved {
		return errors.BadRequest(
			errors.Message("identity: reserved issuer identifier"),
		)
	}

	if idToken.Dc < 1 {
		// invalid or not assigned !
		idToken.Dc = app.GetDc()
	}
	// Ensure App.Dc tenant match !
	if idToken.Dc != app.GetDc() {
		return errors.BadRequest(
			errors.Message("identity: invalid business identifier"),
		)
	}
	// Validate [idToken.Sub] subject identifier
	// /[A-Za-z0-9\-\.]+/
	if idToken.Sub == "" {
		return errors.BadRequest(
			errors.Message("identity: subject identifier is missing"),
		)
	}

	if idToken.App == "" {
		// current
		idToken.App = app.ClientId()
	}

	// Validate [idToken.Sub] subject identifier
	// /[A-Za-z0-9\-\.]+/
	commonName := ContactName{
		CommonName: idToken.Name,
		GivenName:  idToken.GivenName,
		MiddleName: idToken.MiddleName,
		FamilyName: idToken.FamilyName,
	}

	if !commonName.IsValid() {
		return errors.BadRequest(
			errors.Message("identity: subject name is missing"),
		)
	}
	// build ; normalize
	idToken.Name = commonName.String()

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
