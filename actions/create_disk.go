package actions

import (
	"fmt"
	"strings"

	uuid "github.com/nu7hatch/gouuid"
	"github.com/sykesm/kubernetes-cpi/cpi"
	"github.com/sykesm/kubernetes-cpi/kubecluster"
	"k8s.io/client-go/1.4/pkg/api/resource"
	"k8s.io/client-go/1.4/pkg/api/v1"
)

type CreateDiskCloudProperties struct {
	Context string `json:"context"`
}

// DiskCreator simply creates a PersistentVolumeClaim. The attach process will
// turn the claim into a volume mounted into the pod.
type DiskCreator struct {
	ClientProvider    kubecluster.ClientProvider
	GUIDGeneratorFunc func() (string, error)
}

func (d *DiskCreator) CreateDisk(size uint, cloudProps CreateDiskCloudProperties, vmcid cpi.VMCID) (cpi.DiskCID, error) {
	diskID, err := d.GUIDGeneratorFunc()
	if err != nil {
		return "", err
	}

	volumeSize, err := resource.ParseQuantity(fmt.Sprintf("%dMi", size))
	if err != nil {
		return "", err
	}

	client, err := d.ClientProvider.New(cloudProps.Context)
	if err != nil {
		return "", err
	}

	_, err = client.PersistentVolumeClaims().Create(&v1.PersistentVolumeClaim{
		ObjectMeta: v1.ObjectMeta{
			Name:      "disk-" + diskID,
			Namespace: client.Namespace(),
			Labels: map[string]string{
				"bosh.cloudfoundry.org/disk-id": diskID,
			},
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: volumeSize,
				},
			},
		},
	})
	if err != nil {
		return "", err
	}

	return NewDiskCID(client.Context(), diskID), nil
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
