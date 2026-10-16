# Development and cloud acceptance

Run commands from the repository root. Keep credentials and private resource configuration in ignored local files.

Run `make test plugin-check` locally and `go generate ./...` to regenerate the HCL schema. Cloud acceptance is opt-in: set `ZSVIRT_ACC=1`, `ZSVIRT_ACC_IMAGE_UUID`, `ZSVIRT_ACC_NETWORK_UUID`, `ZSVIRT_ACC_IMAGE_STORAGE_UUID`, and the usual credentials, then run `go test ./builder/zsvirt -run '^TestAcceptanceCloud$' -v -count=1 -timeout=35m`. This creates temporary cloud resources and deletes its output image. Set `ZSVIRT_ACC_COMMUNICATOR=ssh` and `ZSVIRT_ACC_SSH_USERNAME` to exercise SSH; otherwise the communicator is disabled.

For URL-import acceptance, set `ZSVIRT_ACC_IMAGE_URL` instead of `ZSVIRT_ACC_IMAGE_UUID` (exactly one is required). With SSH enabled, `ZSVIRT_ACC_SSH_PASSWORD` selects password authentication; leave it empty to exercise generated-key authentication. Keep image URLs and credentials in the ignored local `.env`. The test checks the actual SSH user, runs system commands through the provision hook, and verifies that its imported source image is removed.

Generated-key authentication requires a guest image that processes the cloud's SSH-key injection (for example, enabled and correctly configured cloud-init) and the matching SSH user. A successful password login does not establish that generated-key injection works. Use an existing password/private key for images with cloud-init disabled.

To prepare such an image, boot a disposable VM with password access, attach the Guest Tools ISO and run its `zs-tools-install.sh` over SSH. Guest Tools installation and cloud-init configuration are separate steps: check for `cloud-init.disabled` and installer configuration that pins the datasource to `None`. For example, with Ubuntu 24.04.2 / cloud-init 24.4, `DataSourceEc2` recognizes ZSphere; the prepared image uses `datasource_list: [ Ec2, None ]` and a default user matching `ssh_username`. Remove the installer override, enable cloud-init, run `cloud-init clean --logs --machine-id`, then stop the VM and create a new template before testing a fresh VM. Apply changes to a disposable copy and retain the original source image. See the [cloud-init datasource implementation](https://github.com/canonical/cloud-init/blob/24.4/cloudinit/sources/DataSourceEc2.py) and [image cleanup command](https://docs.cloud-init.io/en/latest/reference/cli.html#clean).

The metadata endpoint must also be reachable from the guest. Guest Tools installation or an enabled cloud-init service alone does not prove that key injection works; validate a fresh boot and successful cloud-init completion.

Set `ZSVIRT_ACC_SCENARIO=key-cleanup` with SSH enabled and no password to verify generated-key injection and removal. The test image needs cloud-init, Python 3 and passwordless sudo. It waits for cloud-init to finish, confirms the temporary key exists for the SSH user, and checks both that user's and root's `authorized_keys` after the cleanup step and before shutdown/image creation.

For multi-NIC acceptance, replace `ZSVIRT_ACC_NETWORK_UUID` with comma-separated `ZSVIRT_ACC_NETWORK_UUIDS` and set `ZSVIRT_ACC_DEFAULT_NETWORK_UUID`. Optional `ZSVIRT_ACC_ZONE_UUID`, `ZSVIRT_ACC_CLUSTER_UUID`, and `ZSVIRT_ACC_HOST_UUID` constrain placement. The test checks actual NIC IP allocations, the default network and selected communicator IP, and requested placement. With SSH enabled it also verifies login through the selected network; it does not certify guest routing on every NIC. Set `ZSVIRT_ACC_CHECK_EXPORT=1` with export enabled to read four bytes from the output URL before destruction and require HTTP 404/410 afterward.

On the tested ZSphere 5.5.38 ImageStore backend, the delete-export API returned success while the exported raw file remained downloadable (HTTP 206). This is tracked as Q12-E in the acceptance record; Ceph passed the same deletion probe. Verify export cleanup on your deployed backend. `skip_export = true` avoids creating a download copy when one is not needed.

Set `ZSVIRT_ACC_ARCHITECTURE=aarch64` for ARM acceptance. This passes the architecture to the builder, checks the output image metadata, and, with SSH enabled, requires the guest to report `aarch64` from `uname -m`.

For cloud-init key acceptance, prepare a disposable source image with cloud-init enabled, its cache cleaned, and a default user matching `ZSVIRT_ACC_SSH_USERNAME`. The official [upstream ZStack creation guide](https://www.zstack.io/docs/4.5.0/html5/a_print_ZStack_User_Guide/topic/ZStack_User_Guide_0033_2.html) requires working DHCP/Userdata services and a User Data payload for cloud-init images. Pass plaintext cloud-config through `userdata` in HCL or `ZSVIRT_ACC_USER_DATA` in acceptance (for example, `#cloud-config\npreserve_hostname: true\n`). The SSH tag also generates cloud-config on the server; an extra payload cannot repair an unreachable metadata service. Platform key removal does not guarantee removal from guest files, so keep `ssh_clear_authorized_keys = true` for generated-key builds. Optional `ZSVIRT_ACC_SSH_BASTION_HOST`, `ZSVIRT_ACC_SSH_BASTION_USERNAME` and `ZSVIRT_ACC_SSH_BASTION_PASSWORD` exercise the SSH communicator through a reachable bastion.

For Windows acceptance, configure WinRM in a disposable source image first, then set `ZSVIRT_ACC_COMMUNICATOR=winrm`, `ZSVIRT_ACC_PLATFORM=WindowsVirtio`, and `ZSVIRT_ACC_WINRM_USERNAME`/`ZSVIRT_ACC_WINRM_PASSWORD`. Optional `ZSVIRT_ACC_CPU_NUM` and `ZSVIRT_ACC_MEMORY_SIZE` override test sizing (memory is in MiB). `ZSVIRT_ACC_WINRM_USE_SSL=1` selects HTTPS; `ZSVIRT_ACC_WINRM_INSECURE=1` skips certificate validation for a self-signed test certificate. The test verifies Administrator identity (or the configured username), a 64-bit OS, file upload/readback/removal, output platform/architecture, and image lifecycle. This does not certify Sysprep/generalization or every Windows version.

## ISO installation acceptance

Use an installer-specific configuration based on the [ISO examples](https://github.com/ZSvirt/packer-plugin-zsvirty/tree/main/examples/iso). The configuration must include the installation check command and credentials configured by its answer file.

```sh
ZSVIRT_ACC_ISO=1 \
ZSVIRT_ACC_ISO_CONFIG_FILE=/path/to/private-config.json \
go test ./builder/zsvirt -run '^TestAcceptanceISO$' -v -count=1 -timeout=120m
```

This test installs the OS, verifies hard-disk reboot and file transfer, creates/exports the image, starts a new VM from the output template and deletes test-owned resources. Configure an isolated test environment with sufficient capacity. Existing source images are retained and temporary imported media are cleaned. Set `ZSVIRT_ACC_KEEP_OUTPUTS=1` to retain generated images and exports for manual verification; otherwise outputs are deleted. The ISO test has a 90-minute context; nested environments can use `winrm_timeout = "75m"`.

Answer files are delivered on a second CD through `cd_content`; this does not require a guest connection to the Packer HTTP server. SSH/WinRM verification and file transfers still require a reachable guest IP. Windows optical boot prompts have a short key-input window: use `boot_menu_timeout = "20s"`, `boot_wait = "0s"`, and repeated space keys. `WindowsVirtio` explicitly requests VirtIO devices and requires matching guest drivers. `hyperv_enlightenments` selects the platform Windows optimization setting. Windows 11 TPM creation uses an existing `tpm_key_provider_uuid`; the provider is never deleted by the plugin.

## Network acceptance matrix

Run both port-group and ordinary L3 scenarios: name lookup, UUID passthrough, multiple NICs and default-network selection. With the same name on a port group and an ordinary L3 network, verify port-group priority and explicit L3 selection by UUID. Duplicate port-group names must require a UUID.

Cover platform IPAM and external DHCP/static addressing. A newly created VM with no IP must wait; timeout/cancellation must retain the existing cleanup guarantees. Supply `ssh_host` / `winrm_host` if the platform cannot report the address. ISO acceptance must refresh the address after hard-disk reboot. Run normal, ISO, snapshot and failure-cleanup paths on the target cloud; local mock tests alone do not establish cloud compatibility.

For offline plugin loading and HCL validation of all three builder aliases, run `make plugin-smoke PACKER=/path/to/packer`.

Windows examples use `shutdown_command = "shutdown.exe /s /t 10 /f"` and `shutdown_timeout = "10m"` for both hard-disk reboot and image sealing. Confirm that HTTPS WinRM survives reboot; Windows System events 41/6008 indicate an unclean shutdown even when the platform reports Stopped. Guest shutdown timeout fails the build without falling back to an API power-off.

## Repository files and generated documentation

Keep run reports, screenshots, console/session data, rendered answer files and private test configuration under the ignored `.acceptance/` directory. Keep credentials in `.env` or ignored `*.pkrvars.hcl` / `*.pkrvars.json` files. Only empty or placeholder values belong in `.env.example` and example/sample variable files. Local plans, tasks and reports under `docs/plans/`, `docs/tasks/` and `docs/reports/` are also ignored. Do not force-add these files.

Commit reusable test code, scripts, example templates and these testing guides. They document how to reproduce verification; individual runs and their private resource inventories are not repository documentation.

After changing configuration fields or public docs, run `make generate`. It runs `go generate ./...`, renders `docs/` into temporary `.docs/`, then compiles the public pages into `.web-docs/` and removes `.docs/`. Commit the generated `*.hcl2spec.go` and `.web-docs/` changes with their sources. The test CI regenerates and checks for differences; the manual/tag integration-release workflows also regenerate and require a clean working tree before notifying the integration release. CI does not commit generated changes for you.
