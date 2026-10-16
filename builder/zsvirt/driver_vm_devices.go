// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0
package zsvirt

import "github.com/terraform-zstack-modules/zsphere-sdk-go/pkg/param"

type vmCreateWithDevices struct {
	param.CreateVmInstanceParamDetail
	Devices vmDevices `json:"devices"`
}
type vmDevices struct {
	TPM vmTPM `json:"tpm"`
}
type vmTPM struct {
	Enable          bool   `json:"enable"`
	KeyProviderUUID string `json:"keyProviderUuid"`
}
