package config

type Agent struct {
	Blobstore  interface{} `json:"blobstore,omitempty"`
	MessageBus string      `json:"mbus"`
	NTPServers []string    `json:"ntp,omitempty"`
}
