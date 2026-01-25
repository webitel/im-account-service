package server

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"
	"github.com/webitel/im-account-service/cmd"
	"github.com/webitel/im-account-service/config"
)

func CMD() *cli.Command {
	return &cli.Command{
		Name:  "server",
		Usage: "Run the gRPC server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config_file",
				Aliases: []string{"c"},
				Usage:   "Configuration file path",
			},
		},
		Action: func(c *cli.Context) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}
			app := NewApp(cfg)

			if err := app.Start(c.Context); err != nil {
				return err
			}

			stop := make(chan os.Signal, 1)
			signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
			<-stop

			slog.Info("Shutting down...")

			return app.Stop(context.Background())
		},
	}
}

func init() {
	cmd.Register(CMD())
}
