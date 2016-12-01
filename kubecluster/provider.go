package kubecluster

import (
	"k8s.io/client-go/1.4/kubernetes"
	"k8s.io/client-go/1.4/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/1.4/tools/clientcmd/api"
)

const DefaultContext = ""

//go:generate counterfeiter -o fakes/client_provider.go --fake-name ClientProvider . ClientProvider
type ClientProvider interface {
	New(context string) (Client, error)
}

type Provider struct {
	clientcmdapi.Config
}

func (p *Provider) New(context string) (Client, error) {
	if context == DefaultContext {
		context = p.Config.CurrentContext
	}

	kubeClientConfig := clientcmd.NewNonInteractiveClientConfig(
		p.Config,
		context,
		&clientcmd.ConfigOverrides{},
		&clientcmd.ClientConfigLoadingRules{},
	)

	restConfig, err := kubeClientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	ns, _, err := kubeClientConfig.Namespace()
	if err != nil {
		return nil, err
	}

	return &client{
		context:   context,
		namespace: ns,
		Clientset: kubeClient,
	}, nil
}
