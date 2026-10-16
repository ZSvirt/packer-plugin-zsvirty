# Packer Plugin ZSphere（中文）

`ZSphere` 是一个 HashiCorp [Packer](https://www.packer.io) 多组件插件，用于基于 ZSphere 云平台的 VM 实例或卷快照构建自定义镜像。完整的功能列表请参见 [docs](docs)。

> English version: see [README.md](README.md)。

## 功能概览

- **两种构建路径**
  - VM 构建：导入/选择源镜像 → 创建 VM → SSH 连接 → 执行 provisioner → 关机 → 从 root volume 生成镜像 → 导出
  - 快照直接构建（设置 `source_volume_snapshot_uuid`）：跳过 VM 创建、SSH 与 provisioner，直接基于已有卷快照生成镜像模板
- **两种鉴权方式**
  - 账号密码：`account_name` + `account_password`
  - AccessKey（推荐）：`access_key_id` + `access_key_secret`
- **多种镜像源**：已有镜像名 / `image_uuid` 直传 / `source_image_url` 从 HTTP(S) 远程导入
- **网络与规格**：支持名称解析为 UUID，或直接传入 `network_uuid` / `instance_offering_uuid`；可用 `cpu_num` + `memory_size` 替代实例规格
- **可配置超时**：`image_ready_timeout`、`vm_running_timeout`
- **导出兼容**：镜像存储不支持镜像导出时降级为告警，不会让整个构建失败
- **敏感信息脱敏**：日志摘要只输出 `*_set=true/false` 而不会泄漏密钥/密码

## 安装

### 使用 `packer init`

从 Packer 1.7 开始可用：

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

随后执行 `packer init`。

## 兼容性与本地开发

本项目复用 `packer-plugin-zstack v1.2.1`（提交 `989d0ca`），SDK 替换为 `zsphere-sdk-go v1.1.2`，保留原有构建、provisioner 和资源清理流程。

仓库与发行二进制名为 `packer-plugin-zsvirty`；安装配置中的 `zsvirt` 别名提供 `zsvirt`、`zsvirt-iso`、`zsvirt-snapshot` 三个模板入口。请保留该别名。

本地安装需要支持 `plugins install --path` 的 Packer（CI 使用 1.16.1）。尚未发布 release 时执行 `make dev` 本地构建并安装，模板版本约束使用 `version = ">= 0.1.0"`。发布后使用上方安装配置。

`network_name` 优先查询端口组；没有匹配项，或端点返回 HTTP 404、405、501 时，回退查询普通 L3 网络。权限、认证及其他 API 错误直接返回。同一查询有多个匹配时必须指定 UUID；端口组与普通 L3 网络同名时优先端口组，可用 L3 UUID 明确选择普通网络。

`network_uuid`、`network_uuids` 支持端口组和 L3 网络 UUID，`default_network_uuid` 指定连接网卡；未指定时优先平台报告的默认网络，再在无默认网络时选择首个可用网卡。SSH/WinRM 连接前会按 `vm_running_timeout` 等待 IP，ISO 重启后重新获取。平台无法报告外部 DHCP 或静态地址时，请显式设置 `ssh_host` 或 `winrm_host`。插件使用已有网络，不创建交换机、端口组或 L3 网络。

## 鉴权环境变量

| 变量 | 说明 |
| --- | --- |
| `ZSVIRT_MANAGEMENT_IP` | 管理节点地址（必填） |
| `ZSVIRT_PORT` | 管理节点端口（默认 8080） |
| `ZSVIRT_ACCOUNT_NAME` / `ZSVIRT_ACCOUNT_PASSWORD` | 账号密码方式 |
| `ZSVIRT_ACCESS_KEY_ID` / `ZSVIRT_ACCESS_KEY_SECRET` | AccessKey 方式（推荐） |

两种方式互斥，只能配置其一。

## 快速开始

### 1. 基于已有镜像构建

```hcl
source "zsvirt" "example" {
  management_ip       = "192.0.2.10"
  access_key_id     = env("ZSVIRT_ACCESS_KEY_ID")
  access_key_secret = env("ZSVIRT_ACCESS_KEY_SECRET")

  source_image           = "ubuntu-22.04"
  network_name           = "build-port-group"
  instance_offering_name = "medium-vm"
  image_storage_name    = "local-backup"

  instance_name     = "packer-example"
  image_name        = "packer-example-image"
  image_description = "Built by Packer"

  ssh_username = "root"
  ssh_password = "your-ssh-password"
}

build {
  sources = ["source.zsvirt.example"]

  provisioner "shell" {
    inline = ["echo hello"]
  }
}
```

### 2. 从远程 URL 导入并构建

```hcl
source "zsvirt" "from_url" {
  management_ip      = "192.0.2.10"
  account_name     = env("ZSVIRT_ACCOUNT_NAME")
  account_password = env("ZSVIRT_ACCOUNT_PASSWORD")

  source_image     = "ubuntu-22.04-from-url"
  source_image_url = "http://mirrors.example.com/ubuntu-22.04.qcow2"
  format           = "qcow2"     # 默认 qcow2
  platform         = "Linux"     # 默认 Linux
  guest_os_type    = "Ubuntu"

  network_name        = "build-port-group"
  cpu_num             = 2
  memory_size         = 4096
  image_storage_name = "local-backup"

  instance_name = "packer-from-url"
  image_name    = "packer-from-url-image"
  ssh_username  = "root"
  ssh_password  = "your-ssh-password"
}
```

### 3. UUID 直通模式（跳过名称查询）

```hcl
source "zsvirt" "uuid_passthrough" {
  management_ip       = "192.0.2.10"
  access_key_id     = env("ZSVIRT_ACCESS_KEY_ID")
  access_key_secret = env("ZSVIRT_ACCESS_KEY_SECRET")

  image_uuid             = "xxxxxxxx-image-uuid-xxxxxxxx"
  network_uuid           = "xxxxxxxx-net-uuid-xxxxxxxx"
  instance_offering_uuid = "xxxxxxxx-offer-uuid-xxxxxxxx"
  image_storage_uuid    = "xxxxxxxx-bs-uuid-xxxxxxxx"

  instance_name = "packer-uuid"
  image_name    = "packer-uuid-image"
  ssh_username  = "root"
  ssh_password  = "your-ssh-password"
}
```

### 4. 从已有根云盘快照构建（跳过 VM/SSH）

`zsvirt-snapshot` 从 v0.1.0 开始提供。兼容的 `zsvirt + source_volume_snapshot_uuid` 用法继续兼容，新构建会提示迁移。新入口不接受 SSH、网络或 VM 规格配置，也不会执行 provisioner。

```hcl
source "zsvirt-snapshot" "from_snapshot" {
  management_ip      = "192.0.2.10"
  account_name     = env("ZSVIRT_ACCOUNT_NAME")
  account_password = env("ZSVIRT_ACCOUNT_PASSWORD")

  source_volume_snapshot_uuid = "snapshot-uuid-here"

  image_name          = "packer-from-snapshot"
  image_description   = "Built from ZSphere volume snapshot"
  platform            = "Linux"
  architecture        = "x86_64"
  image_storage_name = "local-backup"
}

build {
  sources = ["source.zsvirt-snapshot.from_snapshot"]
}
```

> 该模式下 Packer 不会创建 VM，也不会建立 SSH 连接、不会执行 provisioner。

## 配置项参考

### 鉴权（必填一组）

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `management_ip` | string | 管理节点 IP，必填 |
| `port` | int | 管理节点端口，默认 8080 |
| `account_name` / `account_password` | string | 账号密码（仅支持平台管理员 `admin`） |
| `access_key_id` / `access_key_secret` | string | AccessKey 方式（推荐） |

### 镜像参数

| 字段 | 说明 |
| --- | --- |
| `source_image` | 源镜像名称 |
| `image_uuid` | 源镜像 UUID（提供后跳过名称查询） |
| `source_image_url` | HTTP/HTTPS 远程镜像 URL |
| `format` | 镜像格式（qcow2、raw 等，默认 qcow2） |
| `platform` | 平台类型（Linux/Windows，默认 Linux） |
| `guest_os_type` | 客户机 OS 类型（Ubuntu、CentOS …） |
| `architecture` | CPU 架构（x86_64、aarch64） |
| `image_name` | 目标镜像名称 |
| `image_description` | 目标镜像描述（默认与名称相同） |
| `source_volume_snapshot_uuid` | 已有卷快照 UUID，启用快照直建模式 |

### 网络与实例

| 字段 | 说明 |
| --- | --- |
| `network_name` / `network_uuid` | 端口组优先，兼容 L3 网络名称或 UUID |
| `instance_offering_name` / `instance_offering_uuid` | 实例规格 |
| `hyperv_enlightenments` | 可选 Windows Hyper-V 优化开关；不填继承平台设置，可按已验证的 VM 配置启用 |
| `tpm_key_provider_uuid` | 可选的现有密钥提供程序 UUID；创建 VM 时配置 vTPM，须配合 `boot_mode = "UEFI"`；不会删除密钥提供程序 |
| `cpu_vendor` | 可选 `None`（关闭 CPU 厂商伪装）或 `AuthenticAMD`；不填继承平台设置。嵌套 Intel 环境中的 Windows 可尝试 `None` |
| `cpu_mode` | 可选 `none` / `host-model` / `host-passthrough`；不填时遵循平台默认值 |
| `cpu_num` / `memory_size` | 自定义 CPU / 内存（MiB），可替代实例规格 |
| `instance_name` | 临时 VM 名称 |

### 镜像存储

| 字段 | 说明 |
| --- | --- |
| `image_storage_name` / `image_storage_uuid` | 镜像存放的镜像存储；**两者至少配置一个** |

### SSH（仅常规构建路径需要）

| 字段 | 说明 |
| --- | --- |
| `ssh_username` | SSH 用户名 |
| `ssh_password` | SSH 密码 |

### 超时

| 字段 | 默认值 | 说明 |
| --- | --- | --- |
| `image_ready_timeout` | `5m` | 等待镜像变为 Ready 状态的最长时间 |
| `vm_running_timeout` | `5m` | 等待 VM 进入 Running 状态的最长时间 |

## 构建步骤说明

**常规构建路径：**

```
StepPreValidate → StepAddImage（仅 source_image_url）→ StepWaitForImageReady
                → StepSourceImageValidate（仅命名/UUID 模式）
                → StepInstanceOfferingValidate（可选）
                → StepCreateSSHKey → StepCreateVMInstance → StepWaitForRunning
                → StepAttachGuestTools → StepWaitForIP → StepConnect → StepProvision
                → StepStopVmInstance → StepCreateImage → StepExpungeVmInstance
                → StepExportImage
```

**快照直建路径：**

```
StepPreValidate → StepCreateImageFromSnapshot → StepWaitForImageReady → StepExportImage
```

## E2E 测试

参考 [`examples/basic.pkr.hcl`](examples/basic.pkr.hcl) 与 [`examples/load_images.sh`](examples/load_images.sh)。

```bash
PACKER_PLUGIN_PATH=$(pwd) packer validate examples/basic.pkr.hcl
PACKER_PLUGIN_PATH=$(pwd) packer build    examples/basic.pkr.hcl
```

> **建议**：`source_image_url` 模式下，镜像存储优先用 `image_storage_name`，插件会自动解析为 UUID，免去记忆 UUID。

## 行为约定

- 镜像存储参数必填（无论是常规构建还是快照构建），因为生成的镜像会存放在镜像存储中，并可能从该镜像存储导出。
- 镜像存储不支持导出时，构建会跳过导出步骤并打印警告，整个构建仍判定为成功。
- 创建临时 SSH 密钥仅在 `ssh_password` 与 `ssh_private_key_file` 都未提供时发生，密钥仅保留在内存中。

## ISO 无人值守安装

使用 `source "zsvirt-iso"`，指定空白根盘规格，通过 HTTP 或辅助 CD 提供安装应答文件。安装器负责配置 SSH/WinRM 账号和凭证，不依赖 ZSphere userdata。插件在卸载安装介质、从硬盘重启后再次检查系统，确认安装成功再制作镜像。

详细配置见 [ISO builder 文档](docs/builders/iso.mdx)和 [Debian、Ubuntu、Rocky ARM、Windows 示例](examples/iso/README.md)。网络、放置、超时和资源清理配置见 [ZSphere builder 文档](docs/builders/zsvirt.mdx)。

## 开发与测试

本地测试与插件校验：`make test plugin-check`。`make plugin-smoke` 在临时插件目录安装并验证三个 Builder，不访问云平台；可用 `make plugin-smoke PACKER=/path/to/packer` 指定 Packer。生成 HCL schema 和官网文档：`make generate`。真实云端验收步骤见[验收指南](docs/testing_zh.md)。

各版本的改动记录见 [GitHub Releases](https://github.com/ZSvirt/packer-plugin-zsvirty/releases)。

## 贡献

- 发现 Bug 或对使用有疑问，请在 GitHub Issue 中反馈。
- 欢迎 PR；功能性改动请先开 Issue 讨论。

推荐字段为 `management_ip`（管理节点 IP）、`image_storage_name` / `image_storage_uuid`（镜像存储）。旧字段 `zstack_host`、`backup_storage_name`、`backup_storage_uuid` 作为弃用别名继续可用；新旧字段同时填写时必须一致。
环境变量优先级为显式 HCL > `ZSVIRT_*` > `ZSPHERE_*` > `ZSTACK_*`。管理 IP 对应 `ZSVIRT_MANAGEMENT_IP`，兼容 `ZSPHERE_HOST` / `ZSTACK_HOST`；端口和认证变量沿用相同后缀。认证按整组选择，部分填写会报错，不跨来源拼接凭据。
镜像存储不自动读取专用环境变量；如需环境传参，在 HCL 声明同名变量后使用 `PKR_VAR_image_storage_name` 或 `PKR_VAR_image_storage_uuid`。
普通与 ISO 构建可选择 `instance_offering_name` / `instance_offering_uuid`，或者同时填写正整数 `cpu_num` 和 `memory_size`；两种方式互斥。内存单位为 MiB，例如 `2048` = 2 GiB。快照构建无需虚拟机规格。
