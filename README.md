# Packer Plugin ZSphere

The `ZSphere` multi-component plugin can be used with HashiCorp [Packer](https://www.packer.io)
to create custom images from ZSphere VM instances. For the full list of available features for this plugin see [docs](docs).

## Installation

### Using pre-built releases

#### Using the `packer init` command

Starting from version 1.7, Packer supports a new `packer init` command allowing
automatic installation of Packer plugins. Read the
[Packer documentation](https://www.packer.io/docs/commands/init) for more information.

To install this plugin, copy and paste this code into your Packer configuration. Then, run [`packer init`](https://www.packer.io/docs/commands/init).

```hcl
packer {
  required_plugins {
    zsvirt = {
      version = ">= 0.1.0"
      source  = "github.com/ZSvirt/zsvirty"
    }
  }
}
```

## Compatibility and local development

This project reuses [packer-plugin-zstack v1.2.1](https://github.com/chijiajian/packer-plugin-zstack/tree/989d0cae2fce87e551d0ef73415afde93b42d7f1) and uses `github.com/terraform-zstack-modules/zsphere-sdk-go v1.1.2`. The upstream build, provisioning and cleanup flows are retained.

The repository and release binary are named `packer-plugin-zsvirty`; the `zsvirt` alias in `required_plugins` provides the template entries `zsvirt`, `zsvirt-iso` and `zsvirt-snapshot`. Keep this alias in your templates.

Use Packer with `plugins install --path` support (CI uses 1.16.1). Before a release is published, build and install locally with `make dev`. Local builds use version `0.1.0-dev`; select `version = ">= 0.1.0"` in your template when using this build. Released versions use the installation block above.

`network_name` resolves port groups first, then ordinary L3 networks when no port group matches or the port-group endpoint returns HTTP 404, 405 or 501. Authentication, permission and other API errors are reported instead of falling back. Multiple matches within the selected query require an explicit UUID. If a port group and an ordinary L3 network share a name, the port group wins; use the L3 UUID to select the other network.

`network_uuid` and `network_uuids` accept port-group or L3-network UUIDs directly, with `default_network_uuid` selecting the connection NIC. Otherwise the VM's default network is preferred, then the first usable NIC when the platform reports no default. Before SSH/WinRM, including after an ISO reboot, the plugin polls for an IP using `vm_running_timeout`. If the platform cannot report an external DHCP/static address, set `ssh_host` or `winrm_host` explicitly. Packer does not create switches, port groups or L3 networks.

## Quick Start

The plugin supports two authentication methods:

- Account/password authentication with `ZSVIRT_MANAGEMENT_IP`, `ZSVIRT_ACCOUNT_NAME`, and `ZSVIRT_ACCOUNT_PASSWORD`
- Access key/secret authentication with `ZSVIRT_MANAGEMENT_IP`, `ZSVIRT_ACCESS_KEY_ID`, and `ZSVIRT_ACCESS_KEY_SECRET`

You can build from resource names or use UUID passthrough with `image_uuid`, `network_uuid`, and `instance_offering_uuid` to skip name-based lookups.

Image storage is required for image creation and export. In the normal VM build path, the builder stops the VM and creates the image template directly from the stopped root volume. When `source_volume_snapshot_uuid` is set, the builder creates the template from that existing volume snapshot instead.

Use `image_description` to set a custom description for the generated image.

The `zsvirt-snapshot` builder creates an image directly from an existing **root-volume snapshot** using `source_volume_snapshot_uuid`. Available starting with v0.1.0. The legacy `zsvirt` snapshot input remains supported with a deprecation warning in newer builds. In this mode no VM is created and SSH/provisioners are skipped; the builder creates the image template from that existing snapshot — see [`examples/from_snapshot.pkr.hcl`](examples/from_snapshot.pkr.hcl).

See the [`examples/`](examples) directory for ready-to-run HCL examples covering account/password auth, AK/SK auth, and UUID passthrough.

## E2E Test Method

Use the basic template to verify VM creation, SSH provisioning and image creation from the root volume. The optional shell script checks the guest environment; the configuration below also demonstrates URL-based image import:

- Template: [`examples/basic.pkr.hcl`](examples/basic.pkr.hcl)
- Provisioner script: [`examples/load_images.sh`](examples/load_images.sh)

### Recommended Test Config

For `source_image_url` imports, configure image storage by **name** whenever possible. The plugin resolves `image_storage_name` to UUID automatically, so users do not need to memorize UUID values.

```hcl
source "zsvirt" "e2e-test" {
  management_ip      = "192.0.2.10"
  port             = 8080
  account_name     = "admin"
  account_password = "pwd"

  source_image     = "docker-by-packer-image-compressed"
  source_image_url = "http://imageUrl/packer/docker-by-packer-image-compressed.qcow2"
  format           = "qcow2"
  platform         = "Linux"

  network_name           = "build-port-group"
  instance_offering_name = "medium-vm"
  image_storage_name    = "sftp-bs"

  instance_name = "packer-e2e-test"
  image_name    = "packer-e2e-test-image"
  ssh_username  = "root"
  ssh_password  = "pwd"
}

build {
  sources = ["source.zsvirt.e2e-test"]

  provisioner "shell" {
    script = "examples/load_images.sh"
  }
}
```

### Run

```bash
PACKER_PLUGIN_PATH=$(pwd) packer validate examples/basic.pkr.hcl
PACKER_PLUGIN_PATH=$(pwd) packer build examples/basic.pkr.hcl
```

### Backup Storage / Export Behavior

- `image_storage_name` or `image_storage_uuid` is required for both normal VM builds and `source_volume_snapshot_uuid` builds, because the generated image is stored in image storage and may be exported from it.
- If image storage does not support image export, export is skipped with a warning instead of failing the entire build.

### Configuration

For more information on how to configure the plugin, please read the
documentation located in the [`docs/`](docs) directory.

## Configuration and lifecycle

See the [ZSphere builder guide](docs/builders/zsvirt.mdx) for authentication, SSH/WinRM, networking, placement, timeouts and resource cleanup. Output images contain the root disk only; existing source images and snapshots are retained.

## Installing an operating system from ISO

Use `source "zsvirt-iso"` with a blank root-disk offering and installer answer files over HTTP or CD. The installer configures SSH/WinRM credentials; ZSphere userdata is not used. The builder verifies the installed OS again after detaching installation media and restarting from disk.

See the [ISO builder guide](docs/builders/iso.mdx) and [unattended installation examples](examples/iso/README.md) for Debian, Ubuntu, Rocky ARM and Windows.

## Development and testing

Run `make test plugin-check` for local tests and plugin validation. Run `make plugin-smoke` to install into a temporary plugin directory and validate all three builder aliases without contacting a cloud; use `make plugin-smoke PACKER=/path/to/packer` to select a Packer binary. Run `make generate` to regenerate the HCL schema and public web documentation. For opt-in tests against a real cloud, see the [acceptance guide](docs/testing.md).

Version-specific changes are listed in [GitHub Releases](https://github.com/ZSvirt/packer-plugin-zsvirty/releases).

## Contributing

* If you think you've found a bug in the code or you have a question regarding
  the usage of this software, please reach out to us by opening an issue in
  this GitHub repository.
* Contributions to this project are welcome: if you want to add a feature or a
  fix a bug, please do so by opening a Pull Request in this GitHub repository.
  In case of feature contribution, we kindly ask you to open an issue to
  discuss it beforehand.

Use `management_ip` for the management node IP and `image_storage_name` / `image_storage_uuid` for Image Storage. Deprecated aliases `zstack_host`, `backup_storage_name` and `backup_storage_uuid` remain supported; canonical and legacy values must agree if both are supplied.
Explicit HCL takes precedence over `ZSVIRT_*`, then `ZSPHERE_*`, then `ZSTACK_*`. The IP variable is `ZSVIRT_MANAGEMENT_IP`, with `ZSPHERE_HOST` / `ZSTACK_HOST` fallbacks; port and authentication use the same suffixes across namespaces. Credentials are selected as a whole group; incomplete groups fail validation and are never combined across sources.
Image Storage has no automatic dedicated environment defaults. Declare HCL variables and use `PKR_VAR_image_storage_name` or `PKR_VAR_image_storage_uuid` to supply these values.
Image and ISO builds accept either `instance_offering_name` / `instance_offering_uuid`, or both positive integers `cpu_num` and `memory_size`. These modes are mutually exclusive. Memory is in MiB: `2048` means 2 GiB. Snapshot builds do not need VM sizing.
