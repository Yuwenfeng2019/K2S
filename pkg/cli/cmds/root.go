package cmds

import (
	"fmt"
	"os"

	"github.com/Yuwenfeng2019/K2S/pkg/version"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	debug bool
)

func init() {
	// hack - force "file,dns" lookup order if go dns is used
	if os.Getenv("RES_OPTIONS") == "" {
		os.Setenv("RES_OPTIONS", " ")
	}
}

func NewApp() *cli.App {
	app := cli.NewApp()
	app.Name = appName
	app.Usage = "Kubernetes, but small and simple"
	app.Version = fmt.Sprintf("%s (%s)", version.Version, version.GitCommit)
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Printf("%s version %s\n", app.Name, app.Version)
	}
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:        "debug",
			Usage:       "Turn on debug logs",
			Destination: &debug,
			EnvVar:      "K3S_DEBUG",
		},
	}

	app.Before = func(ctx *cli.Context) error {
		if debug {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return nil
	}

	return app
}
