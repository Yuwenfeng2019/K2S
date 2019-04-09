package agent

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Yuwenfeng2019/K2S/pkg/agent/config"
	"github.com/Yuwenfeng2019/K2S/pkg/agent/containerd"
	"github.com/Yuwenfeng2019/K2S/pkg/agent/flannel"
	"github.com/Yuwenfeng2019/K2S/pkg/agent/proxy"
	"github.com/Yuwenfeng2019/K2S/pkg/agent/syssetup"
	"github.com/Yuwenfeng2019/K2S/pkg/agent/tunnel"
	"github.com/Yuwenfeng2019/K2S/pkg/cli/cmds"
	"github.com/Yuwenfeng2019/K2S/pkg/daemons/agent"
	"github.com/Yuwenfeng2019/K2S/pkg/rootless"
	"github.com/rancher/norman/pkg/clientaccess"
	"github.com/sirupsen/logrus"
)

func run(ctx context.Context, cfg cmds.Agent) error {
	nodeConfig := config.Get(ctx, cfg)

	if !nodeConfig.NoFlannel {
		if err := flannel.Prepare(ctx, nodeConfig); err != nil {
			return err
		}
	}

	if nodeConfig.Docker || nodeConfig.ContainerRuntimeEndpoint != "" {
		nodeConfig.AgentConfig.RuntimeSocket = nodeConfig.ContainerRuntimeEndpoint
	} else {
		if err := containerd.Run(ctx, nodeConfig); err != nil {
			return err
		}
	}

	if err := syssetup.Configure(); err != nil {
		return err
	}

	if err := tunnel.Setup(nodeConfig); err != nil {
		return err
	}

	if err := proxy.Run(nodeConfig); err != nil {
		return err
	}

	if err := agent.Agent(&nodeConfig.AgentConfig); err != nil {
		return err
	}

	if !nodeConfig.NoFlannel {
		if err := flannel.Run(ctx, nodeConfig); err != nil {
			return err
		}
	}

	<-ctx.Done()
	return ctx.Err()
}

func Run(ctx context.Context, cfg cmds.Agent) error {
	if err := validate(); err != nil {
		return err
	}

	if cfg.Rootless {
		if err := rootless.Rootless(cfg.DataDir); err != nil {
			return err
		}
	}

	cfg.DataDir = filepath.Join(cfg.DataDir, "agent")

	if cfg.ClusterSecret != "" {
		cfg.Token = "K10node:" + cfg.ClusterSecret
	}

	for {
		tmpFile, err := clientaccess.AgentAccessInfoToTempKubeConfig("", cfg.ServerURL, cfg.Token)
		if err != nil {
			logrus.Error(err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(2 * time.Second):
			}
			continue
		}
		os.Remove(tmpFile)
		break
	}

	os.MkdirAll(cfg.DataDir, 0700)
	return run(ctx, cfg)
}

func validate() error {
	cgroups, err := ioutil.ReadFile("/proc/self/cgroup")
	if err != nil {
		return err
	}

	if !strings.Contains(string(cgroups), "cpuset") {
		logrus.Warn("Failed to find cpuset cgroup, you may need to add \"cgroup_enable=cpuset\" to your linux cmdline (/boot/cmdline.txt on a Raspberry Pi)")
	}

	if !strings.Contains(string(cgroups), "memory") {
		msg := "ailed to find memory cgroup, you may need to add \"cgroup_memory=1 cgroup_enable=memory\" to your linux cmdline (/boot/cmdline.txt on a Raspberry Pi)"
		logrus.Error("F" + msg)
		return errors.New("f" + msg)
	}

	return nil
}
