package config

import (
	"crypto/tls"
	"net"
	"net/http"
	"strings"

	"k8s.io/apiserver/pkg/authentication/authenticator"
)

type Node struct {
	Docker                   bool
	ContainerRuntimeEndpoint string
	NoFlannel                bool
	FlannelConf              string
	LocalAddress             string
	Containerd               Containerd
	Images                   string
	AgentConfig              Agent
	CACerts                  []byte
	ServerAddress            string
	Certificate              *tls.Certificate
}

type Containerd struct {
	Address string
	Log     string
	Root    string
	State   string
	Config  string
	Opt     string
}

type Agent struct {
	NodeName           string
	ClusterCIDR        net.IPNet
	ClusterDNS         net.IP
	RootDir            string
	KubeConfig         string
	NodeIP             string
	RuntimeSocket      string
	ListenAddress      string
	CACertPath         string
	CNIBinDir          string
	CNIConfDir         string
	ExtraKubeletArgs   []string
	ExtraKubeProxyArgs []string
}

type Control struct {
	AdvertisePort         int
	ListenPort            int
	ClusterSecret         string
	ClusterIPRange        *net.IPNet
	ServiceIPRange        *net.IPNet
	ClusterDNS            net.IP
	NoCoreDNS             bool
	KubeConfigOutput      string
	KubeConfigMode        string
	DataDir               string
	Skips                 []string
	ETCDEndpoints         []string
	ETCDKeyFile           string
	ETCDCertFile          string
	ETCDCAFile            string
	NoScheduler           bool
	ExtraAPIArgs          []string
	ExtraControllerArgs   []string
	ExtraSchedulerAPIArgs []string
	NoLeaderElect         bool

	Runtime *ControlRuntime `json:"-"`
}

type ControlRuntime struct {
	TLSCert          string
	TLSKey           string
	TLSCA            string
	TLSCAKey         string
	TokenCA          string
	TokenCAKey       string
	ServiceKey       string
	PasswdFile       string
	KubeConfigSystem string

	NodeCert      string
	NodeKey       string
	ClientToken   string
	NodeToken     string
	Handler       http.Handler
	Tunnel        http.Handler
	Authenticator authenticator.Request
}

type ArgString []string

func (a ArgString) String() string {
	b := strings.Builder{}
	for _, s := range a {
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		b.WriteString(s)
	}
	return b.String()
}
