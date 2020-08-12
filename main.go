/*
Copyright (c) 2019 VMware, Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/kr/pretty"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vim25/debug"
	"github.com/vmware/govmomi/vim25/soap"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/cns"
	cnstypes "github.com/vmware/govmomi/cns/types"
	vim25types "github.com/vmware/govmomi/vim25/types"
)

func main() {
	// set CNS_DEBUG to true if you need to emit soap traces from these tests
	// soap traces will be emitted in the govmomi/cns/.soap directory
	// example export CNS_DEBUG='true'
	enableDebug := os.Getenv("CNS_DEBUG")
	soapTraceDirectory := ".soap"

	url := os.Getenv("CNS_VC_URL") // example: export CNS_VC_URL='https://username:password@vc-ip/sdk'
	datacenter := os.Getenv("CNS_DATACENTER")
	datastore := os.Getenv("CNS_DATASTORE")

	if url == "" || datacenter == "" || datastore == "" {
		panic("CNS_VC_URL or CNS_DATACENTER or CNS_DATASTORE is not set")
	}
	u, err := soap.ParseURL(url)
	if err != nil {
		panic(err)
	}

	if enableDebug == "true" {
		if _, err := os.Stat(soapTraceDirectory); os.IsNotExist(err) {
			os.Mkdir(soapTraceDirectory, 0755)
		}
		p := debug.FileProvider{
			Path: soapTraceDirectory,
		}
		debug.SetProvider(&p)
	}

	ctx := context.Background()
	c, err := govmomi.NewClient(ctx, u, true)
	if err != nil {
		panic(err)
	}
	// UseServiceVersion sets soap.Client.Version to the current version of the service endpoint via /sdk/vsanServiceVersions.xml
	c.UseServiceVersion("vsan")
	cnsClient, err := cns.NewClient(ctx, c.Client)
	if err != nil {
		panic(err)
	}
	finder := find.NewFinder(c.Client, false)
	dc, err := finder.Datacenter(ctx, datacenter)
	if err != nil {
		panic(err)
	}
	finder.SetDatacenter(dc)
	ds, err := finder.Datastore(ctx, datastore)
	if err != nil {
		panic(err)
	}

	var dsList []vim25types.ManagedObjectReference
	dsList = append(dsList, ds.Reference())

	var containerClusterArray []cnstypes.CnsContainerCluster
	containerCluster := cnstypes.CnsContainerCluster{
		ClusterType:   string(cnstypes.CnsClusterTypeKubernetes),
		ClusterId:     "demo-cluster-id",
		VSphereUser:   "Administrator@vsphere.local",
		ClusterFlavor: string(cnstypes.CnsClusterFlavorVanilla),
	}
	containerClusterArray = append(containerClusterArray, containerCluster)

	// Test CreateVolume API
	var cnsVolumeCreateSpecList []cnstypes.CnsVolumeCreateSpec
	cnsVolumeCreateSpec := cnstypes.CnsVolumeCreateSpec{
		Name:       "pvc-901e87eb-c2bd-11e9-806f-005056a0c9a0",
		VolumeType: string(cnstypes.CnsVolumeTypeBlock),
		Datastores: dsList,
		Metadata: cnstypes.CnsVolumeMetadata{
			ContainerCluster: containerCluster,
		},
		BackingObjectDetails: &cnstypes.CnsBlockBackingDetails{
			CnsBackingObjectDetails: cnstypes.CnsBackingObjectDetails{
				CapacityInMb: 5120,
			},
		},
	}
	cnsVolumeCreateSpecList = append(cnsVolumeCreateSpecList, cnsVolumeCreateSpec)
	fmt.Printf("Creating volume using the spec: %+v", pretty.Sprint(cnsVolumeCreateSpec))
	createTask, err := cnsClient.CreateVolume(ctx, cnsVolumeCreateSpecList)
	if err != nil {
		panic(err)
	}
	createTaskInfo, err := cns.GetTaskInfo(ctx, createTask)
	if err != nil {
		panic(err)
	}
	createTaskResult, err := cns.GetTaskResult(ctx, createTaskInfo)
	if err != nil {
		panic(err)
	}
	if createTaskResult == nil {
		panic("Empty create task results")
	}
	createVolumeOperationRes := createTaskResult.GetCnsVolumeOperationResult()
	if createVolumeOperationRes.Fault != nil {
		panic(createVolumeOperationRes.Fault)
	}
	volumeId := createVolumeOperationRes.VolumeId.Id
	fmt.Printf("Volume created sucessfully. volumeId: %s", volumeId)

}
