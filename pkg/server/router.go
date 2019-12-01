package server

import (
	"crypto"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	certutil "github.com/rancher/dynamiclistener/cert"
	"github.com/Yuwenfeng2019/K2S/pkg/bootstrap"
	"github.com/Yuwenfeng2019/K2S/pkg/daemons/config"
	"github.com/Yuwenfeng2019/K2S/pkg/passwd"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/json"
)

const (
	staticURL = "/static/"
)

func router(serverConfig *config.Control, tunnel http.Handler, ca []byte) http.Handler {
	authed := mux.NewRouter()
	authed.Use(authMiddleware(serverConfig, "k2s:agent"))
	authed.NotFoundHandler = serverConfig.Runtime.Handler
	authed.Path("/v1-k2s/serving-kubelet.crt").Handler(servingKubeletCert(serverConfig, serverConfig.Runtime.ServingKubeletKey))
	authed.Path("/v1-k2s/client-kubelet.crt").Handler(clientKubeletCert(serverConfig, serverConfig.Runtime.ClientKubeletKey))
	authed.Path("/v1-k2s/client-kube-proxy.crt").Handler(fileHandler(serverConfig.Runtime.ClientKubeProxyCert, serverConfig.Runtime.ClientKubeProxyKey))
	authed.Path("/v1-k2s/client-k2s-controller.crt").Handler(fileHandler(serverConfig.Runtime.ClientK2sControllerCert, serverConfig.Runtime.ClientK2sControllerKey))
	authed.Path("/v1-k2s/client-ca.crt").Handler(fileHandler(serverConfig.Runtime.ClientCA))
	authed.Path("/v1-k2s/server-ca.crt").Handler(fileHandler(serverConfig.Runtime.ServerCA))
	authed.Path("/v1-k2s/config").Handler(configHandler(serverConfig))

	nodeAuthed := mux.NewRouter()
	nodeAuthed.Use(authMiddleware(serverConfig, "system:nodes"))
	nodeAuthed.Path("/v1-k2s/connect").Handler(tunnel)
	nodeAuthed.NotFoundHandler = authed

	serverAuthed := mux.NewRouter()
	serverAuthed.Use(authMiddleware(serverConfig, "k2s:server"))
	serverAuthed.NotFoundHandler = nodeAuthed
	serverAuthed.Path("/db/info").Handler(nodeAuthed)
	if serverConfig.Runtime.HTTPBootstrap {
		serverAuthed.Path("/v1-k2s/server-bootstrap").Handler(bootstrap.Handler(&serverConfig.Runtime.ControlRuntimeBootstrap))
	}

	staticDir := filepath.Join(serverConfig.DataDir, "static")
	router := mux.NewRouter()
	router.NotFoundHandler = serverAuthed
	router.PathPrefix(staticURL).Handler(serveStatic(staticURL, staticDir))
	router.Path("/cacerts").Handler(cacerts(ca))
	router.Path("/ping").Handler(ping())

	return router
}

func cacerts(ca []byte) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Set("content-type", "text/plain")
		resp.Write(ca)
	})
}

func getNodeInfo(req *http.Request) (string, string, error) {
	nodeNames := req.Header["K2s-Node-Name"]
	if len(nodeNames) != 1 || nodeNames[0] == "" {
		return "", "", errors.New("node name not set")
	}

	nodePasswords := req.Header["K2s-Node-Password"]
	if len(nodePasswords) != 1 || nodePasswords[0] == "" {
		return "", "", errors.New("node password not set")
	}

	return strings.ToLower(nodeNames[0]), nodePasswords[0], nil
}

func getCACertAndKeys(caCertFile, caKeyFile, signingKeyFile string) ([]*x509.Certificate, crypto.Signer, crypto.Signer, error) {
	keyBytes, err := ioutil.ReadFile(signingKeyFile)
	if err != nil {
		return nil, nil, nil, err
	}

	key, err := certutil.ParsePrivateKeyPEM(keyBytes)
	if err != nil {
		return nil, nil, nil, err
	}

	caKeyBytes, err := ioutil.ReadFile(caKeyFile)
	if err != nil {
		return nil, nil, nil, err
	}

	caKey, err := certutil.ParsePrivateKeyPEM(caKeyBytes)
	if err != nil {
		return nil, nil, nil, err
	}

	caBytes, err := ioutil.ReadFile(caCertFile)
	if err != nil {
		return nil, nil, nil, err
	}

	caCert, err := certutil.ParseCertsPEM(caBytes)
	if err != nil {
		return nil, nil, nil, err
	}

	return caCert, caKey.(crypto.Signer), key.(crypto.Signer), nil
}

func servingKubeletCert(server *config.Control, keyFile string) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if req.TLS == nil {
			resp.WriteHeader(http.StatusNotFound)
			return
		}

		nodeName, nodePassword, err := getNodeInfo(req)
		if err != nil {
			sendError(err, resp)
			return
		}

		if err := ensureNodePassword(server.Runtime.NodePasswdFile, nodeName, nodePassword); err != nil {
			sendError(err, resp, http.StatusForbidden)
			return
		}

		caCert, caKey, key, err := getCACertAndKeys(server.Runtime.ServerCA, server.Runtime.ServerCAKey, server.Runtime.ServingKubeletKey)
		if err != nil {
			sendError(err, resp)
			return
		}

		cert, err := certutil.NewSignedCert(certutil.Config{
			CommonName: nodeName,
			Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			AltNames: certutil.AltNames{
				DNSNames: []string{nodeName, "localhost"},
				IPs:      []net.IP{net.ParseIP("127.0.0.1")},
			},
		}, key, caCert[0], caKey)
		if err != nil {
			sendError(err, resp)
			return
		}

		keyBytes, err := ioutil.ReadFile(keyFile)
		if err != nil {
			http.Error(resp, err.Error(), http.StatusInternalServerError)
			return
		}

		resp.Write(append(certutil.EncodeCertPEM(cert), certutil.EncodeCertPEM(caCert[0])...))
		resp.Write(keyBytes)
	})
}

func clientKubeletCert(server *config.Control, keyFile string) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if req.TLS == nil {
			resp.WriteHeader(http.StatusNotFound)
			return
		}

		nodeName, nodePassword, err := getNodeInfo(req)
		if err != nil {
			sendError(err, resp)
			return
		}

		if err := ensureNodePassword(server.Runtime.NodePasswdFile, nodeName, nodePassword); err != nil {
			sendError(err, resp, http.StatusForbidden)
			return
		}

		caCert, caKey, key, err := getCACertAndKeys(server.Runtime.ClientCA, server.Runtime.ClientCAKey, server.Runtime.ClientKubeletKey)
		if err != nil {
			sendError(err, resp)
			return
		}

		cert, err := certutil.NewSignedCert(certutil.Config{
			CommonName:   "system:node:" + nodeName,
			Organization: []string{"system:nodes"},
			Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}, key, caCert[0], caKey)
		if err != nil {
			sendError(err, resp)
			return
		}

		keyBytes, err := ioutil.ReadFile(keyFile)
		if err != nil {
			http.Error(resp, err.Error(), http.StatusInternalServerError)
			return
		}

		resp.Write(append(certutil.EncodeCertPEM(cert), certutil.EncodeCertPEM(caCert[0])...))
		resp.Write(keyBytes)
	})
}

func fileHandler(fileName ...string) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if req.TLS == nil {
			resp.WriteHeader(http.StatusNotFound)
			return
		}
		resp.Header().Set("Content-Type", "text/plain")

		if len(fileName) == 1 {
			http.ServeFile(resp, req, fileName[0])
			return
		}

		for _, f := range fileName {
			bytes, err := ioutil.ReadFile(f)
			if err != nil {
				logrus.Errorf("Failed to read %s: %v", f, err)
				resp.WriteHeader(http.StatusInternalServerError)
				return
			}
			resp.Write(bytes)
		}
	})
}

func configHandler(server *config.Control) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if req.TLS == nil {
			resp.WriteHeader(http.StatusNotFound)
			return
		}
		resp.Header().Set("content-type", "application/json")
		json.NewEncoder(resp).Encode(server)
	})
}

func ping() http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		data := []byte("pong")
		resp.Header().Set("Content-Type", "text/plain")
		resp.Header().Set("Content-Length", strconv.Itoa(len(data)))
		resp.Write(data)
	})
}

func serveStatic(urlPrefix, staticDir string) http.Handler {
	return http.StripPrefix(urlPrefix, http.FileServer(http.Dir(staticDir)))
}

func sendError(err error, resp http.ResponseWriter, status ...int) {
	code := http.StatusInternalServerError
	if len(status) == 1 {
		code = status[0]
	}

	logrus.Error(err)
	resp.WriteHeader(code)
	resp.Write([]byte(err.Error()))
}

func ensureNodePassword(passwdFile, nodeName, pass string) error {
	passwd, err := passwd.Read(passwdFile)
	if err != nil {
		return err
	}
	match, exists := passwd.Check(nodeName, pass)
	if exists {
		if !match {
			return fmt.Errorf("Node password validation failed for '%s', using passwd file '%s'", nodeName, passwdFile)
		}
		return nil
	}
	// If user doesn't exist we save this password for future validation
	passwd.EnsureUser(nodeName, "", pass)
	return passwd.Write(passwdFile)
}
