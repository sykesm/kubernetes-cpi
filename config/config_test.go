package config_test

import (
	"encoding/json"

	"k8s.io/client-go/1.4/pkg/runtime"
	clientcmdapi "k8s.io/client-go/1.4/tools/clientcmd/api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sykesm/kubernetes-cpi/config"
)

var _ = Describe("Config", func() {
	var configData []byte

	BeforeEach(func() {
		configData = []byte(`{
			"clusters": {
				"minikube": { "certificate_authority_data": "certificate-authority-data", "server": "https://192.168.64.17:8443" }
			},
			"contexts": {
				"minikube": { "cluster": "minikube", "user": "minikube", "namespace": "minikube" },
				"bosh": { "cluster": "minikube", "user": "minikube", "namespace": "bosh" }
			},
			"current_context": "minikube",
			"users": {
				"minikube": { "client_certificate_data": "client-certificate-data", "client_key_data": "client-key-data" }
			}
		}`)
	})

	It("deserializes the config file", func() {
		var conf config.Config
		err := json.Unmarshal([]byte(configData), &conf)
		Expect(err).NotTo(HaveOccurred())

		Expect(conf.Clusters).To(HaveLen(1))
		Expect(conf.Clusters["minikube"]).To(Equal(&config.Cluster{
			Server: "https://192.168.64.17:8443",
			CertificateAuthorityData: "certificate-authority-data",
		}))

		Expect(conf.Contexts).To(HaveLen(2))
		Expect(conf.Contexts["minikube"]).To(Equal(&config.Context{
			Cluster:   "minikube",
			AuthInfo:  "minikube",
			Namespace: "minikube",
		}))
		Expect(conf.Contexts["bosh"]).To(Equal(&config.Context{
			Cluster:   "minikube",
			AuthInfo:  "minikube",
			Namespace: "bosh",
		}))

		Expect(conf.AuthInfos).To(HaveLen(1))
		Expect(conf.AuthInfos["minikube"]).To(Equal(&config.AuthInfo{
			ClientCertificateData: "client-certificate-data",
			ClientKeyData:         "client-key-data",
		}))

		Expect(conf.CurrentContext).To(Equal("minikube"))
	})

	Describe("ClientConfig", func() {
		var conf config.Config

		BeforeEach(func() {
			conf = config.Config{
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
			cc := conf.ClientConfig()
			Expect(cc).To(Equal(&clientcmdapi.Config{
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

	Describe("DefaultClientConfig", func() {
		var conf config.Config

		BeforeEach(func() {
			err := json.Unmarshal([]byte(configData), &conf)
			Expect(err).NotTo(HaveOccurred())
		})

		It("wraps the result of ClientConfig", func() {
			defaultCC := conf.DefaultClientConfig()
			Expect(defaultCC.RawConfig()).To(Equal(*conf.ClientConfig()))
		})

		It("is associated with the current context's namespace", func() {
			Expect(conf.Contexts[conf.CurrentContext].Namespace).To(Equal("minikube"))

			ns, override, err := conf.DefaultClientConfig().Namespace()
			Expect(err).NotTo(HaveOccurred())
			Expect(ns).To(Equal("minikube"))
			Expect(override).To(BeFalse())
		})
	})

	Describe("NonInteractiveClientConfig", func() {
		var conf config.Config

		BeforeEach(func() {
			err := json.Unmarshal([]byte(configData), &conf)
			Expect(err).NotTo(HaveOccurred())
		})

		It("is associated with the current context's namespace", func() {
			Expect(conf.Contexts[conf.CurrentContext].Namespace).NotTo(Equal("bosh"))

			ns, override, err := conf.NonInteractiveClientConfig("bosh").Namespace()
			Expect(err).NotTo(HaveOccurred())
			Expect(ns).To(Equal("bosh"))
			Expect(override).To(BeFalse())
		})
	})
})
