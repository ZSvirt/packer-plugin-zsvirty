# Requires plugin v0.1.0 or later with ISO/boot_mode support.
packer {
  required_plugins {
    zsvirt = {
      source  = "github.com/ZSvirt/zsvirty"
      version = ">= 0.1.0"
    }
  }
}
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

variable "iso_url" { type = string }
variable "network_name" { type = string }
variable "image_storage_uuid" { type = string }
variable "root_disk_offering_uuid" { type = string }
variable "http_ip" { type = string }
variable "boot_mode" {
  type    = string
  default = "UEFI"
  validation {
    condition     = contains(["UEFI", "Legacy"], var.boot_mode)
    error_message = "Boot mode must be UEFI or Legacy."
  }
}
variable "ssh_password" {
  type      = string
  sensitive = true
}
variable "password_hash" {
  type      = string
  sensitive = true
}

source "zsvirt-iso" "rocky" {
  management_ip = var.management_ip

  # Choose one authentication mode; leave the other pair empty.
  # Account: ZSVIRT_ACCOUNT_NAME + ZSVIRT_ACCOUNT_PASSWORD.
  # AK/SK:   ZSVIRT_ACCESS_KEY_ID + ZSVIRT_ACCESS_KEY_SECRET.
  account_name      = var.account_name
  account_password  = var.account_password
  access_key_id     = var.access_key_id
  access_key_secret = var.access_key_secret


  source_image_url        = var.iso_url
  source_image            = "packer-rocky-installer"
  instance_name           = "packer-rocky-install"
  image_name              = "packer-rocky-template"
  format                  = "iso"
  platform                = "Linux"
  architecture            = "x86_64"
  boot_mode               = var.boot_mode
  cpu_mode                = "host-model"
  network_name            = var.network_name
  image_storage_uuid      = var.image_storage_uuid
  root_disk_offering_uuid = var.root_disk_offering_uuid
  cpu_num                 = 2
  memory_size             = 4096 # MiB
  # Alternative: remove cpu_num and memory_size, then use instance_offering_name.
  # instance_offering_name = "build-offering"
  http_ip             = var.http_ip
  boot_wait           = "15s"
  boot_key_interval   = "100ms"
  image_ready_timeout = "20m"
  communicator        = "ssh"
  ssh_username        = "root"
  ssh_password        = var.ssh_password
  ssh_timeout         = "35m"
  cd_label            = "PACKER_ANSWER"
  cd_content = {
    "ks.cfg" = templatefile("${path.root}/../rocky.ks.pkrtpl", { root_password_hash = var.password_hash })
  }
  # Legacy preserves the installer's defaults; UEFI loads its kernel via GRUB.
  # sr0 is the installer ISO and sr1 is the generated answer CD.
  boot_command = var.boot_mode == "UEFI" ? [
    "c<wait>linux /images/pxeboot/vmlinuz inst.stage2=hd:sr0 inst.ks=hd:sr1:/ks.cfg<enter><wait>initrd /images/pxeboot/initrd.img<enter><wait>boot<enter>"
  ] : ["<up><tab><end> inst.ks=hd:sr1:/ks.cfg<enter>"]
  installation_check_command = "test -f /etc/packer-installed && . /etc/os-release && test \"$ID\" = rocky${var.boot_mode == "UEFI" ? " && test -d /sys/firmware/efi" : ""}"

}

build {
  sources = ["source.zsvirt-iso.rocky"]
}
