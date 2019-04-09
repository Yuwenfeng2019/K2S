package cmds

import (
	"os"
	"path/filepath"

	"github.com/urfave/cli"
)

type Agent struct {
	Token                    string
	TokenFile                string
	ServerURL                string
	ResolvConf               string
	DataDir                  string
	NodeIP                   string
	NodeName                 string
	ClusterSecret            string
	Docker                   bool
	ContainerRuntimeEndpoint string
	NoFlannel                bool
	Debug                    bool
	Rootless                 bool
	AgentShared
	ExtraKubeletArgs   cli.StringSlice
	ExtraKubeProxyArgs cli.StringSlice
}

type AgentShared struct {
	NodeIP string
}

var (
	appName     = filepath.Base(os.Args[0])
	AgentConfig Agent
	NodeIPFlag  = cli.StringFlag{
		Name:        "node-ip,i",
		Usage:       "(agent) IP address to advertise for node",
		Destination: &AgentConfig.NodeIP,
	}
	NodeNameFlag = cli.StringFlag{
		Name:        "node-name",
		Usage:       "(agent) Node name",
		EnvVar:      "K2S_NODE_NAME",
		Destination: &AgentConfig.NodeName,
	}
	DockerFlag = cli.BoolFlag{
		Name:        "docker",
		Usage:       "(agent) Use docker instead of containerd",
		Destination: &AgentConfig.Docker,
	}
	FlannelFlag = cli.BoolFlag{
		Name:        "no-flannel",
		Usage:       "(agent) Disable embedded flannel",
		Destination: &AgentConfig.NoFlannel,
	}
	CRIEndpointFlag = cli.StringFlag{
		Name:        "container-runtime-endpoint",
		Usage:       "(agent) Disable embedded containerd and use alternative CRI implementation",
		Destination: &AgentConfig.ContainerRuntimeEndpoint,
	}
	ResolvConfFlag = cli.StringFlag{
		Name:        "resolv-conf",
		Usage:       "Kubelet resolv.conf file",
		EnvVar:      "K3S_RESOLV_CONF",
		Destination: &AgentConfig.ResolvConf,
	}
	ExtraKubeletArgs = cli.StringSliceFlag{
		Name:  "kubelet-arg",
		Usage: "(agent) Customized flag for kubelet process",
		Value: &AgentConfig.ExtraKubeletArgs,
	}
	ExtraKubeProxyArgs = cli.StringSliceFlag{
		Name:  "kube-proxy-arg",
		Usage: "(agent) Customized flag for kube-proxy process",
		Value: &AgentConfig.ExtraKubeProxyArgs,
	}
)

func NewAgentCommand(action func(ctx *cli.Context) error) cli.Command {
	return cli.Command{
		Name:      "agent",
		Usage:     "Run node agent",
		UsageText: appName + " agent [OPTIONS]",
		Action:    action,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:        "token,t",
				Usage:       "Token to use for authentication",
				EnvVar:      "K2S_TOKEN",
				Destination: &AgentConfig.Token,
			},
			cli.StringFlag{
				Name:        "token-file",
				Usage:       "Token file to use for authentication",
				EnvVar:      "K2S_TOKEN_FILE",
				Destination: &AgentConfig.TokenFile,
			},
			cli.StringFlag{
				Name:        "server,s",
				Usage:       "Server to connect to",
				EnvVar:      "K2S_URL",
				Destination: &AgentConfig.ServerURL,
			},
			cli.StringFlag{
				Name:        "data-dir,d",
				Usage:       "Folder to hold state",
				Destination: &AgentConfig.DataDir,
				Value:       "/var/lib/k2s",
			},
			cli.StringFlag{
				Name:        "cluster-secret",
				Usage:       "Shared secret used to bootstrap a cluster",
				Destination: &AgentConfig.ClusterSecret,
				EnvVar:      "K2S_CLUSTER_SECRET",
			},
			cli.BoolFlag{
				Name:        "rootless",
				Usage:       "(experimental) Run rootless",
				Destination: &AgentConfig.Rootless,
			},
			cli.BoolFlag{
				Name:        "rootless",
				Usage:       "(experimental) Run rootless",
				Destination: &AgentConfig.Rootless,
			},
			DockerFlag,
			FlannelFlag,
			NodeNameFlag,
			NodeIPFlag,
			CRIEndpointFlag,
			ResolvConfFlag,
			ExtraKubeletArgs,
			ExtraKubeProxyArgs,
		},
	}
}
