package actions_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sykesm/kubernetes-cpi/actions"
	"github.com/sykesm/kubernetes-cpi/cpi"
	"github.com/sykesm/kubernetes-cpi/kubecluster/fakes"
	"k8s.io/client-go/1.4/kubernetes/fake"
	"k8s.io/client-go/1.4/pkg/api/v1"
	"k8s.io/client-go/1.4/pkg/runtime"
	"k8s.io/client-go/1.4/testing"
)

var _ = Describe("DeleteVm", func() {
	var (
		fakeClient   *fakes.Client
		fakeProvider *fakes.ClientProvider

		vmDeleter *actions.VMDeleter
	)

	BeforeEach(func() {
		fakeClient = fakes.NewClient()
		fakeClient.ContextReturns("bosh")
		fakeClient.NamespaceReturns("bosh-namespace")

		fakeProvider = &fakes.ClientProvider{}
		fakeProvider.NewReturns(fakeClient, nil)

		vmDeleter = &actions.VMDeleter{ClientProvider: fakeProvider}
	})

	Describe("Delete", func() {
		var (
			agentID string
			vmcid   cpi.VMCID
		)

		BeforeEach(func() {
			agentID = "agent-id"
			vmcid = actions.NewVMCID("bosh", agentID)

			fakeClient.Clientset = *fake.NewSimpleClientset(
				&v1.Pod{ObjectMeta: v1.ObjectMeta{Name: "agent-agent-id", Namespace: "bosh-namespace"}},
				&v1.Service{ObjectMeta: v1.ObjectMeta{Name: "agent-agent-id", Namespace: "bosh-namespace"}},
				&v1.ConfigMap{ObjectMeta: v1.ObjectMeta{Name: "agent-agent-id", Namespace: "bosh-namespace"}},
			)
		})

		It("gets a client for the appropriate context", func() {
			err := vmDeleter.Delete(vmcid)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeProvider.NewCallCount()).To(Equal(1))
			Expect(fakeProvider.NewArgsForCall(0)).To(Equal("bosh"))
		})

		It("deletes the pod", func() {
			err := vmDeleter.Delete(vmcid)
			Expect(err).NotTo(HaveOccurred())

			matches := fakeClient.MatchingActions("delete", "pods")
			Expect(matches).To(HaveLen(1))

			Expect(matches[0].(testing.DeleteAction).GetName()).To(Equal("agent-" + agentID))
			Expect(matches[0].(testing.DeleteAction).GetNamespace()).To(Equal("bosh-namespace"))
		})

		It("deletes the service", func() {
			err := vmDeleter.Delete(vmcid)
			Expect(err).NotTo(HaveOccurred())

			matches := fakeClient.MatchingActions("delete", "services")
			Expect(matches).To(HaveLen(1))

			Expect(matches[0].(testing.DeleteAction).GetName()).To(Equal("agent-" + agentID))
			Expect(matches[0].(testing.DeleteAction).GetNamespace()).To(Equal("bosh-namespace"))
		})

		It("deletes the config map", func() {
			err := vmDeleter.Delete(vmcid)
			Expect(err).NotTo(HaveOccurred())

			matches := fakeClient.MatchingActions("delete", "configmaps")
			Expect(matches).To(HaveLen(1))

			Expect(matches[0].(testing.DeleteAction).GetName()).To(Equal("agent-" + agentID))
			Expect(matches[0].(testing.DeleteAction).GetNamespace()).To(Equal("bosh-namespace"))
		})

		Context("when objects have already been deleted", func() {
			BeforeEach(func() {
				err := vmDeleter.Delete(vmcid)
				Expect(err).NotTo(HaveOccurred())
			})

			It("continues with the delete process", func() {
				err := vmDeleter.Delete(vmcid)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeClient.Actions()).To(HaveLen(6))
				Expect(fakeClient.MatchingActions("delete", "pods")).To(HaveLen(2))
				Expect(fakeClient.MatchingActions("delete", "services")).To(HaveLen(2))
				Expect(fakeClient.MatchingActions("delete", "configmaps")).To(HaveLen(2))
			})
		})

		Context("when deleting the pod fails", func() {
			BeforeEach(func() {
				fakeClient.PrependReactor("delete", "pods", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("pods-welp")
				})
			})

			It("returns an error", func() {
				err := vmDeleter.Delete(vmcid)
				Expect(err).To(MatchError("pods-welp"))
				Expect(fakeClient.MatchingActions("delete", "pods")).To(HaveLen(1))
			})
		})

		Context("when deleting the config map fails", func() {
			BeforeEach(func() {
				fakeClient.PrependReactor("delete", "configmaps", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("configmaps-welp")
				})
			})

			It("returns an error", func() {
				err := vmDeleter.Delete(vmcid)
				Expect(err).To(MatchError("configmaps-welp"))
				Expect(fakeClient.MatchingActions("delete", "configmaps")).To(HaveLen(1))
			})
		})

		Context("when deleting the service fails", func() {
			BeforeEach(func() {
				fakeClient.PrependReactor("delete", "services", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("services-welp")
				})
			})

			It("returns an error", func() {
				err := vmDeleter.Delete(vmcid)
				Expect(err).To(MatchError("services-welp"))
				Expect(fakeClient.MatchingActions("delete", "services")).To(HaveLen(1))
			})
		})
	})
})
