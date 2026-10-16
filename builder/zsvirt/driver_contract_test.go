// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0
package zsvirt

import (
	"context"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/terraform-zstack-modules/zsphere-sdk-go/pkg/param"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestResourceReadContracts(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		call       func(*ZSvirtDriver) (any, error)
	}{
		{"GetBackupStorage", "/zstack/v1/backup-storage/id", func(d *ZSvirtDriver) (any, error) { return d.GetBackupStorage("id") }},
		{"GetImage", "/zstack/v1/images/id", func(d *ZSvirtDriver) (any, error) { return d.GetImage("id") }},
		{"GetVmInstance", "/zstack/v1/vm-instances/id", func(d *ZSvirtDriver) (any, error) { return d.GetVmInstance("id") }},
		{"GetL3Network", "/zstack/v1/l3-networks/id", func(d *ZSvirtDriver) (any, error) { return d.GetL3Network("id") }},
		{"GetInstanceOffering", "/zstack/v1/instance-offerings/id", func(d *ZSvirtDriver) (any, error) { return d.GetInstanceOffering("id") }},
		{"GetVolume", "/zstack/v1/volumes/id", func(d *ZSvirtDriver) (any, error) { return d.GetVolume("id") }},
		{"GetZone", "/zstack/v1/zones/id", func(d *ZSvirtDriver) (any, error) { return d.GetZone("id") }},
		{"GetVolumeSnapshot", "/zstack/v1/volume-snapshots/id", func(d *ZSvirtDriver) (any, error) { return d.GetVolumeSnapshot("id") }},
	} {
		for _, response := range []struct {
			name, body string
			status     int
			invalid    bool
		}{
			{"success", `{"inventories":[{"uuid":"id","name":"resource"}]}`, 200, false},
			{"absent", `{"inventories":[]}`, 200, true},
			{"ambiguous", `{"inventories":[{"uuid":"id"},{"uuid":"other"}]}`, 200, true},
			{"missing UUID", `{"inventories":[{"name":"resource"}]}`, 200, true},
			{"forbidden", `{"error":{"code":"AUTH","details":"denied"}}`, 403, true},
		} {
			t.Run(tc.name+"/"+response.name, func(t *testing.T) {
				d := httpDriver(t, func(w http.ResponseWriter, r *http.Request) {
					require.Equal(t, http.MethodGet, r.Method)
					require.Equal(t, tc.path, r.URL.Path)
					w.WriteHeader(response.status)
					fmt.Fprint(w, response.body)
				})
				result, err := tc.call(d)
				if response.invalid {
					require.Error(t, err)
					require.Nil(t, result)
				} else {
					require.NoError(t, err)
					require.NotNil(t, result)
				}
			})
		}
	}
}

func TestResourceCreateContracts(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		call       func(*ZSvirtDriver) (any, error)
	}{
		{"CreateVmInstance", "/zstack/v1/vm-instances", func(d *ZSvirtDriver) (any, error) {
			return d.CreateVmInstance(param.CreateVmInstanceParam{Params: param.CreateVmInstanceParamDetail{Name: "build"}}, "")
		}},
		{"AddImage", "/zstack/v1/images", func(d *ZSvirtDriver) (any, error) {
			return d.AddImage(param.AddImageParam{Params: param.AddImageParamDetail{Name: "build"}})
		}},
		{"CreateImage", "/zstack/v1/images/root-volume-templates/from/volumes/root", func(d *ZSvirtDriver) (any, error) {
			return d.CreateImage("root", param.CreateRootVolumeTemplateFromRootVolumeParam{Params: param.CreateRootVolumeTemplateFromRootVolumeParamDetail{Name: "build"}})
		}},
		{"CreateImageFromVolumeSnapshot", "/zstack/v1/images/root-volume-templates/from/volume-snapshots/snapshot", func(d *ZSvirtDriver) (any, error) {
			return d.CreateImageFromVolumeSnapshot("snapshot", param.CreateRootVolumeTemplateFromVolumeSnapshotParam{Params: param.CreateRootVolumeTemplateFromVolumeSnapshotParamDetail{Name: "build"}})
		}},
		{"CreateVolumeSnapshot", "/zstack/v1/volumes/root/volume-snapshots", func(d *ZSvirtDriver) (any, error) {
			return d.CreateVolumeSnapshot("root", param.CreateVolumeSnapshotParam{Params: param.CreateVolumeSnapshotParamDetail{Name: "build"}})
		}},
		{"CreateDataVolume", "/zstack/v1/volumes/data", func(d *ZSvirtDriver) (any, error) {
			return d.CreateDataVolume(param.CreateDataVolumeParam{Params: param.CreateDataVolumeParamDetail{Name: "build"}})
		}},
	} {
		for _, response := range []struct {
			name, body string
			invalid    bool
		}{
			{"success", `{"inventory":{"uuid":"created"}}`, false},
			{"missing UUID", `{"inventory":{}}`, true},
			{"API error", `{"error":{"code":"FAILED","details":"cannot create"}}`, true},
		} {
			t.Run(tc.name+"/"+response.name, func(t *testing.T) {
				d := httpDriver(t, func(w http.ResponseWriter, r *http.Request) {
					require.Equal(t, http.MethodPost, r.Method)
					require.Equal(t, tc.path, r.URL.Path)
					var body struct {
						Params struct {
							Name string `json:"name"`
						} `json:"params"`
					}
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					require.Equal(t, "build", body.Params.Name)
					fmt.Fprint(w, response.body)
				})
				result, err := tc.call(d)
				if response.invalid {
					require.Error(t, err)
					require.Nil(t, result)
				} else {
					require.NoError(t, err)
					require.NotNil(t, result)
				}
			})
		}
	}
}

func TestAccessClientAuthenticationLifecycle(t *testing.T) {
	for _, mode := range []string{"account", "key", "failure", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if mode == "failure" {
					w.WriteHeader(401)
					fmt.Fprint(w, `{"error":{"details":"secret-password"}}`)
					return
				}
				if mode == "account" {
					require.Equal(t, "/zstack/v1/accounts/login", r.URL.Path)
					var body map[string]map[string]string
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					require.Equal(t, "admin", body["logInByAccount"]["accountName"])
					require.Equal(t, fmt.Sprintf("%x", sha512.Sum512([]byte("secret-password"))), body["logInByAccount"]["password"])
					fmt.Fprint(w, `{"inventory":{"uuid":"session"}}`)
				} else {
					require.Equal(t, "/zstack/v1/zones", r.URL.Path)
					require.Contains(t, r.Header.Get("Authorization"), "ZStack key:")
					fmt.Fprint(w, `{"inventories":[]}`)
				}
			}))
			defer server.Close()
			c := AccessConfig{Host: server.URL, AccountName: "admin", AccountPassword: "secret-password"}
			if mode == "key" {
				c.AccountName = ""
				c.AccountPassword = ""
				c.AccessKeyId = "key"
				c.AccessKeySecret = "secret"
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "cancelled" {
				cancel()
			}
			d, err := c.driver(ctx, defaultAPITimeout)
			if mode == "failure" || mode == "cancelled" {
				require.Error(t, err)
				require.NotContains(t, err.Error(), "secret-password")
				require.Nil(t, d)
				if mode == "cancelled" {
					require.Zero(t, calls)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, d)
				require.Equal(t, 1, calls)
			}
		})
	}
}

func TestMutationActionContracts(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, action string
		call                       func(*ZSvirtDriver) error
	}{
		{"destroy VM", "DELETE", "/zstack/v1/vm-instances/id", "", func(d *ZSvirtDriver) error { return d.DestroyVmInstance("id") }},
		{"expunge VM", "PUT", "/zstack/v1/vm-instances/id/actions", "expungeVmInstance", func(d *ZSvirtDriver) error { return d.DeleteVmInstance("id") }},
		{"delete disk", "DELETE", "/zstack/v1/volumes/id", "", func(d *ZSvirtDriver) error { return d.DeleteDataVolume("id") }},
		{"expunge disk", "PUT", "/zstack/v1/volumes/id/actions", "expungeDataVolume", func(d *ZSvirtDriver) error { return d.ExpungeDataVolume("id") }},
		{"delete export", "DELETE", "/zstack/v1/backup-storage/bs/exported-images/id", "", func(d *ZSvirtDriver) error { return d.DeleteExportedImage("bs", "id") }},
		{"guest tools", "PUT", "/zstack/v1/vm-instances/id/actions", "attachGuestToolsIsoToVm", func(d *ZSvirtDriver) error { return d.AttachGuestToolsToVm("id") }},
	} {
		for _, fail := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/fail=%t", tc.name, fail), func(t *testing.T) {
				d := httpDriver(t, func(w http.ResponseWriter, r *http.Request) {
					require.Equal(t, tc.method, r.Method)
					require.Equal(t, tc.path, r.URL.Path)
					if tc.action != "" {
						var body map[string]any
						require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
						require.Contains(t, body, tc.action)
					} else {
						require.Equal(t, "Permissive", r.URL.Query().Get("deleteMode"))
					}
					if fail {
						w.WriteHeader(403)
						fmt.Fprint(w, `{"error":{"code":"DENIED"}}`)
					} else {
						fmt.Fprint(w, `{}`)
					}
				})
				err := tc.call(d)
				if fail {
					require.ErrorContains(t, err, "DENIED")
				} else {
					require.NoError(t, err)
				}
			})
		}
	}
}

func TestQueryResourceNames(t *testing.T) {
	for _, tc := range []struct {
		path string
		call func(*ZSvirtDriver) (any, error)
	}{
		{"backup-storage", func(d *ZSvirtDriver) (any, error) { return d.QueryBackStorage("image store") }},
		{"images", func(d *ZSvirtDriver) (any, error) { return d.QueryImage("image store") }},
		{"instance-offerings", func(d *ZSvirtDriver) (any, error) { return d.QueryInstanceOffering("image store") }},
	} {
		t.Run(tc.path, func(t *testing.T) {
			d := httpDriver(t, func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/zstack/v1/"+tc.path, r.URL.Path)
				require.Equal(t, "name=image store", r.URL.Query().Get("q"))
				fmt.Fprint(w, `{"inventories":[{"uuid":"one"},{"uuid":"two"}]}`)
			})
			rows, err := tc.call(d)
			require.NoError(t, err)
			require.Len(t, rows, 2, "preserve ambiguity for validation")
		})
	}
}

func TestDataVolumeExpungeRetriesOnlyBusy(t *testing.T) {
	for _, code := range []string{"VOLUME.1000", "PERMISSION_DENIED"} {
		t.Run(code, func(t *testing.T) {
			calls := 0
			d := httpDriver(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if calls == 1 {
					fmt.Fprintf(w, `{"error":{"code":%q}}`, code)
				} else {
					fmt.Fprint(w, `{}`)
				}
			})
			err := expungeDataVolumeWithRetry(context.Background(), d, "owned", time.Millisecond)
			if code == "VOLUME.1000" {
				require.NoError(t, err)
				require.Equal(t, 2, calls)
			} else {
				require.Error(t, err)
				require.Equal(t, 1, calls)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d := &MockDriver{ExpungeDataVolumeErr: &apiError{Code: "VOLUME.1000"}}
	require.ErrorIs(t, expungeDataVolumeWithRetry(ctx, d, "owned", time.Millisecond), context.Canceled)
}

func TestVMTPMCreatePayload(t *testing.T) {
	for _, provider := range []string{"", "existing-provider"} {
		t.Run(provider, func(t *testing.T) {
			d := httpDriver(t, func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/zstack/v1/vm-instances", r.URL.Path)
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				params := body["params"].(map[string]any)
				require.Equal(t, "build", params["name"])
				require.Equal(t, "CreateStopped", params["strategy"])
				require.Equal(t, []any{"net"}, params["l3NetworkUuids"])
				require.Equal(t, []any{"bootMode::UEFI"}, body["systemTags"])
				if provider == "" {
					require.NotContains(t, params, "devices")
				} else {
					require.Equal(t, map[string]any{"tpm": map[string]any{"enable": true, "keyProviderUuid": provider}}, params["devices"])
				}
				fmt.Fprint(w, `{"inventory":{"uuid":"created"}}`)
			})
			_, err := d.CreateVmInstance(param.CreateVmInstanceParam{
				BaseParam: param.BaseParam{SystemTags: []string{"bootMode::UEFI"}},
				Params:    param.CreateVmInstanceParamDetail{Name: "build", Strategy: strPtr("CreateStopped"), L3NetworkUuids: []string{"net"}},
			}, provider)
			require.NoError(t, err)
		})
	}
}
