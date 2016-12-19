package actions_test

import (
	"errors"

	"k8s.io/client-go/1.4/pkg/api/resource"
	"k8s.io/client-go/1.4/pkg/api/v1"
	"k8s.io/client-go/1.4/pkg/runtime"
	"k8s.io/client-go/1.4/testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sykesm/kubernetes-cpi/actions"
	"github.com/sykesm/kubernetes-cpi/cpi"
	"github.com/sykesm/kubernetes-cpi/kubecluster/fakes"
)

var _ = Describe("DeleteDisk", func() {
	var (
		fakeClient   *fakes.Client
		fakeProvider *fakes.ClientProvider
		diskCID      cpi.DiskCID

		diskDeleter *actions.DiskDeleter
	)

	BeforeEach(func() {
		diskCID = actions.NewDiskCID("bosh", "disk-id")

		fakeClient = fakes.NewClient(&v1.PersistentVolumeClaim{
			ObjectMeta: v1.ObjectMeta{
				Name:      "disk-disk-id",
				Namespace: "bosh-namespace",
				Labels: map[string]string{
					"bosh.cloudfoundry.org/disk-id": "disk-id",
				},
			},
			Spec: v1.PersistentVolumeClaimSpec{
				AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceStorage: resource.MustParse("20Mi"),
					},
				},
			},
		})
		fakeClient.ContextReturns("bosh")
		fakeClient.NamespaceReturns("bosh-namespace")

		fakeProvider = &fakes.ClientProvider{}
		fakeProvider.NewReturns(fakeClient, nil)

		diskDeleter = &actions.DiskDeleter{ClientProvider: fakeProvider}
	})

	It("gets a client for the appropriate context", func() {
		err := diskDeleter.DeleteDisk(diskCID)
		Expect(err).NotTo(HaveOccurred())

		Expect(fakeProvider.NewCallCount()).To(Equal(1))
		Expect(fakeProvider.NewArgsForCall(0)).To(Equal("bosh"))
	})

	It("deletes the persistent volume claim", func() {
		err := diskDeleter.DeleteDisk(diskCID)
		Expect(err).NotTo(HaveOccurred())

		matches := fakeClient.MatchingActions("delete", "persistentvolumeclaims")
		Expect(matches).To(HaveLen(1))

		Expect(matches[0].(testing.DeleteAction).GetName()).To(Equal("disk-disk-id"))
		Expect(matches[0].(testing.DeleteAction).GetNamespace()).To(Equal("bosh-namespace"))
	})

	Context("when getting the client fails", func() {
		BeforeEach(func() {
			fakeProvider.NewReturns(nil, errors.New("boom"))
		})

		It("gets a client for the appropriate context", func() {
			err := diskDeleter.DeleteDisk(diskCID)
			Expect(err).To(MatchError("boom"))
		})
	})

	Context("when deleting the pv claim fails", func() {
		BeforeEach(func() {
			fakeClient.PrependReactor("delete", "persistentvolumeclaims", func(action testing.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("pvc-welp")
			})
		})

		It("returns an error", func() {
			err := diskDeleter.DeleteDisk(diskCID)
			Expect(err).To(MatchError("pvc-welp"))
			Expect(fakeClient.MatchingActions("delete", "persistentvolumeclaims")).To(HaveLen(1))
		})
	})
})
