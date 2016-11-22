package cpi_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sykesm/kubernetes-cpi/cpi"
)

var _ = Describe("Remarshal", func() {
	type TypedStruct struct {
		String  string                 `json:"string"`
		Int     int                    `json:"int"`
		Untyped map[string]interface{} `json:"untyped"`
	}

	It("remarshals a map of string:interface{} into the typed struct", func() {
		typed := TypedStruct{}
		untyped := map[string]interface{}{
			"string": "string",
			"int":    12345,
			"untyped": map[string]interface{}{
				"foo": "bar",
			},
			"extra": "goo",
		}

		err := cpi.Remarshal(untyped, &typed)
		Expect(err).NotTo(HaveOccurred())

		Expect(typed).To(Equal(TypedStruct{
			String: "string",
			Int:    12345,
			Untyped: map[string]interface{}{
				"foo": "bar",
			},
		}))
	})

	Context("when the source can't be marshaled", func() {
		It("returns an error", func() {
			untyped := map[string]interface{}{}
			ch := make(chan struct{})

			err := cpi.Remarshal(ch, &untyped)
			Expect(err).To(BeAssignableToTypeOf(&json.UnsupportedTypeError{}))
		})
	})

	Context("when the encoded json can't be unmarshaled into the target", func() {
		It("returns an error", func() {
			untyped := map[string]interface{}{}

			err := cpi.Remarshal("simple-string", &untyped)
			Expect(err).To(BeAssignableToTypeOf(&json.UnmarshalTypeError{}))
		})
	})
})
