#!/usr/bin/env bash
# Copyright ZStack.io, Inc. 2013, 2026
# SPDX-License-Identifier: MPL-2.0
set -euo pipefail

repo_dir="$(cd "$(dirname "$0")/.." && pwd)"
plugin_binary="${1:-$repo_dir/packer-plugin-zsvirty}"
packer_cmd="${PACKER:-packer}"
task_dir=$(mktemp -d)
trap 'rm -rf "$task_dir"' EXIT
export PACKER_PLUGIN_PATH="$task_dir/plugins"
unset ZSVIRT_MANAGEMENT_IP ZSVIRT_PORT ZSVIRT_ACCOUNT_NAME ZSVIRT_ACCOUNT_PASSWORD
unset ZSVIRT_ACCESS_KEY_ID ZSVIRT_ACCESS_KEY_SECRET
unset ZSPHERE_HOST ZSPHERE_PORT ZSPHERE_ACCOUNT_NAME ZSPHERE_ACCOUNT_PASSWORD
unset ZSPHERE_ACCESS_KEY_ID ZSPHERE_ACCESS_KEY_SECRET
unset ZSTACK_HOST ZSTACK_PORT ZSTACK_ACCOUNT_NAME ZSTACK_ACCOUNT_PASSWORD
unset ZSTACK_ACCESS_KEY_ID ZSTACK_ACCESS_KEY_SECRET
"$packer_cmd" plugins install --path "$plugin_binary" github.com/ZSvirt/zsvirty
"$packer_cmd" validate "$repo_dir/builder/zsvirt/test-fixtures/template.pkr.hcl"

# Validate the actual user-facing examples in both authentication modes.
export ZSVIRT_MANAGEMENT_IP=192.0.2.10
export PKR_VAR_ssh_password=offline-password
export PKR_VAR_snapshot_uuid=offline-snapshot
export PKR_VAR_iso_url=http://192.0.2.20/installer.iso
export PKR_VAR_root_disk_offering_uuid=offline-disk
export PKR_VAR_http_ip=192.0.2.30
export PKR_VAR_root_password_hash=offline-hash
export PKR_VAR_password_hash=offline-hash
export PKR_VAR_network_name=offline-port-group
export PKR_VAR_image_storage_uuid=offline-image-storage
export PKR_VAR_iso_uuid=offline-iso
export PKR_VAR_tpm_key_provider_uuid=offline-key-provider
export PKR_VAR_winrm_client_address=192.0.2.30
export PKR_VAR_winrm_password=offline-password
mkdir -p "$task_dir/drivers"
export PKR_VAR_driver_directory="$task_dir/drivers"
for mode in account aksk; do
  unset ZSVIRT_ACCOUNT_NAME ZSVIRT_ACCOUNT_PASSWORD ZSVIRT_ACCESS_KEY_ID ZSVIRT_ACCESS_KEY_SECRET
  if [[ "$mode" == account ]]; then
    export ZSVIRT_ACCOUNT_NAME=offline-admin ZSVIRT_ACCOUNT_PASSWORD=offline-password
  else
    export ZSVIRT_ACCESS_KEY_ID=offline-key ZSVIRT_ACCESS_KEY_SECRET=offline-secret
  fi
  for example in basic.pkr.hcl iso/debian.pkr.hcl from_snapshot.pkr.hcl iso/rocky-amd64/rocky.pkr.hcl iso/windows11/windows11.pkr.hcl iso/windows-server2022/windows-server2022.pkr.hcl; do
    "$packer_cmd" validate "$repo_dir/examples/$example"
  done
  "$packer_cmd" validate -var 'boot_mode=Legacy' "$repo_dir/examples/iso/rocky-amd64/rocky.pkr.hcl"
done
# Both pairs supplied must be rejected, including by the restricted snapshot schema.
export ZSVIRT_ACCOUNT_NAME=offline-admin ZSVIRT_ACCOUNT_PASSWORD=offline-password
for example in basic.pkr.hcl iso/debian.pkr.hcl from_snapshot.pkr.hcl; do
  if "$packer_cmd" validate "$repo_dir/examples/$example" > "$task_dir/conflict.log" 2>&1; then
    echo "Expected authentication conflict: $example" >&2
    exit 1
  fi
  grep -q 'mutually exclusive' "$task_dir/conflict.log"
done
