package cache

import (
	"context"
	"log/slog"
	"time"
)

// IndexFunc used to project index key(s) for given cache entry.
// All keys MUST be comparable and unique within all cache entries.
type IndexFunc func(data any) (keys []any)

type Options struct {
	Logger  *slog.Logger
	IndexKeys IndexFunc
	OnEvicted func(data any) // [NOTE]: not emited on TTL timeout  =((

	Size int
	TTL  time.Duration
}

type Option func(opts *Options)

func TTL(exp time.Duration) Option {
	return func(opts *Options) {
		opts.TTL = exp
	}
}

func Size(max int) Option {
	return func(opts *Options) {
		opts.Size = max
	}
}

func Logger(std *slog.Logger) Option {
	return func(opts *Options) {
		// if std != nil {
		// 	std = std.With("typeof", fmt.Sprintf(
		// 		"(%T)", reflect.Zero(reflect.TypeFor[T]()).Interface(),
		// 	)) // reflect.TypeFor[T]().Name(),
		// }
		opts.Logger = std
	}
}

// OnEvicted callback while LRU [add|del] eviction
// Does NOT emit when entry's TTL timed out =((
func OnEvicted[T any](cb func(T)) Option {
	return func(opts *Options) {
    if cb == nil {
      return
    }
    next := opts.OnEvicted
    opts.OnEvicted = func(data any) {
      if e, ok := data.(T); ok {
        cb(e)
      }
      if next != nil {
        next(data)
      }
    }
	}
}

func newOptions(opts []Option) Options {

  options := Options{
    // defaults
  }

	for _, setup := range opts {
		setup(&options)
	}

	return options
}

func (c *LRU) Options() Options {
	return c.opts
}

func (opts *Options) Log(level slog.Level, msg string, args ...any) {
	if out := opts.Logger; out != nil {
    ctx := context.Background()
		out.Log(ctx, level, msg, args...)
	}
}
