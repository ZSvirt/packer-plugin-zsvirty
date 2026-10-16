# ZSphere Packer Plugin

Build custom ZSphere images from an existing VM image, a volume snapshot, or an unattended operating-system installation from ISO. The output is a root-volume template stored in ZSphere; supported image storage can also provide an export URL.

## Installation

Declare `github.com/ZSvirt/zsvirty` in Packer's `required_plugins` block, select a plugin version containing the features you need, and run `packer init`. All three builders are included from v0.1.0.

```hcl
packer {
  required_plugins {
    zsvirt = {
      source  = "github.com/ZSvirt/zsvirty"
      version = ">= 0.1.0"
    }
  }
}
```

Authenticate with either `ZSVIRT_ACCESS_KEY_ID`/`ZSVIRT_ACCESS_KEY_SECRET` or `ZSVIRT_ACCOUNT_NAME`/`ZSVIRT_ACCOUNT_PASSWORD`, together with `ZSVIRT_MANAGEMENT_IP`. Explicit HCL values override environment defaults. Load local `.env` files yourself; Packer does not load them automatically.

The `zsvirt` local plugin alias is required for the builder names shown below. Keep existing `management_ip` and `ZSVIRT_*` configuration. Before publishing a release, run `make dev` and use version `>= 0.1.0`.

Network names prefer port groups and fall back to ordinary L3 networks; UUID inputs support both. See the [network configuration](/packer/plugins/builders/zsvirty/zsvirt) for IP discovery and explicit connection hosts.

## Builders

- [ZSphere](/packer/plugins/builders/zsvirty/zsvirt) (`zsvirt`): provision a VM from an existing image and create a template from its root disk.
- [ZSphere ISO](/packer/plugins/builders/zsvirty/iso) (`zsvirt-iso`): install an OS onto a blank root disk using HTTP/CD answer files, verify hard-disk boot, then provision and create a template. ZSphere userdata is not used for installation.

- [ZSphere Snapshot](/packer/plugins/builders/zsvirty/snapshot) (`zsvirt-snapshot`): convert an existing root-volume snapshot directly into an image, without starting a VM or running provisioners. Available starting with v0.1.0. The old `zsvirt` snapshot input remains compatible but is deprecated.

## Examples and requirements

The [examples directory](https://github.com/ZSvirt/packer-plugin-zsvirty/tree/main/examples) includes authentication, resource UUIDs, snapshot builds, placement and unattended ISO installation. Choose one template rather than loading all top-level examples together.

ISO examples cover Debian, Ubuntu Server, Rocky ARM and Windows Server. Configure the installer credentials, firmware, root-disk offering and an installation check for your OS. The runner must reach SSH/WinRM and, for boot commands, the host's VNC console; HTTP/CD delivery must be reachable from the guest and image storage.

Only the root disk is included in the resulting image. Build-owned temporary VMs, data disks and imported media are cleaned by default; existing source resources are retained. See each builder's configuration for retention settings, timeouts and export behavior. Keep credentials and rendered answer files in ignored local files.
