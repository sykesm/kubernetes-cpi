package kubecluster_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/sykesm/kubernetes-cpi/config"
	"github.com/sykesm/kubernetes-cpi/kubecluster"
	"k8s.io/client-go/1.4/pkg/api/v1"
)

var _ = Describe("Provider", func() {
	var server *ghttp.Server
	var provider *kubecluster.Provider

	BeforeEach(func() {
		server = ghttp.NewTLSServer()
		kubeConf := config.Kubernetes{
			Clusters: map[string]*config.Cluster{
				"test_cluster": &config.Cluster{
					InsecureSkipTLSVerify: true,
					Server:                server.URL(),
				},
			},
			AuthInfos: map[string]*config.AuthInfo{
				"default_user": &config.AuthInfo{
					Username: "default-user",
					Password: "default-password",
				},
				"test_user": &config.AuthInfo{
					Username: "user",
					Password: "password",
				},
			},
			Contexts: map[string]*config.Context{
				"default": &config.Context{
					Cluster:   "test_cluster",
					AuthInfo:  "default_user",
					Namespace: "default-context-namespace",
				},
				"test_context": &config.Context{
					Cluster:   "test_cluster",
					AuthInfo:  "test_user",
					Namespace: "test-context-namespace",
				},
				"no_namespace": &config.Context{
					Cluster:  "test_cluster",
					AuthInfo: "test_user",
				},
			},
			CurrentContext: "default",
		}

		provider = &kubecluster.Provider{
			Config: kubeConf.ClientConfig(),
		}
	})

	Context("when an empty context is specified", func() {
		BeforeEach(func() {
			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/api/v1/namespaces/default-context-namespace/pods/podname"),
				ghttp.VerifyBasicAuth("default-user", "default-password"),
				ghttp.RespondWithJSONEncoded(
					http.StatusOK,
					v1.Pod{ObjectMeta: v1.ObjectMeta{Name: "podname", Namespace: "default-context-namespace"}},
				),
			))
		})

		It("creates a kubernetes client for the default context", func() {
			client, err := provider.New("")
			Expect(err).NotTo(HaveOccurred())

			pod, err := client.Pods().Get("podname")
			Expect(err).NotTo(HaveOccurred())
			Expect(server.ReceivedRequests()).To(HaveLen(1))

			Expect(pod.Name).To(Equal("podname"))
			Expect(pod.Namespace).To(Equal("default-context-namespace"))
		})
	})

	Context("when a context name is specified", func() {
		BeforeEach(func() {
			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/api/v1/namespaces/test-context-namespace/pods/podname"),
				ghttp.VerifyBasicAuth("user", "password"),
				ghttp.RespondWithJSONEncoded(
					http.StatusOK,
					v1.Pod{ObjectMeta: v1.ObjectMeta{Name: "podname", Namespace: "test-context-namespace"}},
				),
			))
		})

		It("creates a kubernetes client for the defautl context", func() {
			client, err := provider.New("test_context")
			Expect(err).NotTo(HaveOccurred())

			pod, err := client.Pods().Get("podname")
			Expect(err).NotTo(HaveOccurred())
			Expect(server.ReceivedRequests()).To(HaveLen(1))

			Expect(pod.Name).To(Equal("podname"))
			Expect(pod.Namespace).To(Equal("test-context-namespace"))
		})
	})

	Context("when a context without a namespace is specified", func() {
		BeforeEach(func() {
			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/api/v1/namespaces/default/pods/podname"),
				ghttp.VerifyBasicAuth("user", "password"),
				ghttp.RespondWithJSONEncoded(
					http.StatusOK,
					v1.Pod{ObjectMeta: v1.ObjectMeta{Name: "podname", Namespace: "default"}},
				),
			))
		})

		It("creates a kubernetes client for the default context", func() {
			client, err := provider.New("no_namespace")
			Expect(err).NotTo(HaveOccurred())

			pod, err := client.Pods().Get("podname")
			Expect(err).NotTo(HaveOccurred())
			Expect(server.ReceivedRequests()).To(HaveLen(1))

			Expect(pod.Name).To(Equal("podname"))
			Expect(pod.Namespace).To(Equal("default"))
		})
	})

	Context("when an invalid context name is specified", func() {
		It("raises an error", func() {
			_, err := provider.New("does-not-exist")
			Expect(err).To(MatchError("invalid configuration: no configuration has been provided"))

			Expect(server.ReceivedRequests()).To(HaveLen(0))
		})
	})
})
