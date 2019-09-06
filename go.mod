module github.com/rancher/k3s

go 1.12

replace (
	github.com/containerd/btrfs => github.com/containerd/btrfs v0.0.0-20181101203652-af5082808c83
	github.com/containerd/cgroups => github.com/containerd/cgroups v0.0.0-20190717030353-c4b9ac5c7601
	github.com/containerd/console => github.com/containerd/console v0.0.0-20181022165439-0650fd9eeb50
	github.com/containerd/containerd => github.com/rancher/containerd v1.3.0-k3s.1
	github.com/containerd/continuity => github.com/containerd/continuity v0.0.0-20190815185530-f2a389ac0a02
	github.com/containerd/fifo => github.com/containerd/fifo v0.0.0-20190816180239-bda0ff6ed73c
	github.com/containerd/go-runc => github.com/containerd/go-runc v0.0.0-20190911050354-e029b79d8cda
	github.com/containerd/typeurl => github.com/containerd/typeurl v0.0.0-20180627222232-a93fcdb778cd
	github.com/containernetworking/plugins => github.com/rancher/plugins v0.8.2-k3s.2
	github.com/coreos/flannel => github.com/rancher/flannel v0.11.0-k3s.1
	github.com/coreos/go-systemd => github.com/coreos/go-systemd v0.0.0-20180511133405-39ca1b05acc7
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20190205005809-0d3efadf0154
	github.com/docker/libnetwork => github.com/docker/libnetwork v0.8.0-dev.2.0.20190624125649-f0e46a78ea34
	github.com/kubernetes-sigs/cri-tools => github.com/rancher/cri-tools v1.16.0-k3s.1
	github.com/matryer/moq => github.com/rancher/moq v0.0.0-20190404221404-ee5226d43009
	github.com/opencontainers/runtime-spec => github.com/opencontainers/runtime-spec v0.0.0-20180911193056-5684b8af48c1
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.2
	github.com/prometheus/client_model => github.com/prometheus/client_model v0.0.0-20180712105110-5c3871d89910
	github.com/prometheus/common => github.com/prometheus/common v0.0.0-20181126121408-4724e9255275
	github.com/prometheus/procfs => github.com/prometheus/procfs v0.0.0-20181204211112-1dc9a6cbc91a
	github.com/rancher/dynamiclistener => github.com/erikwilson/rancher-dynamiclistener v0.0.0-20190717164634-c08b499d1719
	github.com/rancher/kine => github.com/ibuildthecloud/kine v0.1.0
	k8s.io/api => github.com/rancher/kubernetes/staging/src/k8s.io/api v1.16.0-k3s.1
	k8s.io/apiextensions-apiserver => github.com/rancher/kubernetes/staging/src/k8s.io/apiextensions-apiserver v1.16.0-k3s.1
	k8s.io/apimachinery => github.com/rancher/kubernetes/staging/src/k8s.io/apimachinery v1.16.0-k3s.1
	k8s.io/apiserver => github.com/rancher/kubernetes/staging/src/k8s.io/apiserver v1.16.0-k3s.1
	k8s.io/cli-runtime => github.com/rancher/kubernetes/staging/src/k8s.io/cli-runtime v1.16.0-k3s.1
	k8s.io/client-go => github.com/rancher/kubernetes/staging/src/k8s.io/client-go v1.16.0-k3s.1
	k8s.io/cloud-provider => github.com/rancher/kubernetes/staging/src/k8s.io/cloud-provider v1.16.0-k3s.1
	k8s.io/cluster-bootstrap => github.com/rancher/kubernetes/staging/src/k8s.io/cluster-bootstrap v1.16.0-k3s.1
	k8s.io/code-generator => github.com/rancher/kubernetes/staging/src/k8s.io/code-generator v1.16.0-k3s.1
	k8s.io/component-base => github.com/rancher/kubernetes/staging/src/k8s.io/component-base v1.16.0-k3s.1
	k8s.io/cri-api => github.com/rancher/kubernetes/staging/src/k8s.io/cri-api v1.16.0-k3s.1
	k8s.io/csi-translation-lib => github.com/rancher/kubernetes/staging/src/k8s.io/csi-translation-lib v1.16.0-k3s.1
	k8s.io/kube-aggregator => github.com/rancher/kubernetes/staging/src/k8s.io/kube-aggregator v1.16.0-k3s.1
	k8s.io/kube-controller-manager => github.com/rancher/kubernetes/staging/src/k8s.io/kube-controller-manager v1.16.0-k3s.1
	k8s.io/kube-proxy => github.com/rancher/kubernetes/staging/src/k8s.io/kube-proxy v1.16.0-k3s.1
	k8s.io/kube-scheduler => github.com/rancher/kubernetes/staging/src/k8s.io/kube-scheduler v1.16.0-k3s.1
	k8s.io/kubectl => github.com/rancher/kubernetes/staging/src/k8s.io/kubectl v1.16.0-k3s.1
	k8s.io/kubelet => github.com/rancher/kubernetes/staging/src/k8s.io/kubelet v1.16.0-k3s.1
	k8s.io/kubernetes => github.com/rancher/kubernetes v1.16.0-k3s.1
	k8s.io/legacy-cloud-providers => github.com/rancher/kubernetes/staging/src/k8s.io/legacy-cloud-providers v1.16.0-k3s.1
	k8s.io/metrics => github.com/rancher/kubernetes/staging/src/k8s.io/metrics v1.16.0-k3s.1
	k8s.io/node-api => github.com/rancher/kubernetes/staging/src/k8s.io/node-api v1.16.0-k3s.1
	k8s.io/sample-apiserver => github.com/rancher/kubernetes/staging/src/k8s.io/sample-apiserver v1.16.0-k3s.1
	k8s.io/sample-cli-plugin => github.com/rancher/kubernetes/staging/src/k8s.io/sample-cli-plugin v1.16.0-k3s.1
	k8s.io/sample-controller => github.com/rancher/kubernetes/staging/src/k8s.io/sample-controller v1.16.0-k3s.1

)

require (
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/NYTimes/gziphandler v1.1.1 // indirect
	github.com/bhendo/go-powershell v0.0.0-20190719160123-219e7fb4e41e // indirect
	github.com/bronze1man/goStrongswanVici v0.0.0-20190828090544-27d02f80ba40 // indirect
	github.com/buger/jsonparser v0.0.0-20181115193947-bf1c66bbce23 // indirect
	github.com/containerd/cgroups v0.0.0-20190923161937-abd0b19954a6 // indirect
	github.com/containerd/containerd v1.2.8
	github.com/containerd/continuity v0.0.0-20190827140505-75bee3e2ccb6 // indirect
	github.com/containerd/cri v1.11.1-0.20190909171321-f4d75d321c89
	github.com/containerd/fifo v0.0.0-20190816180239-bda0ff6ed73c // indirect
	github.com/containerd/go-cni v0.0.0-20190904155053-d20b7eebc7ee // indirect
	github.com/containerd/go-runc v0.0.0-20190923131748-a2952bc25f51 // indirect
	github.com/containerd/ttrpc v0.0.0-20190828172938-92c8520ef9f8 // indirect
	github.com/containernetworking/plugins v0.8.2
	github.com/coreos/flannel v0.11.0
	github.com/coreos/go-systemd v0.0.0-20190321100706-95778dfbb74e
	github.com/docker/docker v0.7.3-0.20190731001754-589f1dad8dad
	github.com/docker/go-events v0.0.0-20190806004212-e31b211e4f1c // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/libnetwork v0.8.0-dev.2.0.20190624125649-f0e46a78ea34 // indirect
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/go-sql-driver/mysql v1.4.1
	github.com/gofrs/flock v0.7.1 // indirect
	github.com/gogo/googleapis v1.3.0 // indirect
	github.com/google/tcpproxy v0.0.0-20180808230851-dfa16c61dad2
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/websocket v1.4.0
	github.com/j-keck/arping v1.0.0 // indirect
	github.com/juju/errors v0.0.0-20190806202954-0232dcc7464d // indirect
	github.com/juju/testing v0.0.0-20190723135506-ce30eb24acd2 // indirect
	github.com/kubernetes-sigs/cri-tools v0.0.0-00010101000000-000000000000
	github.com/lib/pq v1.1.1
	github.com/mattn/go-sqlite3 v1.10.0
	github.com/mindprince/gonvml v0.0.0-20190828220739-9ebdce4bb989 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible
	github.com/opencontainers/runc v1.0.0-rc2.0.20190611121236-6cc515888830
	github.com/pkg/errors v0.8.1
	github.com/rakelkar/gonetsh v0.0.0-20190719023240-501daadcadf8 // indirect
	github.com/rancher/dynamiclistener v0.0.0-20190717164634-c08b499d1719
	github.com/rancher/helm-controller v0.2.2
	github.com/rancher/kine v0.0.0-00010101000000-000000000000
	github.com/rancher/remotedialer v0.2.0
	github.com/rancher/wrangler v0.2.0
	github.com/rancher/wrangler-api v0.2.0
	github.com/rootless-containers/rootlesskit v0.6.0
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/pflag v1.0.3
	github.com/tchap/go-patricia v2.3.0+incompatible // indirect
	github.com/theckman/go-flock v0.7.1 // indirect
	github.com/urfave/cli v1.21.0
	go.etcd.io/bbolt v1.3.3 // indirect
	golang.org/x/net v0.0.0-20190812203447-cdfb69ac37fc
	golang.org/x/sys v0.0.0-20190812073006-9eafafc0a87e
	google.golang.org/grpc v1.23.0
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
	k8s.io/api v0.0.0
	k8s.io/apimachinery v0.0.0
	k8s.io/apiserver v0.0.0
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/component-base v0.0.0
	k8s.io/cri-api v0.0.0
	k8s.io/klog v0.4.0
	k8s.io/kubernetes v1.16.0
	k8s.io/utils v0.0.0-20190829053155-3a4a5477acf8 // indirect
)
