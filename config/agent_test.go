package config_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sykesm/kubernetes-cpi/config"
)

var _ = Describe("Agent Config", func() {
	var configData []byte
	var agentConf config.Agent

	BeforeEach(func() {
		configData = []byte(`{
			"blobstore": {
					"provider": "local",
					"options": { "blobstore_path": "/var/vcap/blobs" }
			},
			"mbus": "https://mbus:mbus-password@0.0.0.0:6868",
			"ntp": [ "0.pool.ntp.org", "1.pool.ntp.org" ]
		}`)

		err := json.Unmarshal([]byte(configData), &agentConf)
		Expect(err).NotTo(HaveOccurred())
	})

	It("deserializes the config data", func() {
		Expect(agentConf.MessageBus).To(Equal("https://mbus:mbus-password@0.0.0.0:6868"))
		Expect(agentConf.NTPServers).To(ConsistOf("0.pool.ntp.org", "1.pool.ntp.org"))

		blobstoreBytes, err := json.Marshal(agentConf.Blobstore)
		Expect(err).NotTo(HaveOccurred())
		Expect(blobstoreBytes).To(MatchJSON(`{ "provider": "local", "options": { "blobstore_path": "/var/vcap/blobs" } }`))
	})
})
