package actions

import (
	"encoding/json"

	"github.com/sykesm/kubernetes-cpi/agent"
	"github.com/sykesm/kubernetes-cpi/config"
	"github.com/sykesm/kubernetes-cpi/cpi"
	"github.com/sykesm/kubernetes-cpi/kubecluster"

	core "k8s.io/client-go/1.4/kubernetes/typed/core/v1"
	kubeerrors "k8s.io/client-go/1.4/pkg/api/errors"
	"k8s.io/client-go/1.4/pkg/api/resource"
	"k8s.io/client-go/1.4/pkg/api/unversioned"
	"k8s.io/client-go/1.4/pkg/api/v1"
)

type VMCreator struct {
	AgentConfig    *config.Agent
	ClientProvider kubecluster.ClientProvider
}

type Service struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	ClusterIP string `json:"cluster_ip"`
	Ports     []Port `json:"ports"`
}

type Port struct {
	Name     string `json:"name"`
	NodePort int32  `json:"node_port"`
	Port     int32  `json:"port"`
	Protocol string `json:"protocol"`
}

type VMCloudProperties struct {
	Context  string    `json:"context"`
	Services []Service `json:"services,omitempty"`
}

func (v *VMCreator) Create(
	agentID string,
	stemcellCID cpi.StemcellCID,
	cloudProps VMCloudProperties,
	networks cpi.Networks,
	diskCIDs []cpi.DiskCID,
	env cpi.Environment,
) (cpi.VMCID, error) {

	// create the client set
	client, err := v.ClientProvider.New(cloudProps.Context)
	if err != nil {
		return "", err
	}

	// create the target namespace if it doesn't already exist
	err = createNamespace(client.Core(), client.Namespace())
	if err != nil {
		return "", err
	}

	// NOTE: This is a workaround for the fake Clientset. This should be
	// removed once https://github.com/kubernetes/client-go/issues/48 is
	// resolved.
	ns := client.Namespace()
	instanceSettings, err := v.InstanceSettings(agentID, networks, env)
	if err != nil {
		return "", err
	}

	// create the config map
	_, err = createConfigMap(client.ConfigMaps(), ns, agentID, instanceSettings)
	if err != nil {
		return "", err
	}

	// create the service
	err = createServices(client.Services(), ns, agentID, cloudProps.Services)
	if err != nil {
		return "", err
	}

	// create the pod
	_, err = createPod(client.Pods(), ns, agentID, string(stemcellCID))
	if err != nil {
		return "", err
	}

	return NewVMCID(client.Context(), agentID), nil
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
		Blobstore:  v.AgentConfig.Blobstore,
		MessageBus: v.AgentConfig.MessageBus,
		NTPServers: v.AgentConfig.NTPServers,

		AgentID: agentID,
		VM:      agent.VM{Name: agentID},

		Env:      env,
		Networks: agentNetworks,
		Disks:    agent.Disks{},
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

func createConfigMap(configMapService core.ConfigMapInterface, ns, agentID string, instanceSettings *agent.Settings) (*v1.ConfigMap, error) {
	instanceJSON, err := json.Marshal(instanceSettings)
	if err != nil {
		return nil, err
	}

	return configMapService.Create(&v1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      "agent-" + agentID,
			Namespace: ns,
			Labels: map[string]string{
				"bosh.cloudfoundry.org/agent-id": agentID,
			},
		},
		Data: map[string]string{
			"instance_settings": string(instanceJSON),
		},
	})
}

func createServices(serviceClient core.ServiceInterface, ns, agentID string, services []Service) error {
	for _, svc := range services {
		serviceType := v1.ServiceTypeClusterIP
		if svc.Type == "NodePort" {
			serviceType = v1.ServiceTypeNodePort
		}

		var ports []v1.ServicePort
		for _, port := range svc.Ports {
			port := v1.ServicePort{
				Name:     port.Name,
				Protocol: v1.Protocol(port.Protocol),
				Port:     port.Port,
				NodePort: port.NodePort,
			}
			ports = append(ports, port)
		}

		service := &v1.Service{
			ObjectMeta: v1.ObjectMeta{
				Name:      svc.Name,
				Namespace: ns,
				Labels: map[string]string{
					"bosh.cloudfoundry.org/agent-id": agentID,
				},
			},
			Spec: v1.ServiceSpec{
				Type:      serviceType,
				ClusterIP: svc.ClusterIP,
				Ports:     ports,
				Selector: map[string]string{
					"bosh.cloudfoundry.org/agent-id": agentID,
				},
			},
		}

		_, err := serviceClient.Create(service)
		if err != nil {
			return err
		}
	}

	return nil
}

func createPod(podClient core.PodInterface, ns, agentID string, image string) (*v1.Pod, error) {
	trueValue := true
	rootUID := int64(0)

	resourceRequest := v1.ResourceList{
		v1.ResourceMemory: resource.MustParse("1Gi"),
	}

	return podClient.Create(&v1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      "agent-" + agentID,
			Namespace: ns,
			Labels: map[string]string{
				"bosh.cloudfoundry.org/agent-id": agentID,
			},
		},
		Spec: v1.PodSpec{
			Hostname: agentID,
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
					Name:      "bosh-ephemeral",
					MountPath: "/var/vcap/data",
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
				Name: "bosh-ephemeral",
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
			}},
		},
	})
}
