package actions_test

import (
	"github.com/sykesm/kubernetes-cpi/actions"
	"github.com/sykesm/kubernetes-cpi/cpi"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Stemcell", func() {
	Describe("CreateStemcell", func() {
		var cloudProps actions.StemcellCloudProperties

		BeforeEach(func() {
			cloudProps = actions.StemcellCloudProperties{
				Image: "cloudfoundry/kubernetes-stemcell:999",
			}
		})

		It("returns the image as a stemcell ID", func() {
			stemcellCID, err := actions.CreateStemcell("/ignored/path", cloudProps)
			Expect(err).NotTo(HaveOccurred())
			Expect(stemcellCID).To(Equal(cpi.StemcellCID("cloudfoundry/kubernetes-stemcell:999")))
		})
	})

	Describe("DeleteStemcell", func() {
		var stemcellCID cpi.StemcellCID

		BeforeEach(func() {
			stemcellCID = cpi.StemcellCID("image-id:version")
		})

		It("succeeds without error", func() {
			Expect(actions.DeleteStemcell(stemcellCID)).To(Succeed())
		})
	})
})
