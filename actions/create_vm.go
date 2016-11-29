package actions

import (
	"encoding/json"

	"github.com/sykesm/kubernetes-cpi/agent"
	"github.com/sykesm/kubernetes-cpi/config"
	"github.com/sykesm/kubernetes-cpi/cpi"

	core "k8s.io/client-go/1.4/kubernetes/typed/core/v1"
	kubeerrors "k8s.io/client-go/1.4/pkg/api/errors"
	"k8s.io/client-go/1.4/pkg/api/resource"
	"k8s.io/client-go/1.4/pkg/api/unversioned"
	"k8s.io/client-go/1.4/pkg/api/v1"
)

type VMCreator struct {
	Config config.Config
}

type VMCloudProperties struct {
	Context   string `json:"context"`
	Namespace string `json:"namespace"`
}

func (v *VMCreator) Create(
	agentID string,
	stemcellCID cpi.StemcellCID,
	cloudProps VMCloudProperties,
	networks cpi.Networks,
	diskCIDs []cpi.DiskCID,
	env cpi.Environment,
) (cpi.VMCID, error) {
	if len(cloudProps.Context) == 0 {
		cloudProps.Context = v.Config.Context()
	}

	if len(cloudProps.Namespace) == 0 {
		cloudProps.Namespace = v.Config.Namespace()
	}

	// create the client set
	clientSet, err := v.Config.NewClient(cloudProps.Context)
	if err != nil {
		return "", err
	}

	// create the target namespace if it doesn't already exist
	err = createNamespace(clientSet.Core(), cloudProps.Namespace)
	if err != nil {
		return "", err
	}

	instanceSettings, err := v.InstanceSettings(agentID, networks, env)
	if err != nil {
		return "", err
	}

	// create the config map
	_, err = createConfigMap(clientSet.Core().ConfigMaps(cloudProps.Namespace), agentID, instanceSettings)
	if err != nil {
		return "", err
	}

	// create the service
	_, err = createService(clientSet.Core().Services(cloudProps.Namespace), agentID, "")
	if err != nil {
		return "", err
	}

	// create the pod
	_, err = createPod(clientSet.Core().Pods(cloudProps.Namespace), agentID, string(stemcellCID))
	if err != nil {
		return "", err
	}

	return cpi.VMCID("foo"), nil
}

func (v *VMCreator) InstanceSettings(agentID string, networks cpi.Networks, env cpi.Environment) (*agent.Settings, error) {
	agentNetworks := agent.Networks{}
	for name, cpiNetwork := range networks {
		agentNetwork := agent.Network{}
		if err := cpi.Remarshal(cpiNetwork, &agentNetwork); err != nil {
			return nil, err
		}
		agentNetwork.Preconfigured = true
		agentNetworks[name] = agentNetwork
	}

	settings := &agent.Settings{
		AgentID: agentID,
		VM:      agent.VM{Name: agentID},
		Env:     env,

		Networks: agentNetworks,
		Disks: agent.Disks{
			Persistent: map[string]string{
				"not-implemented": "/mnt/persistent",
			},
		},

		// TODO: Get from config file
		NTPServers: []string{"0.pool.ntp.org", "1.pool.ntp.org"},
		MessageBus: "https://admin:adminpass@0.0.0.0:6868",
		Blobstore: agent.Blobstore{
			Type: "local",
			Options: map[string]interface{}{
				"blobstore_path": "/var/vcap/micro_bosh/data/cache",
			},
		},
	}
	return settings, nil
}

func createNamespace(coreClient core.CoreInterface, namespace string) error {
	_, err := coreClient.Namespaces().Get(namespace)
	if err == nil {
		return nil
	}

	_, err = coreClient.Namespaces().Create(&v1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: namespace},
	})
	if err == nil {
		return nil
	}

	if statusError, ok := err.(*kubeerrors.StatusError); ok {
		if statusError.Status().Reason == unversioned.StatusReasonAlreadyExists {
			return nil
		}
	}
	return err
}

func createConfigMap(configMapService core.ConfigMapInterface, agentID string, instanceSettings *agent.Settings) (*v1.ConfigMap, error) {
	instanceJSON, err := json.Marshal(instanceSettings)
	if err != nil {
		return nil, err
	}

	return configMapService.Create(&v1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name: "agent-" + agentID,
			Labels: map[string]string{
				"bosh.cloudfoundry.org/agent-id": agentID,
			},
		},
		Data: map[string]string{
			"instance_settings": string(instanceJSON),
		},
	})
}

func createService(serviceClient core.ServiceInterface, agentID string, vip string) (*v1.Service, error) {
	// Need to provide a way to explicitly associate services.
	// For the director, we will need 22 (ssh) and 25555 (director).
	// During bosh-init, the agent will need to expose 6868.
	return serviceClient.Create(&v1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name: "agent-" + agentID,
			Labels: map[string]string{
				"bosh.cloudfoundry.org/agent-id": agentID,
			},
		},
		Spec: v1.ServiceSpec{
			Type: v1.ServiceTypeNodePort,
			Ports: []v1.ServicePort{{
				NodePort: 32068, // FIXME
				Port:     6868,
			}},
			ClusterIP: vip,
			Selector: map[string]string{
				"bosh.cloudfoundry.org/agent-id": agentID,
			},
		},
	})
}

func createPod(podClient core.PodInterface, agentID string, image string) (*v1.Pod, error) {
	trueValue := true
	rootUID := int64(0)

	resourceRequest := v1.ResourceList{
		v1.ResourceMemory: resource.MustParse("1Gi"),
	}

	return podClient.Create(&v1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name: "agent-" + agentID,
			Labels: map[string]string{
				"bosh.cloudfoundry.org/agent-id": agentID,
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{{
				Name:            "bosh-job",
				Image:           image,
				ImagePullPolicy: v1.PullAlways,
				Command:         []string{"/usr/sbin/runsvdir-start"},
				Args:            []string{},
				Resources: v1.ResourceRequirements{
					Limits:   resourceRequest,
					Requests: resourceRequest,
				},
				SecurityContext: &v1.SecurityContext{
					Privileged: &trueValue,
					RunAsUser:  &rootUID,
				},
				VolumeMounts: []v1.VolumeMount{{
					Name:      "bosh-config",
					MountPath: "/var/vcap/bosh/instance_settings.json",
					SubPath:   "instance_settings.json",
				}, {
					Name:      "agent-pv",
					MountPath: "/mnt/persistent",
				}},
			}},
			Volumes: []v1.Volume{{
				Name: "bosh-config",
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{
						LocalObjectReference: v1.LocalObjectReference{
							Name: "agent-" + agentID,
						},
						Items: []v1.KeyToPath{{
							Key:  "instance_settings",
							Path: "instance_settings.json",
						}},
					},
				},
			}, {
				Name: "agent-pv",
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						ClaimName: "agent-pv-claim",
					},
				},
			}},
		},
	})
}
