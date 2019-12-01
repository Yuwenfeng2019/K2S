package syssetup

import (
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/sirupsen/logrus"
)

func loadKernelModule(moduleName string) {
	if _, err := os.Stat("/sys/module/" + moduleName); err == nil {
		logrus.Infof("module %s was already loaded", moduleName)
		return
	}

	if err := exec.Command("modprobe", moduleName).Run(); err != nil {
		logrus.Warnf("failed to start %s module", moduleName)
	}
}

func enableSystemControl(file string) {
	if err := ioutil.WriteFile(file, []byte("1"), 0640); err != nil {
		logrus.Warnf("failed to write value 1 at %s: %v", file, err)
	}
}

func Configure() {
	loadKernelModule("overlay")
	loadKernelModule("nf_conntrack")
	loadKernelModule("br_netfilter")

	enableSystemControl("/proc/sys/net/ipv4/ip_forward")
	enableSystemControl("/proc/sys/net/ipv6/conf/all/forwarding")
	enableSystemControl("/proc/sys/net/bridge/bridge-nf-call-iptables")
	enableSystemControl("/proc/sys/net/bridge/bridge-nf-call-ip6tables")
}
