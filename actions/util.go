package actions

import (
	"strings"

	uuid "github.com/nu7hatch/gouuid"
	"github.com/sykesm/kubernetes-cpi/cpi"
)

func NewVMCID(context, agentID string) cpi.VMCID {
	return cpi.VMCID(context + ":" + agentID)
}

func ParseVMCID(vmcid cpi.VMCID) (context, agentID string) {
	parts := strings.SplitN(string(vmcid), ":", 2)
	return parts[0], parts[1]
}

func NewDiskCID(context, diskID string) cpi.DiskCID {
	return cpi.DiskCID(context + ":" + diskID)
}

func ParseDiskCID(diskCID cpi.DiskCID) (context, diskID string) {
	parts := strings.SplitN(string(diskCID), ":", 2)
	return parts[0], parts[1]
}

func CreateGUID() (string, error) {
	guid, err := uuid.NewV4()
	if err != nil {
		return "", nil
	}

	return guid.String(), nil
}
