package server

import (
	"github.com/Yuwenfeng2019/K2S/pkg/daemons/config"
	"github.com/rancher/norman/pkg/dynamiclistener"
)

type Config struct {
	DisableAgent     bool
	DisableServiceLB bool
	TLSConfig        dynamiclistener.UserConfig
	ControlConfig    config.Control
}
