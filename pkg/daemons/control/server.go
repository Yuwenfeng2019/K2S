package control

import (
	"context"
	"crypto"
	"crypto/x509"
	"fmt"
	"html/template"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	// registering k2s cloud provider
	_ "github.com/Yuwenfeng2019/K2S/pkg/cloudprovider"

	certutil "github.com/rancher/dynamiclistener/cert"
	"github.com/Yuwenfeng2019/K2S/pkg/cluster"
	"github.com/Yuwenfeng2019/K2S/pkg/daemons/config"
	"github.com/Yuwenfeng2019/K2S/pkg/passwd"
	"github.com/Yuwenfeng2019/K2S/pkg/token"
	"github.com/rancher/kine/pkg/endpoint"
	"github.com/rancher/wrangler-api/pkg/generated/controllers/rbac"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/tools/clientcmd"
	ccmapp "k8s.io/kubernetes/cmd/cloud-controller-manager/app"
	"k8s.io/kubernetes/cmd/kube-apiserver/app"
	cmapp "k8s.io/kubernetes/cmd/kube-controller-manager/app"
	sapp "k8s.io/kubernetes/cmd/kube-scheduler/app"
	_ "k8s.io/kubernetes/pkg/client/metrics/prometheus" // for client metric registration
	"k8s.io/kubernetes/pkg/kubeapiserver/authorizer/modes"
	"k8s.io/kubernetes/pkg/master"
	"k8s.io/kubernetes/pkg/proxy/util"
)

var (
	localhostIP        = net.ParseIP("127.0.0.1")
	requestHeaderCN    = "system:auth-proxy"
	kubeconfigTemplate = template.Must(template.New("kubeconfig").Parse(`apiVersion: v1
clusters:
- cluster:
    server: {{.URL}}
    certificate-authority: {{.CACert}}
  name: local
contexts:
- context:
    cluster: local
    namespace: default
    user: user
  name: Default
current-context: Default
kind: Config
preferences: {}
users:
- name: user
  user:
    client-certificate: {{.ClientCert}}
    client-key: {{.ClientKey}}
`))
)

const (
	userTokenSize  = 16
	ipsecTokenSize = 48
)

func Server(ctx context.Context, cfg *config.Control) error {
	rand.Seed(time.Now().UTC().UnixNano())

	runtime := &config.ControlRuntime{}
	cfg.Runtime = runtime

	if err := prepare(ctx, cfg, runtime); err != nil {
		return err
	}

	cfg.Runtime.Tunnel = setupTunnel()
	util.DisableProxyHostnameCheck = true

	auth, handler, err := apiServer(ctx, cfg, runtime)
	if err != nil {
		return err
	}

	if err := waitForAPIServer(runtime); err != nil {
		return err
	}

	runtime.Handler = handler
	runtime.Authenticator = auth

	if !cfg.NoScheduler {
		scheduler(cfg, runtime)
	}

	controllerManager(cfg, runtime)

	if !cfg.DisableCCM {
		cloudControllerManager(cfg, runtime)
	}

	return nil
}

func controllerManager(cfg *config.Control, runtime *config.ControlRuntime) {
	argsMap := map[string]string{
		"kubeconfig":                       runtime.KubeConfigController,
		"service-account-private-key-file": runtime.ServiceKey,
		"allocate-node-cidrs":              "true",
		"cluster-cidr":                     cfg.ClusterIPRange.String(),
		"root-ca-file":                     runtime.ServerCA,
		"port":                             "10252",
		"bind-address":                     localhostIP.String(),
		"secure-port":                      "0",
		"use-service-account-credentials":  "true",
		"cluster-signing-cert-file":        runtime.ServerCA,
		"cluster-signing-key-file":         runtime.ServerCAKey,
	}
	offset := cfg.HTTPSPort - 6443
	if offset > 0 {
		argsMap["port"] = strconv.Itoa(10252 + offset)
	}
	if cfg.NoLeaderElect {
		argsMap["leader-elect"] = "false"
	}

	args := config.GetArgsList(argsMap, cfg.ExtraControllerArgs)

	command := cmapp.NewControllerManagerCommand()
	command.SetArgs(args)

	go func() {
		logrus.Infof("Running kube-controller-manager %s", config.ArgString(args))
		logrus.Fatalf("controller-manager exited: %v", command.Execute())
	}()
}

func scheduler(cfg *config.Control, runtime *config.ControlRuntime) {
	argsMap := map[string]string{
		"kubeconfig":   runtime.KubeConfigScheduler,
		"port":         "10251",
		"bind-address": "127.0.0.1",
		"secure-port":  "0",
	}
	offset := cfg.HTTPSPort - 6443
	if offset > 0 {
		argsMap["port"] = strconv.Itoa(10251 + offset)
	}
	if cfg.NoLeaderElect {
		argsMap["leader-elect"] = "false"
	}
	args := config.GetArgsList(argsMap, cfg.ExtraSchedulerAPIArgs)

	command := sapp.NewSchedulerCommand()
	command.SetArgs(args)

	go func() {
		logrus.Infof("Running kube-scheduler %s", config.ArgString(args))
		logrus.Fatalf("scheduler exited: %v", command.Execute())
	}()
}

func apiServer(ctx context.Context, cfg *config.Control, runtime *config.ControlRuntime) (authenticator.Request, http.Handler, error) {
	argsMap := make(map[string]string)

	setupStorageBackend(argsMap, cfg)

	certDir := filepath.Join(cfg.DataDir, "tls/temporary-certs")
	os.MkdirAll(certDir, 0700)

	argsMap["cert-dir"] = certDir
	argsMap["allow-privileged"] = "true"
	argsMap["authorization-mode"] = strings.Join([]string{modes.ModeNode, modes.ModeRBAC}, ",")
	argsMap["service-account-signing-key-file"] = runtime.ServiceKey
	argsMap["service-cluster-ip-range"] = cfg.ServiceIPRange.String()
	argsMap["advertise-port"] = strconv.Itoa(cfg.AdvertisePort)
	if cfg.AdvertiseIP != "" {
		argsMap["advertise-address"] = cfg.AdvertiseIP
	}
	argsMap["insecure-port"] = "0"
	argsMap["secure-port"] = strconv.Itoa(cfg.ListenPort)
	argsMap["bind-address"] = localhostIP.String()
	argsMap["tls-cert-file"] = runtime.ServingKubeAPICert
	argsMap["tls-private-key-file"] = runtime.ServingKubeAPIKey
	argsMap["service-account-key-file"] = runtime.ServiceKey
	argsMap["service-account-issuer"] = "k2s"
	argsMap["api-audiences"] = "unknown"
	argsMap["basic-auth-file"] = runtime.PasswdFile
	argsMap["kubelet-certificate-authority"] = runtime.ServerCA
	argsMap["kubelet-client-certificate"] = runtime.ClientKubeAPICert
	argsMap["kubelet-client-key"] = runtime.ClientKubeAPIKey
	argsMap["requestheader-client-ca-file"] = runtime.RequestHeaderCA
	argsMap["requestheader-allowed-names"] = requestHeaderCN
	argsMap["proxy-client-cert-file"] = runtime.ClientAuthProxyCert
	argsMap["proxy-client-key-file"] = runtime.ClientAuthProxyKey
	argsMap["requestheader-extra-headers-prefix"] = "X-Remote-Extra-"
	argsMap["requestheader-group-headers"] = "X-Remote-Group"
	argsMap["requestheader-username-headers"] = "X-Remote-User"
	argsMap["client-ca-file"] = runtime.ClientCA
	argsMap["enable-admission-plugins"] = "NodeRestriction"
	argsMap["anonymous-auth"] = "false"

	args := config.GetArgsList(argsMap, cfg.ExtraAPIArgs)

	command := app.NewAPIServerCommand(ctx.Done())
	command.SetArgs(args)

	go func() {
		logrus.Infof("Running kube-apiserver %s", config.ArgString(args))
		logrus.Fatalf("apiserver exited: %v", command.Execute())
	}()

	startupConfig := <-app.StartupConfig

	return startupConfig.Authenticator, startupConfig.Handler, nil
}

func defaults(config *config.Control) {
	if config.ClusterIPRange == nil {
		_, clusterIPNet, _ := net.ParseCIDR("10.42.0.0/16")
		config.ClusterIPRange = clusterIPNet
	}

	if config.ServiceIPRange == nil {
		_, serviceIPNet, _ := net.ParseCIDR("10.43.0.0/16")
		config.ServiceIPRange = serviceIPNet
	}

	if len(config.ClusterDNS) == 0 {
		config.ClusterDNS = net.ParseIP("10.43.0.10")
	}

	if config.AdvertisePort == 0 {
		config.AdvertisePort = config.HTTPSPort
	}

	if config.ListenPort == 0 {
		if config.HTTPSPort != 0 {
			config.ListenPort = config.HTTPSPort + 1
		} else {
			config.ListenPort = 6444
		}
	}

	if config.DataDir == "" {
		config.DataDir = "./management-state"
	}
}

func prepare(ctx context.Context, config *config.Control, runtime *config.ControlRuntime) error {
	var err error

	defaults(config)

	if err := os.MkdirAll(config.DataDir, 0700); err != nil {
		return err
	}

	config.DataDir, err = filepath.Abs(config.DataDir)
	if err != nil {
		return err
	}

	os.MkdirAll(path.Join(config.DataDir, "tls"), 0700)
	os.MkdirAll(path.Join(config.DataDir, "cred"), 0700)

	runtime.ClientCA = path.Join(config.DataDir, "tls", "client-ca.crt")
	runtime.ClientCAKey = path.Join(config.DataDir, "tls", "client-ca.key")
	runtime.ServerCA = path.Join(config.DataDir, "tls", "server-ca.crt")
	runtime.ServerCAKey = path.Join(config.DataDir, "tls", "server-ca.key")
	runtime.RequestHeaderCA = path.Join(config.DataDir, "tls", "request-header-ca.crt")
	runtime.RequestHeaderCAKey = path.Join(config.DataDir, "tls", "request-header-ca.key")
	runtime.IPSECKey = path.Join(config.DataDir, "cred", "ipsec.psk")

	runtime.ServiceKey = path.Join(config.DataDir, "tls", "service.key")
	runtime.PasswdFile = path.Join(config.DataDir, "cred", "passwd")
	runtime.NodePasswdFile = path.Join(config.DataDir, "cred", "node-passwd")

	runtime.KubeConfigAdmin = path.Join(config.DataDir, "cred", "admin.kubeconfig")
	runtime.KubeConfigController = path.Join(config.DataDir, "cred", "controller.kubeconfig")
	runtime.KubeConfigScheduler = path.Join(config.DataDir, "cred", "scheduler.kubeconfig")
	runtime.KubeConfigAPIServer = path.Join(config.DataDir, "cred", "api-server.kubeconfig")
	runtime.KubeConfigCloudController = path.Join(config.DataDir, "cred", "cloud-controller.kubeconfig")

	runtime.ClientAdminCert = path.Join(config.DataDir, "tls", "client-admin.crt")
	runtime.ClientAdminKey = path.Join(config.DataDir, "tls", "client-admin.key")
	runtime.ClientControllerCert = path.Join(config.DataDir, "tls", "client-controller.crt")
	runtime.ClientControllerKey = path.Join(config.DataDir, "tls", "client-controller.key")
	runtime.ClientCloudControllerCert = path.Join(config.DataDir, "tls", "client-cloud-controller.crt")
	runtime.ClientCloudControllerKey = path.Join(config.DataDir, "tls", "client-cloud-controller.key")
	runtime.ClientSchedulerCert = path.Join(config.DataDir, "tls", "client-scheduler.crt")
	runtime.ClientSchedulerKey = path.Join(config.DataDir, "tls", "client-scheduler.key")
	runtime.ClientKubeAPICert = path.Join(config.DataDir, "tls", "client-kube-apiserver.crt")
	runtime.ClientKubeAPIKey = path.Join(config.DataDir, "tls", "client-kube-apiserver.key")
	runtime.ClientKubeProxyCert = path.Join(config.DataDir, "tls", "client-kube-proxy.crt")
	runtime.ClientKubeProxyKey = path.Join(config.DataDir, "tls", "client-kube-proxy.key")
	runtime.ClientK2sControllerCert = path.Join(config.DataDir, "tls", "client-k2s-controller.crt")
	runtime.ClientK2sControllerKey = path.Join(config.DataDir, "tls", "client-k2s-controller.key")

	runtime.ServingKubeAPICert = path.Join(config.DataDir, "tls", "serving-kube-apiserver.crt")
	runtime.ServingKubeAPIKey = path.Join(config.DataDir, "tls", "serving-kube-apiserver.key")

	runtime.ClientKubeletKey = path.Join(config.DataDir, "tls", "client-kubelet.key")
	runtime.ServingKubeletKey = path.Join(config.DataDir, "tls", "serving-kubelet.key")

	runtime.ClientAuthProxyCert = path.Join(config.DataDir, "tls", "client-auth-proxy.crt")
	runtime.ClientAuthProxyKey = path.Join(config.DataDir, "tls", "client-auth-proxy.key")

	if err := genCerts(config, runtime); err != nil {
		return err
	}

	if err := cluster.New(config).Start(ctx); err != nil {
		return err
	}

	if err := genServiceAccount(runtime); err != nil {
		return err
	}

	if err := genUsers(config, runtime); err != nil {
		return err
	}

	if err := genEncryptedNetworkInfo(config, runtime); err != nil {
		return err
	}

	if err := prepareStorageBackend(ctx, config); err != nil {
		return err
	}

	return readTokens(runtime)
}

func prepareStorageBackend(ctx context.Context, config *config.Control) error {
	etcdConfig, err := endpoint.Listen(ctx, config.Storage)
	if err != nil {
		return err
	}

	config.Storage.Config = etcdConfig.TLSConfig
	config.Storage.Endpoint = strings.Join(etcdConfig.Endpoints, ",")
	config.NoLeaderElect = !etcdConfig.LeaderElect
	return nil
}

func readTokens(runtime *config.ControlRuntime) error {
	tokens, err := passwd.Read(runtime.PasswdFile)
	if err != nil {
		return err
	}

	if nodeToken, ok := tokens.Pass("node"); ok {
		runtime.AgentToken = "node:" + nodeToken
	}
	if serverToken, ok := tokens.Pass("server"); ok {
		runtime.AgentToken = "server:" + serverToken
	}
	if clientToken, ok := tokens.Pass("admin"); ok {
		runtime.ClientToken = "admin:" + clientToken
	}

	return nil
}

func genEncryptedNetworkInfo(controlConfig *config.Control, runtime *config.ControlRuntime) error {
	if s, err := os.Stat(runtime.IPSECKey); err == nil && s.Size() > 0 {
		psk, err := ioutil.ReadFile(runtime.IPSECKey)
		if err != nil {
			return err
		}
		controlConfig.IPSECPSK = strings.TrimSpace(string(psk))
		return nil
	}

	psk, err := token.Random(ipsecTokenSize)
	if err != nil {
		return err
	}

	controlConfig.IPSECPSK = psk
	if err := ioutil.WriteFile(runtime.IPSECKey, []byte(psk+"\n"), 0600); err != nil {
		return err
	}

	return nil
}

func migratePassword(p *passwd.Passwd) error {
	server, _ := p.Pass("server")
	node, _ := p.Pass("node")
	if server == "" && node != "" {
		return p.EnsureUser("server", "k2s:server", node)
	}
	return nil
}

func getServerPass(passwd *passwd.Passwd, config *config.Control) (string, error) {
	var (
		err error
	)

	serverPass := config.Token
	if serverPass == "" {
		serverPass, _ = passwd.Pass("server")
	}
	if serverPass == "" {
		serverPass, err = token.Random(16)
		if err != nil {
			return "", err
		}
	}

	return serverPass, nil
}

func getNodePass(config *config.Control, serverPass string) string {
	if config.AgentToken == "" {
		return serverPass
	}
	return config.AgentToken
}

func genUsers(config *config.Control, runtime *config.ControlRuntime) error {
	passwd, err := passwd.Read(runtime.PasswdFile)
	if err != nil {
		return err
	}

	if err := migratePassword(passwd); err != nil {
		return err
	}

	serverPass, err := getServerPass(passwd, config)
	if err != nil {
		return err
	}

	nodePass := getNodePass(config, serverPass)

	if err := passwd.EnsureUser("admin", "system:masters", ""); err != nil {
		return err
	}

	if err := passwd.EnsureUser("node", "k2s:agent", nodePass); err != nil {
		return err
	}

	if err := passwd.EnsureUser("server", "k2s:server", serverPass); err != nil {
		return err
	}

	return passwd.Write(runtime.PasswdFile)
}

func genCerts(config *config.Control, runtime *config.ControlRuntime) error {
	if err := genClientCerts(config, runtime); err != nil {
		return err
	}
	if err := genServerCerts(config, runtime); err != nil {
		return err
	}
	if err := genRequestHeaderCerts(config, runtime); err != nil {
		return err
	}
	return nil
}

type signedCertFactory = func(commonName string, organization []string, certFile, keyFile string) (bool, error)

func getSigningCertFactory(regen bool, altNames *certutil.AltNames, extKeyUsage []x509.ExtKeyUsage, caCertFile, caKeyFile string) signedCertFactory {
	return func(commonName string, organization []string, certFile, keyFile string) (bool, error) {
		return createClientCertKey(regen, commonName, organization, altNames, extKeyUsage, caCertFile, caKeyFile, certFile, keyFile)
	}
}

func genClientCerts(config *config.Control, runtime *config.ControlRuntime) error {
	regen, err := createSigningCertKey("k2s-client", runtime.ClientCA, runtime.ClientCAKey)
	if err != nil {
		return err
	}

	factory := getSigningCertFactory(regen, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, runtime.ClientCA, runtime.ClientCAKey)

	var certGen bool
	apiEndpoint := fmt.Sprintf("https://127.0.0.1:%d", config.ListenPort)

	certGen, err = factory("system:admin", []string{"system:masters"}, runtime.ClientAdminCert, runtime.ClientAdminKey)
	if err != nil {
		return err
	}
	if certGen {
		if err := KubeConfig(runtime.KubeConfigAdmin, apiEndpoint, runtime.ServerCA, runtime.ClientAdminCert, runtime.ClientAdminKey); err != nil {
			return err
		}
	}

	certGen, err = factory("system:kube-controller-manager", nil, runtime.ClientControllerCert, runtime.ClientControllerKey)
	if err != nil {
		return err
	}
	if certGen {
		if err := KubeConfig(runtime.KubeConfigController, apiEndpoint, runtime.ServerCA, runtime.ClientControllerCert, runtime.ClientControllerKey); err != nil {
			return err
		}
	}

	certGen, err = factory("system:kube-scheduler", nil, runtime.ClientSchedulerCert, runtime.ClientSchedulerKey)
	if err != nil {
		return err
	}
	if certGen {
		if err := KubeConfig(runtime.KubeConfigScheduler, apiEndpoint, runtime.ServerCA, runtime.ClientSchedulerCert, runtime.ClientSchedulerKey); err != nil {
			return err
		}
	}

	certGen, err = factory("kube-apiserver", nil, runtime.ClientKubeAPICert, runtime.ClientKubeAPIKey)
	if err != nil {
		return err
	}
	if certGen {
		if err := KubeConfig(runtime.KubeConfigAPIServer, apiEndpoint, runtime.ServerCA, runtime.ClientKubeAPICert, runtime.ClientKubeAPIKey); err != nil {
			return err
		}
	}

	if _, err = factory("system:kube-proxy", nil, runtime.ClientKubeProxyCert, runtime.ClientKubeProxyKey); err != nil {
		return err
	}
	if _, err = factory("system:k2s-controller", nil, runtime.ClientK2sControllerCert, runtime.ClientK2sControllerKey); err != nil {
		return err
	}

	if _, _, err := certutil.LoadOrGenerateKeyFile(runtime.ClientKubeletKey, regen); err != nil {
		return err
	}

	certGen, err = factory("cloud-controller-manager", nil, runtime.ClientCloudControllerCert, runtime.ClientCloudControllerKey)
	if err != nil {
		return err
	}
	if certGen {
		if err := KubeConfig(runtime.KubeConfigCloudController, apiEndpoint, runtime.ServerCA, runtime.ClientCloudControllerCert, runtime.ClientCloudControllerKey); err != nil {
			return err
		}
	}

	return nil
}

func createServerSigningCertKey(config *config.Control, runtime *config.ControlRuntime) (bool, error) {
	TokenCA := path.Join(config.DataDir, "tls", "token-ca.crt")
	TokenCAKey := path.Join(config.DataDir, "tls", "token-ca.key")

	if exists(TokenCA, TokenCAKey) && !exists(runtime.ServerCA) && !exists(runtime.ServerCAKey) {
		logrus.Infof("Upgrading token-ca files to server-ca")
		if err := os.Link(TokenCA, runtime.ServerCA); err != nil {
			return false, err
		}
		if err := os.Link(TokenCAKey, runtime.ServerCAKey); err != nil {
			return false, err
		}
		return true, nil
	}
	return createSigningCertKey("k2s-server", runtime.ServerCA, runtime.ServerCAKey)
}

func genServerCerts(config *config.Control, runtime *config.ControlRuntime) error {
	regen, err := createServerSigningCertKey(config, runtime)
	if err != nil {
		return err
	}

	_, apiServerServiceIP, err := master.DefaultServiceIPRange(*config.ServiceIPRange)
	if err != nil {
		return err
	}

	if _, err := createClientCertKey(regen, "kube-apiserver", nil,
		&certutil.AltNames{
			DNSNames: []string{"kubernetes.default.svc", "kubernetes.default", "kubernetes", "localhost"},
			IPs:      []net.IP{apiServerServiceIP, localhostIP},
		}, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		runtime.ServerCA, runtime.ServerCAKey,
		runtime.ServingKubeAPICert, runtime.ServingKubeAPIKey); err != nil {
		return err
	}

	if _, _, err := certutil.LoadOrGenerateKeyFile(runtime.ServingKubeletKey, regen); err != nil {
		return err
	}

	return nil
}

func genRequestHeaderCerts(config *config.Control, runtime *config.ControlRuntime) error {
	regen, err := createSigningCertKey("k2s-request-header", runtime.RequestHeaderCA, runtime.RequestHeaderCAKey)
	if err != nil {
		return err
	}

	if _, err := createClientCertKey(regen, requestHeaderCN, nil,
		nil, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		runtime.RequestHeaderCA, runtime.RequestHeaderCAKey,
		runtime.ClientAuthProxyCert, runtime.ClientAuthProxyKey); err != nil {
		return err
	}

	return nil
}

func createClientCertKey(regen bool, commonName string, organization []string, altNames *certutil.AltNames, extKeyUsage []x509.ExtKeyUsage, caCertFile, caKeyFile, certFile, keyFile string) (bool, error) {
	// check for certificate expiration
	if !regen {
		regen = expired(certFile)
	}

	if !regen {
		if exists(certFile, keyFile) {
			return false, nil
		}
	}

	caKeyBytes, err := ioutil.ReadFile(caKeyFile)
	if err != nil {
		return false, err
	}

	caKey, err := certutil.ParsePrivateKeyPEM(caKeyBytes)
	if err != nil {
		return false, err
	}

	caBytes, err := ioutil.ReadFile(caCertFile)
	if err != nil {
		return false, err
	}

	caCert, err := certutil.ParseCertsPEM(caBytes)
	if err != nil {
		return false, err
	}
	keyBytes, _, err := certutil.LoadOrGenerateKeyFile(keyFile, regen)
	if err != nil {
		return false, err
	}

	key, err := certutil.ParsePrivateKeyPEM(keyBytes)
	if err != nil {
		return false, err
	}

	cfg := certutil.Config{
		CommonName:   commonName,
		Organization: organization,
		Usages:       extKeyUsage,
	}
	if altNames != nil {
		cfg.AltNames = *altNames
	}
	cert, err := certutil.NewSignedCert(cfg, key.(crypto.Signer), caCert[0], caKey.(crypto.Signer))
	if err != nil {
		return false, err
	}

	return true, certutil.WriteCert(certFile, append(certutil.EncodeCertPEM(cert), certutil.EncodeCertPEM(caCert[0])...))
}

func exists(files ...string) bool {
	for _, file := range files {
		if _, err := os.Stat(file); err != nil {
			return false
		}
	}
	return true
}

func genServiceAccount(runtime *config.ControlRuntime) error {
	_, keyErr := os.Stat(runtime.ServiceKey)
	if keyErr == nil {
		return nil
	}

	key, err := certutil.NewPrivateKey()
	if err != nil {
		return err
	}

	return certutil.WriteKey(runtime.ServiceKey, certutil.EncodePrivateKeyPEM(key))
}

func createSigningCertKey(prefix, certFile, keyFile string) (bool, error) {
	if exists(certFile, keyFile) {
		return false, nil
	}

	caKeyBytes, _, err := certutil.LoadOrGenerateKeyFile(keyFile, false)
	if err != nil {
		return false, err
	}

	caKey, err := certutil.ParsePrivateKeyPEM(caKeyBytes)
	if err != nil {
		return false, err
	}

	cfg := certutil.Config{
		CommonName: fmt.Sprintf("%s-ca@%d", prefix, time.Now().Unix()),
	}

	cert, err := certutil.NewSelfSignedCACert(cfg, caKey.(crypto.Signer))
	if err != nil {
		return false, err
	}

	if err := certutil.WriteCert(certFile, certutil.EncodeCertPEM(cert)); err != nil {
		return false, err
	}
	return true, nil
}

func KubeConfig(dest, url, caCert, clientCert, clientKey string) error {
	data := struct {
		URL        string
		CACert     string
		ClientCert string
		ClientKey  string
	}{
		URL:        url,
		CACert:     caCert,
		ClientCert: clientCert,
		ClientKey:  clientKey,
	}

	output, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer output.Close()

	return kubeconfigTemplate.Execute(output, &data)
}

func setupStorageBackend(argsMap map[string]string, cfg *config.Control) {
	argsMap["storage-backend"] = "etcd3"
	// specify the endpoints
	if len(cfg.Storage.Endpoint) > 0 {
		argsMap["etcd-servers"] = cfg.Storage.Endpoint
	}
	// storage backend tls configuration
	if len(cfg.Storage.CAFile) > 0 {
		argsMap["etcd-cafile"] = cfg.Storage.CAFile
	}
	if len(cfg.Storage.CertFile) > 0 {
		argsMap["etcd-certfile"] = cfg.Storage.CertFile
	}
	if len(cfg.Storage.KeyFile) > 0 {
		argsMap["etcd-keyfile"] = cfg.Storage.KeyFile
	}
}

func expired(certFile string) bool {
	certBytes, err := ioutil.ReadFile(certFile)
	if err != nil {
		return false
	}
	certificates, err := certutil.ParseCertsPEM(certBytes)
	if err != nil {
		return false
	}
	return certutil.IsCertExpired(certificates[0])
}

func cloudControllerManager(cfg *config.Control, runtime *config.ControlRuntime) {
	argsMap := map[string]string{
		"kubeconfig":                   runtime.KubeConfigCloudController,
		"allocate-node-cidrs":          "true",
		"cluster-cidr":                 cfg.ClusterIPRange.String(),
		"bind-address":                 localhostIP.String(),
		"secure-port":                  "0",
		"cloud-provider":               "k2s",
		"allow-untagged-cloud":         "true",
		"node-status-update-frequency": "1m",
	}
	if cfg.NoLeaderElect {
		argsMap["leader-elect"] = "false"
	}

	args := config.GetArgsList(argsMap, cfg.ExtraCloudControllerArgs)

	command := ccmapp.NewCloudControllerManagerCommand()
	command.SetArgs(args)
	// register k2s cloud provider

	go func() {
		for {
			// check for the cloud controller rbac binding
			if err := checkForCloudControllerPrivileges(runtime); err != nil {
				logrus.Infof("Waiting for cloudcontroller rbac role to be created")
				time.Sleep(time.Second)
				continue
			}
			break
		}
		logrus.Infof("Running cloud-controller-manager %s", config.ArgString(args))
		logrus.Fatalf("cloud-controller-manager exited: %v", command.Execute())
	}()
}

func checkForCloudControllerPrivileges(runtime *config.ControlRuntime) error {
	restConfig, err := clientcmd.BuildConfigFromFlags("", runtime.KubeConfigAdmin)
	if err != nil {
		return err
	}
	crb := rbac.NewFactoryFromConfigOrDie(restConfig).Rbac().V1().ClusterRoleBinding()
	_, err = crb.Get("cloud-controller-manager", metav1.GetOptions{})
	if err != nil {
		return err
	}
	return nil
}

func waitForAPIServer(runtime *config.ControlRuntime) error {
	restConfig, err := clientcmd.BuildConfigFromFlags("", runtime.KubeConfigAdmin)
	if err != nil {
		return err
	}

	discoveryclient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return err
	}

	for i := 0; i < 60; i++ {
		info, err := discoveryclient.ServerVersion()
		if err == nil {
			logrus.Infof("apiserver %s is up and running", info)
			return nil
		}
		logrus.Infof("waiting for apiserver to become available")
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("timeout waiting for apiserver")
}
