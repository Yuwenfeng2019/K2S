package server

import (
	"github.com/rancher/dynamiclistener"
	"github.com/Yuwenfeng2019/K2S/pkg/daemons/config"
)

type Config struct {
	DisableAgent     bool
	DisableServiceLB bool
	TLSConfig        dynamiclistener.UserConfig
	ControlConfig    config.Control
	Rootless         bool
}
