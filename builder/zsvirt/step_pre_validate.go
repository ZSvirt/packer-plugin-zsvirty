// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zsvirt

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/terraform-zstack-modules/zsphere-sdk-go/pkg/view"
)

type StepPreValidate struct {
}

func (s *StepPreValidate) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	ui.Say("Validating configuration...")

	snapshotOnly := config.SourceVolumeSnapshotUuid != ""

	if snapshotOnly {
		log.Printf("[INFO] Snapshot-only build: skipping L3 network validation")
	} else if len(config.L3NetworkUuids) > 0 {
		config.L3NetworkUuid = config.L3NetworkUuids[0]
	} else if config.NetworkConfig.L3NetworkUuid != "" {
		log.Printf("[INFO] Using provided network UUID: %s", config.NetworkConfig.L3NetworkUuid)
		ui.Say(fmt.Sprintf("Using provided network UUID: %s", config.NetworkConfig.L3NetworkUuid))
	} else {
		networks, err := validateNetwork(ctx, state)
		if err != nil {
			state.Put("error", err)
			ui.Errorf("Network validation failed: %s", err)
			return multistep.ActionHalt
		}
		config.NetworkConfig.L3NetworkUuid = networks[0].UUID
		ui.Say("Port group or L3 network validated")
	}

	if config.BackupStorageConfig.BackupStorageUuid == "" && config.BackupStorageConfig.BackupStorageName == "" {
		err := fmt.Errorf("image_storage_name or image_storage_uuid is required")
		state.Put("error", err)
		ui.Errorf("Image storage validation failed: %s", err)
		return multistep.ActionHalt
	}

	// AC-002-04: Skip image storage query when image_storage_uuid is provided.
	if config.BackupStorageConfig.BackupStorageUuid != "" {
		log.Printf("[INFO] Using provided image storage UUID: %s", config.BackupStorageConfig.BackupStorageUuid)
		ui.Say(fmt.Sprintf("Using provided image storage UUID: %s", config.BackupStorageConfig.BackupStorageUuid))
	} else if config.BackupStorageConfig.BackupStorageName != "" {
		backupStorages, err := validateBackupStorage(ctx, state)
		if err != nil {
			state.Put("error", err)
			ui.Errorf("Image storage validation failed: %s", err)
			return multistep.ActionHalt
		}
		config.BackupStorageConfig.BackupStorageUuid = backupStorages[0].UUID
		ui.Say("Image storage validated")
	}

	return multistep.ActionContinue
}

func validateNetwork(ctx context.Context, state multistep.StateBag) ([]view.L3NetworkInventoryView, error) {
	config := state.Get("config").(*Config)
	driver := stepDriver(ctx, state)

	networks, err := driver.QueryL3Network(config.L3NetworkName)
	if err != nil {
		return nil, fmt.Errorf("error querying port group or L3 network: %w", err)
	}
	if len(networks) == 0 {
		return nil, fmt.Errorf("network '%s' not found", config.L3NetworkName)
	}
	if len(networks) > 1 {
		return nil, fmt.Errorf("multiple network resources match the name; specify UUID")
	}
	return networks, nil
}

func validateBackupStorage(ctx context.Context, state multistep.StateBag) ([]view.BackupStorageInventoryView, error) {
	config := state.Get("config").(*Config)
	driver := stepDriver(ctx, state)

	backupStorage, err := driver.QueryBackStorage(config.BackupStorageName)
	if err != nil {
		return nil, fmt.Errorf("error querying image storage: %s", err)
	}
	if len(backupStorage) == 0 {
		return nil, fmt.Errorf("image storage '%s' not found", config.BackupStorageName)
	}
	if len(backupStorage) > 1 {
		return nil, fmt.Errorf("multiple image storage resources match the name; specify UUID")
	}
	return backupStorage, nil
}

func (s *StepPreValidate) Cleanup(state multistep.StateBag) {}
