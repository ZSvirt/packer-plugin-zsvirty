// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zsvirt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/terraform-zstack-modules/zsphere-sdk-go/pkg/param"
	"github.com/terraform-zstack-modules/zsphere-sdk-go/pkg/view"
)

// Retain SDK wire models with a transport that enforces cancellation and deadlines.
type ZSvirtDriver struct {
	client  *apiClient
	ctx     context.Context
	timeout time.Duration
	secrets []string
}

func (d *ZSvirtDriver) WithContext(ctx context.Context) Driver {
	copy := *d
	copy.ctx = ctx
	return &copy
}

func (d *ZSvirtDriver) request(fn func(context.Context) error) error {
	ctx := d.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := d.timeout
	if timeout <= 0 {
		timeout = defaultAPITimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return err
	}
	err := fn(ctx)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return redactError(err, d.secrets)
}

type Driver interface {
	DeleteExportedImage(backupStorageUUID, imageUUID string) error
	WithContext(context.Context) Driver
	DeleteDataVolume(string) error
	ExpungeDataVolume(string) error
	GetBackupStorage(uuid string) (*view.BackupStorageInventoryView, error)
	QueryBackStorage(backupStorageName string) ([]view.BackupStorageInventoryView, error)
	GetImage(uuid string) (*view.ImageInventoryView, error)
	QueryImage(imageName string) ([]view.ImageInventoryView, error)
	GetVmInstance(uuid string) (*view.VmInstanceInventoryView, error)
	GetL3Network(uuid string) (*view.L3NetworkInventoryView, error)
	QueryL3Network(networkName string) ([]view.L3NetworkInventoryView, error)
	GetInstanceOffering(uuid string) (*view.InstanceOfferingInventoryView, error)
	QueryInstanceOffering(instanceOfferingName string) ([]view.InstanceOfferingInventoryView, error)
	GetVolume(uuid string) (*view.VolumeInventoryView, error)
	GetZone(uuid string) (*view.ZoneInventoryView, error)

	CreateVmInstance(vmInstance param.CreateVmInstanceParam, tpmKeyProviderUUID string) (*view.VmInstanceInventoryView, error)
	StopVminstance(uuid string) (*view.VmInstanceInventoryView, error)
	DestroyVmInstance(uuid string) error
	DeleteVmInstance(uuid string) error

	CreateVolumeSnapshot(volumeUuid string, params param.CreateVolumeSnapshotParam) (*view.VolumeSnapshotInventoryView, error)
	CreateImage(rootVolumeUuid string, params param.CreateRootVolumeTemplateFromRootVolumeParam) (*view.ImageInventoryView, error)
	CreateImageFromVolumeSnapshot(snapshotUuid string, params param.CreateRootVolumeTemplateFromVolumeSnapshotParam) (*view.ImageInventoryView, error)
	GetVolumeSnapshot(uuid string) (*view.VolumeSnapshotInventoryView, error)
	AddImage(param param.AddImageParam) (*view.ImageInventoryView, error)
	DeleteImage(uuid string) error
	ExpungeImage(uuid string) error
	CreateDataVolume(volume param.CreateDataVolumeParam) (*view.VolumeInventoryView, error)
	ExportImage(backupStorageUuid string, params param.ExportImageFromBackupStorageParam) (*view.ExportImageFromBackupStorageEventView, error)

	AttachGuestToolsToVm(vmUuid string) error
	AttachDataVolumeToVm(vmUuid, volumeUuid string) (*view.VolumeInventoryView, error)

	ValidateCredentials() error
}

func (d *ZSvirtDriver) GetBackupStorage(uuid string) (*view.BackupStorageInventoryView, error) {
	var out view.BackupStorageInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.Get(ctx, "v1/backup-storage", uuid, nil, &out)
	})
	if err != nil {
		return nil, fmt.Errorf("get BackupStorage: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("BackupStorage %s not found or returned empty inventory", uuid)
	}
	return &out, nil
}
func (d *ZSvirtDriver) GetImage(uuid string) (*view.ImageInventoryView, error) {
	var out view.ImageInventoryView
	err := d.request(func(ctx context.Context) error { return d.client.Get(ctx, "v1/images", uuid, nil, &out) })
	if err != nil {
		return nil, fmt.Errorf("get Image: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("Image %s not found or returned empty inventory", uuid)
	}
	return &out, nil
}
func (d *ZSvirtDriver) GetVmInstance(uuid string) (*view.VmInstanceInventoryView, error) {
	var out view.VmInstanceInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.Get(ctx, "v1/vm-instances", uuid, nil, &out)
	})
	if err != nil {
		return nil, fmt.Errorf("get VmInstance: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("VmInstance %s not found or returned empty inventory", uuid)
	}
	return &out, nil
}
func (d *ZSvirtDriver) GetL3Network(uuid string) (*view.L3NetworkInventoryView, error) {
	var out view.L3NetworkInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.Get(ctx, "v1/l3-networks", uuid, nil, &out)
	})
	if err != nil {
		return nil, fmt.Errorf("get L3Network: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("L3Network %s not found or returned empty inventory", uuid)
	}
	return &out, nil
}
func (d *ZSvirtDriver) GetInstanceOffering(uuid string) (*view.InstanceOfferingInventoryView, error) {
	var out view.InstanceOfferingInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.Get(ctx, "v1/instance-offerings", uuid, nil, &out)
	})
	if err != nil {
		return nil, fmt.Errorf("get InstanceOffering: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("InstanceOffering %s not found or returned empty inventory", uuid)
	}
	return &out, nil
}
func (d *ZSvirtDriver) GetVolume(uuid string) (*view.VolumeInventoryView, error) {
	var out view.VolumeInventoryView
	err := d.request(func(ctx context.Context) error { return d.client.Get(ctx, "v1/volumes", uuid, nil, &out) })
	if err != nil {
		return nil, fmt.Errorf("get Volume: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("Volume %s not found or returned empty inventory", uuid)
	}
	return &out, nil
}
func (d *ZSvirtDriver) GetZone(uuid string) (*view.ZoneInventoryView, error) {
	var out view.ZoneInventoryView
	err := d.request(func(ctx context.Context) error { return d.client.Get(ctx, "v1/zones", uuid, nil, &out) })
	if err != nil {
		return nil, fmt.Errorf("get Zone: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("Zone %s not found or returned empty inventory", uuid)
	}
	return &out, nil
}
func (d *ZSvirtDriver) GetVolumeSnapshot(uuid string) (*view.VolumeSnapshotInventoryView, error) {
	var out view.VolumeSnapshotInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.Get(ctx, "v1/volume-snapshots", uuid, nil, &out)
	})
	if err != nil {
		return nil, fmt.Errorf("get VolumeSnapshot: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("VolumeSnapshot %s not found or returned empty inventory", uuid)
	}
	return &out, nil
}
func (d *ZSvirtDriver) QueryBackStorage(name string) ([]view.BackupStorageInventoryView, error) {
	params := param.NewQueryParam()
	params.AddQ("name=" + name)
	var out []view.BackupStorageInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.List(ctx, "v1/backup-storage", &params, &out)
	})
	return out, err
}
func (d *ZSvirtDriver) QueryImage(name string) ([]view.ImageInventoryView, error) {
	params := param.NewQueryParam()
	params.AddQ("name=" + name)
	var out []view.ImageInventoryView
	err := d.request(func(ctx context.Context) error { return d.client.List(ctx, "v1/images", &params, &out) })
	return out, err
}
func (d *ZSvirtDriver) QueryL3Network(name string) ([]view.L3NetworkInventoryView, error) {
	params := param.NewQueryParam()
	params.AddQ("name=" + name)
	var out []view.L3NetworkInventoryView
	err := d.request(func(ctx context.Context) error {
		// Port groups share the L3 network wire fields and VM attachment UUIDs.
		// Prefer them by name, retaining ordinary L3 networks as a fallback.
		err := d.client.List(ctx, "v1/l3-networks/port-group", &params, &out)
		if err == nil && len(out) > 0 {
			return nil
		}
		if err != nil {
			var apiErr *apiError
			if !errors.As(err, &apiErr) || (apiErr.statusCode != http.StatusNotFound && apiErr.statusCode != http.StatusMethodNotAllowed && apiErr.statusCode != http.StatusNotImplemented) {
				return err
			}
		}
		out = nil
		return d.client.List(ctx, "v1/l3-networks", &params, &out)
	})
	return out, err
}
func (d *ZSvirtDriver) QueryInstanceOffering(name string) ([]view.InstanceOfferingInventoryView, error) {
	params := param.NewQueryParam()
	params.AddQ("name=" + name)
	var out []view.InstanceOfferingInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.List(ctx, "v1/instance-offerings", &params, &out)
	})
	return out, err
}
func (d *ZSvirtDriver) CreateVmInstance(params param.CreateVmInstanceParam, tpmKeyProviderUUID string) (*view.VmInstanceInventoryView, error) {
	var out view.VmInstanceInventoryView
	// SDK v1.1.2 omits devices. Extend only the documented create payload,
	// keeping the existing transport, polling, authentication and SDK fields.
	var request any = params
	if tpmKeyProviderUUID != "" {
		request = struct {
			param.BaseParam
			Params vmCreateWithDevices `json:"params"`
		}{params.BaseParam, vmCreateWithDevices{params.Params, vmDevices{vmTPM{true, tpmKeyProviderUUID}}}}
	}
	err := d.request(func(ctx context.Context) error {
		return d.client.Post(ctx, "v1/vm-instances", request, &out)
	})
	if err != nil {
		return nil, fmt.Errorf("CreateVmInstance: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("CreateVmInstance returned empty inventory UUID")
	}
	return &out, nil
}
func (d *ZSvirtDriver) AddImage(params param.AddImageParam) (*view.ImageInventoryView, error) {
	var out view.ImageInventoryView
	err := d.request(func(ctx context.Context) error { return d.client.Post(ctx, "v1/images", params, &out) })
	if err != nil {
		return nil, fmt.Errorf("AddImage: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("AddImage returned empty inventory UUID")
	}
	return &out, nil
}
func (d *ZSvirtDriver) CreateImage(rootVolumeUuid string, params param.CreateRootVolumeTemplateFromRootVolumeParam) (*view.ImageInventoryView, error) {
	var out view.ImageInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.Post(ctx, "v1/images/root-volume-templates/from/volumes/"+rootVolumeUuid, params, &out)
	})
	if err != nil {
		return nil, fmt.Errorf("CreateImage: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("CreateImage returned empty inventory UUID")
	}
	return &out, nil
}
func (d *ZSvirtDriver) CreateImageFromVolumeSnapshot(snapshotUuid string, params param.CreateRootVolumeTemplateFromVolumeSnapshotParam) (*view.ImageInventoryView, error) {
	var out view.ImageInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.Post(ctx, "v1/images/root-volume-templates/from/volume-snapshots/"+snapshotUuid, params, &out)
	})
	if err != nil {
		return nil, fmt.Errorf("CreateImageFromVolumeSnapshot: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("CreateImageFromVolumeSnapshot returned empty inventory UUID")
	}
	return &out, nil
}
func (d *ZSvirtDriver) CreateVolumeSnapshot(volumeUuid string, params param.CreateVolumeSnapshotParam) (*view.VolumeSnapshotInventoryView, error) {
	var out view.VolumeSnapshotInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.Post(ctx, "v1/volumes/"+volumeUuid+"/volume-snapshots", params, &out)
	})
	if err != nil {
		return nil, fmt.Errorf("CreateVolumeSnapshot: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("CreateVolumeSnapshot returned empty inventory UUID")
	}
	return &out, nil
}
func (d *ZSvirtDriver) CreateDataVolume(params param.CreateDataVolumeParam) (*view.VolumeInventoryView, error) {
	var out view.VolumeInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.Post(ctx, "v1/volumes/data", params, &out)
	})
	if err != nil {
		return nil, fmt.Errorf("CreateDataVolume: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("CreateDataVolume returned empty inventory UUID")
	}
	return &out, nil
}
func (d *ZSvirtDriver) StopVminstance(uuid string) (*view.VmInstanceInventoryView, error) {
	var out view.VmInstanceInventoryView
	params := map[string]any{"stopVmInstance": map[string]any{"type": "grace", "stopHA": "true"}}
	err := d.request(func(ctx context.Context) error {
		return d.client.Put(ctx, "v1/vm-instances", uuid, params, &out)
	})
	if err != nil {
		return nil, err
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("StopVmInstance returned empty inventory UUID")
	}
	return &out, nil
}
func (d *ZSvirtDriver) ExportImage(uuid string, params param.ExportImageFromBackupStorageParam) (*view.ExportImageFromBackupStorageEventView, error) {
	var out view.ExportImageFromBackupStorageEventView
	err := d.request(func(ctx context.Context) error {
		return d.client.PutWithRespKey(ctx, "v1/backup-storage", uuid, "", map[string]any{"exportImageFromBackupStorage": params.Params}, &out)
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}
func (d *ZSvirtDriver) AttachGuestToolsToVm(uuid string) error {
	return d.request(func(ctx context.Context) error {
		return d.client.Put(ctx, "v1/vm-instances", uuid, map[string]any{"attachGuestToolsIsoToVm": map[string]any{}}, nil)
	})
}
func (d *ZSvirtDriver) AttachDataVolumeToVm(vmUuid, volumeUuid string) (*view.VolumeInventoryView, error) {
	var out view.VolumeInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.Post(ctx, "v1/volumes/"+volumeUuid+"/vm-instances/"+vmUuid, param.AttachDataVolumeToVmParam{}, &out)
	})
	return &out, err
}
func (d *ZSvirtDriver) ValidateCredentials() error {
	params := param.NewQueryParam()
	var out []view.ZoneInventoryView
	return d.request(func(ctx context.Context) error { return d.client.List(ctx, "v1/zones", &params, &out) })
}
func (d *ZSvirtDriver) DestroyVmInstance(uuid string) error {
	return d.request(func(ctx context.Context) error {
		return d.client.Delete(ctx, "v1/vm-instances", uuid, string(param.DeleteModePermissive))
	})
}
func (d *ZSvirtDriver) DeleteImage(uuid string) error {
	err := d.request(func(ctx context.Context) error {
		return d.client.Delete(ctx, "v1/images", uuid, string(param.DeleteModePermissive))
	})
	if imageAlreadyAbsent(err, uuid) {
		return nil
	}
	return err
}
func (d *ZSvirtDriver) DeleteDataVolume(uuid string) error {
	return d.request(func(ctx context.Context) error {
		return d.client.Delete(ctx, "v1/volumes", uuid, string(param.DeleteModePermissive))
	})
}
func (d *ZSvirtDriver) DeleteVmInstance(uuid string) error {
	return d.request(func(ctx context.Context) error {
		return d.client.Put(ctx, "v1/vm-instances", uuid, map[string]any{"expungeVmInstance": map[string]any{}}, nil)
	})
}
func (d *ZSvirtDriver) ExpungeImage(uuid string) error {
	err := d.request(func(ctx context.Context) error {
		return d.client.Put(ctx, "v1/images", uuid, map[string]any{"expungeImage": map[string]any{}}, nil)
	})
	if imageAlreadyAbsent(err, uuid) {
		return nil
	}
	return err
}
func (d *ZSvirtDriver) ExpungeDataVolume(uuid string) error {
	return d.request(func(ctx context.Context) error {
		return d.client.Put(ctx, "v1/volumes", uuid, map[string]any{"expungeDataVolume": map[string]any{}}, nil)
	})
}

func (d *ZSvirtDriver) DeleteExportedImage(backupStorageUUID, imageUUID string) error {
	return d.request(func(ctx context.Context) error {
		return d.client.Delete(ctx, "v1/backup-storage/"+backupStorageUUID+"/exported-images", imageUUID, string(param.DeleteModePermissive))
	})
}

// A failed/cancelled import may be removed by the cloud before our cleanup.
// Only ignore a structured not-found error for this exact image, never text
// matches or missing dependencies belonging to another resource.
func imageAlreadyAbsent(err error, uuid string) bool {
	var apiErr *apiError
	if uuid == "" || !errors.As(err, &apiErr) || apiErr.Code != "SYS.1003" {
		return false
	}
	if len(apiErr.FormatArgs) == 3 {
		return apiErr.FormatArgs[0] == "imageUuid" && apiErr.FormatArgs[1] == uuid && apiErr.FormatArgs[2] == "ImageVO"
	}
	// ZSphere reports missing-resource arguments under opaque instead of formatArgs.
	value := func(key string) string {
		var s string
		_ = json.Unmarshal(apiErr.Opaque[key], &s)
		return s
	}
	action := value("arg.1")
	return value("arg.0") == "imageUuid" && value("arg.2") == uuid && value("arg.3") == "ImageVO" &&
		(action == "APIDeleteImageMsg" || action == "APIExpungeImageMsg")
}
