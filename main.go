// Copyright (c) HashiCorp, Inc.
// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"os"

	"github.com/ZSvirt/packer-plugin-zsvirty/builder/zsvirt"
	zsvirtVersion "github.com/ZSvirt/packer-plugin-zsvirty/version"

	"github.com/hashicorp/packer-plugin-sdk/plugin"
)

func main() {
	pps := plugin.NewSet()
	pps.RegisterBuilder(plugin.DEFAULT_NAME, new(zsvirt.Builder))
	pps.RegisterBuilder("iso", new(zsvirt.ISOBuilder))
	pps.RegisterBuilder("snapshot", new(zsvirt.SnapshotBuilder))
	pps.SetVersion(zsvirtVersion.PluginVersion)
	err := pps.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

}
