package fakes

import (
	"github.com/sykesm/kubernetes-cpi/kubecluster"
	"k8s.io/client-go/1.4/kubernetes/fake"
	core "k8s.io/client-go/1.4/kubernetes/typed/core/v1"
	"k8s.io/client-go/1.4/pkg/runtime"
)

//go:generate counterfeiter -o client_context.go --fake-name ClientContext . clientContext
type clientContext interface {
	Context() string
	Namespace() string
}

func NewClient(objects ...runtime.Object) *Client {
	return &Client{
		ClientContext: ClientContext{},
		Clientset:     *fake.NewSimpleClientset(objects...),
	}
}

var _ kubecluster.Client = NewClient()

// Client is a combination of a counterfeiter fake that exposes Namespace and Context
// and a Kubernetes fake.Clientset.
type Client struct {
	ClientContext
	fake.Clientset
}

func (c *Client) Pods() core.PodInterface {
	return c.Core().Pods(c.Namespace())
}

func (c *Client) Services() core.ServiceInterface {
	return c.Core().Services(c.Namespace())
}

func (c *Client) ConfigMaps() core.ConfigMapInterface {
	return c.Core().ConfigMaps(c.Namespace())
}
