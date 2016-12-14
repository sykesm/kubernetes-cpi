package actions_test

import (
	"errors"

	"k8s.io/client-go/1.4/pkg/api/resource"
	"k8s.io/client-go/1.4/pkg/api/v1"
	"k8s.io/client-go/1.4/pkg/runtime"
	"k8s.io/client-go/1.4/testing"

	"github.com/sykesm/kubernetes-cpi/actions"
	"github.com/sykesm/kubernetes-cpi/cpi"
	"github.com/sykesm/kubernetes-cpi/kubecluster/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateDisk", func() {
	var (
		fakeClient   *fakes.Client
		fakeProvider *fakes.ClientProvider
		vmcid        cpi.VMCID
		cloudProps   actions.CreateDiskCloudProperties

		diskCreator *actions.DiskCreator
	)

	BeforeEach(func() {
		fakeClient = fakes.NewClient()
		fakeClient.ContextReturns("bosh")
		fakeClient.NamespaceReturns("bosh-namespace")

		fakeProvider = &fakes.ClientProvider{}
		fakeProvider.NewReturns(fakeClient, nil)

		vmcid = actions.NewVMCID("bosh", "agent-id")
		cloudProps = actions.CreateDiskCloudProperties{
			Context: "bosh",
		}

		diskCreator = &actions.DiskCreator{
			ClientProvider:    fakeProvider,
			GUIDGeneratorFunc: func() (string, error) { return "disk-guid", nil },
		}
	})

	It("gets a client for the appropriate context", func() {
		_, err := diskCreator.CreateDisk(1000, cloudProps, vmcid)
		Expect(err).NotTo(HaveOccurred())

		Expect(fakeProvider.NewCallCount()).To(Equal(1))
		Expect(fakeProvider.NewArgsForCall(0)).To(Equal("bosh"))
	})

	It("creates a persistent volume claim", func() {
		diskCID, err := diskCreator.CreateDisk(1000, cloudProps, vmcid)
		Expect(err).NotTo(HaveOccurred())
		Expect(diskCID).To(Equal(cpi.DiskCID("bosh:disk-guid")))

		matches := fakeClient.MatchingActions("create", "persistentvolumeclaims")
		Expect(matches).To(HaveLen(1))

		createAction := matches[0].(testing.CreateAction)
		Expect(createAction.GetNamespace()).To(Equal("bosh-namespace"))

		pvc := createAction.GetObject().(*v1.PersistentVolumeClaim)
		Expect(pvc).To(Equal(&v1.PersistentVolumeClaim{
			ObjectMeta: v1.ObjectMeta{
				Name:      "disk-disk-guid",
				Namespace: "bosh-namespace",
				Labels: map[string]string{
					"bosh.cloudfoundry.org/disk-id": "disk-guid",
				},
			},
			Spec: v1.PersistentVolumeClaimSpec{
				AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceStorage: resource.MustParse("1000Mi"),
					},
				},
			},
		}))
	})

	Context("when getting the client fails", func() {
		BeforeEach(func() {
			fakeProvider.NewReturns(nil, errors.New("boom"))
		})

		It("gets a client for the appropriate context", func() {
			_, err := diskCreator.CreateDisk(1000, cloudProps, vmcid)
			Expect(err).To(MatchError("boom"))
		})
	})

	Context("when creating the persistent volume claim fails", func() {
		BeforeEach(func() {
			fakeClient.PrependReactor("create", "persistentvolumeclaims", func(action testing.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("create-pvc-welp")
			})
		})

		It("returns an error", func() {
			_, err := diskCreator.CreateDisk(1000, cloudProps, vmcid)
			Expect(err).To(MatchError("create-pvc-welp"))
			Expect(fakeClient.MatchingActions("create", "persistentvolumeclaims")).To(HaveLen(1))
		})
	})
})
