package main

import (
	"os"

	"github.com/Yuwenfeng2019/K2S/pkg/cli/agent"
	"github.com/Yuwenfeng2019/K2S/pkg/cli/cmds"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func main() {
	app := cmds.NewApp()
	app.Commands = []cli.Command{
		cmds.NewAgentCommand(agent.Run),
	}

	err := app.Run(os.Args)
	if err != nil {
		logrus.Fatal(err)
	}
}
