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
	"github.com/sykesm/kubernetes-cpi/kubecluster"
)

var agentConfigFlag = flag.String(
	"agentConfig",
	"",
	"Path to serialized agent configuration data",
)

var kubeConfigFlag = flag.String(
	"kubeConfig",
	"",
	"Path to the serialized kubernetes configuration file",
)

func main() {
	flag.Parse()

	kubeConf, err := loadKubeConfig(*kubeConfigFlag)
	if err != nil {
		panic(err)
	}

	agentConf, err := loadAgentConfig(*agentConfigFlag)
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

	provider := &kubecluster.Provider{
		Config: kubeConf.ClientConfig(),
	}

	var result *cpi.Response
	switch req.Method {

	// Stemcell Management
	case "create_stemcell":
		result, err = cpi.Dispatch(&req, actions.CreateStemcell)

	case "delete_stemcell":
		result, err = cpi.Dispatch(&req, actions.DeleteStemcell)

	// VM management
	case "create_vm":
		vmCreator := &actions.VMCreator{
			AgentConfig:    agentConf,
			ClientProvider: provider,
		}
		result, err = cpi.Dispatch(&req, vmCreator.Create)

	case "delete_vm":
		vmDeleter := &actions.VMDeleter{ClientProvider: provider}
		result, err = cpi.Dispatch(&req, vmDeleter.Delete)

	case "has_vm":
		vmFinder := &actions.VMFinder{ClientProvider: provider}
		result, err = cpi.Dispatch(&req, vmFinder.HasVM)

	case "reboot_vm":
		result, err = cpi.Dispatch(&req, RebootVM)

	case "set_vm_metadata":
		vmMetadataSetter := actions.VMMetadataSetter{ClientProvider: provider}
		result, err = cpi.Dispatch(&req, vmMetadataSetter.SetVMMetadata)

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

	err = json.NewEncoder(os.Stdout).Encode(result)
	if err != nil {
		panic(err)
	}
}

func loadKubeConfig(path string) (*config.Kubernetes, error) {
	kubeConfigFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer kubeConfigFile.Close()

	var kubeConf config.Kubernetes
	err = json.NewDecoder(kubeConfigFile).Decode(&kubeConf)
	if err != nil {
		return nil, err
	}

	return &kubeConf, nil
}

func loadAgentConfig(path string) (*config.Agent, error) {
	agentConfigFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer agentConfigFile.Close()

	var agentConf config.Agent
	err = json.NewDecoder(agentConfigFile).Decode(&agentConf)
	if err != nil {
		return nil, err
	}

	return &agentConf, nil
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

func RebootVM(vmcid cpi.VMCID) error {
	return nil
}
