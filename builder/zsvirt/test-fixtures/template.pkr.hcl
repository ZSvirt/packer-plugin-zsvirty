# Copyright ZStack.io, Inc. 2013, 2026
# SPDX-License-Identifier: MPL-2.0

# Offline validation only: these IDs and credentials are deliberately fake.
packer {
  required_plugins {
    zsvirt = {
      source  = "github.com/ZSvirt/zsvirty"
      version = ">= 0.1.0"
    }
  }
}

source "zsvirt" "image" {
  management_ip      = "192.0.2.10"
  access_key_id      = "test-key"
  access_key_secret  = "test-secret"
  image_uuid         = "source-image"
  network_name       = "build-port-group"
  cpu_num            = 2
  memory_size        = 2048
  image_storage_uuid = "image-storage"
  instance_name      = "packer-test"
  image_name         = "output"
  communicator       = "none"
}

source "zsvirt-iso" "installer" {
  management_ip              = "192.0.2.10"
  access_key_id              = "test-key"
  access_key_secret          = "test-secret"
  image_uuid                 = "installer-iso"
  network_uuids              = ["port-group", "l3-network"]
  default_network_uuid       = "port-group"
  cpu_num                    = 2
  memory_size                = 2048
  cpu_mode                   = "host-model"
  root_disk_offering_uuid    = "root-disk"
  image_storage_uuid         = "image-storage"
  instance_name              = "packer-iso-test"
  image_name                 = "iso-output"
  ssh_username               = "root"
  ssh_password               = "test-password"
  installation_check_command = "test -f /etc/os-release"
}

source "zsvirt-snapshot" "snapshot" {
  management_ip               = "192.0.2.10"
  access_key_id               = "test-key"
  access_key_secret           = "test-secret"
  source_volume_snapshot_uuid = "root-snapshot"
  image_storage_uuid          = "image-storage"
  image_name                  = "snapshot-output"
}

build {
  sources = ["source.zsvirt.image", "source.zsvirt-iso.installer", "source.zsvirt-snapshot.snapshot"]
}
