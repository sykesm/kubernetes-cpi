package actions

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"code.cloudfoundry.org/clock"

	"github.com/sykesm/kubernetes-cpi/agent"
	"github.com/sykesm/kubernetes-cpi/cpi"
	"github.com/sykesm/kubernetes-cpi/kubecluster"
	core "k8s.io/client-go/1.4/kubernetes/typed/core/v1"
	"k8s.io/client-go/1.4/pkg/api"
	"k8s.io/client-go/1.4/pkg/api/v1"
	"k8s.io/client-go/1.4/pkg/labels"
	"k8s.io/client-go/1.4/pkg/watch"
)

type VolumeManager struct {
	ClientProvider kubecluster.ClientProvider

	Clock             clock.Clock
	PodReadyTimeout   time.Duration
	PostRecreateDelay time.Duration
}

type Operation int

const (
	Add Operation = iota
	Remove
)

func (v *VolumeManager) AttachDisk(vmcid cpi.VMCID, diskCID cpi.DiskCID) error {
	vmContext, agentID := ParseVMCID(vmcid)
	context, diskID := ParseDiskCID(diskCID)
	if context != vmContext {
		return fmt.Errorf("Kubernetes disk and resource pool contexts must be the same: disk: %q, resource pool: %q", context, vmContext)
	}

	client, err := v.ClientProvider.New(context)
	if err != nil {
		return err
	}

	err = v.recreatePod(client, Add, agentID, diskID)
	if err != nil {
		return err
	}

	return nil
}

func (v *VolumeManager) DetachDisk(vmcid cpi.VMCID, diskCID cpi.DiskCID) error {
	vmContext, agentID := ParseVMCID(vmcid)
	context, diskID := ParseDiskCID(diskCID)
	if context != vmContext {
		return fmt.Errorf("Kubernetes disk and resource pool contexts must be the same: disk: %q, resource pool: %q", context, vmContext)
	}

	client, err := v.ClientProvider.New(context)
	if err != nil {
		return err
	}

	err = v.recreatePod(client, Remove, agentID, diskID)
	if err != nil {
		return err
	}

	return nil
}

func (v *VolumeManager) recreatePod(client kubecluster.Client, op Operation, agentID string, diskID string) error {
	podService := client.Pods()
	pod, err := podService.Get("agent-" + agentID)
	if err != nil {
		return err
	}

	err = updateConfigMapDisks(client, op, agentID, diskID)
	if err != nil {
		return err
	}

	updateVolumes(op, &pod.Spec, diskID)
	pod.ObjectMeta = v1.ObjectMeta{
		Name:        pod.Name,
		Namespace:   pod.Namespace,
		Annotations: pod.Annotations,
		Labels:      pod.Labels,
	}
	pod.Status = v1.PodStatus{}

	err = podService.Delete("agent-"+agentID, &api.DeleteOptions{GracePeriodSeconds: int64Ptr(0)})
	if err != nil {
		return err
	}

	updated, err := podService.Create(pod)
	if err != nil {
		return err
	}

	ready, err := v.waitForPod(podService, agentID, updated.ResourceVersion)
	if err != nil {
		return err
	}

	if !ready {
		return errors.New("Pod recreate failed with a timeout")
	}

	// TODO: Need an agent readiness check that's real
	v.Clock.Sleep(v.PostRecreateDelay)

	return nil
}

func updateConfigMapDisks(client kubecluster.Client, op Operation, agentID, diskID string) error {
	configMapService := client.ConfigMaps()
	cm, err := configMapService.Get("agent-" + agentID)
	if err != nil {
		return err
	}

	var settings agent.Settings
	err = json.Unmarshal([]byte(cm.Data["instance_settings"]), &settings)
	if err != nil {
		return err
	}

	diskCID := string(NewDiskCID(client.Context(), diskID))
	if settings.Disks.Persistent == nil {
		settings.Disks.Persistent = map[string]string{}
	}

	switch op {
	case Add:
		settings.Disks.Persistent[diskCID] = "/mnt/" + diskID
	case Remove:
		delete(settings.Disks.Persistent, diskCID)
	}

	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return err
	}

	cm.Data["instance_settings"] = string(settingsJSON)

	_, err = configMapService.Update(cm)
	if err != nil {
		return err
	}

	return nil
}

func updateVolumes(op Operation, spec *v1.PodSpec, diskID string) {
	switch op {
	case Add:
		addVolume(spec, diskID)
	case Remove:
		removeVolume(spec, diskID)
	}
}

func addVolume(spec *v1.PodSpec, diskID string) {
	spec.Volumes = append(spec.Volumes, v1.Volume{
		Name: "disk-" + diskID,
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
				ClaimName: "disk-" + diskID,
			},
		},
	})

	for i, c := range spec.Containers {
		if c.Name == "bosh-job" {
			spec.Containers[i].VolumeMounts = append(c.VolumeMounts, v1.VolumeMount{
				Name:      "disk-" + diskID,
				MountPath: "/mnt/" + diskID,
			})
			break
		}
	}
}

func removeVolume(spec *v1.PodSpec, diskID string) {
	for i, v := range spec.Volumes {
		if v.Name == "disk-"+diskID {
			spec.Volumes = append(spec.Volumes[:i], spec.Volumes[i+1:]...)
			break
		}
	}

	for i, c := range spec.Containers {
		if c.Name == "bosh-job" {
			for j, v := range c.VolumeMounts {
				if v.Name == "disk-"+diskID {
					spec.Containers[i].VolumeMounts = append(c.VolumeMounts[:j], c.VolumeMounts[j+1:]...)
					break
				}
			}
		}
	}
}

func (v *VolumeManager) waitForPod(podService core.PodInterface, agentID string, resourceVersion string) (bool, error) {
	agentSelector, err := labels.Parse("bosh.cloudfoundry.org/agent-id=" + agentID)
	if err != nil {
		return false, err
	}

	listOptions := api.ListOptions{
		LabelSelector:   agentSelector,
		ResourceVersion: resourceVersion,
		Watch:           true,
	}

	timer := v.Clock.NewTimer(v.PodReadyTimeout)
	defer timer.Stop()

	podWatch, err := podService.Watch(listOptions)
	if err != nil {
		return false, err
	}
	defer podWatch.Stop()

	for {
		select {
		case event := <-podWatch.ResultChan():
			switch event.Type {
			case watch.Modified:
				pod, ok := event.Object.(*v1.Pod)
				if !ok {
					return false, fmt.Errorf("Unexpected object type: %v", reflect.TypeOf(event.Object))
				}

				if isAgentContainerRunning(pod) {
					return true, nil
				}

			default:
				return false, fmt.Errorf("Unexpected pod watch event: %s", event.Type)
			}

		case <-timer.C():
			return false, nil
		}
	}
}

func isAgentContainerRunning(pod *v1.Pod) bool {
	if pod.Status.Phase != v1.PodRunning {
		return false
	}

	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Name == "bosh-job" {
			return containerStatus.Ready && containerStatus.State.Running != nil
		}
	}

	return false
}
