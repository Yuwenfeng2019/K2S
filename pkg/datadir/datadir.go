package datadir

import (
	"os"

	"github.com/pkg/errors"
	"github.com/rancher/wrangler/pkg/resolvehome"
)

const (
	DefaultDataDir     = "/var/lib/k2s"
	DefaultHomeDataDir = "${HOME}//k2s"
	HomeConfig         = "${HOME}/.kube/k2s.yaml"
	GlobalConfig       = "/etc/k2s/k2s.yaml"
)

func Resolve(dataDir string) (string, error) {
	return LocalHome(dataDir, false)
}

func LocalHome(dataDir string, forceLocal bool) (string, error) {
	if dataDir == "" {
		if os.Getuid() == 0 && !forceLocal {
			dataDir = DefaultDataDir
		} else {
			dataDir = DefaultHomeDataDir
		}
	}

	dataDir, err := resolvehome.Resolve(dataDir)
	if err != nil {
		return "", errors.Wrapf(err, "resolving %s", dataDir)
	}

	return dataDir, nil
}
