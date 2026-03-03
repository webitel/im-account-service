package jwks

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/lestrrat-go/httprc/v3"
	"github.com/lestrrat-go/httprc/v3/errsink"
	"github.com/lestrrat-go/httprc/v3/tracesink"
	"github.com/lestrrat-go/jwx/v3/jwk"
)

type Options struct {
  Logger *slog.Logger
  NumWorkers int
  HttpClient jwk.HTTPClient
  AddResource ResourceOptions
}

type Option func(opts *Options)

func cacheOptions(opts []Option) (Options) {
  options := Options{
    Logger: nil, // slog.Default()
    HttpClient: nil, // http.DefaultClient
    NumWorkers: runtime.NumCPU(),
    AddResource: ResourceOptions{
    	ParseOptions:       nil, // []jwk.ParseOption{},
    	RefreshInterval:    0, // ConstantInterval > 0 // skip: [Cache-Control] header spec.
    	RefreshMinInterval: (1 * time.Minute), // default: (15 * time.Minute) // 15 min
    	RefreshMaxInterval: (24 * time.Hour), // default: (24 * time.Hour * 30) // 30 days
    },
  }
  options.init(opts)
  return options
}

func (x *Options) init(opts []Option) {
  for _, setup := range opts {
    setup(x)
  }
}

func (x Options) httprcOptions(with ...httprc.NewClientOption) (opts []httprc.NewClientOption) {
  // httprc.WithHTTPClient(cl HTTPClient) NewClientResourceOption
  // httprc.WithErrorSink(sink httprc.ErrorSink) NewClientOption
  // httprc.WithTraceSink(sink httprc.TraceSink) NewClientOption
  // httprc.WithWhitelist(wl httprc.Whitelist) NewClientOption
  // httprc.WithWorkers(n int) NewClientOption // default: 5

  // defaults
  if x.Logger != nil {
    ctx := context.TODO()
    verb := slog.LevelError
    if x.Logger.Enabled(ctx, verb) {
      opts = append(opts, httprc.WithErrorSink(
        errsink.NewSlog(x.Logger),
      ))
    }
    verb = slog.LevelDebug
    if x.Logger.Enabled(ctx, verb) {
      opts = append(opts, httprc.WithTraceSink(
        tracesink.NewSlog(x.Logger),
      ))
    }
  }
  if x.NumWorkers != 0 {
    opts = append(opts, httprc.WithWorkers(x.NumWorkers))
  }
  if x.HttpClient != nil {
    opts = append(opts, httprc.WithHTTPClient(x.HttpClient))
  }

  // custom
  opts = append(opts, with...)

  // override ..
  
  return opts
}

type Cache struct {
  mx sync.Mutex // guard
  opts Options // configuration
  cache *jwk.Cache // backend management
  rsync sync.Map // httpClient handle stats
}

func NewCache(opts ...Option) *Cache {
  return &Cache{opts: cacheOptions(opts)}
}

func (c *Cache) Init(opts ...Option) error {

  c.mx.Lock()
  defer c.mx.Unlock()

  // once ...
  if c.cache != nil {
    // started
    return fmt.Errorf("jwks.Cache: backend is already running")
  }

  c.opts.init(opts)
  return nil
}

func (c *Cache) Start(_ context.Context, opts ...httprc.NewClientOption) error {

  c.mx.Lock()
  defer c.mx.Unlock()

  // once ...
  if c.cache != nil {
    // started
    return nil
  }

  // opts = c.opts.httprcOptions(opts...)
  opts = append(
    c.opts.httprcOptions(opts...),
    // httprc.WithHTTPClient(httpClient{c}),
    httprc.WithHTTPClient(httpProxy{}),
  )

  cache, err := jwk.NewCache(
    context.Background(), httprc.NewClient(opts...),
  )

  if err != nil {
    return err
  }

  c.cache = cache
  return nil
}

func (c *Cache) Stop(ctx context.Context) error {

  c.mx.Lock()
  defer c.mx.Unlock()

  // once ...
  cache := c.cache
  c.cache = nil // destroy

  if cache == nil {
    // stopped
    return nil
  }

  err := cache.Shutdown(ctx)

  if err != nil {
    // FIXME
  }

  return err
}

// // Fetch fetches a JWKS from the cache. If the JWKS URL has not been registered with
// // the cache, an error is returned.
// func (c *Cache) Fetch(ctx context.Context, uri string, _ ...jwk.FetchOption) (jwk.Set, error) {
// 	if !c.cache.IsRegistered(ctx, uri) {
// 		return nil, fmt.Errorf(`jwks.Cache: resource url %q has not been registered`, uri)
// 	}
// 	return c.cache.Lookup(ctx, uri)
// }

func (c *Cache) Refresh(ctx context.Context, uri string, opts ...ResourceOption) (jwk.Set, error) {

  // return c.Register(ctx, uri, opts...)

  // ensure: backend httprc.(Client).Controller is running
  err := c.Start(context.Background())
  if err != nil {
    return nil, err
  }

  // if !c.cache.IsRegistered(ctx, uri) {
  src, err := c.cache.LookupResource(
    ctx, uri,
  )

  if err != nil {
    // httprc.ErrResourceNotFound(!)
    err = c.cache.Register(
      ctx, uri, c.opts.AddResource.registerOptions(
        jwk.WithWaitReady(false), // NON BLOCKED !
      )...,
    )
    if err != nil {
      // - option(s) error
      // - URL parse error
      return nil, err
    }
    // registered resource port
    src, err = c.cache.LookupResource(
      ctx, uri,
    )

    if err != nil {
      // panic("unreachable code")
      return nil, err
    }
  }

    ctx, cancel := context.WithTimeoutCause(
      ctx, (time.Second * 5), fmt.Errorf("jwks.Cache.Fetch( timeout )"),
    )
    defer cancel()

    // _ = c.cache.Register(
    //   ctx, uri, c.opts.AddResource.registerOptions(
    //     jwk.WithWaitReady(false), // NON BLOCKED !
    //   )...,
    // )

    // if err != nil {
    //   // httprc.ErrResourceAlreadyExists(!)
    // }

    // Fetch its content NOW !
    _, err = c.cache.Refresh(
      ctx, uri, // context.WithTimeout(5s)
    )
    if err != nil {
      // - invalid URL
      // - unreachable URL
      // - invalid content
      _ = c.cache.Unregister(
        context.WithoutCancel(ctx), uri,
      )
      return nil, err
    }
    
    // if err != nil {
    //   // httprc.ErrResourceAlreadyExists()
    //   return nil, err
    // }

    // _, err = c.cache.Refresh(ctx, uri)
    // if err != nil {
    //   // - invalid URL
    //   // - unreachable URL
    //   // - invalid content
    //   _ = c.cache.Unregister(ctx, uri)
    //   return nil, err
    // }
  // }

  // return c.cache.CachedSet(uri)
  src, err = c.cache.LookupResource(ctx, uri)
	if err != nil {
		return nil, fmt.Errorf(`failed to lookup resource %q: %w`, uri, err)
	}
  err = src.Ready(ctx)
  if err != nil {
    // - context.Err()
    return nil, fmt.Errorf(`failed to lookup resource %q: not available`, uri)
  }
	return jwk.NewCachedSet(src), nil

  // cache, err := c.cache.LookupResource(ctx, uri)
  // if err != nil {
  //   // - not regustered
  //   return nil, err
  // }

  // if cache.IsBusy()

}

func (c *Cache) Register(ctx context.Context, uri string, opts ...ResourceOption) (jwk.Set, error) {

  // ensure: backend httprc.(Client).Controller is running
  err := c.Start(context.Background())
  if err != nil {
    return nil, err
  }

  reg, _ := c.lookup(uri)
  if reg == nil {

    reg = &resource{
    	opts: c.opts.AddResource, // clone
    	// data: nil, // &httprc.ResourceBase[jwk.Set]{},
    	// date: time.Time{},
    	// err:  err,
    }
    reg.opts.init(opts)
    reg.data, err = c.cache.LookupResource(
      ctx, uri,
    )
    
    if err != nil {
      // - httprc.ErrResourceNotFound(!)
      err = c.cache.Register(
        ctx, uri, reg.opts.registerOptions(
          jwk.WithWaitReady(false), // NON BLOCKED !
        )...,
      )
      
      if err != nil {
        // - option(s) error
        // - URL parse error
        return nil, err
      }

      // registered resource data
      reg.data, err = c.cache.LookupResource(
        ctx, uri,
      )

      if err != nil {
        // - httprc.ErrResourceAlreadyExists(!)
        return nil, err
      }

      // +[ OK ] registered
    }

    c.register(reg)
  }

  // // if !c.cache.IsRegistered(ctx, uri) {
  // src, err := c.cache.LookupResource(
  //   ctx, uri,
  // )

  // if err != nil {
  //   // httprc.ErrResourceNotFound(!)
  //   _ = c.cache.Register(
  //     ctx, uri, c.opts.AddResource.registerOptions(
  //       jwk.WithWaitReady(false), // NON BLOCKED !
  //     )...,
  //   )
  //   if err != nil {
  //     // - option(s) error
  //     // - URL parse error
  //     return nil, err
  //   }
  //   // registered resource port
  //   src, err = c.cache.LookupResource(
  //     ctx, uri,
  //   )
  // }

  // reg.data.IsBusy() // sync in progress ..
  // reg.data.Ready(ctx) // blocks until FIRST sync success

  // if reg.data.Ready() {

  // }

    ctx, cancel := context.WithTimeoutCause(
      ctx, (time.Second * 5), fmt.Errorf("jwks.Cache.Fetch( timeout )"),
    )
    defer cancel()

    // _ = c.cache.Register(
    //   ctx, uri, c.opts.AddResource.registerOptions(
    //     jwk.WithWaitReady(false), // NON BLOCKED !
    //   )...,
    // )

    // if err != nil {
    //   // httprc.ErrResourceAlreadyExists(!)
    // }

    // Fetch its content NOW !
    _, err = c.cache.Refresh(
      ctx, uri, // context.WithTimeout(5s)
    )
    if err != nil {
      // // - invalid URL
      // // - unreachable URL
      // // - invalid content
      // _ = c.cache.Unregister(
      //   context.WithoutCancel(ctx), uri,
      // )
      // c.deregister(reg.data.URL())
      return nil, err
    }
    
    // if err != nil {
    //   // httprc.ErrResourceAlreadyExists()
    //   return nil, err
    // }

    // _, err = c.cache.Refresh(ctx, uri)
    // if err != nil {
    //   // - invalid URL
    //   // - unreachable URL
    //   // - invalid content
    //   _ = c.cache.Unregister(ctx, uri)
    //   return nil, err
    // }
  // }

  // return c.cache.CachedSet(uri) with Context
  keyset, err := c.cache.LookupResource(ctx, uri)
	if err != nil {
		return nil, fmt.Errorf(`failed to lookup resource %q: %w`, uri, err)
	}
  err = keyset.Ready(ctx)
  if err != nil {
    // - context.Err()
    return nil, fmt.Errorf(`failed to lookup resource %q: not available`, uri)
  }
	return jwk.NewCachedSet(keyset), nil

  // cache, err := c.cache.LookupResource(ctx, uri)
  // if err != nil {
  //   // - not regustered
  //   return nil, err
  // }

  // if cache.IsBusy()

}

func (c *Cache) lookup(uri string) (*resource, bool) {
  reg, _ := c.rsync.Load(uri)
  res, _ := reg.(*resource)
  return res, (res != nil)
}

func (c *Cache) register(add *resource) {
  c.rsync.Store(add.data.URL(), add)
}

func (c *Cache) deregister(uri string) {
  c.rsync.Delete(uri)
}

// func (c *Cache) syncTrip(uri string, err error) {
//   rst, _ := c.lookup(uri)
// }

// func (c *Cache) Fetch(ctx context.Context, uri string, opts ...ResourceOption) (jwk.Set, error) {
//   // ensure is running

//   ctx, cancel := context.WithTimeoutCause(
//     ctx, (time.Second * 5), fmt.Errorf("jwks.Cache.Fetch( timeout )"),
//   )
//   defer cancel()

//   forever := context.Background()
//   err := c.Start(forever)
//   if err != nil {
//     return nil, err
//   }

//   // if !c.cache.IsRegistered(ctx, uri) {

//     err = c.cache.Register(
//       forever, uri, c.opts.AddResource.registerOptions(
//         jwk.WithWaitReady(false), // NON BLOCKED !
//       )...,
//     )

//     // httprc.ErrResourceAlreadyExists(?)
//     if err == nil {
//       // New resource registered
//       // Fetch its content NOW !
//       _, err = c.cache.Refresh(ctx, uri)
//       if err != nil {
//         // - invalid URL
//         // - unreachable URL
//         // - invalid content
//         _ = c.cache.Unregister(ctx, uri)
//         return nil, err
//       }
//     }
    
//     // if err != nil {
//     //   // httprc.ErrResourceAlreadyExists()
//     //   return nil, err
//     // }

//     // _, err = c.cache.Refresh(ctx, uri)
//     // if err != nil {
//     //   // - invalid URL
//     //   // - unreachable URL
//     //   // - invalid content
//     //   _ = c.cache.Unregister(ctx, uri)
//     //   return nil, err
//     // }
//   // }

//   // return c.cache.CachedSet(uri)
//   e, err := c.cache.LookupResource(ctx, uri)
// 	if err != nil {
// 		return nil, fmt.Errorf(`failed to lookup resource %q: %w`, uri, err)
// 	}
// 	return jwk.NewCachedSet(e), nil

//   // cache, err := c.cache.LookupResource(ctx, uri)
//   // if err != nil {
//   //   // - not regustered
//   //   return nil, err
//   // }

//   // if cache.IsBusy()

// }