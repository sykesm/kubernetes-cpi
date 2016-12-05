package actions

import (
	"encoding/json"
	"strings"

	"github.com/sykesm/kubernetes-cpi/cpi"
	"github.com/sykesm/kubernetes-cpi/kubecluster"
	"k8s.io/client-go/1.4/pkg/api"
	"k8s.io/client-go/1.4/pkg/util/strategicpatch"
	"k8s.io/client-go/1.4/pkg/util/validation"
)

type VMMetadataSetter struct {
	ClientProvider kubecluster.ClientProvider
}

func (v *VMMetadataSetter) SetVMMetadata(vmcid cpi.VMCID, metadata map[string]string) error {
	context, agentID := ParseVMCID(vmcid)

	client, err := v.ClientProvider.New(context)
	if err != nil {
		return err
	}

	pod, err := client.Pods().Get("agent-" + agentID)
	if err != nil {
		return err
	}

	old, err := json.Marshal(pod)
	if err != nil {
		return err
	}

	for k, v := range metadata {
		k = "bosh.cloudfoundry.org/" + strings.ToLower(k)
		if len(validation.IsQualifiedName(k)) == 0 && len(validation.IsValidLabelValue(v)) == 0 {
			pod.ObjectMeta.Labels[k] = v
		}
	}

	new, err := json.Marshal(pod)
	if err != nil {
		return err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(old, new, pod)
	if err != nil {
		return err
	}

	_, err = client.Pods().Patch(pod.Name, api.StrategicMergePatchType, patch)
	if err != nil {
		return err
	}

	return nil
}
