package actions

import (
	"github.com/sykesm/kubernetes-cpi/config"
	"github.com/sykesm/kubernetes-cpi/cpi"
	"k8s.io/client-go/1.4/pkg/api"
	"k8s.io/client-go/1.4/pkg/labels"
)

type VMFinder struct {
	KubeConfig  config.Kubernetes
	AgentConfig config.Agent
}

func (f *VMFinder) HasVM(vmcid cpi.VMCID) (bool, error) {
	agentSelector, err := labels.Parse("bosh.cloudfoundry.org/agent-id=" + string(vmcid))
	if err != nil {
		return false, err
	}

	contexts := []string{f.KubeConfig.DefaultContext()}
	for name := range f.KubeConfig.Contexts {
		if name != contexts[0] {
			contexts = append(contexts, name)
		}
	}

	errs := []error{}
	listOptions := api.ListOptions{LabelSelector: agentSelector}
	for _, context := range contexts {
		clientSet, err := f.KubeConfig.NewClient(context)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		podList, err := clientSet.Core().Pods(f.KubeConfig.Namespace(context)).List(listOptions)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if len(podList.Items) > 0 {
			return true, nil
		}
	}

	return false, errs[0]
}
