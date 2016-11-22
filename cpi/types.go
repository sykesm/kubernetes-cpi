package cpi

type Context struct {
	DirectorUUID string `json:"director_uuid"`
}

type Network struct {
	Type string `json:"type"`

	IP      string   `json:"ip"`
	Netmask string   `json:"netmask"`
	Gateway string   `json:"gateway"`
	DNS     []string `json:"dns"`
	Default []string `json:"default"`

	CloudProperties map[string]interface{} `json:"cloud_properties"`
}

type DiskCID string

type Environment map[string]interface{}

type Metadata map[string]string

type Networks map[string]Network

type StemcellCID string

type SnapshotCID string

type VMCID string
