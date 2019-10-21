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
	DisableLoadBalancer      bool
	ResolvConf               string
	DataDir                  string
	NodeIP                   string
	NodeExternalIP           string
	NodeName                 string
	ClusterSecret            string
	PauseImage               string
	Docker                   bool
	ContainerRuntimeEndpoint string
	NoFlannel                bool
	FlannelIface             string
	FlannelConf              string
	Debug                    bool
	Rootless                 bool
	AgentShared
	ExtraKubeletArgs   cli.StringSlice
	ExtraKubeProxyArgs cli.StringSlice
	Labels             cli.StringSlice
	Taints             cli.StringSlice
	PrivateRegistry    string
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
	NodeExternalIPFlag = cli.StringFlag{
		Name:        "node-external-ip",
		Usage:       "(agent) External IP address to advertise for node",
		Destination: &AgentConfig.NodeExternalIP,
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
	FlannelIfaceFlag = cli.StringFlag{
		Name:        "flannel-iface",
		Usage:       "(agent) Override default flannel interface",
		Destination: &AgentConfig.FlannelIface,
	}
	FlannelConfFlag = cli.StringFlag{
		Name:        "flannel-conf",
		Usage:       "(agent) (experimental) Override default flannel config file",
		Destination: &AgentConfig.FlannelConf,
	}
	CRIEndpointFlag = cli.StringFlag{
		Name:        "container-runtime-endpoint",
		Usage:       "(agent) Disable embedded containerd and use alternative CRI implementation",
		Destination: &AgentConfig.ContainerRuntimeEndpoint,
	}
	PauseImageFlag = cli.StringFlag{
		Name:        "pause-image",
		Usage:       "(agent) Customized pause image for containerd sandbox",
		Destination: &AgentConfig.PauseImage,
	}
	ResolvConfFlag = cli.StringFlag{
		Name:        "resolv-conf",
		Usage:       "(agent) Kubelet resolv.conf file",
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
	NodeTaints = cli.StringSliceFlag{
		Name:  "node-taint",
		Usage: "(agent) Registering kubelet with set of taints",
		Value: &AgentConfig.Taints,
	}
	NodeLabels = cli.StringSliceFlag{
		Name:  "node-label",
		Usage: "(agent) Registering kubelet with set of labels",
		Value: &AgentConfig.Labels,
	}
	PrivateRegistryFlag = cli.StringFlag{
		Name:        "private-registry",
		Usage:       "(agent) Private registry configuration file",
		Destination: &AgentConfig.PrivateRegistry,
		Value:       "/etc/rancher/k3s/registries.yaml",
	}
)

func NewAgentCommand(action func(ctx *cli.Context) error) cli.Command {
	return cli.Command{
		Name:      "agent",
		Usage:     "Run node agent",
		UsageText: appName + " agent [OPTIONS]",
		Action:    action,
		Flags: []cli.Flag{
			VLevel,
			VModule,
			LogFile,
			AlsoLogToStderr,
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
			FlannelIfaceFlag,
			FlannelConfFlag,
			NodeNameFlag,
			NodeIPFlag,
			CRIEndpointFlag,
			PauseImageFlag,
			ResolvConfFlag,
			ExtraKubeletArgs,
			ExtraKubeProxyArgs,
			NodeLabels,
			NodeTaints,
			PrivateRegistryFlag,
			NodeExternalIPFlag,
		},
	}
}
