package kubecluster_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestKubecluster(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Kubecluster Suite")
}
