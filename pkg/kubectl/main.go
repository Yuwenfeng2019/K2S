package kubectl

import (
	goflag "flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/Yuwenfeng2019/K2S/pkg/server"
	"github.com/spf13/pflag"
	utilflag "k8s.io/apiserver/pkg/util/flag"
	"k8s.io/apiserver/pkg/util/logs"
	"k8s.io/kubernetes/pkg/kubectl/cmd"
)

func Main() {
	kubenv := os.Getenv("KUBECONFIG")
	if kubenv == "" {
		config, err := server.HomeKubeConfig(false)
		if _, serr := os.Stat(config); err == nil && serr == nil {
			os.Setenv("KUBECONFIG", config)
		}
	}

	main()
}

func main() {
	rand.Seed(time.Now().UnixNano())

	command := cmd.NewDefaultKubectlCommand()

	// TODO: once we switch everything over to Cobra commands, we can go back to calling
	// utilflag.InitFlags() (by removing its pflag.Parse() call). For now, we have to set the
	// normalize func and add the go flag set by hand.
	pflag.CommandLine.SetNormalizeFunc(utilflag.WordSepNormalizeFunc)
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	// utilflag.InitFlags()
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
