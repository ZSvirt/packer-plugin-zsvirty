// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zsvirt

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/terraform-zstack-modules/zsphere-sdk-go/pkg/view"
)

// StepWaitForIP runs after boot commands, including after the ISO hard-disk
// reboot. VM creation and Running state alone do not guarantee an address.
type StepWaitForIP struct{}

func (*StepWaitForIP) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	c := state.Get("config").(*Config)
	if c.Comm.Type == "none" || (c.Comm.Type == "ssh" && c.Comm.SSHHost != "") || (c.Comm.Type == "winrm" && c.Comm.WinRMHost != "") {
		return multistep.ActionContinue
	}
	if c.InstanceUuid == "" {
		return halt(state, fmt.Errorf("instance UUID is empty, cannot discover its IP"))
	}
	ctx, cancel := context.WithTimeout(ctx, c.VmRunningTimeout())
	defer cancel()
	d := stepDriver(ctx, state)
	state.Get("ui").(packersdk.Ui).Say("Waiting for a port group or L3 network IP address...")
	c.IP = "" // Refresh after reboot; do not reuse a stale DHCP lease.
	for {
		if err := ctx.Err(); err != nil {
			return halt(state, fmt.Errorf("waiting for VM %s IP; configure ssh_host/winrm_host if the platform cannot report its address: %w", c.InstanceUuid, err))
		}
		vm, err := d.GetVmInstance(c.InstanceUuid)
		if err != nil {
			return halt(state, fmt.Errorf("querying VM IP: %w", err))
		}
		if vm == nil {
			return halt(state, fmt.Errorf("empty VM inventory while waiting for IP"))
		}
		if c.IP = selectVMIP(c, vm); c.IP != "" {
			return multistep.ActionContinue
		}
		timer := time.NewTimer(c.PollInterval())
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
}

func (*StepWaitForIP) Cleanup(multistep.StateBag) {}

func selectVMIP(c *Config, vm *view.VmInstanceInventoryView) string {
	network := c.DefaultL3NetworkUuid
	if network == "" {
		network = vm.DefaultL3NetworkUuid
	}
	for _, nic := range vm.VmNics {
		if network != "" && nic.L3NetworkUuid != network {
			continue
		}
		ip := net.ParseIP(nic.Ip)
		if ip != nil && !ip.IsUnspecified() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() {
			return nic.Ip
		}
	}
	return ""
}
