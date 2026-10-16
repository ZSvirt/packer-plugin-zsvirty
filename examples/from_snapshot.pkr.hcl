# Copyright ZStack.io, Inc. 2013, 2026
# SPDX-License-Identifier: MPL-2.0

# Requires plugin v0.1.0 or later.
# Build an image from an existing ZSphere volume snapshot (no VM/SSH/provisioning).
variable "management_ip" {
  type    = string
  default = env("ZSVIRT_MANAGEMENT_IP")
}

variable "account_name" {
  type    = string
  default = env("ZSVIRT_ACCOUNT_NAME")
}

variable "account_password" {
  type      = string
  sensitive = true
  default   = env("ZSVIRT_ACCOUNT_PASSWORD")
}

variable "access_key_id" {
  type      = string
  sensitive = true
  default   = env("ZSVIRT_ACCESS_KEY_ID")
}

variable "access_key_secret" {
  type      = string
  sensitive = true
  default   = env("ZSVIRT_ACCESS_KEY_SECRET")
}

variable "snapshot_uuid" {
  type        = string
  description = "UUID of the volume snapshot to convert into an image"
}

packer {
  required_plugins {
    zsvirt = {
      version = ">= 0.1.0"
      source  = "github.com/ZSvirt/zsvirty"
    }
  }
}

source "zsvirt-snapshot" "from_snapshot" {
  management_ip = var.management_ip

  # Choose one authentication mode; leave the other pair empty.
  # Account: ZSVIRT_ACCOUNT_NAME + ZSVIRT_ACCOUNT_PASSWORD.
  # AK/SK:   ZSVIRT_ACCESS_KEY_ID + ZSVIRT_ACCESS_KEY_SECRET.
  account_name      = var.account_name
  account_password  = var.account_password
  access_key_id     = var.access_key_id
  access_key_secret = var.access_key_secret

  source_volume_snapshot_uuid = var.snapshot_uuid

  image_name         = "packer-from-snapshot-image"
  image_description  = "Built with Packer from a ZSphere volume snapshot"
  platform           = "Linux"
  architecture       = "x86_64"
  image_storage_name = "local-backup"
}

build {
  sources = ["source.zsvirt-snapshot.from_snapshot"]
}
