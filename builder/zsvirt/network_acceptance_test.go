// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0
package zsvirt

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sdk "github.com/terraform-zstack-modules/zsphere-sdk-go/pkg/client"
	"github.com/terraform-zstack-modules/zsphere-sdk-go/pkg/param"
)

// Read-only compatibility check: port groups also have an L3 API identity.
// This does not require provisioning a separate legacy L3 network.
func TestAcceptanceSDKNetworkCompatibility(t *testing.T) {
	if os.Getenv("ZSVIRT_ACC_NETWORK") != "1" {
		t.Skip("set ZSVIRT_ACC_NETWORK=1 to query network compatibility through the SDK")
	}
	var access AccessConfig
	require.Empty(t, access.Prepare())
	config := sdk.NewZSConfig(access.Host, access.Port, "zstack").Debug(false).RetryTimes(2)
	if access.AccessKeyId != "" {
		config.AccessKey(access.AccessKeyId, access.AccessKeySecret)
	} else {
		config.LoginAccount(access.AccountName, access.AccountPassword)
	}
	client := sdk.NewZSClient(config)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if access.AccessKeyId == "" {
		_, err := client.Login(ctx)
		require.NoError(t, err)
		defer func() { require.NoError(t, client.Logout(context.Background())) }()
	}
	query := param.NewQueryParam()
	groups, err := client.QueryPortGroup(ctx, &query)
	require.NoError(t, err)
	l3s, err := client.QueryL3Network(ctx, &query)
	require.NoError(t, err)
	require.NotEmpty(t, groups)
	driver, err := access.Driver(false)
	require.NoError(t, err)
	for _, group := range groups {
		t.Run(group.Name, func(t *testing.T) {
			found := false
			for _, network := range l3s {
				if network.UUID == group.UUID {
					found = true
					require.Equal(t, group.Name, network.Name)
					require.Equal(t, group.L2NetworkUuid, network.L2NetworkUuid)
				}
			}
			require.True(t, found, "port group must be visible through the L3 query API")
			byID, err := client.GetL3Network(ctx, group.UUID)
			require.NoError(t, err)
			require.Equal(t, group.UUID, byID.UUID)
			byName, err := driver.QueryL3Network(group.Name)
			require.NoError(t, err)
			found = false
			for _, network := range byName {
				found = found || network.UUID == group.UUID
			}
			require.True(t, found, "builder name lookup must retain the SDK L3 identity")
			resolved, err := driver.GetL3Network(group.UUID)
			require.NoError(t, err)
			require.Equal(t, group.UUID, resolved.UUID)
			t.Logf("SDK port-group and L3 identity agree: name=%s uuid=%s system=%t", group.Name, group.UUID, group.System)
		})
	}
}
