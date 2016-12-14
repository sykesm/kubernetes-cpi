package kubecluster

import (
	"k8s.io/client-go/1.4/kubernetes"
	core "k8s.io/client-go/1.4/kubernetes/typed/core/v1"
)

type Client interface {
	Context() string
	Namespace() string

	Core() core.CoreInterface

	ConfigMaps() core.ConfigMapInterface
	PersistentVolumeClaims() core.PersistentVolumeClaimInterface
	Pods() core.PodInterface
	Services() core.ServiceInterface
}

type client struct {
	context   string
	namespace string

	*kubernetes.Clientset
}

var _ Client = &client{}

func (c *client) Context() string {
	return c.context
}

func (c *client) Namespace() string {
	return c.namespace
}

func (c *client) ConfigMaps() core.ConfigMapInterface {
	return c.Core().ConfigMaps(c.namespace)
}

func (c *client) PersistentVolumeClaims() core.PersistentVolumeClaimInterface {
	return c.Core().PersistentVolumeClaims(c.namespace)
}

func (c *client) Pods() core.PodInterface {
	return c.Core().Pods(c.namespace)
}

func (c *client) Services() core.ServiceInterface {
	return c.Core().Services(c.namespace)
}
