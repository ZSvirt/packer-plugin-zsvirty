// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0
package zsvirt

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/terraform-zstack-modules/zsphere-sdk-go/pkg/param"
	"github.com/terraform-zstack-modules/zsphere-sdk-go/pkg/view"
	"os"
	"testing"
	"time"
)

func TestAcceptanceDataDisk(t *testing.T) {
	if os.Getenv("ZSVIRT_ACC_DATA_DISK") != "1" {
		t.Skip("set ZSVIRT_ACC_DATA_DISK=1 for a disposable data disk")
	}
	var access AccessConfig
	require.Empty(t, access.Prepare())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	d, err := access.driver(ctx, time.Minute)
	require.NoError(t, err)
	id := newResource().UUID
	var offering view.DiskOfferingInventoryView
	require.NoError(t, d.client.Post(ctx, "v1/disk-offerings", param.CreateDiskOfferingParam{Params: param.CreateDiskOfferingParamDetail{Name: "packer-data-" + id, DiskSize: 1024 * 1024 * 1024, ResourceUuid: &id}}, &offering))
	t.Cleanup(func() {
		ctx, stop := context.WithTimeout(context.Background(), time.Minute)
		defer stop()
		require.NoError(t, d.client.Delete(ctx, "v1/disk-offerings", id, "Permissive"))
	})
	t.Setenv("ZSVIRT_ACC", "1")
	t.Setenv("ZSVIRT_ACC_DATA_DISK_OFFERING_UUID", id)
	t.Setenv("ZSVIRT_ACC_SCENARIO", "")
	TestAcceptanceCloud(t)
}
