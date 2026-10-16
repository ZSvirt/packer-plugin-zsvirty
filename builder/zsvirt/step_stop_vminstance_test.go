// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zsvirt

import (
	"context"
	"errors"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"testing"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/stretchr/testify/assert"
	"github.com/terraform-zstack-modules/zsphere-sdk-go/pkg/view"
)

func TestStepStopVmInstance_Run(t *testing.T) {
	t.Run("StopVMSuccess", func(t *testing.T) {
		config := &Config{InstanceConfig: InstanceConfig{InstanceUuid: "vm-1"}}
		driver := &MockDriver{StopVminstanceResult: &view.VmInstanceInventoryView{State: "Stopped", BaseInfoView: view.BaseInfoView{Name: "vm-name"}}}
		state := testStateBag(config, driver)

		action := (&StepStopVmInstance{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.True(t, driver.StopVminstanceCalled)
		assert.Equal(t, "vm-1", driver.StopVminstanceUuid)
	})

	t.Run("StopVMError", func(t *testing.T) {
		config := &Config{InstanceConfig: InstanceConfig{InstanceUuid: "vm-1"}}
		driver := &MockDriver{StopVminstanceErr: errors.New("stop failed")}
		state := testStateBag(config, driver)

		action := (&StepStopVmInstance{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		assert.True(t, driver.StopVminstanceCalled)
		errVal, ok := state.GetOk("error")
		assert.True(t, ok)
		assert.Error(t, errVal.(error))
		assert.Contains(t, errVal.(error).Error(), "failed to stop VM instance 'vm-1'")
	})
}

func TestGuestShutdownWaitsWithoutAPIPowerOff(t *testing.T) {
	for _, tc := range []struct {
		name, state string
		code        int
		connected   bool
		want        multistep.StepAction
	}{
		{"stopped", "Stopped", 0, true, multistep.ActionContinue},
		{"timeout", "Running", 0, true, multistep.ActionHalt},
		{"command failed", "Stopped", 5, true, multistep.ActionHalt},

		{"no communicator", "Stopped", 0, false, multistep.ActionHalt},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &Config{ShutdownCommand: "shutdown.exe /s /t 10 /f", shutdownTimeout: 20 * time.Millisecond, pollInterval: time.Millisecond, InstanceConfig: InstanceConfig{InstanceUuid: "vm"}}
			d := &MockDriver{GetVmInstanceResult: &view.VmInstanceInventoryView{State: tc.state}}
			state := testStateBag(c, d)
			comm := &packersdk.MockCommunicator{StartExitStatus: tc.code}
			if tc.connected {
				state.Put("communicator", comm)
			}
			assert.Equal(t, tc.want, (&StepStopVmInstance{}).Run(context.Background(), state))
			assert.False(t, d.StopVminstanceCalled, "guest shutdown must never fall back to power-off")
			if tc.want == multistep.ActionContinue {
				assert.True(t, d.GetVmInstanceCalled)
				assert.Equal(t, c.ShutdownCommand, comm.StartCmd.Command)
			}
		})
	}
}

type shutdownFailingCommunicator struct{ packersdk.MockCommunicator }

func (*shutdownFailingCommunicator) Start(context.Context, *packersdk.RemoteCmd) error {
	return errors.New("connection failed")
}
func TestGuestShutdownErrorsDoNotFallBack(t *testing.T) {
	for _, name := range []string{"transport", "query", "cancel"} {
		t.Run(name, func(t *testing.T) {
			c := &Config{ShutdownCommand: "shutdown", InstanceConfig: InstanceConfig{InstanceUuid: "vm"}}
			d := &MockDriver{GetVmInstanceResult: &view.VmInstanceInventoryView{State: "Running"}}
			state := testStateBag(c, d)
			state.Put("communicator", &packersdk.MockCommunicator{})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch name {
			case "transport":
				state.Put("communicator", &shutdownFailingCommunicator{})
			case "query":
				d.GetVmInstanceErr = errors.New("query failed")
			case "cancel":
				cancel()
			}
			assert.Equal(t, multistep.ActionHalt, (&StepStopVmInstance{}).Run(ctx, state))
			assert.False(t, d.StopVminstanceCalled)
		})
	}
}
