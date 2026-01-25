package auth

import (
	"cmp"
	"context"
	"time"

	adpb "github.com/webitel/im-account-service/internal/client/webitel/proto/gen/auth"
	"github.com/webitel/im-account-service/internal/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Inspect Webitel Authorization token
func (c *Client) Inspect(ctx context.Context, token string, opts ...InspectOption) (debug *adpb.Userinfo, err error) {

	tx := NewInspectRequest(ctx, opts...)

	defer func() {
		// NOT Found !
		if err == nil && !checkTokenIsValid(&tx, debug) {
			// debug, err = nil, ErrTokenIsInvalid
			err = ErrTokenIsInvalid
		}
	}()

	// from cache ..
	var found bool
	if debug, found = c.cache.Get(token); found {
		return // debug, nil
	}
	// DO send request !
	md := c.creds.Copy()
	md.Set("x-webitel-access", token)
	reqCtx := metadata.NewOutgoingContext(ctx, md)
	debug, err = c.authz.UserInfo(
		reqCtx, &adpb.UserinfoRequest{
			// AccessToken: token,
		},
		// // ...client.CallOption
		// client.WithAddress(a ...string),
		// client.WithBackoff(fn BackoffFunc),
		// client.WithCache(c time.Duration),
		// client.WithCallWrapper(cw ...CallWrapper),
		// client.WithConnClose(),
		// client.WithDialTimeout(d time.Duration),
		// client.WithRequestTimeout(d time.Duration),
		// client.WithRetries(i int),
		// client.WithRetry(fn RetryFunc),
		// client.WithSelectOption(so ...selector.SelectOption),
		// client.WithServiceToken(),
		// client.WithStreamTimeout(d time.Duration),
	)

	if err != nil {
		return nil, err
	}

	if debug != nil {
		_ = c.cache.Add(token, debug)
	}

	return // debug?, nil

}

// Indicates ANY token clams violation
var ErrTokenIsInvalid = status.New(
	codes.Unauthenticated,
	"[webitel]: token is invalid",
).Err()

// Indicates [NotBefore] claim violation
var ErrTokenNotActive = status.New(
	codes.Unauthenticated,
	"[webitel]: token not active",
).Err()

// Indicates [NotAfter] claim violation
var ErrTokenIsExpired = status.New(
	codes.Unauthenticated,
	"[webitel]: token is expired",
).Err()

func checkDateIsZero(date *time.Time) (ok bool) {
	return (date == nil) || date.IsZero() || (0 < date.Unix())
}

func checkNotBefore(ctx *InspectRequest, nbf int64) (ok bool) {
	// [notBefore] date specified ?
	if ok = !(0 < nbf); ok {
		return // true // no date to check !
	}
	ok = (nbf <= ctx.Date.UnixMilli())
	return // ok?
}

func checkNotAfter(ctx *InspectRequest, exp int64) (ok bool) {
	// [notAfter] date specified ?
	if ok = !(0 < exp); ok {
		return // true // no date to check !
	}
	ok = (ctx.Date.UnixMilli() < exp)
	return // ok?
}

func checkTokenIsValid(ctx *InspectRequest, token *adpb.Userinfo) (ok bool) {
	// Found ?
	if ok = (token != nil); !ok {
		ctx.Error = ErrTokenIsInvalid
		return // false ; token is invalid
	}
	// NotBefore
	if ok = checkNotBefore(ctx, token.GetUpdatedAt()); !ok {
		ctx.Error = ErrTokenNotActive
		return // false ; token not active yet
	}
	// NotAfter
	if ok = checkNotAfter(ctx, token.GetExpiresAt()); !ok {
		ctx.Error = ErrTokenIsExpired
		return // false ; token is expired
	}
	// OK
	return // true
}

// func checkTokenNotBefore(token *adpb.Userinfo, check ...time.Time) (ok bool) {
// 	// [notBefore] date specified ?
// 	if ok = !(0 < token.GetUpdatedAt()); ok {
// 		return // true // no date to check !
// 	}
// 	var date time.Time
// 	if n := len(check); n > 0 {
// 		date = check[n-1]
// 	}
// 	if date.IsZero() || date.Unix() <= 0 {
// 		date = ad.Local.Now()
// 	}
// 	ok = (token.UpdatedAt <= date.UnixMilli())
// 	return // ok?
// }

// func checkTokenNotAfter(token *adpb.Userinfo, check ...time.Time) (ok bool) {
// 	// [notAfter] date specified ?
// 	if ok = !(0 < token.GetExpiresAt()); ok {
// 		return // true // no date to check !
// 	}
// 	var date time.Time
// 	if n := len(check); n > 0 {
// 		date = check[n-1]
// 	}
// 	if date.IsZero() || date.Unix() <= 0 {
// 		date = ad.Local.Now()
// 	}
// 	ok = (date.UnixMilli() < token.ExpiresAt)
// 	return // ok?
// }

type InspectRequest struct {
	context.Context
	Date  time.Time
	Error error
}

type InspectOption func(req *InspectRequest)

func NewInspectRequest(ctx context.Context, opts ...InspectOption) (req InspectRequest) {
	req = InspectRequest{
		Context: cmp.Or(ctx, context.Background()),
	}
	for _, option := range opts {
		option(&req)
	}
	if checkDateIsZero(&req.Date) {
		req.Date = model.LocalTime.Now()
	}
	return // req
}

func InspectDate(at time.Time) InspectOption {
	return func(req *InspectRequest) {
		if !checkDateIsZero(&at) {
			req.Date = at
		}
		// invalid ; ignore
	}
}

type VerifyOption func(token *adpb.Userinfo) error

// Verify token's claims to satisfy your needs
func (ctx *InspectRequest) Verify(token *adpb.Userinfo, claims ...VerifyOption) (ok bool) {
	// lastError := ctx.Error
	// ctx.Error = nil // sanitize
	if ok = checkTokenIsValid(ctx, token); !ok {
		return // false
	}
	var err error
	for _, claim := range claims {
		err = claim(token)
		// Failed ?
		if err != nil {
			ctx.Error = err
			return false
		}
	}
	return true
}
