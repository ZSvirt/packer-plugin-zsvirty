// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zsvirt

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func preserveZStackEnvVars() func() {
	saved := map[string]*string{}
	for _, name := range accessEnvironmentNames() {
		if value, ok := os.LookupEnv(name); ok {
			valueCopy := value
			saved[name] = &valueCopy
		} else {
			saved[name] = nil
		}
	}
	return func() {
		for name, value := range saved {
			if value == nil {
				os.Unsetenv(name)
			} else {
				os.Setenv(name, *value)
			}
		}
	}
}

func accessEnvironmentNames() []string {
	names := []string{"ZSVIRT_MANAGEMENT_IP", "ZSPHERE_HOST", "ZSTACK_HOST"}
	for _, prefix := range []string{"ZSVIRT_", "ZSPHERE_", "ZSTACK_"} {
		for _, suffix := range []string{"PORT", "ACCOUNT_NAME", "ACCOUNT_PASSWORD", "ACCESS_KEY_ID", "ACCESS_KEY_SECRET"} {
			names = append(names, prefix+suffix)
		}
	}
	return names
}

func TestBuilderPrepare_UsesAccessConfigEnvVars(t *testing.T) {
	restore := preserveZStackEnvVars()
	defer restore()

	clearZStackEnvVars()
	os.Setenv("ZSTACK_HOST", "env-host")
	os.Setenv("ZSTACK_ACCOUNT_NAME", "env-admin")
	os.Setenv("ZSTACK_ACCOUNT_PASSWORD", "env-password")

	var b Builder
	_, _, err := b.Prepare(map[string]interface{}{
		"communicator":        "none",
		"backup_storage_name": "local-backup",
		"image_name":          "target", "image_uuid": "source", "instance_name": "temporary", "network_uuid": "net", "instance_offering_uuid": "offering",
	})

	assert.NoError(t, err)
	assert.Equal(t, "env-host", b.config.Host)
	assert.Equal(t, "env-admin", b.config.AccountName)
	assert.Equal(t, "env-password", b.config.AccountPassword)
	assert.Equal(t, "none", b.config.Comm.Type)
}

func TestBuilderPrepare_ValidatesMissingAuth(t *testing.T) {
	restore := preserveZStackEnvVars()
	defer restore()

	clearZStackEnvVars()

	var b Builder
	_, _, err := b.Prepare(map[string]interface{}{
		"communicator": "none",
		"zstack_host":  "example.com",
	})

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "either account_name + account_password or access_key_id + access_key_secret is required")
	}
}

func TestBuilderPrepare_InvalidImageReadyTimeout(t *testing.T) {
	restore := preserveZStackEnvVars()
	defer restore()

	clearZStackEnvVars()

	var b Builder
	_, _, err := b.Prepare(map[string]interface{}{
		"communicator":        "none",
		"zstack_host":         "example.com",
		"account_name":        "admin",
		"account_password":    "pw",
		"image_ready_timeout": "not-a-duration",
	})

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "image_ready_timeout")
	}
}

func TestBuilderPrepare_ValidTimeouts(t *testing.T) {
	restore := preserveZStackEnvVars()
	defer restore()

	clearZStackEnvVars()

	var b Builder
	_, _, err := b.Prepare(map[string]interface{}{
		"communicator":        "none",
		"zstack_host":         "example.com",
		"account_name":        "admin",
		"account_password":    "pw",
		"backup_storage_name": "local-backup",
		"image_name":          "target", "image_uuid": "source", "instance_name": "temporary", "network_uuid": "net", "instance_offering_uuid": "offering",
		"image_ready_timeout": "10m",
		"vm_running_timeout":  "2m",
	})
	assert.NoError(t, err)
	assert.Equal(t, 10*time.Minute, b.config.ImageReadyTimeout())
	assert.Equal(t, 2*time.Minute, b.config.VmRunningTimeout())
}

func TestCommHost_ReturnsProvidedSshHost(t *testing.T) {
	state := testStateBag(&Config{}, &MockDriver{})
	fn := commHost("10.0.0.5")
	ip, err := fn(state)
	assert.NoError(t, err)
	assert.Equal(t, "10.0.0.5", ip)
}

func TestCommHost_FallsBackToConfigIP(t *testing.T) {
	config := &Config{InstanceConfig: InstanceConfig{IP: "192.168.1.10"}}
	state := testStateBag(config, &MockDriver{})
	fn := commHost("")
	ip, err := fn(state)
	assert.NoError(t, err)
	assert.Equal(t, "192.168.1.10", ip)
}

func TestConfigRedactedSummary_DoesNotLeakSecrets(t *testing.T) {
	cfg := Config{
		AccessConfig: AccessConfig{
			Host:            "example.com",
			AccountName:     "admin",
			AccountPassword: "super-secret-account-password",
			AccessKeyId:     "access-key-id",
			AccessKeySecret: "super-secret-access-key",
		},
		ImageConfig: ImageConfig{
			ImageName:      "packer-image",
			SourceImage:    "base-image",
			SourceImageUrl: "https://example.com/image.qcow2",
		},
		NetworkConfig: NetworkConfig{
			L3NetworkName: "default-network",
			L3NetworkUuid: "network-uuid",
		},
		InstanceConfig: InstanceConfig{
			InstanceName:         "packer-vm",
			InstanceOfferingName: "small-vm",
			InstanceOfferingUuid: "offering-uuid",
			UserData:             "cloud-init-secret",
		},
		BackupStorageConfig: BackupStorageConfig{
			BackupStorageName: "backup-storage",
			BackupStorageUuid: "backup-storage-uuid",
		},
	}
	cfg.Comm.Type = "ssh"
	cfg.Comm.SSHHost = "10.0.0.10"
	cfg.Comm.SSHPort = 22
	cfg.Comm.SSHUsername = "root"
	cfg.Comm.SSHPassword = "super-secret-ssh-password"
	cfg.Comm.SSHPrivateKeyFile = "/tmp/private-key.pem"
	cfg.PackerDebug = true

	summary := cfg.RedactedSummary()

	assert.Contains(t, summary, `auth_mode=account`)
	assert.Contains(t, summary, `ssh_password_set=true`)
	assert.Contains(t, summary, `ssh_private_key_file_set=true`)
	assert.Contains(t, summary, `user_data_set=true`)
	assert.NotContains(t, summary, "super-secret-account-password")
	assert.NotContains(t, summary, "access-key-id")
	assert.NotContains(t, summary, "super-secret-access-key")
	assert.NotContains(t, summary, "super-secret-ssh-password")
	assert.NotContains(t, summary, "cloud-init-secret")
	assert.NotContains(t, summary, "/tmp/private-key.pem")
}
