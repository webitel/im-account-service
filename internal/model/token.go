package model

import (
	"cmp"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/webitel/im-account-service/internal/errors"
)

// AccessToken GRANT for Contact (Account) at Device (Client) session Authorization
type AccessToken struct {
	Id      UUID       // subscription id ; e.g.: session.id, [creds].id
	Date    time.Time  // [re]generated date ; [not_before]
	Type    string     // token type ; default: "bearer"
	Token   string     // [access_token] string ; REQUIRED
	Scope   []string   // permissions granted ; OPTIONAL
	Expires *time.Time // [access_token] absolute expiry date
	Revoked *time.Time // [access_token] revocation date ; Invalidated -if- non-empty
	Refresh string     // [refresh_token] string ; OPTIONAL
	// MaxAge  *time.Time // [max_age] for GRANT [re]generation ; no [refresh_token] after ..
}

// Indicates ANY token clams violation
var ErrTokenIsInvalid = errors.Unauthorized(
	errors.Message("messaging: token is invalid"),
)

// Indicates [NotBefore] claim violation
var ErrTokenNotActive = errors.Unauthorized(
	errors.Message("messaging: token not active"),
)

// Indicates [NotAfter] claim violation
var ErrTokenIsExpired = errors.Unauthorized(
	errors.Message("messaging: token is expired"),
)

func (e *AccessToken) Verify(date time.Time) error {
	// assigned ?
	if e == nil || e.Token == "" {
		return ErrTokenIsInvalid
	}
	if date.IsZero() {
		date = LocalTime.Now()
	}
	// revoked ?
	if e.Revoked != nil && date.After(*e.Revoked) {
		return ErrTokenIsInvalid // revoked
	}
	// expired ?
	if e.Expires != nil && e.Expires.Before(date) {
		return ErrTokenIsExpired
	}
	// [ OK ]
	return nil
}

type GenerateOptions struct {
	Type    string
	Length  int           // Token length
	Expires time.Duration // TTL period
	Refresh *GenerateOptions
	GenOpts []GenerateOption // generate default options
}

type generateRequest struct {
	NoRefresh bool       // disallow refresh_token generation
	NotBefore time.Time  // generation date
	NotAfter  *time.Time // absolute expiration date
	Scope     []string   // GRANT scope(s)
}

type GenerateOption func(gen *generateRequest)

func TokenNotBefore(date time.Time) GenerateOption {
	return func(gen *generateRequest) {
		gen.NotBefore = date
	}
}

func TokenScope(scope []string) GenerateOption {
	return func(gen *generateRequest) {
		gen.Scope = scope
	}
}

func TokenNoRefresh() GenerateOption {
	return func(gen *generateRequest) {
		gen.NoRefresh = true
	}
}

// Generate NEW token grant
func (gen GenerateOptions) Generate(opts ...GenerateOption) (grant AccessToken, err error) {
	req := generateRequest{}
	for _, option := range gen.GenOpts {
		option(&req) // default
	}
	for _, option := range opts {
		option(&req) // request
	}
	if req.NotBefore.IsZero() {
		req.NotBefore = LocalTime.Now()
	}
	token, err := GenerateSecureToken(gen.Length)
	if err != nil {
		return grant, err
	}
	grant = AccessToken{
		Date:  req.NotBefore,
		Type:  cmp.Or(gen.Type, "bearer"),
		Token: token,
		// Refresh: "",
		// Expires: &time.Time{},
		Scope: req.Scope,
	}
	if gen.Expires > 0 {
		expiry := grant.Date.Add(gen.Expires)
		grant.Expires = &expiry
	}
	if !req.NoRefresh {
		// refresh_token policy defined ?
		refresh := gen.Refresh
		if refresh != nil {
			token, err := GenerateSecureToken(refresh.Length)
			if err != nil {
				return grant, err
			}
			// generated
			grant.Refresh = token
		}
	}

	return grant, nil
}

// GenerateSecureToken generates a cryptographically secure random token of a specified length.
func GenerateSecureToken(length int) (string, error) {
	if length < 16 {
		length = 16
	}
	codec := base64.RawURLEncoding
	bin := make([]byte, codec.EncodedLen(length))
	_, err := rand.Read(bin)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return codec.EncodeToString(bin), nil
}
