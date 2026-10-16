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

variable "iso_uuid" { type = string }
variable "network_name" { type = string }
variable "image_storage_uuid" { type = string }
variable "root_disk_offering_uuid" { type = string }
variable "http_ip" { type = string }
# The runner's source IP as seen by the guest; used by the WinRM firewall rule.
variable "winrm_client_address" { type = string }
# Directory named drivers containing 2k22/amd64 viostor, vioscsi and NetKVM folders.
variable "driver_directory" { type = string }
variable "winrm_password" {
  type      = string
  sensitive = true
}
locals {
  password_xml = replace(replace(replace(replace(replace(var.winrm_password, "&", "&amp;"), "<", "&lt;"), ">", "&gt;"), "\"", "&quot;"), "'", "&apos;")
}
source "zsvirt-iso" "windows-server2022" {
  management_ip = var.management_ip

  # Choose one authentication mode; leave the other pair empty.
  # Account: ZSVIRT_ACCOUNT_NAME + ZSVIRT_ACCOUNT_PASSWORD.
  # AK/SK:   ZSVIRT_ACCESS_KEY_ID + ZSVIRT_ACCESS_KEY_SECRET.
  account_name      = var.account_name
  account_password  = var.account_password
  access_key_id     = var.access_key_id
  access_key_secret = var.access_key_secret

  image_uuid              = var.iso_uuid
  instance_name           = "packer-server2022-install"
  image_name              = "packer-server2022-template"
  platform                = "WindowsVirtio"
  guest_os_type           = "Windows Server 2022 (64-bit)"
  architecture            = "x86_64"
  boot_mode               = "UEFI"
  cpu_mode                = "host-model" # Compatibility CPU mode.
  cpu_vendor              = "None"       # Preserve the native CPU vendor in this tested configuration.
  hyperv_enlightenments   = true
  machine_type            = "q35"
  network_name            = var.network_name # Port group name; L3 lookup is also supported.
  image_storage_uuid      = var.image_storage_uuid
  root_disk_offering_uuid = var.root_disk_offering_uuid
  cpu_num                 = 4
  memory_size             = 8192 # MiB
  # Alternative: remove cpu_num and memory_size, then use instance_offering_name.
  # instance_offering_name = "windows-build-offering"
  http_ip             = var.http_ip
  boot_menu_timeout   = "20s"
  boot_wait           = "0s"
  boot_key_interval   = "100ms"
  boot_command        = [for i in range(30) : "<spacebar><wait>"]
  shutdown_command    = "shutdown.exe /s /t 10 /f"
  shutdown_timeout    = "10m"
  communicator        = "winrm"
  winrm_username      = "Administrator"
  winrm_password      = var.winrm_password
  winrm_use_ssl       = true
  winrm_use_ntlm      = true
  winrm_insecure      = true # Temporary self-signed certificate created by setup.ps1.
  winrm_timeout       = "75m"
  image_ready_timeout = "30m"
  api_timeout         = "30m"
  cd_label            = "PACKER_ANSWER"
  cd_files            = [var.driver_directory]
  cd_content = {
    "Autounattend.xml" = templatefile("${path.root}/../windows-Autounattend.xml.pkrtpl", {
      admin_password_xml   = local.password_xml
      winrm_client_address = var.winrm_client_address
    })
    "setup.ps1"          = file("${path.root}/../windows-setup.ps1")
    "verify-install.ps1" = file("${path.root}/../windows-verify-install.ps1")
  }
  installation_check_command = "powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -File C:\\ProgramData\\Packer\\verify-install.ps1"
}
build {
  sources = ["source.zsvirt-iso.windows-server2022"]
}
