package cmds

import (
	"fmt"

	"github.com/rancher/k3s/pkg/daemons/config"
	"github.com/urfave/cli"
)

type Server struct {
	ClusterCIDR         string
	ClusterSecret       string
	ServiceCIDR         string
	ClusterDNS          string
	ClusterDomain       string
	HTTPSPort           int
	HTTPPort            int
	DataDir             string
	DisableAgent        bool
	KubeConfigOutput    string
	KubeConfigMode      string
	TLSSan              cli.StringSlice
	BindAddress         string
	ExtraAPIArgs        cli.StringSlice
	ExtraSchedulerArgs  cli.StringSlice
	ExtraControllerArgs cli.StringSlice
	Rootless            bool
	StoreBootstrap      bool
	StorageEndpoint     string
	StorageCAFile       string
	StorageCertFile     string
	StorageKeyFile      string
	AdvertiseIP         string
	AdvertisePort       int
	DisableScheduler    bool
	FlannelBackend      string
}

var ServerConfig Server

func NewServerCommand(action func(*cli.Context) error) cli.Command {
	return cli.Command{
		Name:      "server",
		Usage:     "Run management server",
		UsageText: appName + " server [OPTIONS]",
		Action:    action,
		Flags: []cli.Flag{
			VLevel,
			VModule,
			LogFile,
			AlsoLogToStderr,
			cli.StringFlag{
				Name:        "bind-address",
				Usage:       "k3s bind address (default: localhost)",
				Destination: &ServerConfig.BindAddress,
			},
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
			cli.StringFlag{
				Name:        "cluster-domain",
				Usage:       "Cluster Domain",
				Destination: &ServerConfig.ClusterDomain,
				Value:       "cluster.local",
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
				Name:  "tls-san",
				Usage: "Add additional hostname or IP as a Subject Alternative Name in the TLS cert",
				Value: &ServerConfig.TLSSan,
			},
			cli.StringSliceFlag{
				Name:  "kube-apiserver-arg",
				Usage: "Customized flag for kube-apiserver process",
				Value: &ServerConfig.ExtraAPIArgs,
			},
			cli.StringSliceFlag{
				Name:  "kube-scheduler-arg",
				Usage: "Customized flag for kube-scheduler process",
				Value: &ServerConfig.ExtraSchedulerArgs,
			},
			cli.StringSliceFlag{
				Name:  "kube-controller-arg",
				Usage: "Customized flag for kube-controller-manager process",
				Value: &ServerConfig.ExtraControllerArgs,
			},
			cli.BoolFlag{
				Name:        "rootless",
				Usage:       "(experimental) Run rootless",
				Destination: &ServerConfig.Rootless,
			},
			cli.BoolFlag{
				Name:        "bootstrap-save",
				Usage:       "(experimental) Save bootstrap information in the storage endpoint",
				Hidden:      true,
				Destination: &ServerConfig.StoreBootstrap,
			},
			cli.StringFlag{
				Name:        "storage-endpoint",
				Usage:       "Specify etcd, Mysql, Postgres, or Sqlite (default) data source name",
				Destination: &ServerConfig.StorageEndpoint,
				EnvVar:      "K3S_STORAGE_ENDPOINT",
			},
			cli.StringFlag{
				Name:        "storage-cafile",
				Usage:       "SSL Certificate Authority file used to secure storage backend communication",
				Destination: &ServerConfig.StorageCAFile,
				EnvVar:      "K3S_STORAGE_CAFILE",
			},
			cli.StringFlag{
				Name:        "storage-certfile",
				Usage:       "SSL certification file used to secure storage backend communication",
				Destination: &ServerConfig.StorageCertFile,
				EnvVar:      "K3S_STORAGE_CERTFILE",
			},
			cli.StringFlag{
				Name:        "storage-keyfile",
				Usage:       "SSL key file used to secure storage backend communication",
				Destination: &ServerConfig.StorageKeyFile,
				EnvVar:      "K3S_STORAGE_KEYFILE",
			},
			cli.StringFlag{
				Name:        "advertise-address",
				Usage:       "IP address that apiserver uses to advertise to members of the cluster",
				Destination: &ServerConfig.AdvertiseIP,
			},
			cli.IntFlag{
				Name:        "advertise-port",
				Usage:       "Port that apiserver uses to advertise to members of the cluster",
				Value:       0,
				Destination: &ServerConfig.AdvertisePort,
			},
			cli.BoolFlag{
				Name:        "disable-scheduler",
				Usage:       "Disable Kubernetes default scheduler",
				Destination: &ServerConfig.DisableScheduler,
			},
			cli.StringFlag{
				Name:        "flannel-backend",
				Usage:       fmt.Sprintf("(experimental) One of '%s', '%s', '%s', or '%s'", config.FlannelBackendNone, config.FlannelBackendVXLAN, config.FlannelBackendIPSEC, config.FlannelBackendWireguard),
				Destination: &ServerConfig.FlannelBackend,
				Value:       config.FlannelBackendVXLAN,
			},
			NodeIPFlag,
			NodeNameFlag,
			DockerFlag,
			FlannelFlag,
			FlannelIfaceFlag,
			FlannelConfFlag,
			CRIEndpointFlag,
			PauseImageFlag,
			ResolvConfFlag,
			ExtraKubeletArgs,
			ExtraKubeProxyArgs,
			NodeLabels,
			NodeTaints,
		},
	}
}
