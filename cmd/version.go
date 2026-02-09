package cmd

import (
	"fmt"
	"time"

	"github.com/urfave/cli/v2"
)

var (
	version        = "0.0.0"
	commit         = "hash"
	commitDate     = time.Now().String()
	branch         = "branch"
	buildTimestamp = ""
)

func Version() string {
	return branch
}

func FullVersion() string {
	return fmt.Sprintf("%s-%s-%s",
		branch, buildTimestamp, commit,
	)
}

func init() {

	// defaults

	switch branch {
	case "branch", "":
		{
			branch = "dev" // development
		}
	}

	switch commit {
	case "hash", "":
		{
			commit = "000000000000"
		}
	}

	if buildTimestamp == "" {
		buildTimestamp = time.Now().UTC().Format("20060102150405")
	}

	// version

	Register(&cli.Command{
		Name:  "version",
		Usage: "Print build version & exit",
		// Flags: []cli.Flag{
		// 	&cli.BoolFlag{
		// 		Name:  "short",
		// 		Usage: "",
		// 		Value: false,
		// 	},
		// 	&cli.BoolFlag{
		// 		Name:  "build",
		// 		Usage: "Build date",
		// 		Value: false,
		// 	},
		// 	&cli.BoolFlag{
		// 		Name:  "patch",
		// 		Usage: "",
		// 		Value: false,
		// 	},
		// },
		Action: func(_ *cli.Context) error {
			fmt.Println(FullVersion())
			return nil
		},
	})

}