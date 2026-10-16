// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0
package zsvirt

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/terraform-zstack-modules/zsphere-sdk-go/pkg/param"
	"github.com/terraform-zstack-modules/zsphere-sdk-go/pkg/view"
)

func TestAcceptanceOffering(t *testing.T) {
	if os.Getenv("ZSVIRT_ACC_OFFERING") != "1" {
		t.Skip("set ZSVIRT_ACC_OFFERING=1 to test disposable offerings")
	}
	var access AccessConfig
	require.Empty(t, access.Prepare())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	d, err := access.driver(ctx, time.Minute)
	require.NoError(t, err)
	id := newResource().UUID
	name := "packer-offering-" + id
	var offering view.InstanceOfferingInventoryView
	require.NoError(t, d.client.Post(ctx, "v1/instance-offerings", param.CreateInstanceOfferingParam{Params: param.CreateInstanceOfferingParamDetail{Name: name, CpuNum: 2, MemorySize: 2 * 1024 * 1024 * 1024, ResourceUuid: &id}}, &offering))
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), time.Minute)
		defer stop()
		require.NoError(t, d.client.Delete(cleanup, "v1/instance-offerings", id, "Permissive"))
	})
	t.Setenv("ZSVIRT_ACC", "1")
	t.Setenv("ZSVIRT_ACC_SCENARIO", "")
	for _, mode := range []string{"NAME", "UUID"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("ZSVIRT_ACC_INSTANCE_OFFERING_NAME", "")
			t.Setenv("ZSVIRT_ACC_INSTANCE_OFFERING_UUID", "")
			value := name
			if mode == "UUID" {
				value = id
			}
			t.Setenv("ZSVIRT_ACC_INSTANCE_OFFERING_"+mode, value)
			TestAcceptanceCloud(t)
		})
	}
}
