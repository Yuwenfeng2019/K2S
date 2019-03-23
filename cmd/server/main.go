package main

import (
	"os"
	"path/filepath"

	"github.com/docker/docker/pkg/reexec"
	crictl2 "github.com/kubernetes-sigs/cri-tools/cmd/crictl"
	"github.com/Yuwenfeng2019/K2S/pkg/cli/agent"
	"github.com/Yuwenfeng2019/K2S/pkg/cli/cmds"
	"github.com/Yuwenfeng2019/K2S/pkg/cli/crictl"
	"github.com/Yuwenfeng2019/K2S/pkg/cli/kubectl"
	"github.com/Yuwenfeng2019/K2S/pkg/cli/server"
	"github.com/Yuwenfeng2019/K2S/pkg/containerd"
	kubectl2 "github.com/Yuwenfeng2019/K2S/pkg/kubectl"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func init() {
	reexec.Register("containerd", containerd.Main)
	reexec.Register("kubectl", kubectl2.Main)
	reexec.Register("crictl", crictl2.Main)
}

func main() {
	cmd := os.Args[0]
	os.Args[0] = filepath.Base(os.Args[0])
	if reexec.Init() {
		return
	}
	os.Args[0] = cmd

	app := cmds.NewApp()
	app.Commands = []cli.Command{
		cmds.NewServerCommand(server.Run),
		cmds.NewAgentCommand(agent.Run),
		cmds.NewKubectlCommand(kubectl.Run),
		cmds.NewCRICTL(crictl.Run),
	}

	err := app.Run(os.Args)
	if err != nil {
		logrus.Fatal(err)
	}
}
