package jwks

import (
	"context"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/lestrrat-go/httprc/v3"
)

// NOTE: httprc.ResourceBase[jwk.Set] failed to Sync.(Send) request
// which cause to query resource.Next() attempt(s) every (default: time.Second ; without httprc.WithConstantInterval )

type httpClient struct {
  manager *Cache
}

var _ httprc.HTTPClient = httpClient{}

var onHttpError = &backoff.ExponentialBackOff{
  InitialInterval:     (time.Second * 2),
  RandomizationFactor: 0, // 0.5,
  Multiplier:          2,
  MaxInterval:         (time.Second * 16), // (time.Minute),
}

func (c httpClient) Do(req *http.Request) (res *http.Response, err error) {

  client := c.manager.opts.HttpClient
  if client == nil {
    client = http.DefaultClient
  }

  // ctx := req.Context()
  uri := req.URL.String()
  rst, _ := c.manager.lookup(uri)
  
  // begin
  if rst != nil {
    rst.date = time.Now()
  }
  // invoke ..
  res, err = client.Do(req)
  // end ..
  if rst != nil {
    rst.err = err // last error
  }

  // [ OK ]
  if err == nil {
    // // RESET
    // if retry != nil {
    //   retry.Reset()
    //   // resource.SetNext(time.Time{}) // Zero()
    // }
    return res, nil
  }

  if rst == nil {
    // no registration track found
    return // res, err
  }

  // BackOff on HTTP error ..
  next := rst.opts.NextBackOff.next(err)

  // rw := &c.manager.mx
  // rw.Lock() // ---------------------------- +[RW]

  // next, _ := c.manager.retry[uri]

  // if next != nil {
  //   after = next(err)
  // }

  // if after < 1 {
  //   delete(c.manager.retry, uri)
  //   rw.Unlock() // ------------------------ -[RW]
  //   _ = c.manager.cache.Unregister(
  //     context.Background(), uri,
  //   )
  //   return res, err
  // }
  // rw.Unlock() // -------------------------- -[RW]

  // // failed to reach resource ..
  // if c.retry == nil {
  //   c.retry = make(map[string]backoff.BackOff)
  // }
  // if retry == nil {
  //   // retry = backoff.NewExponentialBackOff()
  //   retry = &backoff.ExponentialBackOff{
  //   	RandomizationFactor: 0, // 0.5,
  //   	InitialInterval:     (time.Second),
  //   	Multiplier:          2,
  //   	MaxInterval:         (time.Second * 16), // (time.Minute),
  //   }
  //   // retry = &IncrementBackOff{
  //   // 	InitialInterval:   (time.Second * 2),
  //   // 	IncrementInterval: (time.Second * 2),
  //   // 	MaxInterval:       time.Minute,
  //   // }
  //   c.retry[uri] = retry
  // }

  // next := retry.NextBackOff()
  // slog.Default().Info("[ BACKOFF ]", "uri", uri, "next", next)
  // next := time.Now() // resource.Next()
  if time.Second < next {
    rst.data.SetNext(time.Now().Add(next)) // (time.Second * 3)
  } else if next <= 0 {
    next = c.manager.opts.AddResource.RefreshMinInterval
    rst.data.SetNext(time.Now().Add(next))
    go func(rst *resource) {
      uri := rst.data.URL()
      _ = c.manager.cache.Unregister(
        context.Background(), uri,
      )
      c.manager.deregister(uri)
    }(rst)
  }

  return // res, err
}

