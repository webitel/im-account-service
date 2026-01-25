package migrate

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	"github.com/webitel/im-account-service/cmd"
	"github.com/webitel/im-account-service/config"
	"github.com/webitel/im-account-service/migrations"
)

func CMD() *cli.Command {
	return &cli.Command{
		Name:  "migrate",
		Usage: "Execute database migrations",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config_file",
				Aliases: []string{"c"},
				Usage:   "Path to the configuration file",
			},
		},
		Action: func(c *cli.Context) error {

			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}

			app := fx.New(
				fx.Provide(
					func() *config.Config { return cfg },
					cmd.ProvideLogger,
				),
				fx.Invoke(func(cfg *config.Config, log *slog.Logger, lc fx.Lifecycle) error {
					m := NewMigrator(cfg, log)
					if err := m.Run(c.Context); err != nil {
						return err
					}

					return nil
				}),
				fx.NopLogger,
			)

			return app.Start(c.Context)
		},
	}
}

func init() {
	cmd.Register(CMD())
}

type migrator struct {
	cfg *config.Config
	log *slog.Logger
}

func NewMigrator(cfg *config.Config, log *slog.Logger) *migrator {
	return &migrator{
		cfg: cfg,
		log: log,
	}
}

func (m *migrator) Run(ctx context.Context) error {
	conf, err := pgxpool.ParseConfig(m.cfg.Postgres.DSN)
	if err != nil {
		return err
	}

	db := stdlib.OpenDB(*conf.ConnConfig)
	defer db.Close()

	goose.SetLogger(newLogger(m.log))
	goose.SetVerbose(true)
	store, err := database.NewStore(database.DialectPostgres, "im_account_schema_version")
	if err != nil {
		return err
	}

	noopDialect := goose.Dialect("")
	provider, err := goose.NewProvider(noopDialect, db, migrations.EmbedMigrations, goose.WithStore(store))
	if err != nil {
		return err
	}

	res, err := provider.Up(ctx)
	if err != nil {
		return err
	}

	for _, r := range res {
		if r.Error != nil {
			m.log.Error("unable to apply migration", "err", r.Error)
		} else {
			m.log.Info("applied migration")
		}
	}

	return nil
}

type migrateLogger struct {
	log *slog.Logger
}

func newLogger(log *slog.Logger) *migrateLogger {
	return &migrateLogger{log: log}
}

func (l *migrateLogger) Printf(format string, args ...any) {
	l.log.Info(fmt.Sprintf(format, args...))
}

func (l *migrateLogger) Fatalf(format string, args ...any) {
	l.log.Error(fmt.Sprintf(format, args...))
}
