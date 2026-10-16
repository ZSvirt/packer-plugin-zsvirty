// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zsvirt

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/terraform-zstack-modules/zsphere-sdk-go/pkg/param"
	"github.com/terraform-zstack-modules/zsphere-sdk-go/pkg/view"
)

type StepExportImage struct{}

type exportImageResult struct {
	view *view.ExportImageFromBackupStorageEventView
	err  error
}

func (s *StepExportImage) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	driver := stepDriver(ctx, state)

	// AC-003-01/03: Skip export when no image storage configured
	if config.SkipExport || config.BackupStorageUuid == "" {
		log.Printf("[INFO] Skipping image export: disabled or no image storage configured")
		ui.Say("Skipping image export: disabled or no image storage configured")
		return multistep.ActionContinue
	}

	if config.ImageUuid == "" {
		err := fmt.Errorf("image UUID is empty, cannot export")
		ui.Error(err.Error())
		state.Put("error", err)
		return multistep.ActionHalt
	}

	ui.Say("Exporting image to image storage...")

	exportImageParam := param.ExportImageFromBackupStorageParam{
		Params: param.ExportImageFromBackupStorageParamDetail{
			ImageUuid: config.ImageUuid,
		},
	}

	value, err := driver.ExportImage(config.BackupStorageUuid, exportImageParam)
	if ctx.Err() != nil {
		return halt(state, ctx.Err())
	}
	res := exportImageResult{view: value, err: err}

	if res.err != nil {
		if isUnsupportedExportError(res.err) {
			msg := fmt.Sprintf("Skipping image export: image storage may not support export (%v)", res.err)
			log.Printf("[WARN] %s", msg)
			ui.Say(msg)
			return multistep.ActionContinue
		}
		ui.Error("Failed to export image: " + res.err.Error())
		state.Put("error", res.err)
		return multistep.ActionHalt
	}

	if res.view == nil || res.view.ImageUrl == "" {
		return halt(state, fmt.Errorf("export returned no image URL"))
	}
	config.ImageUrl = res.view.ImageUrl
	config.Success = true
	ui.Say("Successfully exported image; URL is available in artifact metadata")
	return multistep.ActionContinue
}

func (s *StepExportImage) Cleanup(state multistep.StateBag) {}

func isUnsupportedExportError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no service deals with message") && strings.Contains(msg, "apiexportimagefrombackupstoragemsg") ||
		strings.Contains(msg, "does not support image export") || strings.Contains(msg, "image export is not supported")
}
