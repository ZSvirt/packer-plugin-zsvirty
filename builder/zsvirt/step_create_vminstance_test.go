// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zsvirt

import (
	"context"
	"errors"
	"testing"

	"github.com/ZSvirt/packer-plugin-zsvirty/builder/zsvirt/utils"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/stretchr/testify/assert"
	"github.com/terraform-zstack-modules/zsphere-sdk-go/pkg/view"
)

func TestStepCreateVMInstance_Run(t *testing.T) {
	t.Run("CreateVMSuccess", func(t *testing.T) {
		config := &Config{
			ImageConfig:   ImageConfig{ImageUuid: "img-1"},
			NetworkConfig: NetworkConfig{L3NetworkUuid: "l3-1"},
			InstanceConfig: InstanceConfig{
				InstanceName:         "vm-1",
				InstanceOfferingUuid: "offering-1",
			},
		}
		driver := &MockDriver{CreateVmInstanceResult: &view.VmInstanceInventoryView{
			BaseInfoView:   view.BaseInfoView{UUID: "vm-uuid-1"},
			RootVolumeUuid: "root-vol-1",
			VmNics:         []view.VmNicInventoryView{{Ip: "192.168.0.10"}},
		}}
		state := testStateBag(config, driver)
		if config.InstanceUuid != "" {
			resources(state).VM = ownedResource{UUID: config.InstanceUuid, Created: true}
		}

		action := (&StepCreateVMInstance{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.True(t, driver.CreateVmInstanceCalled)
		assert.Equal(t, "vm-uuid-1", config.InstanceUuid)
		assert.Equal(t, "root-vol-1", config.RootVolumeUuid)
		assert.Equal(t, "192.168.0.10", config.IP)
	})

	t.Run("CreateVMWithCpuMemory", func(t *testing.T) {
		config := &Config{
			ImageConfig:   ImageConfig{ImageUuid: "img-1", Platform: "Linux", GuestOsType: "Linux"},
			NetworkConfig: NetworkConfig{L3NetworkUuid: "l3-1"},
			InstanceConfig: InstanceConfig{
				InstanceName:       "vm-1",
				CPUNum:             4,
				CPUMode:            "host-model",
				CPUVendor:          "None",
				TPMKeyProviderUUID: "provider",
				MachineType:        "q35",
				MemorySize:         8192,
			},
		}
		driver := &MockDriver{CreateVmInstanceResult: &view.VmInstanceInventoryView{
			BaseInfoView:   view.BaseInfoView{UUID: "vm-uuid-1"},
			RootVolumeUuid: "root-vol-1",
			VmNics:         []view.VmNicInventoryView{{Ip: "192.168.0.10"}},
		}}
		state := testStateBag(config, driver)
		if config.InstanceUuid != "" {
			resources(state).VM = ownedResource{UUID: config.InstanceUuid, Created: true}
		}

		action := (&StepCreateVMInstance{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.Contains(t, driver.CreateVmInstanceParam.SystemTags, "resourceConfig::kvm::vm.cpuMode::host-model")
		assert.Contains(t, driver.CreateVmInstanceParam.SystemTags, "vmMachineType::q35")
		assert.Contains(t, driver.CreateVmInstanceParam.SystemTags, "resourceConfig::vm::vm.cpuid.vendor::None")
		assert.Equal(t, "provider", driver.CreateVmTPMKeyProviderUUID)
		assert.Equal(t, strPtr("Linux"), driver.CreateVmInstanceParam.Params.Platform)
		assert.Equal(t, strPtr("Linux"), driver.CreateVmInstanceParam.Params.GuestOsType)
		assert.Nil(t, driver.CreateVmInstanceParam.Params.InstanceOfferingUuid)
		if assert.NotNil(t, driver.CreateVmInstanceParam.Params.CpuNum) {
			assert.Equal(t, 4, *driver.CreateVmInstanceParam.Params.CpuNum)
		}
		if assert.NotNil(t, driver.CreateVmInstanceParam.Params.MemorySize) {
			assert.Equal(t, utils.MBToBytes(8192), *driver.CreateVmInstanceParam.Params.MemorySize)
		}
	})

	t.Run("CreateVMWithSshKey", func(t *testing.T) {
		config := &Config{
			ImageConfig:   ImageConfig{ImageUuid: "img-1"},
			NetworkConfig: NetworkConfig{L3NetworkUuid: "l3-1"},
			InstanceConfig: InstanceConfig{
				InstanceName:         "vm-1",
				InstanceOfferingUuid: "offering-1",
				SshKey:               "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD",
			},
		}
		driver := &MockDriver{CreateVmInstanceResult: &view.VmInstanceInventoryView{
			BaseInfoView:   view.BaseInfoView{UUID: "vm-uuid-1"},
			RootVolumeUuid: "root-vol-1",
			VmNics:         []view.VmNicInventoryView{{Ip: "192.168.0.10"}},
		}}
		state := testStateBag(config, driver)
		if config.InstanceUuid != "" {
			resources(state).VM = ownedResource{UUID: config.InstanceUuid, Created: true}
		}

		action := (&StepCreateVMInstance{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.Contains(t, driver.CreateVmInstanceParam.BaseParam.SystemTags, "sshkey::ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD")
	})

	t.Run("CreateVMError", func(t *testing.T) {
		expectedErr := errors.New("create vm failed")
		config := &Config{
			ImageConfig:   ImageConfig{ImageUuid: "img-1"},
			NetworkConfig: NetworkConfig{L3NetworkUuid: "l3-1"},
			InstanceConfig: InstanceConfig{
				InstanceName:         "vm-1",
				InstanceOfferingUuid: "offering-1",
			},
		}
		driver := &MockDriver{CreateVmInstanceErr: expectedErr}
		state := testStateBag(config, driver)
		if config.InstanceUuid != "" {
			resources(state).VM = ownedResource{UUID: config.InstanceUuid, Created: true}
		}

		action := (&StepCreateVMInstance{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		errVal, ok := state.GetOk("error")
		assert.True(t, ok)
		assert.Equal(t, expectedErr, errVal)
	})
}

func TestStepCreateVMInstance_Cleanup(t *testing.T) {
	t.Run("CleanupOnError", func(t *testing.T) {
		config := &Config{InstanceConfig: InstanceConfig{InstanceUuid: "vm-1"}}
		driver := &MockDriver{}
		state := testStateBag(config, driver)
		if config.InstanceUuid != "" {
			resources(state).VM = ownedResource{UUID: config.InstanceUuid, Created: true}
		}
		state.Put("error", errors.New("build failed"))

		(&StepCreateVMInstance{}).Cleanup(state)

		assert.True(t, driver.DestroyVmInstanceCalled)
		assert.Equal(t, "vm-1", driver.DestroyVmInstanceUuid)
		assert.True(t, driver.DeleteVmInstanceCalled)
		assert.Equal(t, "vm-1", driver.DeleteVmInstanceUuid)
	})

	t.Run("CleanupSkippedAfterExpunge", func(t *testing.T) {
		config := &Config{InstanceConfig: InstanceConfig{InstanceUuid: "vm-1"}}
		driver := &MockDriver{}
		state := testStateBag(config, driver)
		if config.InstanceUuid != "" {
			resources(state).VM = ownedResource{UUID: config.InstanceUuid, Created: true, Removed: true}
		}

		(&StepCreateVMInstance{}).Cleanup(state)

		assert.False(t, driver.DestroyVmInstanceCalled)
		assert.False(t, driver.DeleteVmInstanceCalled)
	})

	t.Run("CleanupSkippedNoInstanceUuid", func(t *testing.T) {
		config := &Config{}
		driver := &MockDriver{}
		state := testStateBag(config, driver)
		if config.InstanceUuid != "" {
			resources(state).VM = ownedResource{UUID: config.InstanceUuid, Created: true}
		}
		state.Put("error", errors.New("build failed"))

		(&StepCreateVMInstance{}).Cleanup(state)

		assert.False(t, driver.DestroyVmInstanceCalled)
		assert.False(t, driver.DeleteVmInstanceCalled)
	})

	t.Run("CleanupDestroyError", func(t *testing.T) {
		config := &Config{InstanceConfig: InstanceConfig{InstanceUuid: "vm-1"}}
		driver := &MockDriver{DestroyVmInstanceErr: errors.New("destroy failed")}
		state := testStateBag(config, driver)
		if config.InstanceUuid != "" {
			resources(state).VM = ownedResource{UUID: config.InstanceUuid, Created: true}
		}
		state.Put("error", errors.New("build failed"))

		(&StepCreateVMInstance{}).Cleanup(state)

		assert.True(t, driver.DestroyVmInstanceCalled)
		assert.False(t, driver.DeleteVmInstanceCalled)
	})
}

func TestStepCreateVMInstance_NoNIC_DefersIPDiscovery(t *testing.T) {
	config := &Config{
		ImageConfig:   ImageConfig{ImageUuid: "img-1"},
		NetworkConfig: NetworkConfig{L3NetworkUuid: "l3-1"},
		InstanceConfig: InstanceConfig{
			InstanceName:         "vm-1",
			InstanceOfferingUuid: "offering-1",
		},
	}
	driver := &MockDriver{CreateVmInstanceResult: &view.VmInstanceInventoryView{
		BaseInfoView: view.BaseInfoView{UUID: "vm-uuid-1"},
		VmNics:       nil,
	}}
	state := testStateBag(config, driver)
	if config.InstanceUuid != "" {
		resources(state).VM = ownedResource{UUID: config.InstanceUuid, Created: true}
	}

	action := (&StepCreateVMInstance{}).Run(context.Background(), state)

	assert.Equal(t, multistep.ActionContinue, action)
	assert.Empty(t, config.IP)
	assert.Equal(t, "vm-uuid-1", config.InstanceUuid)
	assert.True(t, resources(state).VM.Created)
}

func TestStepCreateVMInstance_CleanTrafficTag(t *testing.T) {
	t.Run("DefaultFalse", func(t *testing.T) {
		config := &Config{
			ImageConfig:    ImageConfig{ImageUuid: "img-1"},
			NetworkConfig:  NetworkConfig{L3NetworkUuid: "l3-1"},
			InstanceConfig: InstanceConfig{InstanceName: "vm-1", InstanceOfferingUuid: "o-1"},
		}
		driver := &MockDriver{CreateVmInstanceResult: &view.VmInstanceInventoryView{
			BaseInfoView: view.BaseInfoView{UUID: "vm-uuid-1"},
			VmNics:       []view.VmNicInventoryView{{Ip: "192.168.0.10"}},
		}}
		state := testStateBag(config, driver)
		if config.InstanceUuid != "" {
			resources(state).VM = ownedResource{UUID: config.InstanceUuid, Created: true}
		}

		(&StepCreateVMInstance{}).Run(context.Background(), state)
		assert.Contains(t, driver.CreateVmInstanceParam.BaseParam.SystemTags, "cleanTraffic::false")
	})
	t.Run("TrueWhenConfigured", func(t *testing.T) {
		config := &Config{
			ImageConfig:    ImageConfig{ImageUuid: "img-1"},
			NetworkConfig:  NetworkConfig{L3NetworkUuid: "l3-1"},
			InstanceConfig: InstanceConfig{InstanceName: "vm-1", InstanceOfferingUuid: "o-1"},
			CleanTraffic:   true,
		}
		driver := &MockDriver{CreateVmInstanceResult: &view.VmInstanceInventoryView{
			BaseInfoView: view.BaseInfoView{UUID: "vm-uuid-1"},
			VmNics:       []view.VmNicInventoryView{{Ip: "192.168.0.10"}},
		}}
		state := testStateBag(config, driver)
		if config.InstanceUuid != "" {
			resources(state).VM = ownedResource{UUID: config.InstanceUuid, Created: true}
		}

		(&StepCreateVMInstance{}).Run(context.Background(), state)
		assert.Contains(t, driver.CreateVmInstanceParam.BaseParam.SystemTags, "cleanTraffic::true")
	})
}

func TestEncodeUserData(t *testing.T) {
	t.Run("PlaintextGetsEncoded", func(t *testing.T) {
		got := encodeUserData("#!/bin/bash\necho hi")
		assert.Equal(t, "IyEvYmluL2Jhc2gKZWNobyBoaQ==", got)
	})
	t.Run("AlwaysEncodesRegardlessOfBase64Heuristic", func(t *testing.T) {
		// "password" happens to be valid base64 — we still encode.
		// Callers must pass plaintext, never pre-encoded base64.
		got := encodeUserData("password")
		assert.Equal(t, "cGFzc3dvcmQ=", got)
	})
	t.Run("WhitespaceTrimmed", func(t *testing.T) {
		got := encodeUserData("  hello  ")
		assert.Equal(t, "aGVsbG8=", got)
	})
	t.Run("EmptyStringEncodesToEmpty", func(t *testing.T) {
		assert.Equal(t, "", encodeUserData(""))
	})
}

func TestWindowsVirtioDeviceSelection(t *testing.T) {
	for _, platform := range []string{"Windows", "WindowsVirtio", "Linux"} {
		t.Run(platform, func(t *testing.T) {
			config := &Config{ImageConfig: ImageConfig{ImageUuid: "image", Platform: platform}, InstanceConfig: InstanceConfig{InstanceName: "build"}}
			d := &MockDriver{CreateVmInstanceResult: &view.VmInstanceInventoryView{BaseInfoView: view.BaseInfoView{UUID: "vm"}}}
			state := testStateBag(config, d)
			assert.Equal(t, multistep.ActionContinue, (&StepCreateVMInstance{}).Run(context.Background(), state))
			if platform == "WindowsVirtio" {
				assert.Equal(t, true, *d.CreateVmInstanceParam.Params.Virtio)
			} else {
				assert.Nil(t, d.CreateVmInstanceParam.Params.Virtio)
			}
		})
	}
}

func TestHyperVEnlightenmentSelection(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		config := &Config{ImageConfig: ImageConfig{ImageUuid: "image", Platform: "Windows"}, InstanceConfig: InstanceConfig{InstanceName: "build", HyperVEnlightenments: &enabled}}
		d := &MockDriver{CreateVmInstanceResult: &view.VmInstanceInventoryView{BaseInfoView: view.BaseInfoView{UUID: "vm"}}}
		state := testStateBag(config, d)
		assert.Equal(t, multistep.ActionContinue, (&StepCreateVMInstance{}).Run(context.Background(), state))
		expected := "resourceConfig::vm::emulateHyperV::false"
		if enabled {
			expected = "resourceConfig::vm::emulateHyperV::true"
		}
		assert.Contains(t, d.CreateVmInstanceParam.SystemTags, expected)
	}
}
