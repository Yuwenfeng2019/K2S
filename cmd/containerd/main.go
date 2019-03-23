package main

import (
	"github.com/Yuwenfeng2019/K2S/pkg/containerd"
	"k8s.io/klog"
)

func main() {
	klog.InitFlags(nil)
	containerd.Main()
}
