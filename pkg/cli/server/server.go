package server

import (
	"context"
	"flag"
	"fmt"
	net2 "net"
	"os"
	"path/filepath"
	"strings"

	systemd "github.com/coreos/go-systemd/daemon"
	"github.com/docker/docker/pkg/reexec"
	"github.com/natefinch/lumberjack"
	"github.com/pkg/errors"
	"github.com/Yuwenfeng2019/K2S/pkg/agent"
	"github.com/Yuwenfeng2019/K2S/pkg/cli/cmds"
	"github.com/Yuwenfeng2019/K2S/pkg/server"
	"github.com/rancher/norman/signal"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"k8s.io/apimachinery/pkg/util/net"

	_ "github.com/mattn/go-sqlite3" // ensure we have sqlite
)

func setupLogging(app *cli.Context) {
	if !app.GlobalBool("debug") {
		flag.Set("stderrthreshold", "3")
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

	setupLogging(app)

	if !cfg.DisableAgent && os.Getuid() != 0 {
		return fmt.Errorf("must run as root unless --disable-agent is specified")
	}

	serverConfig := server.Config{}
	serverConfig.ControlConfig.ClusterSecret = cfg.ClusterSecret
	serverConfig.ControlConfig.DataDir = cfg.DataDir
	serverConfig.ControlConfig.KubeConfigOutput = cfg.KubeConfigOutput
	serverConfig.ControlConfig.KubeConfigMode = cfg.KubeConfigMode
	serverConfig.TLSConfig.HTTPSPort = cfg.HTTPSPort
	serverConfig.TLSConfig.HTTPPort = cfg.HTTPPort
	serverConfig.TLSConfig.KnownIPs = knownIPs(cfg.KnownIPs)

	_, serverConfig.ControlConfig.ClusterIPRange, err = net2.ParseCIDR(cfg.ClusterCIDR)
	if err != nil {
		return errors.Wrapf(err, "Invalid CIDR %s: %v", cfg.ClusterCIDR, err)
	}
	_, serverConfig.ControlConfig.ServiceIPRange, err = net2.ParseCIDR(cfg.ServiceCIDR)
	if err != nil {
		return errors.Wrapf(err, "Invalid CIDR %s: %v", cfg.ServiceCIDR, err)
	}

	// If cluster-dns CLI arg is not set, we set ClusterDNS address to be ServiceCIDR network + 10,
	// i.e. when you set service-cidr to 192.168.0.0/16 and don't provide cluster-dns, it will be set to 192.168.0.10
	if cfg.ClusterDNS == "" {
		serverConfig.ControlConfig.ClusterDNS = make(net2.IP, 4)
		copy(serverConfig.ControlConfig.ClusterDNS, serverConfig.ControlConfig.ServiceIPRange.IP.To4())
		serverConfig.ControlConfig.ClusterDNS[3] = 10
	} else {
		serverConfig.ControlConfig.ClusterDNS = net2.ParseIP(cfg.ClusterDNS)
	}

	// TODO: support etcd
	serverConfig.ControlConfig.NoLeaderElect = true

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

	ctx := signal.SigTermCancelContext(context.Background())
	certs, err := server.StartServer(ctx, &serverConfig)
	if err != nil {
		return err
	}

	logrus.Info("k2s is up and running")
	if notifySocket != "" {
		os.Setenv("NOTIFY_SOCKET", notifySocket)
		systemd.SdNotify(true, "READY=1")
	}

	if cfg.DisableAgent {
		<-ctx.Done()
		return nil
	}

	url := fmt.Sprintf("https://localhost:%d", serverConfig.TLSConfig.HTTPSPort)
	token := server.FormatToken(serverConfig.ControlConfig.Runtime.NodeToken, certs)

	agentConfig := cmds.AgentConfig
	agentConfig.Debug = app.GlobalBool("bool")
	agentConfig.DataDir = filepath.Dir(serverConfig.ControlConfig.DataDir)
	agentConfig.ServerURL = url
	agentConfig.Token = token

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
