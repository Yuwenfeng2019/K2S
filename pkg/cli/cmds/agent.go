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
	PauseImage               string
	Docker                   bool
	ContainerRuntimeEndpoint string
	NoFlannel                bool
	FlannelIface             string
	FlannelConf              string
	Debug                    bool
	Rootless                 bool
	RootlessAlreadyUnshared  bool
	WithNodeID               bool
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
		Usage:       "(agent/networking) IP address to advertise for node",
		Destination: &AgentConfig.NodeIP,
	}
	NodeExternalIPFlag = cli.StringFlag{
		Name:        "node-external-ip",
		Usage:       "(agent/networking) External IP address to advertise for node",
		Destination: &AgentConfig.NodeExternalIP,
	}
	NodeNameFlag = cli.StringFlag{
		Name:        "node-name",
		Usage:       "(agent/node) Node name",
		EnvVar:      "K2S_NODE_NAME",
		Destination: &AgentConfig.NodeName,
	}
	WithNodeIDFlag = cli.BoolFlag{
		Name:        "with-node-id",
		Usage:       "(agent/node) Append id to node name",
		Destination: &AgentConfig.WithNodeID,
	}
	DockerFlag = cli.BoolFlag{
		Name:        "docker",
		Usage:       "(agent/runtime) Use docker instead of containerd",
		Destination: &AgentConfig.Docker,
	}
	CRIEndpointFlag = cli.StringFlag{
		Name:        "container-runtime-endpoint",
		Usage:       "(agent/runtime) Disable embedded containerd and use alternative CRI implementation",
		Destination: &AgentConfig.ContainerRuntimeEndpoint,
	}
	PrivateRegistryFlag = cli.StringFlag{
		Name:        "private-registry",
		Usage:       "(agent/runtime) Private registry configuration file",
		Destination: &AgentConfig.PrivateRegistry,
		Value:       "/etc/yuwenfeng/k2s/registries.yaml",
	}
	PauseImageFlag = cli.StringFlag{
		Name:        "pause-image",
		Usage:       "(agent/runtime) Customized pause image for containerd sandbox",
		Destination: &AgentConfig.PauseImage,
	}
	FlannelFlag = cli.BoolFlag{
		Name:        "no-flannel",
		Usage:       "(deprecated) use --flannel-backend=none",
		Destination: &AgentConfig.NoFlannel,
	}
	FlannelIfaceFlag = cli.StringFlag{
		Name:        "flannel-iface",
		Usage:       "(agent/networking) Override default flannel interface",
		Destination: &AgentConfig.FlannelIface,
	}
	FlannelConfFlag = cli.StringFlag{
		Name:        "flannel-conf",
		Usage:       "(agent/networking) Override default flannel config file",
		Destination: &AgentConfig.FlannelConf,
	}
	ResolvConfFlag = cli.StringFlag{
		Name:        "resolv-conf",
		Usage:       "(agent/networking) Kubelet resolv.conf file",
		EnvVar:      "K2S_RESOLV_CONF",
		Destination: &AgentConfig.ResolvConf,
	}
	ExtraKubeletArgs = cli.StringSliceFlag{
		Name:  "kubelet-arg",
		Usage: "(agent/flags) Customized flag for kubelet process",
		Value: &AgentConfig.ExtraKubeletArgs,
	}
	ExtraKubeProxyArgs = cli.StringSliceFlag{
		Name:  "kube-proxy-arg",
		Usage: "(agent/flags) Customized flag for kube-proxy process",
		Value: &AgentConfig.ExtraKubeProxyArgs,
	}
	NodeTaints = cli.StringSliceFlag{
		Name:  "node-taint",
		Usage: "(agent/node) Registering kubelet with set of taints",
		Value: &AgentConfig.Taints,
	}
	NodeLabels = cli.StringSliceFlag{
		Name:  "node-label",
		Usage: "(agent/node) Registering kubelet with set of labels",
		Value: &AgentConfig.Labels,
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
				Usage:       "(cluster) Token to use for authentication",
				EnvVar:      "K2S_TOKEN",
				Destination: &AgentConfig.Token,
			},
			cli.StringFlag{
				Name:        "token-file",
				Usage:       "(cluster) Token file to use for authentication",
				EnvVar:      "K2S_TOKEN_FILE",
				Destination: &AgentConfig.TokenFile,
			},
			cli.StringFlag{
				Name:        "server,s",
				Usage:       "(cluster) Server to connect to",
				EnvVar:      "K2S_URL",
				Destination: &AgentConfig.ServerURL,
			},
			cli.StringFlag{
				Name:        "data-dir,d",
				Usage:       "(agent/data) Folder to hold state",
				Destination: &AgentConfig.DataDir,
				Value:       "/var/lib/k2s",
			},
			NodeNameFlag,
			NodeLabels,
			NodeTaints,
			DockerFlag,
			CRIEndpointFlag,
			PauseImageFlag,
			PrivateRegistryFlag,
			NodeIPFlag,
			NodeExternalIPFlag,
			ResolvConfFlag,
			FlannelIfaceFlag,
			FlannelConfFlag,
			ExtraKubeletArgs,
			ExtraKubeProxyArgs,
			cli.BoolFlag{
				Name:        "rootless",
				Usage:       "(experimental) Run rootless",
				Destination: &AgentConfig.Rootless,
			},

			// Deprecated/hidden below

			FlannelFlag,
			cli.StringFlag{
				Name:        "cluster-secret",
				Usage:       "(deprecated) use --token",
				Destination: &AgentConfig.Token,
				EnvVar:      "K2S_CLUSTER_SECRET",
			},
		},
	}
}
