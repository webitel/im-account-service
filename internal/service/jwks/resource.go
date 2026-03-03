package jwks

import (
	"time"

	"github.com/lestrrat-go/httprc/v3"
	"github.com/lestrrat-go/jwx/v3/jwk"
)

type ResourceOptions struct {

  RefreshInterval time.Duration
  RefreshMinInterval time.Duration
  RefreshMaxInterval time.Duration

  ParseOptions []jwk.ParseOption
  // You can process an error occured while query an URL
  // Return positive [Retry-After] interval to keep sync trying
  // or negative Zero(-0) value to Unregister Resource URL binding
  NextBackOff BackOffFunc
}

type BackOffFunc func(error) time.Duration

func (fn BackOffFunc) next(err error) time.Duration {
  if fn != nil {
    return fn(err)
  }
  return 0
}

func FreeResource(_ error) time.Duration { return 0 }

type ResourceOption func(opts *ResourceOptions)

func resourceOptions(with []ResourceOption) ResourceOptions {
  // defaults
  opts := ResourceOptions{
    RefreshInterval:  0,
  	RefreshMinInterval:  (time.Minute),
  	RefreshMaxInterval:  (time.Hour * 24),
  	ParseOptions: nil,
  	NextBackOff: nil,
  }
  opts.init(with)
  return opts
}

func (opts *ResourceOptions) init(with []ResourceOption) {
  for _, setup := range with {
    setup(opts)
  }
}

func (opts ResourceOptions) registerOptions(with ...jwk.RegisterOption) []jwk.RegisterOption {
  for _, option := range opts.ParseOptions {
    with = append(with, option.(jwk.RegisterOption))
  }
  if opts.RefreshInterval > 0 {
    with = append(with, jwk.WithConstantInterval(opts.RefreshInterval))
  }
  if opts.RefreshMinInterval > 0 {
    with = append(with, jwk.WithMinInterval(opts.RefreshMinInterval))
  }
  if opts.RefreshMaxInterval > 0 {
    with = append(with, jwk.WithMaxInterval(opts.RefreshMaxInterval))
  }
  return with
}

// httpClient resource sync state
type resource struct {
  opts ResourceOptions
  data *httprc.ResourceBase[jwk.Set] // httprc.Client resource registration
  date time.Time // last date of sync request
  err error // last error of sync request
}