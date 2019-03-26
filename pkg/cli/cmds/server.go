package cmds

import (
	"github.com/urfave/cli"
)

type Server struct {
	Log              string
	ClusterCIDR      string
	ClusterSecret    string
	ServiceCIDR      string
	ClusterDNS       string
	HTTPSPort        int
	HTTPPort         int
	DataDir          string
	DisableAgent     bool
	KubeConfigOutput string
	KubeConfigMode   string
	KnownIPs cli.StringSlice
}

var ServerConfig Server

func NewServerCommand(action func(*cli.Context) error) cli.Command {
	return cli.Command{
		Name:      "server",
		Usage:     "Run management server",
		UsageText: appName + " server [OPTIONS]",
		Action:    action,
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:        "https-listen-port",
				Usage:       "HTTPS listen port",
				Value:       6443,
				Destination: &ServerConfig.HTTPSPort,
			},
			cli.IntFlag{
				Name:        "http-listen-port",
				Usage:       "HTTP listen port (for /healthz, HTTPS redirect, and port for TLS terminating LB)",
				Value:       0,
				Destination: &ServerConfig.HTTPPort,
			},
			cli.StringFlag{
				Name:        "data-dir,d",
				Usage:       "Folder to hold state default /var/lib/k2s or ${HOME}/k2s if not root",
				Destination: &ServerConfig.DataDir,
			},
			cli.BoolFlag{
				Name:        "disable-agent",
				Usage:       "Do not run a local agent and register a local kubelet",
				Destination: &ServerConfig.DisableAgent,
			},
			cli.StringFlag{
				Name:        "log,l",
				Usage:       "Log to file",
				Destination: &ServerConfig.Log,
			},
			cli.StringFlag{
				Name:        "cluster-cidr",
				Usage:       "Network CIDR to use for pod IPs",
				Destination: &ServerConfig.ClusterCIDR,
				Value:       "10.42.0.0/16",
			},
			cli.StringFlag{
				Name:        "cluster-secret",
				Usage:       "Shared secret used to bootstrap a cluster",
				Destination: &ServerConfig.ClusterSecret,
				EnvVar:      "K2S_CLUSTER_SECRET",
			},
			cli.StringFlag{
				Name:        "service-cidr",
				Usage:       "Network CIDR to use for services IPs",
				Destination: &ServerConfig.ServiceCIDR,
				Value:       "10.43.0.0/16",
			},
			cli.StringFlag{
				Name:        "cluster-dns",
				Usage:       "Cluster IP for coredns service. Should be in your service-cidr range",
				Destination: &ServerConfig.ClusterDNS,
				Value:       "",
			},
			cli.StringSliceFlag{
				Name:  "no-deploy",
				Usage: "Do not deploy packaged components (valid items: coredns, servicelb, traefik)",
			},
			cli.StringFlag{
				Name:        "write-kubeconfig,o",
				Usage:       "Write kubeconfig for admin client to this file",
				Destination: &ServerConfig.KubeConfigOutput,
				EnvVar:      "K2S_KUBECONFIG_OUTPUT",
			},
			cli.StringFlag{
				Name:        "write-kubeconfig-mode",
				Usage:       "Write kubeconfig with this mode",
				Destination: &ServerConfig.KubeConfigMode,
				EnvVar:      "K3S_KUBECONFIG_MODE",
			},
			cli.StringSliceFlag{
				Name:        "tls-san",
				Usage:       "Add additional hostname or IP as a Subject Alternative Name in the TLS cert",
				Value: &ServerConfig.KnownIPs,
			},
			NodeIPFlag,
			NodeNameFlag,
			DockerFlag,
			FlannelFlag,
			CRIEndpointFlag,
		},
	}
}
