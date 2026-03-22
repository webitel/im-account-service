package cmd

import (
	"os"

	"github.com/urfave/cli/v2"
	"github.com/webitel/im-account-service/internal/service"
)

const (
	ServiceName      = service.ServiceName
	ServiceNamespace = "webitel"
)

func Run() error {

	app := &cli.App{
		Name:  ServiceName,
		Usage: "Webitel IM Account microservice",
		Flags: nil, // []cli.Flag{}
		// Commands: []*cli.Command{
		// 	// serverCmd(),
		// 	server.CMD(),
		// 	migrate.CMD(),
		// },
		Version: Version(),
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
