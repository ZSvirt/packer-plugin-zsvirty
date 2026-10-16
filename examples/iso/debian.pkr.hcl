# Requires a locally installed build of this branch or a release containing zsvirt-iso.
# Install Debian from a standard BIOS installer ISO. Verify its SHA256 against
# the publisher's checksums before making it available to ZSphere.
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
variable "root_disk_offering_uuid" { type = string }
variable "http_ip" { type = string }
variable "ssh_password" {
  type      = string
  sensitive = true
}
# SHA-512 crypt hash of the same installation password; generate locally.
variable "root_password_hash" {
  type      = string
  sensitive = true
}

source "zsvirt-iso" "debian" {
  management_ip = var.management_ip

  # Choose one authentication mode; leave the other pair empty.
  # Account: ZSVIRT_ACCOUNT_NAME + ZSVIRT_ACCOUNT_PASSWORD.
  # AK/SK:   ZSVIRT_ACCESS_KEY_ID + ZSVIRT_ACCESS_KEY_SECRET.
  account_name      = var.account_name
  account_password  = var.account_password
  access_key_id     = var.access_key_id
  access_key_secret = var.access_key_secret

  source_image_url        = var.iso_url
  source_image            = "packer-debian-installer"
  instance_name           = "packer-debian-install"
  image_name              = "packer-debian-template"
  format                  = "iso"
  platform                = "Linux"
  architecture            = "x86_64"
  network_name            = "build-port-group" # Port group name; L3 networks are also supported.
  image_storage_name      = "local-backup"
  root_disk_offering_uuid = var.root_disk_offering_uuid
  cpu_num                 = 2
  memory_size             = 2048 # MiB (2 GiB)
  # Alternative: remove cpu_num and memory_size, then set ONE offering field.
  # instance_offering_name = "min-offering"
  # instance_offering_uuid = "instance-offering-uuid"

  # The guest must reach this address; image storage also needs it for cd_content.
  http_ip = var.http_ip
  http_content = {
    "/preseed.cfg" = templatefile("${path.root}/preseed.cfg.pkrtpl", {
      root_password_hash = var.root_password_hash
    })
  }
  cd_label          = "PACKER_ANSWER"
  cd_files          = ["${path.root}/post-install.sh"]
  boot_wait         = "10s"
  boot_key_interval = "30ms"
  boot_command = [
    "<esc><wait>auto auto=true priority=critical ",
    "url=http://{{ .HTTPIP }}:{{ .HTTPPort }}/preseed.cfg ",
    "netcfg/choose_interface=auto netcfg/get_hostname=packer-debian netcfg/get_domain=local --- <enter>"
  ]
  installation_check_command = "test -f /etc/packer-installed && test -f /etc/packer-answer-cd-used && test \"$(findmnt -n -o FSTYPE /)\" = ext4"
  communicator               = "ssh"
  ssh_username               = "root"
  ssh_password               = var.ssh_password
  ssh_timeout                = "35m"
  image_ready_timeout        = "20m"
}

build {
  sources = ["source.zsvirt-iso.debian"]
  provisioner "shell" {
    inline = ["cat /etc/os-release", "uname -m"]
  }
}
