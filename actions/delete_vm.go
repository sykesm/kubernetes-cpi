package actions

import (
	"github.com/sykesm/kubernetes-cpi/cpi"
	"github.com/sykesm/kubernetes-cpi/kubecluster"

	core "k8s.io/client-go/1.4/kubernetes/typed/core/v1"
	"k8s.io/client-go/1.4/pkg/api"
	kubeerrors "k8s.io/client-go/1.4/pkg/api/errors"
	"k8s.io/client-go/1.4/pkg/api/unversioned"
)

type VMDeleter struct {
	ClientProvider kubecluster.ClientProvider
}

func (v *VMDeleter) Delete(vmcid cpi.VMCID) error {
	context, agentID := ParseVMCID(vmcid)

	client, err := v.ClientProvider.New(context)
	if err != nil {
		return err
	}

	err = deletePod(client.Pods(), agentID)
	if err != nil {
		return err
	}

	err = deleteService(client.Services(), agentID)
	if err != nil {
		return err
	}

	err = deleteConfigMap(client.ConfigMaps(), agentID)
	if err != nil {
		return err
	}

	return nil
}

func deleteConfigMap(configMapService core.ConfigMapInterface, agentID string) error {
	err := configMapService.Delete("agent-"+agentID, &api.DeleteOptions{GracePeriodSeconds: int64Ptr(0)})
	if statusError, ok := err.(*kubeerrors.StatusError); ok {
		if statusError.Status().Reason == unversioned.StatusReasonNotFound {
			return nil
		}
	}
	return err
}

func deleteService(serviceClient core.ServiceInterface, agentID string) error {
	err := serviceClient.Delete("agent-"+agentID, &api.DeleteOptions{GracePeriodSeconds: int64Ptr(1)})
	if statusError, ok := err.(*kubeerrors.StatusError); ok {
		if statusError.Status().Reason == unversioned.StatusReasonNotFound {
			return nil
		}
	}
	return err
}

func deletePod(podClient core.PodInterface, agentID string) error {
	err := podClient.Delete("agent-"+agentID, &api.DeleteOptions{GracePeriodSeconds: int64Ptr(1)})
	if statusError, ok := err.(*kubeerrors.StatusError); ok {
		if statusError.Status().Reason == unversioned.StatusReasonNotFound {
			return nil
		}
	}
	return err
}

func int64Ptr(i int64) *int64 {
	return &i
}
