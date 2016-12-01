package config_test

import (
	"encoding/json"

	"k8s.io/client-go/1.4/pkg/runtime"
	clientcmdapi "k8s.io/client-go/1.4/tools/clientcmd/api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sykesm/kubernetes-cpi/config"
)

var _ = Describe("Kubernetes Config", func() {
	var configData []byte
	var kubeConf config.Kubernetes

	BeforeEach(func() {
		configData = []byte(`{
			"clusters": {
				"bosh": { "server": "https://192.168.64.17:8443", "insecure_skip_tls_verify": true },
				"minikube": { "certificate_authority_data": "certificate-authority-data", "server": "https://192.168.64.17:8443" }
			},
			"contexts": {
				"bosh": { "cluster": "bosh", "user": "bosh", "namespace": "bosh" },
				"minikube": { "cluster": "minikube", "user": "minikube", "namespace": "minikube" },
				"no-namespace": { "cluster": "bosh", "user": "minikube" }
			},
			"current_context": "minikube",
			"users": {
				"bosh": { "username": "user", "password": "password" },
				"minikube": { "client_certificate_data": "client-certificate-data", "client_key_data": "client-key-data" }
			}
		}`)

		err := json.Unmarshal([]byte(configData), &kubeConf)
		Expect(err).NotTo(HaveOccurred())
	})

	It("deserializes the config data", func() {
		Expect(kubeConf.Clusters).To(HaveLen(2))
		Expect(kubeConf.Clusters["bosh"]).To(Equal(&config.Cluster{
			Server:                "https://192.168.64.17:8443",
			InsecureSkipTLSVerify: true,
		}))
		Expect(kubeConf.Clusters["minikube"]).To(Equal(&config.Cluster{
			Server: "https://192.168.64.17:8443",
			CertificateAuthorityData: "certificate-authority-data",
		}))

		Expect(kubeConf.Contexts).To(HaveLen(3))
		Expect(kubeConf.Contexts["bosh"]).To(Equal(&config.Context{
			Cluster:   "bosh",
			AuthInfo:  "bosh",
			Namespace: "bosh",
		}))
		Expect(kubeConf.Contexts["minikube"]).To(Equal(&config.Context{
			Cluster:   "minikube",
			AuthInfo:  "minikube",
			Namespace: "minikube",
		}))
		Expect(kubeConf.Contexts["no-namespace"]).To(Equal(&config.Context{
			Cluster:  "bosh",
			AuthInfo: "minikube",
		}))

		Expect(kubeConf.AuthInfos).To(HaveLen(2))
		Expect(kubeConf.AuthInfos["bosh"]).To(Equal(&config.AuthInfo{
			Username: "user",
			Password: "password",
		}))
		Expect(kubeConf.AuthInfos["minikube"]).To(Equal(&config.AuthInfo{
			ClientCertificateData: "client-certificate-data",
			ClientKeyData:         "client-key-data",
		}))

		Expect(kubeConf.CurrentContext).To(Equal("minikube"))
	})

	Describe("ClientConfig", func() {
		BeforeEach(func() {
			kubeConf = config.Kubernetes{
				Clusters: map[string]*config.Cluster{
					"cluster1": &config.Cluster{Server: "server1"},
					"cluster2": &config.Cluster{Server: "server2", CertificateAuthorityData: "certificate-authority-data-2"},
				},
				AuthInfos: map[string]*config.AuthInfo{
					"user1": &config.AuthInfo{ClientCertificateData: "client-certificate-data", ClientKeyData: "client-key-data"},
					"user2": &config.AuthInfo{Token: "bearer-token"},
					"user3": &config.AuthInfo{Username: "username", Password: "password"},
				},
				Contexts: map[string]*config.Context{
					"context1": &config.Context{
						Cluster:   "cluster1",
						AuthInfo:  "user1",
						Namespace: "namespace1",
					},
					"context2": &config.Context{
						Cluster:   "cluster2",
						AuthInfo:  "user2",
						Namespace: "namespace2",
					},
				},
				CurrentContext: "current-context",
			}
		})

		It("returns an api client config", func() {
			cc := kubeConf.ClientConfig()
			Expect(cc).To(Equal(clientcmdapi.Config{
				Clusters: map[string]*clientcmdapi.Cluster{
					"cluster1": &clientcmdapi.Cluster{
						Server:     "server1",
						Extensions: map[string]runtime.Object{},
					},
					"cluster2": &clientcmdapi.Cluster{
						Server: "server2",
						CertificateAuthorityData: []byte("certificate-authority-data-2"),
						Extensions:               map[string]runtime.Object{},
					},
				},
				AuthInfos: map[string]*clientcmdapi.AuthInfo{
					"user1": &clientcmdapi.AuthInfo{
						ClientCertificateData: []byte("client-certificate-data"),
						ClientKeyData:         []byte("client-key-data"),
						Extensions:            map[string]runtime.Object{},
					},
					"user2": &clientcmdapi.AuthInfo{
						Token:      "bearer-token",
						Extensions: map[string]runtime.Object{},
					},
					"user3": &clientcmdapi.AuthInfo{
						Username:   "username",
						Password:   "password",
						Extensions: map[string]runtime.Object{},
					},
				},
				Contexts: map[string]*clientcmdapi.Context{
					"context1": &clientcmdapi.Context{
						Cluster:    "cluster1",
						AuthInfo:   "user1",
						Namespace:  "namespace1",
						Extensions: map[string]runtime.Object{},
					},
					"context2": &clientcmdapi.Context{
						Cluster:    "cluster2",
						AuthInfo:   "user2",
						Namespace:  "namespace2",
						Extensions: map[string]runtime.Object{},
					},
				},
				CurrentContext: "current-context",
				Extensions:     map[string]runtime.Object{},
				Preferences:    *clientcmdapi.NewPreferences(),
			}))
		})
	})
})
