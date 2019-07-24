package server

import (
	"context"
	"flag"
	"fmt"
	net2 "net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rancher/k3s/pkg/netutil"

	systemd "github.com/coreos/go-systemd/daemon"
	"github.com/docker/docker/pkg/reexec"
	"github.com/natefinch/lumberjack"
	"github.com/pkg/errors"
	"github.com/Yuwenfeng2019/K2S/pkg/agent"
	"github.com/Yuwenfeng2019/K2S/pkg/cli/cmds"
	"github.com/Yuwenfeng2019/K2S/pkg/datadir"
	"github.com/Yuwenfeng2019/K2S/pkg/rootless"
	"github.com/Yuwenfeng2019/K2S/pkg/server"
	"github.com/rancher/wrangler/pkg/signals"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"k8s.io/apimachinery/pkg/util/net"
	"k8s.io/kubernetes/pkg/master"
	"k8s.io/kubernetes/pkg/volume/csi"

	_ "github.com/go-sql-driver/mysql" // ensure we have mysql
	_ "github.com/lib/pq"              // ensure we have postgres
	_ "github.com/mattn/go-sqlite3"    // ensure we have sqlite
)

func setupLogging(app *cli.Context) {
	if !app.GlobalBool("debug") {
		flag.Set("stderrthreshold", "WARNING")
		flag.Set("alsologtostderr", "false")
		flag.Set("logtostderr", "false")
	}
}

func runWithLogging(app *cli.Context, cfg *cmds.Server) error {
	l := &lumberjack.Logger{
		Filename:   cfg.Log,
		MaxSize:    50,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}

	args := append([]string{"k2s"}, os.Args[1:]...)
	cmd := reexec.Command(args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "_RIO_REEXEC_=true")
	cmd.Stderr = l
	cmd.Stdout = l
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func Run(app *cli.Context) error {
	return run(app, &cmds.ServerConfig)
}

func run(app *cli.Context, cfg *cmds.Server) error {
	var (
		err error
	)

	if cfg.Log != "" && os.Getenv("_RIO_REEXEC_") == "" {
		return runWithLogging(app, cfg)
	}

	if err := checkUnixTimestamp(); err != nil {
		return err
	}

	setupLogging(app)

	if !cfg.DisableAgent && os.Getuid() != 0 && !cfg.Rootless {
		return fmt.Errorf("must run as root unless --disable-agent is specified")
	}

	if cfg.Rootless {
		dataDir, err := datadir.LocalHome(cfg.DataDir, true)
		if err != nil {
			return err
		}
		cfg.DataDir = dataDir
		if err := rootless.Rootless(dataDir); err != nil {
			return err
		}
	}

	// If running agent in server, set this so that CSI initializes properly
	csi.WaitForValidHostName = !cfg.DisableAgent

	serverConfig := server.Config{}
	serverConfig.ControlConfig.ClusterSecret = cfg.ClusterSecret
	serverConfig.ControlConfig.DataDir = cfg.DataDir
	serverConfig.ControlConfig.KubeConfigOutput = cfg.KubeConfigOutput
	serverConfig.ControlConfig.KubeConfigMode = cfg.KubeConfigMode
	serverConfig.ControlConfig.NoScheduler = cfg.DisableScheduler
	serverConfig.Rootless = cfg.Rootless
	serverConfig.TLSConfig.HTTPSPort = cfg.HTTPSPort
	serverConfig.TLSConfig.HTTPPort = cfg.HTTPPort
	for _, san := range knownIPs(cfg.TLSSan) {
		addr := net2.ParseIP(san)
		if addr != nil {
			serverConfig.TLSConfig.KnownIPs = append(serverConfig.TLSConfig.KnownIPs, san)
		} else {
			serverConfig.TLSConfig.Domains = append(serverConfig.TLSConfig.Domains, san)
		}
	}
	serverConfig.TLSConfig.BindAddress = cfg.BindAddress
	serverConfig.ControlConfig.HTTPSPort = cfg.HTTPSPort
	serverConfig.ControlConfig.ExtraAPIArgs = cfg.ExtraAPIArgs
	serverConfig.ControlConfig.ExtraControllerArgs = cfg.ExtraControllerArgs
	serverConfig.ControlConfig.ExtraSchedulerAPIArgs = cfg.ExtraSchedulerArgs
	serverConfig.ControlConfig.ClusterDomain = cfg.ClusterDomain
	serverConfig.ControlConfig.StorageEndpoint = cfg.StorageEndpoint
	serverConfig.ControlConfig.StorageBackend = cfg.StorageBackend
	serverConfig.ControlConfig.StorageCAFile = cfg.StorageCAFile
	serverConfig.ControlConfig.StorageCertFile = cfg.StorageCertFile
	serverConfig.ControlConfig.StorageKeyFile = cfg.StorageKeyFile
	serverConfig.ControlConfig.AdvertiseIP = cfg.AdvertiseIP
	serverConfig.ControlConfig.AdvertisePort = cfg.AdvertisePort
	serverConfig.ControlConfig.BootstrapType = cfg.BootstrapType

	if cmds.AgentConfig.FlannelIface != "" && cmds.AgentConfig.NodeIP == "" {
		cmds.AgentConfig.NodeIP = netutil.GetIPFromInterface(cmds.AgentConfig.FlannelIface)
	}

	if serverConfig.ControlConfig.AdvertiseIP == "" && cmds.AgentConfig.NodeIP != "" {
		serverConfig.ControlConfig.AdvertiseIP = cmds.AgentConfig.NodeIP
	}
	if serverConfig.ControlConfig.AdvertiseIP != "" {
		serverConfig.TLSConfig.KnownIPs = append(serverConfig.TLSConfig.KnownIPs, serverConfig.ControlConfig.AdvertiseIP)
	}

	_, serverConfig.ControlConfig.ClusterIPRange, err = net2.ParseCIDR(cfg.ClusterCIDR)
	if err != nil {
		return errors.Wrapf(err, "Invalid CIDR %s: %v", cfg.ClusterCIDR, err)
	}
	_, serverConfig.ControlConfig.ServiceIPRange, err = net2.ParseCIDR(cfg.ServiceCIDR)
	if err != nil {
		return errors.Wrapf(err, "Invalid CIDR %s: %v", cfg.ServiceCIDR, err)
	}

	_, apiServerServiceIP, err := master.DefaultServiceIPRange(*serverConfig.ControlConfig.ServiceIPRange)
	if err != nil {
		return err
	}
	serverConfig.TLSConfig.KnownIPs = append(serverConfig.TLSConfig.KnownIPs, apiServerServiceIP.String())

	// If cluster-dns CLI arg is not set, we set ClusterDNS address to be ServiceCIDR network + 10,
	// i.e. when you set service-cidr to 192.168.0.0/16 and don't provide cluster-dns, it will be set to 192.168.0.10
	if cfg.ClusterDNS == "" {
		serverConfig.ControlConfig.ClusterDNS = make(net2.IP, 4)
		copy(serverConfig.ControlConfig.ClusterDNS, serverConfig.ControlConfig.ServiceIPRange.IP.To4())
		serverConfig.ControlConfig.ClusterDNS[3] = 10
	} else {
		serverConfig.ControlConfig.ClusterDNS = net2.ParseIP(cfg.ClusterDNS)
	}

	if serverConfig.ControlConfig.StorageBackend != "etcd3" {
		serverConfig.ControlConfig.NoLeaderElect = true
	}

	for _, noDeploy := range app.StringSlice("no-deploy") {
		if noDeploy == "servicelb" {
			serverConfig.DisableServiceLB = true
			continue
		}

		if !strings.HasSuffix(noDeploy, ".yaml") {
			noDeploy = noDeploy + ".yaml"
		}
		serverConfig.ControlConfig.Skips = append(serverConfig.ControlConfig.Skips, noDeploy)
	}

	logrus.Info("Starting k2s ", app.App.Version)
	notifySocket := os.Getenv("NOTIFY_SOCKET")
	os.Unsetenv("NOTIFY_SOCKET")

	ctx := signals.SetupSignalHandler(context.Background())
	certs, err := server.StartServer(ctx, &serverConfig)
	if err != nil {
		return err
	}

	logrus.Info("k2s is up and running")
	if notifySocket != "" {
		os.Setenv("NOTIFY_SOCKET", notifySocket)
		systemd.SdNotify(true, "READY=1\n")
	}

	if cfg.DisableAgent {
		<-ctx.Done()
		return nil
	}
	ip := serverConfig.TLSConfig.BindAddress
	if ip == "" {
		ip = "localhost"
	}
	url := fmt.Sprintf("https://%s:%d", ip, serverConfig.TLSConfig.HTTPSPort)
	token := server.FormatToken(serverConfig.ControlConfig.Runtime.NodeToken, certs)

	agentConfig := cmds.AgentConfig
	agentConfig.Debug = app.GlobalBool("bool")
	agentConfig.DataDir = filepath.Dir(serverConfig.ControlConfig.DataDir)
	agentConfig.ServerURL = url
	agentConfig.Token = token
	agentConfig.Labels = append(agentConfig.Labels, "node-role.kubernetes.io/master=true")

	return agent.Run(ctx, agentConfig)
}

func knownIPs(ips []string) []string {
	ips = append(ips, "127.0.0.1")
	ip, err := net.ChooseHostInterface()
	if err == nil {
		ips = append(ips, ip.String())
	}
	return ips
}

func checkUnixTimestamp() error {
	timeNow := time.Now()
	// check if time before 01/01/1980
	if timeNow.Before(time.Unix(315532800, 0)) {
		return fmt.Errorf("server time isn't set properly: %v", timeNow)
	}
	return nil
}
