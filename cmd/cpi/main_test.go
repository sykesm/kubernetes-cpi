package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("main", func() {

	Describe("CreateStemcell", func() {
		var input string

		BeforeEach(func() {
			input = `{
				"method": "create_stemcell",
				"arguments": [
					"/Users/cfdojo/.bosh_init/installations/c281af0b-7cf8-4094-772a-937cee3ec101/tmp/stemcell-manager488652969/image",
					{ "image": "sykesm/kubernetes-stemcell:3309" }
				],
				"context": {
					"director_uuid": "354b13ee-065a-4953-4436-77772e3854a8"
				}
			}`
		})

		It("does nothing", func() {
			Expect(true).To(BeTrue())
		})

		// It("unmarshals the command", func() {
		// 	type Context struct {
		// 		DirectorUUID string `json:"director_uuid"`
		// 	}
		// 	type CPICommand struct {
		// 		Method  string        `json:"method"`
		// 		Args    []interface{} `json:"arguments"`
		// 		Context Context       `json:"context"`
		// 	}

		// 	var command CPICommand
		// 	err := json.Unmarshal([]byte(input), &command)
		// 	Expect(err).NotTo(HaveOccurred())

		// 	type StemcellCloudProperties struct {
		// 		Image string `json:"image"`
		// 	}

		// 	var props StemcellCloudProperties
		// 	err = Remarshal(command.Args[1], &props)
		// 	Expect(err).NotTo(HaveOccurred())
		// })
	})
})
