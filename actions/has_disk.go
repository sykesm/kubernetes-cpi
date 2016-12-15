package actions

import (
	"github.com/sykesm/kubernetes-cpi/cpi"
	"github.com/sykesm/kubernetes-cpi/kubecluster"
	"k8s.io/client-go/1.4/pkg/api"
	"k8s.io/client-go/1.4/pkg/labels"
)

type DiskFinder struct {
	ClientProvider kubecluster.ClientProvider
}

func (d *DiskFinder) HasDisk(diskCID cpi.DiskCID) (bool, error) {
	context, diskID := ParseDiskCID(diskCID)
	diskSelector, err := labels.Parse("bosh.cloudfoundry.org/disk-id=" + diskID)
	if err != nil {
		return false, err
	}

	client, err := d.ClientProvider.New(context)
	if err != nil {
		return false, err
	}

	listOptions := api.ListOptions{LabelSelector: diskSelector}
	pvcList, err := client.PersistentVolumeClaims().List(listOptions)
	if err != nil {
		return false, err
	}

	return len(pvcList.Items) > 0, nil
}
