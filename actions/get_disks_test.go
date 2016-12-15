package actions_test

import (
	"errors"

	"github.com/sykesm/kubernetes-cpi/actions"
	"github.com/sykesm/kubernetes-cpi/cpi"
	"github.com/sykesm/kubernetes-cpi/kubecluster/fakes"
	kubeerrors "k8s.io/client-go/1.4/pkg/api/errors"
	"k8s.io/client-go/1.4/pkg/api/unversioned"
	"k8s.io/client-go/1.4/pkg/api/v1"
	"k8s.io/client-go/1.4/pkg/runtime"
	"k8s.io/client-go/1.4/testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DiskGetter", func() {
	var (
		fakeClient   *fakes.Client
		fakeProvider *fakes.ClientProvider

		diskGetter *actions.DiskGetter
	)

	BeforeEach(func() {
		fakeClient = fakes.NewClient(
			&v1.Pod{
				ObjectMeta: v1.ObjectMeta{
					Name:      "agent-agentID",
					Namespace: "bosh-namespace",
					Labels:    map[string]string{"bosh.cloudfoundry.org/agent-id": "agentID"},
				},
				Spec: v1.PodSpec{
					Volumes: []v1.Volume{{
						Name: "bosh-config",
						VolumeSource: v1.VolumeSource{
							ConfigMap: &v1.ConfigMapVolumeSource{},
						},
					}, {
						Name: "bosh-ephemeral",
						VolumeSource: v1.VolumeSource{
							EmptyDir: &v1.EmptyDirVolumeSource{},
						},
					}, {
						Name: "disk-diskID-1",
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{ClaimName: "disk-diskID-1"},
						},
					}, {
						Name: "disk-diskID-2",
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{ClaimName: "disk-diskID-2"},
						},
					}, {
						Name: "disk-nolabel",
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{ClaimName: "disk-nolabel"},
						},
					}},
				},
			},
			&v1.PersistentVolumeClaimList{
				Items: []v1.PersistentVolumeClaim{{
					ObjectMeta: v1.ObjectMeta{
						Name:      "disk-diskID-1",
						Namespace: "bosh-namespace",
						Labels:    map[string]string{"bosh.cloudfoundry.org/disk-id": "diskID-1"},
					},
				}, {
					ObjectMeta: v1.ObjectMeta{
						Name:      "disk-diskID-2",
						Namespace: "bosh-namespace",
						Labels:    map[string]string{"bosh.cloudfoundry.org/disk-id": "diskID-2-label-value"},
					},
				}, {
					ObjectMeta: v1.ObjectMeta{
						Name:      "disk-nolabel",
						Namespace: "bosh-namespace",
					},
				}},
			},
		)
		fakeClient.NamespaceReturns("bosh-namespace")

		fakeProvider = &fakes.ClientProvider{}
		fakeProvider.NewReturns(fakeClient, nil)

		diskGetter = &actions.DiskGetter{
			ClientProvider: fakeProvider,
		}
	})

	It("gets a client with the context from the DiskCID", func() {
		_, err := diskGetter.GetDisks(cpi.VMCID("context-name:agentID"))
		Expect(err).NotTo(HaveOccurred())

		Expect(fakeProvider.NewCallCount()).To(Equal(1))
		Expect(fakeProvider.NewArgsForCall(0)).To(Equal("context-name"))
	})

	It("retrieves the pod by name", func() {
		_, err := diskGetter.GetDisks(cpi.VMCID("context-name:agentID"))
		Expect(err).NotTo(HaveOccurred())

		matches := fakeClient.MatchingActions("get", "pods")
		Expect(matches).To(HaveLen(1))
		Expect(matches[0].(testing.GetAction).GetName()).To(Equal("agent-agentID"))
	})

	It("retrieves pv claims", func() {
		_, err := diskGetter.GetDisks(cpi.VMCID("context-name:agentID"))
		Expect(err).NotTo(HaveOccurred())

		matches := fakeClient.MatchingActions("get", "persistentvolumeclaims")
		Expect(matches).To(HaveLen(3))
		Expect(matches[0].(testing.GetAction).GetName()).To(Equal("disk-diskID-1"))
		Expect(matches[1].(testing.GetAction).GetName()).To(Equal("disk-diskID-2"))
		Expect(matches[2].(testing.GetAction).GetName()).To(Equal("disk-nolabel"))
	})

	It("returns cloud IDs of claimed disks", func() {
		disks, err := diskGetter.GetDisks(cpi.VMCID("context-name:agentID"))
		Expect(err).NotTo(HaveOccurred())

		Expect(disks).To(ConsistOf(
			cpi.DiskCID("context-name:diskID-1"),
			cpi.DiskCID("context-name:diskID-2-label-value"),
		))
	})

	Context("when the pod isn't found", func() {
		It("returns an empty list", func() {
			disks, err := diskGetter.GetDisks(cpi.VMCID("context-name:missing"))
			Expect(err).NotTo(HaveOccurred())
			Expect(disks).NotTo(BeNil())
			Expect(disks).To(BeEmpty())
		})
	})

	Context("when getting the pod fails", func() {
		BeforeEach(func() {
			fakeClient.PrependReactor("get", "pods", func(action testing.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("get-pod-welp")
			})
		})

		It("returns an error", func() {
			_, err := diskGetter.GetDisks(cpi.VMCID("context-name:agentID"))
			Expect(err).To(MatchError("get-pod-welp"))
		})
	})

	Context("when a pv claim can't be found", func() {
		BeforeEach(func() {
			fakeClient.PrependReactor("get", "persistentvolumeclaims", func(action testing.Action) (bool, runtime.Object, error) {
				if action.(testing.GetAction).GetName() == "disk-diskID-1" {
					return true, nil, kubeerrors.NewNotFound(unversioned.GroupResource{}, "disk-diskID-1")
				}
				return false, nil, nil
			})
		})

		It("it ignores the error", func() {
			disks, err := diskGetter.GetDisks(cpi.VMCID("context-name:agentID"))
			Expect(err).NotTo(HaveOccurred())
			Expect(disks).To(ConsistOf(cpi.DiskCID("context-name:diskID-2-label-value")))
		})
	})

	Context("when getting a pv claim fails", func() {
		BeforeEach(func() {
			fakeClient.PrependReactor("get", "persistentvolumeclaims", func(action testing.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("get-pvc-welp")
			})
		})

		It("returns an error", func() {
			_, err := diskGetter.GetDisks(cpi.VMCID("context-name:agentID"))
			Expect(err).To(MatchError("get-pvc-welp"))
		})
	})
})
