package pg

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	logger *slog.Logger
	client *pgxpool.Pool
	types  atomic.Pointer[pgtype.Map]
}

func New(ctx context.Context, logger *slog.Logger, dataSourceName string) (*DB, error) {

	dsn, err := pgxpool.ParseConfig(dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %v", err)
	}

	db := new(DB)
	{
		db.logger = logger
		// db.client = dbo
	}

	// https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNECT-FALLBACK-APPLICATION-NAME
	// if vs, _ := fallbackApplicationName(opt.Context); vs != "" {
	if dsn.ConnConfig.RuntimeParams["application_name"] == "" {
		dsn.ConnConfig.RuntimeParams["application_name"] = "im-account" // &fallback_application_name=
	}
	// }

	// pgxpool.Config
	dsn.BeforeConnect = func(_ context.Context, _ *pgx.ConnConfig) error {
		// BeforeConnect is called before a new connection is made.
		// It is passed a copy of the underlying pgx.ConnConfig
		// and will not impact any existing open connections.
		return nil
	}
	dsn.AfterConnect = func(_ context.Context, _ *pgx.Conn) error {
		// AfterConnect is called after a connection is established,
		// but before it is added to the pool.
		return nil
	}
	dsn.BeforeAcquire = func(_ context.Context, conn *pgx.Conn) bool {
		// It must return true to allow the acquisition
		// or false to indicate that the connection should be destroyed
		// and a different connection should be acquired.
		_ = db.types.CompareAndSwap(nil, conn.TypeMap())
		return true
	}
	dsn.AfterRelease = func(_ *pgx.Conn) bool {
		// It must return true to return the connection to the pool
		// or false to destroy the connection.
		return true
	}
	dsn.BeforeClose = func(_ *pgx.Conn) {
		// BeforeClose is called right before a connection is closed
		// and removed from the pool.
	}

	// pgx.ConnConfig
	dsn.ConnConfig.Tracer = debugLog(logger)

	//
	// pgconn.Config
	//
	// // BuildContextWatcherHandler is called to create a ContextWatcherHandler for a connection. The handler is called
	// // when a context passed to a PgConn method is canceled.
	// dsn.ConnConfig.Config.BuildContextWatcherHandler = func(*pgconn.PgConn) ctxwatch.Handler {
	// 	panic("TODO")
	// }

	// ValidateConnect is called during a connection attempt after a successful authentication with the PostgreSQL server.
	// It can be used to validate that the server is acceptable. If this returns an error the connection is closed and the next
	// fallback config is tried. This allows implementing high availability behavior such as libpq does with target_session_attrs.
	dsn.ConnConfig.Config.ValidateConnect = func(ctx context.Context, conn *pgconn.PgConn) error {
		return nil
	}

	// AfterConnect is called after ValidateConnect. It can be used to set up the connection (e.g. Set session variables
	// or prepare statements). If this returns an error the connection attempt fails.
	dsn.ConnConfig.Config.AfterConnect = func(ctx context.Context, conn *pgconn.PgConn) error {
		return nil
	}

	// OnNotice is a callback function called when a notice response is received.
	dsn.ConnConfig.Config.OnNotice = func(conn *pgconn.PgConn, notice *pgconn.Notice) {

	}

	// OnNotification is a callback function called when a notification from the LISTEN/NOTIFY system is received.
	dsn.ConnConfig.Config.OnNotification = func(conn *pgconn.PgConn, notify *pgconn.Notification) {

	}

	// // OnPgError is a callback function called when a Postgres error is received by the server. The default handler will close
	// // the connection on any FATAL errors. If you override this handler you should call the previously set handler or ensure
	// // that you close on FATAL errors by returning false.
	// onPgError := dsn.ConnConfig.Config.OnPgError
	// dsn.ConnConfig.Config.OnPgError = func(conn *pgconn.PgConn, err *pgconn.PgError) bool {
	// 	return onPgError(conn, err)
	// }

	dbo, err := pgxpool.NewWithConfig(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %v", err)
	}

	if err := dbo.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %v", err)
	}

	db.client = dbo

	return db, nil
}

func (db *DB) Client() *pgxpool.Pool {
	if db != nil {
		return db.client
	}
	return nil
}

func (db *DB) TypeMap() *pgtype.Map {
	if db != nil {
		types := db.types.Load()
		if types != nil {
			return types
		}
	}
	return defaults.Types.Load()
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
