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

// Creates a temporary key for this account, exercises real signed requests and
// a complete build, then revokes only that key. No existing key is read or changed.
func TestAcceptanceAccessKey(t *testing.T) {
	if os.Getenv("ZSVIRT_ACC_AKSK") != "1" {
		t.Skip("set ZSVIRT_ACC_AKSK=1 to test a disposable access key")
	}
	var config AccessConfig
	require.Empty(t, config.Prepare())
	require.NotEmpty(t, config.AccountName, "account authentication is required to create a disposable key")
	client, err := newAPIClient(&config)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	session, err := client.Login(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, session.AccountUuid)
	id := newResource().UUID
	description := "packer-acceptance-" + id
	var key view.AccessKeyInventoryView
	require.NoError(t, client.Post(ctx, "v1/accesskeys", param.CreateAccessKeyParam{Params: param.CreateAccessKeyParamDetail{AccountUuid: session.AccountUuid, ResourceUuid: &id, Description: &description}}, &key))
	// Register cleanup before asserting response fields.
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), time.Minute)
		defer stop()
		require.NoError(t, client.Delete(cleanup, "v1/accesskeys", id, "Permissive"))
		params := param.NewQueryParam()
		params.AddQ("uuid=" + id)
		var remaining []view.AccessKeyInventoryView
		require.NoError(t, client.List(cleanup, "v1/accesskeys", &params, &remaining))
		require.Empty(t, remaining, "temporary key must be revoked")
	})
	require.Equal(t, id, key.UUID)
	require.NotEmpty(t, key.AccessKeyID)
	require.NotEmpty(t, key.AccessKeySecret)
	// Never print or persist generated keys.
	t.Setenv("ZSVIRT_ACCOUNT_NAME", "")
	t.Setenv("ZSVIRT_ACCOUNT_PASSWORD", "")
	t.Setenv("ZSVIRT_ACCESS_KEY_ID", key.AccessKeyID)
	t.Setenv("ZSVIRT_ACCESS_KEY_SECRET", key.AccessKeySecret)
	t.Setenv("ZSVIRT_ACC", "1")
	t.Setenv("ZSVIRT_ACC_SCENARIO", "")
	t.Run("signed-build", TestAcceptanceCloud)
}
