// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zsvirt

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/stretchr/testify/require"
	"github.com/terraform-zstack-modules/zsphere-sdk-go/pkg/view"
)

func TestNetworkNameResolution(t *testing.T) {
	for _, tc := range []struct {
		name     string
		status   int
		body     string
		fallback bool
		want     string
		wantErr  string
	}{
		{"port group wins", 200, `{"inventories":[{"uuid":"pg"}]}`, false, "pg", ""},
		{"empty port groups use L3", 200, `{"inventories":[]}`, true, "l3", ""},
		{"old endpoint JSON", 404, `{"error":{"code":"NOT_FOUND"}}`, true, "l3", ""},
		{"old endpoint HTML", 404, `<html>Not Found</html>`, true, "l3", ""},
		{"method unsupported", 405, `{}`, true, "l3", ""},
		{"not implemented", 501, `{}`, true, "l3", ""},
		{"authentication error", 401, `{"error":{"code":"AUTH"}}`, false, "", "AUTH"},
		{"authorization error", 403, `{}`, false, "", "403"},
		{"server error", 500, `{}`, false, "", "500"},
		{"API error", 200, `{"error":{"code":"FAILED"}}`, false, "", "FAILED"},
		{"invalid response", 200, `not JSON`, false, "", "invalid JSON"},
		{"ambiguous port groups", 200, `{"inventories":[{"uuid":"pg1"},{"uuid":"pg2"}]}`, false, "", "multiple network resources"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var paths []string
			d := httpDriver(t, func(w http.ResponseWriter, r *http.Request) {
				paths = append(paths, r.URL.Path)
				require.Equal(t, "GET", r.Method)
				require.Equal(t, "name=build network", r.URL.Query().Get("q"))
				if r.URL.Path == "/zstack/v1/l3-networks/port-group" {
					w.WriteHeader(tc.status)
					fmt.Fprint(w, tc.body)
					return
				}
				require.Equal(t, "/zstack/v1/l3-networks", r.URL.Path)
				fmt.Fprint(w, `{"inventories":[{"uuid":"l3"}]}`)
			})
			state := testStateBag(&Config{NetworkConfig: NetworkConfig{L3NetworkName: "build network"}}, d)
			rows, err := validateNetwork(context.Background(), state)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
				require.Len(t, rows, 1)
				require.Equal(t, tc.want, rows[0].UUID)
			}
			wantPaths := []string{"/zstack/v1/l3-networks/port-group"}
			if tc.fallback {
				wantPaths = append(wantPaths, "/zstack/v1/l3-networks")
			}
			require.Equal(t, wantPaths, paths)
		})
	}
}

func TestWaitForIPAfterBoot(t *testing.T) {
	c := &Config{InstanceConfig: InstanceConfig{InstanceUuid: "vm", IP: "192.0.2.99"}, NetworkConfig: NetworkConfig{DefaultL3NetworkUuid: "pg"}}
	c.Comm.Type = "ssh"
	c.pollInterval = time.Millisecond
	c.vmRunningTimeout = time.Second
	calls := 0
	d := httpDriver(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, "/zstack/v1/vm-instances/vm", r.URL.Path)
		if calls == 1 {
			fmt.Fprint(w, `{"inventories":[{"uuid":"vm","vmNics":[{"l3NetworkUuid":"other","ip":"192.0.2.1"}]}]}`)
			return
		}
		fmt.Fprint(w, `{"inventories":[{"uuid":"vm","vmNics":[{"l3NetworkUuid":"other","ip":"192.0.2.1"},{"l3NetworkUuid":"pg","ip":"192.0.2.2"}]}]}`)
	})
	state := testStateBag(c, d)
	require.Equal(t, multistep.ActionContinue, (&StepWaitForIP{}).Run(context.Background(), state))
	require.Equal(t, 2, calls)
	require.Equal(t, "192.0.2.2", c.IP)
	require.Empty(t, c.Comm.SSHHost, "discovered addresses must remain refreshable after reboot")
	host, err := commHost("")(state)
	require.NoError(t, err)
	require.Equal(t, c.IP, host)
}

func TestWaitForIPExplicitHostsAndNoCommunicator(t *testing.T) {
	for _, comm := range []string{"ssh", "winrm", "none"} {
		t.Run(comm, func(t *testing.T) {
			c := &Config{}
			c.Comm.Type = comm
			if comm == "ssh" {
				c.Comm.SSHHost = "ssh.example.invalid"
			}
			if comm == "winrm" {
				c.Comm.WinRMHost = "winrm.example.invalid"
			}
			d := &MockDriver{}
			require.Equal(t, multistep.ActionContinue, (&StepWaitForIP{}).Run(context.Background(), testStateBag(c, d)))
			require.False(t, d.GetVmInstanceCalled)
		})
	}
}

func TestWaitForIPTimeoutAndCancellation(t *testing.T) {
	for _, cancelNow := range []bool{false, true} {
		t.Run(fmt.Sprintf("cancel=%t", cancelNow), func(t *testing.T) {
			c := &Config{InstanceConfig: InstanceConfig{InstanceUuid: "vm"}}
			c.Comm.Type = "winrm"
			c.vmRunningTimeout = 20 * time.Millisecond
			c.pollInterval = time.Millisecond
			d := httpDriver(t, func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `{"inventories":[{"uuid":"vm","vmNics":[{"ip":"0.0.0.0"}]}]}`)
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			want := context.DeadlineExceeded
			if cancelNow {
				cancel()
				want = context.Canceled
			}
			state := testStateBag(c, d)
			require.Equal(t, multistep.ActionHalt, (&StepWaitForIP{}).Run(ctx, state))
			require.ErrorIs(t, state.Get("error").(error), want)
			require.Empty(t, c.IP)
		})
	}
}

func TestL3FallbackValidation(t *testing.T) {
	for _, tc := range []struct{ body, wantErr string }{
		{`{"inventories":[]}`, "not found"},
		{`{"inventories":[{"uuid":"l3a"},{"uuid":"l3b"}]}`, "multiple network resources"},
		{`{"error":{"code":"L3_QUERY_FAILED"}}`, "L3_QUERY_FAILED"},
	} {
		t.Run(tc.wantErr, func(t *testing.T) {
			d := httpDriver(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/zstack/v1/l3-networks/port-group" {
					fmt.Fprint(w, `{"inventories":[]}`)
				} else {
					fmt.Fprint(w, tc.body)
				}
			})
			state := testStateBag(&Config{NetworkConfig: NetworkConfig{L3NetworkName: "build-network"}}, d)
			_, err := validateNetwork(context.Background(), state)
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestWaitForIPUsesPlatformDefaultNetwork(t *testing.T) {
	c := &Config{InstanceConfig: InstanceConfig{InstanceUuid: "vm"}}
	c.Comm.Type = "ssh"
	d := &MockDriver{GetVmInstanceResult: &view.VmInstanceInventoryView{
		DefaultL3NetworkUuid: "l3",
		VmNics:               []view.VmNicInventoryView{{L3NetworkUuid: "pg", Ip: "192.0.2.1"}, {L3NetworkUuid: "l3", Ip: "192.0.2.2"}},
	}}
	require.Equal(t, multistep.ActionContinue, (&StepWaitForIP{}).Run(context.Background(), testStateBag(c, d)))
	require.Equal(t, "192.0.2.2", c.IP)
}

func TestIPTimeoutCleansCreatedVM(t *testing.T) {
	c := &Config{InstanceConfig: InstanceConfig{InstanceName: "test"}}
	c.Comm.Type = "ssh"
	c.vmRunningTimeout = 10 * time.Millisecond
	c.pollInterval = time.Millisecond
	d := &MockDriver{
		CreateVmInstanceResult: &view.VmInstanceInventoryView{BaseInfoView: view.BaseInfoView{UUID: "owned"}},
		GetVmInstanceResult:    &view.VmInstanceInventoryView{BaseInfoView: view.BaseInfoView{UUID: "owned"}, State: "Running"},
	}
	state := testStateBag(c, d)
	(&multistep.BasicRunner{Steps: []multistep.Step{&StepCreateVMInstance{}, &StepWaitForIP{}}}).Run(context.Background(), state)
	require.ErrorIs(t, state.Get("error").(error), context.DeadlineExceeded)
	require.True(t, d.DestroyVmInstanceCalled)
	require.True(t, d.DeleteVmInstanceCalled)
	require.Equal(t, "owned", d.DestroyVmInstanceUuid)
	require.Equal(t, "owned", d.DeleteVmInstanceUuid)
}
