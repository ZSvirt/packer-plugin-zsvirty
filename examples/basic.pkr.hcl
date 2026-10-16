# Copyright ZStack.io, Inc. 2013, 2026
# SPDX-License-Identifier: MPL-2.0

# Account/password or AK/SK authentication with source image name lookup
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

variable "ssh_password" {
  type      = string
  sensitive = true
}

packer {
  required_plugins {
    zsvirt = {
      version = ">= 0.1.0"
      source  = "github.com/ZSvirt/zsvirty"
    }
  }
}

source "zsvirt" "basic" {
  management_ip = var.management_ip

  # Choose one authentication mode; leave the other pair empty.
  # Account: ZSVIRT_ACCOUNT_NAME + ZSVIRT_ACCOUNT_PASSWORD.
  # AK/SK:   ZSVIRT_ACCESS_KEY_ID + ZSVIRT_ACCESS_KEY_SECRET.
  account_name      = var.account_name
  account_password  = var.account_password
  access_key_id     = var.access_key_id
  access_key_secret = var.access_key_secret

  source_image = "CentOS7.9"
  network_name = "build-port-group"
  # Alternative: remove cpu_num and memory_size, then use ONE offering field.
  # instance_offering_name = "min-offering"
  # instance_offering_uuid = "instance-offering-uuid"
  cpu_num           = 2
  memory_size       = 2048 # MiB (2 GiB)
  instance_name     = "packer-basic"
  image_name        = "packer-basic-image"
  image_description = "Built with Packer - basic example"

  image_storage_name = "local-backup"

  ssh_username = "root"
  ssh_password = var.ssh_password
}

build {
  sources = ["source.zsvirt.basic"]

  provisioner "shell" {
    inline = [
      "echo 'Hello from Packer!'",
      "yum update -y",
    ]
  }
}
