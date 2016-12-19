package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"code.cloudfoundry.org/clock"

	"github.com/sykesm/kubernetes-cpi/actions"
	"github.com/sykesm/kubernetes-cpi/config"
	"github.com/sykesm/kubernetes-cpi/cpi"
	"github.com/sykesm/kubernetes-cpi/kubecluster"
)

const (
	DefaultPostRecreateDelay = 5 * time.Second
	DefaultPodReadyTimeout   = 60 * time.Second
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

var debugFlag = flag.Bool(
	"debug",
	false,
	"Write CPI requests and responses to os.Stderr",
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

	debugJSON("request", payload)

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

	case "set_vm_metadata":
		vmMetadataSetter := actions.VMMetadataSetter{ClientProvider: provider}
		result, err = cpi.Dispatch(&req, vmMetadataSetter.SetVMMetadata)

	// Disk management
	case "create_disk":
		diskCreator := actions.DiskCreator{
			ClientProvider:    provider,
			GUIDGeneratorFunc: actions.CreateGUID,
		}
		result, err = cpi.Dispatch(&req, diskCreator.CreateDisk)

	case "attach_disk":
		volumeManager := actions.VolumeManager{
			ClientProvider:    provider,
			Clock:             clock.NewClock(),
			PodReadyTimeout:   DefaultPodReadyTimeout,
			PostRecreateDelay: DefaultPostRecreateDelay,
		}
		result, err = cpi.Dispatch(&req, volumeManager.AttachDisk)

	case "has_disk":
		diskFinder := actions.DiskFinder{ClientProvider: provider}
		result, err = cpi.Dispatch(&req, diskFinder.HasDisk)

	case "delete_disk":
		diskDeleter := actions.DiskDeleter{ClientProvider: provider}
		result, err = cpi.Dispatch(&req, diskDeleter.DeleteDisk)

	case "detach_disk":
		volumeManager := actions.VolumeManager{
			ClientProvider:    provider,
			Clock:             clock.NewClock(),
			PodReadyTimeout:   DefaultPodReadyTimeout,
			PostRecreateDelay: DefaultPostRecreateDelay,
		}
		result, err = cpi.Dispatch(&req, volumeManager.DetachDisk)

	case "get_disks":
		diskGetter := actions.DiskGetter{ClientProvider: provider}
		result, err = cpi.Dispatch(&req, diskGetter.GetDisks)

	// Not implemented
	case "configure_networks":
		result, err = nil, &cpi.NotSupportedError{}

	case "reboot_vm":
		result, err = nil, &cpi.NotSupportedError{}

	case "snapshot_disk":
		result, err = nil, &cpi.NotImplementedError{}

	case "delete_snapshot":
		result, err = nil, &cpi.NotImplementedError{}

	default:
		err = fmt.Errorf("Unexpected method: %q", req.Method)
	}

	if err != nil {
		panic(err)
	}

	response, err := json.Marshal(result)
	if err != nil {
		panic(err)
	}

	debugJSON("response", response)
	fmt.Printf("%s", response)
}

func debugJSON(stem string, payload []byte) {
	if *debugFlag {
		fmt.Fprintf(os.Stderr, `{ "%s": %s }%c`, stem, payload, '\n')
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
