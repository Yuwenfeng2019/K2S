package kubectl

import (
	"github.com/Yuwenfeng2019/K2S/pkg/kubectl"
	"github.com/urfave/cli"
)

func Run(ctx *cli.Context) error {
	kubectl.Main()
	return nil
}
