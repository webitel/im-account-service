package postgres

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Postgres Database Client
type DB struct {
  options Options
  client *pgxpool.Pool
  types atomic.Pointer[pgtype.Map]
}

// New (*DB) configuration
func New(opts ...Option) (*DB, error) {

  dbo := &DB{
    options: newOptions(opts),
  }

  dsn, err := pgxpool.ParseConfig(
    dbo.options.DataSourceName,
  )

  if err != nil {
		return nil, fmt.Errorf("parse dsn: %v", err)
	}

  // [custom] options ..
  for _, setup := range dbo.options.ConnOptions {
    setup(dsn)
  }

	// [default] options
	if dsn.ConnConfig.Tracer == nil && dbo.options.Logger != nil {
		dsn.ConnConfig.Tracer = debugLog(dbo.options.Logger)
	}

	// [system] options ..
	dsn.AfterConnect = dbo.onAfterConnect
  // OnBeforeAcquire(dbo.onBeforeAcquire)(dsn)
	dsn.BeforeAcquire = dbo.onBeforeAcquire

  client, err := pgxpool.NewWithConfig(
    dbo.options.Context, dsn,
  )
	
  if err != nil {
		return nil, fmt.Errorf("create connection pool: %v", err)
	}

	// if err := client.Ping(dbo.options.Context); err != nil {
	// 	return nil, fmt.Errorf("ping database: %v", err)
	// }

	dbo.client = client

	return dbo, nil
}

func (dbo *DB) Init(opts ...Option) {
	dbo.options.setup(opts)
}

func (dbo *DB) Options() Options {
	return dbo.options
}

func (dbo *DB) Client() *pgxpool.Pool {
	if dbo != nil {
		return dbo.client
	}
	return nil
}

func (dbo *DB) TypeMap() *pgtype.Map {
	if dbo != nil {
		types := dbo.types.Load()
		if types != nil {
			return types
		}
	}
	return defaults.Types.Load()
}

func (dbo *DB) onAfterConnect(ctx context.Context, conn *pgx.Conn) (err error) {
	// AfterConnect is called after a connection is established, but before it is added to the pool.
	
	// next := dbo.client.Config().AfterConnect
	// if next != nil {
	// 	return next(ctx, conn)
	// }

	for _, hook := range dbo.options.AfterConnect {
		err = hook(ctx, conn)
		if err != nil {
			return err
		}
	}

	return nil
}

func (dbo *DB) onBeforeAcquire(ctx context.Context, conn *pgx.Conn) (ok bool) {
  // BeforeAcquire is called before a connection is acquired from the pool.
  // It must return true to allow the acquisition or false to indicate that the connection
  // should be destroyed and a different connection should be acquired.
  _ = dbo.types.CompareAndSwap(nil, conn.TypeMap())
	
	// next := dbo.client.Config().BeforeAcquire
	// if next != nil {
	// 	return next(ctx, conn)
	// }

	for _, hook := range dbo.options.BeforeAcquire {
		if ok = hook(ctx, conn); !ok {
			return false
		}
	}

  return true
}

var defaults struct {
	DB    atomic.Pointer[DB]
	Types atomic.Pointer[pgtype.Map]
}

func init() {
	defaults.Types.Store(
		pgtype.NewMap(),
	)
}

func Default() *DB {
	return defaults.DB.Load()
}

func SetDefault(db *DB) {
	defaults.DB.Store(db)
}