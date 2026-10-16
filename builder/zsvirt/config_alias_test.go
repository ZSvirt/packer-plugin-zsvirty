// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zsvirt

import (
	"github.com/ZSvirt/packer-plugin-zsvirty/builder/zsvirt/utils"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAccessEnvironmentPriority(t *testing.T) {
	for _, name := range accessEnvironmentNames() {
		t.Setenv(name, "")
	}
	t.Setenv("ZSTACK_HOST", "192.0.2.1")
	t.Setenv("ZSPHERE_HOST", "192.0.2.2")
	t.Setenv("ZSVIRT_MANAGEMENT_IP", "192.0.2.3")
	t.Setenv("ZSTACK_ACCESS_KEY_ID", "legacy")
	t.Setenv("ZSTACK_ACCESS_KEY_SECRET", "legacy-secret")
	t.Setenv("ZSPHERE_ACCOUNT_NAME", "commercial")
	t.Setenv("ZSPHERE_ACCOUNT_PASSWORD", "commercial-secret")
	var c AccessConfig
	require.Empty(t, c.Prepare())
	require.Equal(t, "192.0.2.3", c.Host)
	require.Equal(t, "commercial", c.AccountName)
	require.Empty(t, c.AccessKeyId)
	t.Setenv("ZSVIRT_ACCESS_KEY_ID", "incomplete")
	c = AccessConfig{}
	require.NotEmpty(t, c.Prepare())
	require.Empty(t, c.AccessKeySecret)
	require.Empty(t, c.AccountName)
	c = AccessConfig{Host: "192.0.2.4", AccountName: "explicit"}
	require.NotEmpty(t, c.Prepare())
	require.Empty(t, c.AccountPassword)
	require.Equal(t, "192.0.2.4", c.Host)
	t.Setenv("ZSVIRT_PORT", "bad")
	c = AccessConfig{}
	require.NotEmpty(t, c.Prepare())
}

func TestCanonicalAndLegacyConfiguration(t *testing.T) {
	for _, name := range accessEnvironmentNames() {
		t.Setenv(name, "")
	}
	for _, tc := range []struct {
		name    string
		extra   map[string]interface{}
		invalid bool
	}{
		{"canonical", nil, false},
		{"guest shutdown", map[string]interface{}{"communicator": "winrm", "winrm_username": "Administrator", "winrm_password": "test", "shutdown_command": "shutdown.exe /s /t 10 /f", "shutdown_timeout": "10m"}, false},
		{"shutdown without communicator", map[string]interface{}{"shutdown_command": "shutdown"}, true},
		{"invalid shutdown timeout", map[string]interface{}{"shutdown_timeout": "invalid"}, true},
		{"nonpositive shutdown timeout", map[string]interface{}{"shutdown_timeout": "0s"}, true},
		{"matching aliases", map[string]interface{}{"zstack_host": "192.0.2.10", "backup_storage_name": "images"}, false},
		{"host conflict", map[string]interface{}{"zstack_host": "192.0.2.11"}, true},
		{"storage conflict", map[string]interface{}{"backup_storage_name": "other"}, true},
		{"uuid conflict", map[string]interface{}{"image_storage_uuid": "one", "backup_storage_uuid": "two"}, true},
		{"offering conflict", map[string]interface{}{"instance_offering_uuid": "offering"}, true},
		{"missing memory", map[string]interface{}{"memory_size": 0}, true},
		{"invalid machine type", map[string]interface{}{"machine_type": "arbitrary"}, true},
		{"firmware boot menu delay", map[string]interface{}{"boot_menu_timeout": "20s"}, false},
		{"invalid firmware boot menu delay", map[string]interface{}{"boot_menu_timeout": "-1s"}, true},
		{"TPM needs UEFI", map[string]interface{}{"tpm_key_provider_uuid": "provider"}, true},
		{"TPM with UEFI", map[string]interface{}{"tpm_key_provider_uuid": "provider", "boot_mode": "UEFI"}, false},
		{"native CPU vendor", map[string]interface{}{"cpu_vendor": "None"}, false},
		{"AMD CPU vendor", map[string]interface{}{"cpu_vendor": "AuthenticAMD"}, false},
		{"invalid CPU vendor", map[string]interface{}{"cpu_vendor": "arbitrary"}, true},
		{"invalid CPU mode", map[string]interface{}{"cpu_mode": "arbitrary"}, true},
		{"negative memory", map[string]interface{}{"memory_size": -1}, true},
		{"overflow memory", map[string]interface{}{"memory_size": int64(8796093022208)}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := map[string]interface{}{"management_ip": "192.0.2.10", "account_name": "admin", "account_password": "test", "image_storage_name": "images", "image_uuid": "source", "image_name": "output", "instance_name": "build", "network_uuid": "net", "communicator": "none", "cpu_num": 2, "memory_size": 2048}
			for k, v := range tc.extra {
				raw[k] = v
			}
			var b Builder
			_, _, err := b.Prepare(raw)
			if tc.invalid {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, int64(2147483648), utils.MiBToBytes(b.config.MemorySize))
			}
		})
	}
}
