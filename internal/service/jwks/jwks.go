package jwks

import (
	"context"

	"github.com/lestrrat-go/httprc/v3"
	"github.com/lestrrat-go/jwx/v3/jwk"
)

// JWKs (global) registry
var Default = NewCache()

// func Init(opts ...httprc.NewClientOption) {
//   Default.Init(opts...)
// }

func Init(opts ...Option) {
  Default.Init(opts...)
}

func Start(ctx context.Context, opts ...httprc.NewClientOption) error {
  return Default.Start(ctx, opts...)
}

func Stop(ctx context.Context) error {
  return Default.Stop(ctx)
}

// func Register(ctx context.Context, uri string, opts ...jwk.RegisterOption) (jwk.Set, error) {
//   return Default.Register(ctx, uri, opts...)
// }

// func Unregister(ctx context.Context, uri string) error {
//   return Default.Unregister(ctx, uri)
// }

func Refresh(ctx context.Context, uri string, opts ...ResourceOption) (jwk.Set, error) {
  return Default.Refresh(ctx, uri, opts...)
}

func Fetch(ctx context.Context, uri string, _ ...ResourceOption) (jwk.Set, error) {
  return jwk.Fetch(ctx, uri)
}