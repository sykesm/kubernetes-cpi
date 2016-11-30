package actions

import (
	"github.com/sykesm/kubernetes-cpi/config"
	"github.com/sykesm/kubernetes-cpi/cpi"
	"k8s.io/client-go/1.4/pkg/api"
	"k8s.io/client-go/1.4/pkg/api/v1"
	"k8s.io/client-go/1.4/pkg/labels"
)

type VMFinder struct {
	KubeConfig config.Kubernetes
}

func (f *VMFinder) HasVM(vmcid cpi.VMCID) (bool, error) {
	_, pod, err := f.FindVM(vmcid)
	return pod != nil, err
}

func (f *VMFinder) FindVM(vmcid cpi.VMCID) (string, *v1.Pod, error) {
	agentSelector, err := labels.Parse("bosh.cloudfoundry.org/agent-id=" + string(vmcid))
	if err != nil {
		return "", nil, err
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
			return context, &podList.Items[0], nil
		}
	}

	return "", nil, errs[0]
}
