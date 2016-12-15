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

type DiskAttacher struct {
	ClientProvider kubecluster.ClientProvider

	Clock             clock.Clock
	PodReadyTimeout   time.Duration
	PostRecreateDelay time.Duration
}

func (d *DiskAttacher) AttachDisk(vmcid cpi.VMCID, diskCID cpi.DiskCID) error {
	vmContext, agentID := ParseVMCID(vmcid)
	context, diskID := ParseDiskCID(diskCID)
	if context != vmContext {
		return fmt.Errorf("Kubernetes disk and resource pool contexts must be the same: disk: %q, resource pool: %q", context, vmContext)
	}

	client, err := d.ClientProvider.New(context)
	if err != nil {
		return err
	}

	err = updateConfigMapDisks(client.ConfigMaps(), agentID, diskCID)
	if err != nil {
		return err
	}

	err = d.recreatePodWithDisk(client.Pods(), agentID, diskID)
	if err != nil {
		return err
	}

	return nil
}

func updateConfigMapDisks(configMapService core.ConfigMapInterface, agentID string, diskCID cpi.DiskCID) error {
	cm, err := configMapService.Get("agent-" + agentID)
	if err != nil {
		return err
	}

	var settings agent.Settings
	err = json.Unmarshal([]byte(cm.Data["instance_settings"]), &settings)
	if err != nil {
		return err
	}

	if settings.Disks.Persistent == nil {
		settings.Disks.Persistent = map[string]string{}
	}

	_, diskID := ParseDiskCID(diskCID)
	settings.Disks.Persistent[string(diskCID)] = "/mnt/" + diskID

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

func (d *DiskAttacher) recreatePodWithDisk(podService core.PodInterface, agentID string, diskID string) error {
	pod, err := podService.Get("agent-" + agentID)
	if err != nil {
		return err
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes, v1.Volume{
		Name: "disk-" + diskID,
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
				ClaimName: "disk-" + diskID,
			},
		},
	})

	for i, container := range pod.Spec.Containers {
		if container.Name != "bosh-job" {
			continue
		}

		container.VolumeMounts = append(container.VolumeMounts, v1.VolumeMount{
			Name:      "disk-" + diskID,
			MountPath: "/mnt/" + diskID,
		})
		pod.Spec.Containers[i] = container
		break
	}

	pod.ObjectMeta = v1.ObjectMeta{
		Name:      pod.Name,
		Namespace: pod.Namespace,
		Labels:    pod.Labels,
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

	ready, err := d.waitForPod(podService, agentID, updated.ResourceVersion)
	if err != nil {
		return err
	}

	if !ready {
		return errors.New("Pod recreate failed with a timeout")
	}

	// Hack - Need an agent readiness check that's real
	d.Clock.Sleep(d.PostRecreateDelay)

	return nil
}

func (d *DiskAttacher) waitForPod(podService core.PodInterface, agentID string, resourceVersion string) (bool, error) {
	agentSelector, err := labels.Parse("bosh.cloudfoundry.org/agent-id=" + agentID)
	if err != nil {
		return false, err
	}

	listOptions := api.ListOptions{
		LabelSelector:   agentSelector,
		ResourceVersion: resourceVersion,
		Watch:           true,
	}

	timer := d.Clock.NewTimer(d.PodReadyTimeout)
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
