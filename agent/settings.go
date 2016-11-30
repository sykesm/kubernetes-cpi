package agent

type Settings struct {
	AgentID    string   `json:"agent_id"`
	Disks      Disks    `json:"disks,omitempty"`
	Networks   Networks `json:"networks,omitempty"`
	MessageBus string   `json:"mbus"`
	NTPServers []string `json:"ntp,omitempty"`
	VM         VM       `json:"vm"`

	// These are just carried along from bosh
	Blobstore interface{} `json:"blobstore,omitempty"`
	Env       interface{} `json:"env,omitempty"`
}

type Disks struct {
	Persistent map[string]string `json:"persistent,omitempty"`
}

type Network struct {
	Type string `json:"type"`

	IP      string   `json:"ip,omitempty"`
	Netmask string   `json:"netmask,omitempty"`
	Gateway string   `json:"gateway,omitempty"`
	Default []string `json:"default,omitempty"`
	DNS     []string `json:"dns,omitempty"`

	Preconfigured bool `json:"preconfigured,omitempty"`
}

type Networks map[string]Network

type VM struct {
	Name string `json:"name"`
}
