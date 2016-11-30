package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/sykesm/kubernetes-cpi/actions"
	"github.com/sykesm/kubernetes-cpi/config"
	"github.com/sykesm/kubernetes-cpi/cpi"
)

var agentConfigFlag = flag.String(
	"agentConfig",
	"",
	"Path to serialized agent configuration data",
)

var kubeConfigFlag = flag.String(
	"kubeConfig",
	"",
	"Path to the serialized configuration file",
)

func main() {
	flag.Parse()

	kubeConfigFile, err := os.Open(*kubeConfigFlag)
	if err != nil {
		panic(err)
	}
	defer kubeConfigFile.Close()

	var kubeConf config.Kubernetes
	err = json.NewDecoder(kubeConfigFile).Decode(&kubeConf)
	if err != nil {
		panic(err)
	}

	agentConfigFile, err := os.Open(*agentConfigFlag)
	if err != nil {
		panic(err)
	}
	defer agentConfigFile.Close()

	var agentConf config.Agent
	err = json.NewDecoder(agentConfigFile).Decode(&agentConf)
	if err != nil {
		panic(err)
	}

	payload, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	var req cpi.Request
	err = json.Unmarshal(payload, &req)
	if err != nil {
		panic(err)
	}

	encoder := json.NewEncoder(os.Stdout)

	var result *cpi.Response
	switch req.Method {

	// Stemcell Management
	case "create_stemcell":
		result, err = cpi.Dispatch(&req, CreateStemcell)

	case "delete_stemcell":
		result, err = cpi.Dispatch(&req, DeleteStemcell)

	// VM management
	case "create_vm":
		vmCreator := &actions.VMCreator{
			KubeConfig:  kubeConf,
			AgentConfig: agentConf,
		}
		result, err = cpi.Dispatch(&req, vmCreator.Create)

	case "delete_vm":
		result, err = cpi.Dispatch(&req, DeleteVM)

	case "has_vm":
		result, err = cpi.Dispatch(&req, HasVM)

	case "reboot_vm":
		result, err = cpi.Dispatch(&req, RebootVM)

	case "set_vm_metadata":
		result, err = cpi.Dispatch(&req, SetVMMetadata)

	case "configure_networks":
		result, err = nil, &cpi.NotSupportedError{}

	// Disk management
	case "create_disk":
		result, err = cpi.Dispatch(&req, CreateDisk)

	case "delete_disk":
		result, err = cpi.Dispatch(&req, DeleteDisk)

	case "has_disk":
		result, err = cpi.Dispatch(&req, HasDisk)

	case "attach_disk":
		result, err = cpi.Dispatch(&req, AttachDisk)

	case "detach_disk":
		result, err = cpi.Dispatch(&req, DetachDisk)

	case "get_disks":
		result, err = cpi.Dispatch(&req, HasDisk)

	case "snapshot_disk":
		result, err = cpi.Dispatch(&req, func(diskCID cpi.DiskCID, meta map[string]interface{}) cpi.SnapshotCID { return "not_implemented" })

	case "delete_snapshot":
		result, err = cpi.Dispatch(&req, func(snapshotCID cpi.SnapshotCID) {})

	default:
		err = fmt.Errorf("Unexpected method: %q", req.Method)
	}

	if err != nil {
		panic(err)
	}

	err = encoder.Encode(result)
	if err != nil {
		panic(err)
	}
}

type StemcellCloudProperties struct {
	Image string `json:"image"`
}

func CreateStemcell(image string, cloudProps StemcellCloudProperties) (cpi.StemcellCID, error) {
	return cpi.StemcellCID(cloudProps.Image), nil
}

func DeleteStemcell(stemcellCID cpi.StemcellCID) error {
	return nil
}

type CreateDiskCloudProperties struct{}

func CreateDisk(size uint, cloudProps CreateDiskCloudProperties, vmcid cpi.VMCID) (cpi.DiskCID, error) {
	return cpi.DiskCID("not-implemented"), nil
}

func DeleteDisk(diskCID cpi.DiskCID) error {
	return nil
}

func AttachDisk(vmcid cpi.VMCID, diskCID cpi.DiskCID) error {
	return nil
}

func DetachDisk(vmcid cpi.VMCID, diskCID cpi.DiskCID) error {
	return nil
}

func HasDisk(diskCID cpi.DiskCID) bool {
	return false
}

func GetDisks(vmcid cpi.VMCID) ([]cpi.DiskCID, error) {
	return []cpi.DiskCID{}, nil
}

func DeleteVM(vmcid cpi.VMCID) error {
	return nil
}

func HasVM(vmcid cpi.VMCID) (bool, error) {
	return false, nil
}

func SetVMMetadata(vmcid cpi.VMCID, metadata map[string]string) error {
	return nil
}

func RebootVM(vmcid cpi.VMCID) error {
	return nil
}
