package cpi_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sykesm/kubernetes-cpi/cpi"
)

var _ = Describe("Dispatch", func() {
	var req *cpi.Request
	var delegate *Delegate

	BeforeEach(func() {
		delegate = &Delegate{}
		req = &cpi.Request{
			Method: "ignored",
			Args:   []interface{}{},
		}
	})

	Context("when the action has no arguments", func() {
		It("calls the target function", func() {
			resp, err := cpi.Dispatch(req, delegate.NoArgs)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Error).To(BeNil())

			Expect(delegate.CallCount).To(Equal(1))
		})
	})

	Context("when the action does not return an error", func() {
		It("calls the action and marshals the result", func() {
			resp, err := cpi.Dispatch(req, delegate.NoArgsReturnBool)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Error).To(BeNil())
			Expect(resp.Result).To(Equal(true))

			Expect(delegate.CallCount).To(Equal(1))
		})
	})

	Context("when the action only returns an error", func() {
		BeforeEach(func() {
			req.Args = []interface{}{"welp"}
		})

		It("calls the action function and marshals the result", func() {
			resp, err := cpi.Dispatch(req, delegate.ReturnErr)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Result).To(BeNil())
			Expect(resp.Error).NotTo(BeNil())
			Expect(resp.Error.Message).To(Equal("welp"))

			Expect(delegate.CallCount).To(Equal(1))
		})
	})

	Context("when the action takes more arguments than were provided", func() {
		It("returns an error", func() {
			_, err := cpi.Dispatch(req, delegate.OneStringArg)
			Expect(err).To(MatchError("Not enough arguments: have 0, want 1"))
		})

		It("does not call the action", func() {
			cpi.Dispatch(req, delegate.OneStringArg)
			Expect(delegate.CallCount).To(Equal(0))
		})
	})

	Context("when the action takes fewer arguments than were provided", func() {
		BeforeEach(func() {
			req.Args = []interface{}{"hello", "world"}
		})

		It("returns an error", func() {
			_, err := cpi.Dispatch(req, delegate.OneStringArg)
			Expect(err).To(MatchError("Too many arguments: have 2, want 1"))
		})

		It("does not call the action", func() {
			cpi.Dispatch(req, delegate.OneStringArg)
			Expect(delegate.CallCount).To(Equal(0))
		})
	})

	Context("when the action returns multiple values", func() {
		BeforeEach(func() {
			req.Args = []interface{}{}
		})

		It("returns an error", func() {
			_, err := cpi.Dispatch(req, delegate.LastNotError)
			Expect(err).To(MatchError("Action error is not an error"))
		})
	})

	Context("when the action returns multiple results and an error", func() {
		BeforeEach(func() {
			req.Args = []interface{}{}
		})

		It("returns an error", func() {
			_, err := cpi.Dispatch(req, delegate.TooManyReturnValues)
			Expect(err).To(MatchError("Too many action results"))
		})
	})

	Context("when the action function is variadic", func() {
		Context("and the required parameters are missing", func() {
			BeforeEach(func() {
				req.Args = []interface{}{}
			})

			It("returns an error", func() {
				_, err := cpi.Dispatch(req, delegate.VariadicStrings)
				Expect(err).To(MatchError("Not enough arguments: have 0, want 1"))
			})

			It("does not call the action", func() {
				cpi.Dispatch(req, delegate.VariadicStrings)
				Expect(delegate.CallCount).To(Equal(0))
			})
		})

		Context("and required parameters are present and optional parameters are missing", func() {
			BeforeEach(func() {
				req.Args = []interface{}{"hello"}
			})

			It("calls the action function and marshals the result", func() {
				resp, err := cpi.Dispatch(req, delegate.VariadicStrings)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.Error).To(BeNil())
				Expect(resp.Result).To(Equal([]string{"hello"}))

				Expect(delegate.CallCount).To(Equal(1))
			})
		})

		Context("and required parameters are present and one optional parameter", func() {
			BeforeEach(func() {
				req.Args = []interface{}{"hello", "world"}
			})

			It("calls the action function and marshals the result", func() {
				resp, err := cpi.Dispatch(req, delegate.VariadicStrings)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.Error).To(BeNil())
				Expect(resp.Result).To(Equal([]string{"hello", "world"}))

				Expect(delegate.CallCount).To(Equal(1))
			})
		})

		Context("and required parameters are present and multiple optional parameters", func() {
			BeforeEach(func() {
				req.Args = []interface{}{"hello", "world", "citizen"}
			})

			It("calls the action function and marshals the result", func() {
				resp, err := cpi.Dispatch(req, delegate.VariadicStrings)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.Error).To(BeNil())
				Expect(resp.Result).To(Equal([]string{"hello", "world", "citizen"}))

				Expect(delegate.CallCount).To(Equal(1))
			})
		})

		Context("when the variadic args are of the wrong type", func() {
			BeforeEach(func() {
				req.Args = []interface{}{"hello", "world", 12345}
			})

			It("calls the action function and marshals the result", func() {
				_, err := cpi.Dispatch(req, delegate.VariadicStrings)
				Expect(err).To(HaveOccurred())
				Expect(delegate.CallCount).To(Equal(0))
			})

			It("does not call the action", func() {
				cpi.Dispatch(req, delegate.VariadicStrings)
				Expect(delegate.CallCount).To(Equal(0))
			})
		})
	})
})

type Delegate struct {
	CallCount int
}

func (d *Delegate) NoArgs() error {
	d.CallCount++
	return nil
}

func (d *Delegate) NoArgsReturnBool() bool {
	d.CallCount++
	return true
}

func (d *Delegate) ReturnErr(msg string) error {
	d.CallCount++
	return errors.New(msg)
}

func (d *Delegate) OneStringArg(s string) error {
	d.CallCount++
	return nil
}

func (d *Delegate) VariadicStrings(s1 string, s2 ...string) ([]string, error) {
	d.CallCount++
	result := []string{s1}
	return append(result, s2...), nil
}

func (d *Delegate) TooManyReturnValues() (string, string, error) {
	d.CallCount++
	return "", "", nil
}

func (d *Delegate) LastNotError() (string, string) {
	d.CallCount++
	return "", ""
}
