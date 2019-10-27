package server

import (
	"github.com/Yuwenfeng2019/K2S/pkg/daemons/config"
)

type Config struct {
	DisableAgent     bool
	DisableServiceLB bool
	ControlConfig    config.Control
	Rootless         bool
}
