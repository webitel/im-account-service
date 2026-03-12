package postgres

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Options struct {

  DataSourceName string
  ConnOptions []ConnOption
  Logger *slog.Logger

  Context context.Context

  // -------------- Hook(s) ------------------- //

  // BeforeConnect is called before a new connection is made.
  // It is passed a copy of the underlying pgx.ConnConfig and
	// will not impact any existing open connections.
	BeforeConnect []func(context.Context, *pgx.ConnConfig) error

	// AfterConnect is called after a connection is established,
  // but before it is added to the pool.
	AfterConnect []func(context.Context, *pgx.Conn) error

	// BeforeAcquire is called before a connection is acquired from the pool.
  // It must return true to allow the acquisition
  // or false to indicate that the connection should be destroyed
  // and a different connection should be acquired.
	BeforeAcquire []func(context.Context, *pgx.Conn) bool

	// AfterRelease is called after a connection is released, but before it is returned to the pool.
  // It must return true to return the connection to the pool or false to destroy the connection.
	AfterRelease []func(*pgx.Conn) bool

	// BeforeClose is called right before a connection is closed and removed from the pool.
	BeforeClose []func(*pgx.Conn)
}

// New.(*DB) Option
type Option func(dbo *Options)

type ConnOption func(dsn *pgxpool.Config)

func (c *Options) setup(opts []Option) {
  for _, setup := range opts {
    setup(c)
  }
}

func newOptions(opts []Option) Options {
  c := Options{
    // defaults
    Context: context.Background(),
  }
  c.setup(opts)
  // normalize
  return c
}

// DataSourceName, connectionString ..
func DataSourceName(connString string) Option {
  return func(dsn *Options) {
    dsn.DataSourceName = connString
  }
}

// *pgxpool.Config Option(s)
func ConnOptions(opts ...ConnOption) Option {
  return func(dbo *Options) {
    dbo.ConnOptions = append(dbo.ConnOptions, opts...)
  }
}

// Logger Option
func Logger(std *slog.Logger) Option {
  return func(dbo *Options) {
    dbo.Logger = std
  }
}

func FallbackApplicationName(app string) ConnOption {
  return func(dsn *pgxpool.Config) {
    // https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNECT-FALLBACK-APPLICATION-NAME
    if dsn.ConnConfig.RuntimeParams["application_name"] == "" {
      dsn.ConnConfig.RuntimeParams["application_name"] = app // &fallback_application_name=
    }
  }
}

// AfterConnect is called after a connection is established,
// but before it is added to the pool.
func OnAfterConnect(hook func(context.Context, *pgx.Conn) error) Option {
  return func(dbo *Options) {
    if hook != nil {
      dbo.AfterConnect = append(dbo.AfterConnect, hook)
    }
  }
}

// BeforeAcquire is called before a connection is acquired from the pool.
// It must return true to allow the acquisition or false to indicate that the connection
// should be destroyed and a different connection should be acquired.
func OnBeforeAcquire(hook func(context.Context, *pgx.Conn) bool) Option {
  return func(dbo *Options) {
    if hook != nil {
      dbo.BeforeAcquire = append(dbo.BeforeAcquire, hook)
    }
  }
}

// // AfterConnect is called after a connection is established,
// // but before it is added to the pool.
// func OnAfterConnect(hook func(context.Context, *pgx.Conn) error) ConnOption {
//   return func(dsn *pgxpool.Config) {
//     if hook == nil {
//       return
//     }
//     next := dsn.AfterConnect
//     dsn.AfterConnect = func(ctx context.Context, conn *pgx.Conn) (err error) {
//       err = hook(ctx, conn)
//       if err != nil {
//         return err
//       }
//       if next != nil {
//         return next(ctx, conn)
//       }
//       return nil
//     }
//   }
// }

// // BeforeAcquire is called before a connection is acquired from the pool.
// // It must return true to allow the acquisition or false to indicate that the connection
// // should be destroyed and a different connection should be acquired.
// func OnBeforeAcquire(hook func(context.Context, *pgx.Conn) bool) ConnOption {
//   return func(dsn *pgxpool.Config) {
//     if hook == nil {
//       return
//     }
//     next := dsn.BeforeAcquire
//     dsn.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) (ok bool) {
//       if ok = hook(ctx, conn); !ok {
//         return false
//       }
//       if next != nil {
//         return next(ctx, conn)
//       }
//       // default
//       return true
//     }
//   }
// }


