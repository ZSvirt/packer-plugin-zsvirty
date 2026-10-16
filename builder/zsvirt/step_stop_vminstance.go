// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zsvirt

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepStopVmInstance struct {
}

func (s *StepStopVmInstance) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	driver := stepDriver(ctx, state)

	instanceUuid := config.InstanceUuid
	ui.Say(fmt.Sprintf("Stopping VM instance '%s'...", instanceUuid))

	if config.ShutdownCommand != "" {
		comm, ok := state.Get("communicator").(packersdk.Communicator)
		if !ok {
			return halt(state, fmt.Errorf("shutdown_command requires a connected communicator"))
		}
		shutdownCtx, cancel := context.WithTimeout(ctx, config.ShutdownTimeout())
		defer cancel()
		cmd := &packersdk.RemoteCmd{Command: config.ShutdownCommand}
		if err := cmd.RunWithUi(shutdownCtx, comm, ui); err != nil {
			return halt(state, fmt.Errorf("shutdown_command: %w", err))
		}
		if cmd.ExitStatus() != 0 {
			return halt(state, fmt.Errorf("shutdown_command exited with status %d", cmd.ExitStatus()))
		}
		// Do not fall back to an API power-off: the guest must finish flushing
		// its filesystem and registry before detaching media or imaging disks.
		ticker := time.NewTicker(config.PollInterval())
		defer ticker.Stop()
		for {
			vm, err := driver.WithContext(shutdownCtx).GetVmInstance(instanceUuid)
			if err != nil {
				return halt(state, fmt.Errorf("waiting for guest shutdown: %w", err))
			}
			if vm != nil && vm.State == "Stopped" {
				return multistep.ActionContinue
			}
			select {
			case <-shutdownCtx.Done():
				return halt(state, fmt.Errorf("waiting for guest shutdown: %w", shutdownCtx.Err()))
			case <-ticker.C:
			}
		}
	}
	vmInstance, err := driver.StopVminstance(instanceUuid)

	if err != nil {
		// AC-004-03: Write error to state so Cleanup can detect failure
		err = fmt.Errorf("failed to stop VM instance '%s': %v", instanceUuid, err)
		ui.Error(err.Error())
		state.Put("error", err)
		return multistep.ActionHalt
	}

	if vmInstance == nil || vmInstance.State != "Stopped" {
		return halt(state, fmt.Errorf("VM did not reach Stopped state"))
	}
	ui.Say(fmt.Sprintf("Successfully stopped VM instance '%s'", vmInstance.Name))
	return multistep.ActionContinue
}

func (s *StepStopVmInstance) Cleanup(state multistep.StateBag) {}
