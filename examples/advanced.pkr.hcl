# Build from an existing image. Credentials use the ZSVIRT_* environment defaults.
packer {
  required_plugins {
    zsvirt = {
      version = ">= 0.1.0"
      source  = "github.com/ZSvirt/zsvirty"
    }
  }
}
variable "image_uuid" { type = string }
variable "network_uuids" { type = list(string) }
variable "image_storage_uuid" { type = string }
variable "ssh_username" { default = "ubuntu" }

source "zsvirt" "advanced" {
  image_uuid                = var.image_uuid
  network_uuids             = var.network_uuids
  default_network_uuid      = var.network_uuids[0]
  image_storage_uuid        = var.image_storage_uuid
  instance_name             = "packer-advanced"
  image_name                = "packer-advanced-output"
  cpu_num                   = 2
  memory_size               = 2048
  ssh_username              = var.ssh_username
  ssh_clear_authorized_keys = true
  api_timeout               = "30m"
  image_ready_timeout       = "20m"
  vm_running_timeout        = "10m"
  cleanup_timeout           = "5m"
  keep_source_image         = false
  keep_image_on_error       = false
  skip_export               = false
  # Optional placement / disks:
  # cluster_uuid = "..."
  # host_uuid = "..."
  # root_disk_offering_uuid = "..."
  # data_disk_offering_uuids = ["..."]
}
build {
  sources = ["source.zsvirt.advanced"]
  provisioner "shell" { inline = ["uname -m"] }
}
