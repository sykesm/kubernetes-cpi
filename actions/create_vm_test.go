package actions_test

import (
	"encoding/json"
	"errors"

	kubeerrors "k8s.io/client-go/1.4/pkg/api/errors"
	"k8s.io/client-go/1.4/pkg/api/resource"
	"k8s.io/client-go/1.4/pkg/api/unversioned"
	"k8s.io/client-go/1.4/pkg/api/v1"
	"k8s.io/client-go/1.4/pkg/runtime"
	"k8s.io/client-go/1.4/testing"

	"github.com/sykesm/kubernetes-cpi/actions"
	"github.com/sykesm/kubernetes-cpi/agent"
	"github.com/sykesm/kubernetes-cpi/config"
	"github.com/sykesm/kubernetes-cpi/cpi"
	"github.com/sykesm/kubernetes-cpi/kubecluster/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateVM", func() {
	var (
		fakeClient   *fakes.Client
		fakeProvider *fakes.ClientProvider
		agentConf    *config.Agent

		agentID  string
		env      cpi.Environment
		networks cpi.Networks

		vmCreator *actions.VMCreator
	)

	BeforeEach(func() {
		fakeClient = fakes.NewClient()
		fakeClient.ContextReturns("bosh")
		fakeClient.NamespaceReturns("bosh-namespace")

		fakeProvider = &fakes.ClientProvider{}
		fakeProvider.NewReturns(fakeClient, nil)

		agentConf = &config.Agent{
			Blobstore:  "some-blbostore-config",
			MessageBus: "message-bus-url",
			NTPServers: []string{"1.example.org", "2.example.org"},
		}

		vmCreator = &actions.VMCreator{
			ClientProvider: fakeProvider,
			AgentConfig:    agentConf,
		}

		agentID = "agent-id"
		env = cpi.Environment{"passed": "along"}
		networks = cpi.Networks{
			"network-1": cpi.Network{
				Type:    "manual",
				IP:      "1.2.3.4",
				Netmask: "255.255.0.0",
				Gateway: "1.2.0.1",
				DNS:     []string{"8.8.8.8", "8.8.4.4"},
				Default: []string{"dns", "gateway"},
				CloudProperties: map[string]interface{}{
					"key": "value",
				},
			},
			"network-2": cpi.Network{
				Type: "dynamic",
				CloudProperties: map[string]interface{}{
					"dynamic-key": "dynamic-value",
				},
			},
		}
	})

	Describe("Create", func() {
		var (
			stemcellCID cpi.StemcellCID
			cloudProps  actions.VMCloudProperties
			diskCIDs    []cpi.DiskCID
		)

		BeforeEach(func() {
			stemcellCID = cpi.StemcellCID("sykesm/kubernetes-stemcell:999")
			cloudProps = actions.VMCloudProperties{Context: "bosh"}
			diskCIDs = []cpi.DiskCID{}
		})

		It("returns a VM Cloud ID", func() {
			vmcid, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
			Expect(err).NotTo(HaveOccurred())
			Expect(vmcid).To(Equal(actions.NewVMCID("bosh", agentID)))
		})

		It("gets a client with the context from the cloud properties", func() {
			_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeProvider.NewCallCount()).To(Equal(1))
			Expect(fakeProvider.NewArgsForCall(0)).To(Equal("bosh"))
		})

		Context("when getting the client fails", func() {
			BeforeEach(func() {
				fakeProvider.NewReturns(nil, errors.New("boom"))
			})

			It("gets a client for the appropriate context", func() {
				_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
				Expect(err).To(MatchError("boom"))
			})
		})

		It("creates the target namespace", func() {
			_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
			Expect(err).NotTo(HaveOccurred())

			matches := fakeClient.MatchingActions("create", "namespaces")
			Expect(matches).To(HaveLen(1))

			namespace := matches[0].(testing.CreateAction).GetObject().(*v1.Namespace)
			Expect(namespace.Name).To(Equal("bosh-namespace"))
		})

		Context("when the namespace already exists", func() {
			BeforeEach(func() {
				fakeClient = fakes.NewClient(
					&v1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "bosh-namespace"}},
				)
				fakeClient.ContextReturns("bosh")
				fakeClient.NamespaceReturns("bosh-namespace")
				fakeProvider.NewReturns(fakeClient, nil)
			})

			It("skips namespace creation", func() {
				_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeClient.MatchingActions("get", "namespaces")).To(HaveLen(1))
				Expect(fakeClient.MatchingActions("create", "namespaces")).To(HaveLen(0))
			})
		})

		Context("when the namespace create fails with StatusReasonAlreadyExists", func() {
			BeforeEach(func() {
				fakeClient.PrependReactor("create", "namespaces", func(action testing.Action) (bool, runtime.Object, error) {
					gr := unversioned.GroupResource{Group: "", Resource: "namespaces"}
					return true, nil, kubeerrors.NewAlreadyExists(gr, "bosh-namespace")
				})
			})

			It("keeps calm and carries on", func() {
				_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeClient.MatchingActions("get", "namespaces")).To(HaveLen(1))
				Expect(fakeClient.MatchingActions("create", "namespaces")).To(HaveLen(1))
			})
		})

		Context("when the namespace create fails", func() {
			BeforeEach(func() {
				fakeClient.PrependReactor("create", "namespaces", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("namespace-welp")
				})
			})

			It("returns an error", func() {
				_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
				Expect(err).To(MatchError("namespace-welp"))
				Expect(fakeClient.MatchingActions("create", "namespaces")).To(HaveLen(1))
			})
		})

		It("creates the config map for agent settings", func() {
			_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
			Expect(err).NotTo(HaveOccurred())

			matches := fakeClient.MatchingActions("create", "configmaps")
			Expect(matches).To(HaveLen(1))

			instanceSettings, err := vmCreator.InstanceSettings(agentID, networks, env)
			Expect(err).NotTo(HaveOccurred())
			instanceJSON, err := json.Marshal(instanceSettings)
			Expect(err).NotTo(HaveOccurred())

			configMap := matches[0].(testing.CreateAction).GetObject().(*v1.ConfigMap)
			Expect(configMap.Name).To(Equal("agent-" + agentID))
			Expect(configMap.Labels["bosh.cloudfoundry.org/agent-id"]).To(Equal(agentID))
			Expect(configMap.Data["instance_settings"]).To(MatchJSON(instanceJSON))
		})

		Context("when the config map create fails", func() {
			BeforeEach(func() {
				fakeClient.PrependReactor("create", "configmaps", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("configmap-welp")
				})
			})

			It("returns an error", func() {
				_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
				Expect(err).To(MatchError("configmap-welp"))
				Expect(fakeClient.MatchingActions("create", "configmaps")).To(HaveLen(1))
			})
		})

		It("creates a NodePort service for the agent", func() {
			_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
			Expect(err).NotTo(HaveOccurred())

			matches := fakeClient.MatchingActions("create", "services")
			Expect(matches).To(HaveLen(1))

			service := matches[0].(testing.CreateAction).GetObject().(*v1.Service)
			Expect(service.Name).To(Equal("agent-" + agentID))
			Expect(service.Labels["bosh.cloudfoundry.org/agent-id"]).To(Equal(agentID))
			Expect(service.Spec).To(Equal(v1.ServiceSpec{
				Type:  v1.ServiceTypeNodePort,
				Ports: []v1.ServicePort{{NodePort: 32068, Port: 6868}},
				Selector: map[string]string{
					"bosh.cloudfoundry.org/agent-id": agentID,
				},
			}))
		})

		Context("when the service create fails", func() {
			BeforeEach(func() {
				fakeClient.PrependReactor("create", "services", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("service-welp")
				})
			})

			It("returns an error", func() {
				_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
				Expect(err).To(MatchError("service-welp"))
				Expect(fakeClient.MatchingActions("create", "services")).To(HaveLen(1))
			})
		})

		It("creates a pod", func() {
			_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
			Expect(err).NotTo(HaveOccurred())

			matches := fakeClient.MatchingActions("create", "pods")
			Expect(matches).To(HaveLen(1))

			trueValue := true
			rootUID := int64(0)
			resourceRequest := v1.ResourceList{
				v1.ResourceMemory: resource.MustParse("1Gi"),
			}

			pod := matches[0].(testing.CreateAction).GetObject().(*v1.Pod)
			Expect(pod.Name).To(Equal("agent-" + agentID))
			Expect(pod.Labels["bosh.cloudfoundry.org/agent-id"]).To(Equal(agentID))
			Expect(pod.Spec).To(Equal(v1.PodSpec{
				Hostname: agentID,
				Containers: []v1.Container{{
					Name:            "bosh-job",
					Image:           "sykesm/kubernetes-stemcell:999",
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
			}))
		})

		Context("when creating the pod fails", func() {
			BeforeEach(func() {
				fakeClient.PrependReactor("create", "pods", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("pods-welp")
				})
			})

			It("returns an error", func() {
				_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
				Expect(err).To(MatchError("pods-welp"))
				Expect(fakeClient.MatchingActions("create", "services")).To(HaveLen(1))
			})
		})
	})

	Describe("InstanceSettings", func() {
		It("copies the blobstore from the agent config", func() {
			agentSettings, err := vmCreator.InstanceSettings(agentID, networks, env)
			Expect(err).NotTo(HaveOccurred())
			Expect(agentSettings.Blobstore).To(Equal(agentConf.Blobstore))
		})

		It("copies the message bus from the agent config", func() {
			agentSettings, err := vmCreator.InstanceSettings(agentID, networks, env)
			Expect(err).NotTo(HaveOccurred())
			Expect(agentSettings.MessageBus).To(Equal(agentConf.MessageBus))
		})

		It("copies the ntp servers from the agent config", func() {
			agentSettings, err := vmCreator.InstanceSettings(agentID, networks, env)
			Expect(err).NotTo(HaveOccurred())
			Expect(agentSettings.NTPServers).To(Equal(agentConf.NTPServers))
		})

		It("sets the agent ID", func() {
			agentSettings, err := vmCreator.InstanceSettings(agentID, networks, env)
			Expect(err).NotTo(HaveOccurred())
			Expect(agentSettings.AgentID).To(Equal(agentID))
		})

		It("sets the VM name", func() {
			agentSettings, err := vmCreator.InstanceSettings(agentID, networks, env)
			Expect(err).NotTo(HaveOccurred())
			Expect(agentSettings.VM).To(Equal(agent.VM{Name: agentID}))
		})

		It("propagates the bosh environment", func() {
			agentSettings, err := vmCreator.InstanceSettings(agentID, networks, env)
			Expect(err).NotTo(HaveOccurred())
			Expect(agentSettings.Env).To(Equal(env))
		})

		It("sets the network configuration", func() {
			agentSettings, err := vmCreator.InstanceSettings(agentID, networks, env)
			Expect(err).NotTo(HaveOccurred())
			Expect(agentSettings.Networks).To(Equal(agent.Networks{
				"network-1": agent.Network{
					Type:          "manual",
					IP:            "1.2.3.4",
					Netmask:       "255.255.0.0",
					Gateway:       "1.2.0.1",
					DNS:           []string{"8.8.8.8", "8.8.4.4"},
					Default:       []string{"dns", "gateway"},
					Preconfigured: true,
				},
				"network-2": agent.Network{
					Type:          "dynamic",
					Preconfigured: true,
				},
			}))
		})

		Context("when the networks fails to remarshal", func() {
			BeforeEach(func() {
				networks["network-2"].CloudProperties["channel"] = make(chan struct{})
			})

			It("returns an error", func() {
				_, err := vmCreator.InstanceSettings(agentID, networks, env)
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(&json.UnsupportedTypeError{}))
			})
		})
	})
})
