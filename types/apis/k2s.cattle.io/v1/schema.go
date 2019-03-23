package v1

import (
	"github.com/rancher/norman/types"
	"github.com/rancher/norman/types/factory"
)

var (
	APIVersion = types.APIVersion{
		Version: "v1",
		Group:   "k2s.cattle.io",
		Path:    "/v1-k2s",
	}

	Schemas = factory.Schemas(&APIVersion).
		MustImport(&APIVersion, Addon{}).
		MustImport(&APIVersion, HelmChart{}).
		MustImport(&APIVersion, ListenerConfig{})
)
