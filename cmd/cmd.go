package cmd

import (
	"os"
	"time"

	"github.com/urfave/cli/v2"
)

const (
	ServiceName      = "im-account-service"
	ServiceNamespace = "webitel"
)

var (
	version        = "0.0.0"
	commit         = "hash"
	commitDate     = time.Now().String()
	branch         = "branch"
	buildTimestamp = ""
)

func Run() error {

	app := &cli.App{
		Name:  ServiceName,
		Usage: "Microservice for Webitel platform",
		Flags: nil, // []cli.Flag{}
		// Commands: []*cli.Command{
		// 	// serverCmd(),
		// 	server.CMD(),
		// 	migrate.CMD(),
		// },
		Commands: commands,
	}

	return app.Run(os.Args)
}

var commands []*cli.Command

func Register(cmds ...*cli.Command) {
	// i := slices.ContainsFunc(
	// 	commands, func(*cli.Command) bool {
	// 		return false
	// 	},
	// )
	commands = append(commands, cmds...)
}
