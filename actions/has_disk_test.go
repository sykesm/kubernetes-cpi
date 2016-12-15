package actions_test

import (
	"errors"

	"github.com/sykesm/kubernetes-cpi/actions"
	"github.com/sykesm/kubernetes-cpi/cpi"
	"github.com/sykesm/kubernetes-cpi/kubecluster/fakes"
	"k8s.io/client-go/1.4/pkg/api/v1"
	"k8s.io/client-go/1.4/testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("HasDisk", func() {
	var (
		fakeClient   *fakes.Client
		fakeProvider *fakes.ClientProvider

		diskFinder *actions.DiskFinder
	)

	BeforeEach(func() {
		fakeClient = fakes.NewClient(
			&v1.PersistentVolumeClaimList{
				Items: []v1.PersistentVolumeClaim{{
					ObjectMeta: v1.ObjectMeta{
						Name:      "disk-diskID-1",
						Namespace: "bosh-namespace",
						Labels:    map[string]string{"bosh.cloudfoundry.org/disk-id": "diskID-1"},
					},
				}},
			},
		)
		fakeClient.NamespaceReturns("bosh-namespace")

		fakeProvider = &fakes.ClientProvider{}
		fakeProvider.NewReturns(fakeClient, nil)

		diskFinder = &actions.DiskFinder{
			ClientProvider: fakeProvider,
		}
	})

	It("gets a client with the context from the DiskCID", func() {
		_, err := diskFinder.HasDisk(cpi.DiskCID("context-name:diskID-1"))
		Expect(err).NotTo(HaveOccurred())

		Expect(fakeProvider.NewCallCount()).To(Equal(1))
		Expect(fakeProvider.NewArgsForCall(0)).To(Equal("context-name"))
	})

	It("lists disks labled with the disk ID", func() {
		_, err := diskFinder.HasDisk(cpi.DiskCID("context-name:diskID-1"))
		Expect(err).NotTo(HaveOccurred())

		Expect(fakeClient.Actions()).To(HaveLen(1))
		listAction := fakeClient.Actions()[0].(testing.ListAction)
		Expect(listAction.GetListRestrictions().Labels.String()).To(Equal("bosh.cloudfoundry.org/disk-id=diskID-1"))
	})

	It("returns true when the disk is found", func() {
		found, err := diskFinder.HasDisk(cpi.DiskCID("context-name:diskID-1"))
		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeTrue())
	})

	It("returns false when the disk is found", func() {
		found, err := diskFinder.HasDisk(cpi.DiskCID("context-name:missing"))
		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeFalse())
	})

	Context("when the client cannot be created", func() {
		BeforeEach(func() {
			fakeProvider.NewReturns(nil, errors.New("welp"))
		})

		It("returns an error", func() {
			_, err := diskFinder.HasDisk(cpi.DiskCID("context-name:missing"))
			Expect(err).To(MatchError("welp"))
		})
	})

	Context("when the label can't be parsed", func() {
		It("returns an error", func() {
			_, err := diskFinder.HasDisk(cpi.DiskCID("context-name:%&^*****@*^"))
			Expect(err).To(HaveOccurred())
		})
	})
})
