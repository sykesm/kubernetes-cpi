package config_test

import (
	"encoding/json"
	"net/http"

	"k8s.io/client-go/1.4/pkg/api/v1"
	"k8s.io/client-go/1.4/pkg/runtime"
	clientcmdapi "k8s.io/client-go/1.4/tools/clientcmd/api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
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

	Describe("Context", func() {
		It("returns the name of the current context", func() {
			Expect(kubeConf.Context()).To(Equal("minikube"))
		})
	})

	Describe("Namespace", func() {
		It("returns the namespace from the current context", func() {
			Expect(kubeConf.Namespace()).To(Equal("minikube"))
		})

		Context("when the current context is missing a namespace", func() {
			BeforeEach(func() {
				kubeConf.CurrentContext = "no-namespace"
			})

			It("uses 'default' as the namespace", func() {
				Expect(kubeConf.Namespace()).To(Equal("default"))
			})
		})
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

	Describe("NonInteractiveClientConfig", func() {
		It("wraps the result of ClientConfig", func() {
			cc := kubeConf.NonInteractiveClientConfig("bosh")
			rawConfig, err := cc.RawConfig()
			Expect(err).NotTo(HaveOccurred())
			Expect(rawConfig).To(Equal(*kubeConf.ClientConfig()))
		})

		It("is associated with the requested context", func() {
			Expect(kubeConf.Contexts[kubeConf.CurrentContext].Namespace).NotTo(Equal("bosh"))

			ns, override, err := kubeConf.NonInteractiveClientConfig("bosh").Namespace()
			Expect(err).NotTo(HaveOccurred())
			Expect(ns).To(Equal("bosh"))
			Expect(override).To(BeFalse())
		})

		Context("when the requested context is empty", func() {
			It("is uses the default context", func() {
				Expect(kubeConf.Contexts[kubeConf.CurrentContext].Namespace).NotTo(Equal("bosh"))

				ns, override, err := kubeConf.NonInteractiveClientConfig("").Namespace()
				Expect(err).NotTo(HaveOccurred())
				Expect(ns).To(Equal("minikube"))
				Expect(override).To(BeFalse())
			})
		})
	})

	Describe("NewClient", func() {
		var server *ghttp.Server

		BeforeEach(func() {
			server = ghttp.NewTLSServer()
			kubeConf.Clusters["bosh"].Server = server.URL()
			kubeConf.Clusters["bosh"].InsecureSkipTLSVerify = true

			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/api/v1/namespaces/namespace/pods/podname"),
				ghttp.VerifyBasicAuth("user", "password"),
				ghttp.RespondWithJSONEncoded(
					http.StatusOK,
					v1.Pod{ObjectMeta: v1.ObjectMeta{Name: "podname", Namespace: "namespace"}},
				),
			))
		})

		It("creates a kubernetes client", func() {
			intf, err := kubeConf.NewClient("bosh")
			Expect(err).NotTo(HaveOccurred())
			Expect(intf).NotTo(BeNil())

			pod, err := intf.Core().Pods("namespace").Get("podname")
			Expect(err).NotTo(HaveOccurred())
			Expect(server.ReceivedRequests()).To(HaveLen(1))

			Expect(pod.Name).To(Equal("podname"))
			Expect(pod.Namespace).To(Equal("namespace"))
		})
	})
})
