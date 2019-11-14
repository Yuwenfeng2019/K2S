package agent

import (
	"context"
	"fmt"
	"os"

	"github.com/Yuwenfeng2019/K2S/pkg/agent"
	"github.com/Yuwenfeng2019/K2S/pkg/cli/cmds"
	"github.com/Yuwenfeng2019/K2S/pkg/datadir"
	"github.com/Yuwenfeng2019/K2S/pkg/netutil"
	"github.com/Yuwenfeng2019/K2S/pkg/token"
	"github.com/rancher/wrangler/pkg/signals"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func Run(ctx *cli.Context) error {
	if err := cmds.InitLogging(); err != nil {
		return err
	}
	if os.Getuid() != 0 {
		return fmt.Errorf("agent must be ran as root")
	}

	if cmds.AgentConfig.TokenFile != "" {
		token, err := token.ReadFile(cmds.AgentConfig.TokenFile)
		if err != nil {
			return err
		}
		cmds.AgentConfig.Token = token
	}

	if cmds.AgentConfig.Token == "" && cmds.AgentConfig.ClusterSecret != "" {
		cmds.AgentConfig.Token = cmds.AgentConfig.ClusterSecret
	}

	if cmds.AgentConfig.Token == "" {
		return fmt.Errorf("--token is required")
	}

	if cmds.AgentConfig.ServerURL == "" {
		return fmt.Errorf("--server is required")
	}

	if cmds.AgentConfig.FlannelIface != "" && cmds.AgentConfig.NodeIP == "" {
		cmds.AgentConfig.NodeIP = netutil.GetIPFromInterface(cmds.AgentConfig.FlannelIface)
	}

	logrus.Infof("Starting k2s agent %s", ctx.App.Version)

	dataDir, err := datadir.LocalHome(cmds.AgentConfig.DataDir, cmds.AgentConfig.Rootless)
	if err != nil {
		return err
	}

	cfg := cmds.AgentConfig
	cfg.Debug = ctx.GlobalBool("debug")
	cfg.DataDir = dataDir

	contextCtx := signals.SetupSignalHandler(context.Background())

	return agent.Run(contextCtx, cfg)
}
