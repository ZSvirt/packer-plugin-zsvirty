# 开发与真实环境验收

命令从仓库根目录执行；凭证和私有资源配置保存在本机忽略文件中。

本地验证：`make test plugin-check`；重建 HCL schema：`go generate ./...`。真实云端测试需显式设置 `ZSVIRT_ACC=1`、`ZSVIRT_ACC_IMAGE_UUID`、`ZSVIRT_ACC_NETWORK_UUID`、`ZSVIRT_ACC_IMAGE_STORAGE_UUID`，然后执行 `go test ./builder/zsvirt -run '^TestAcceptanceCloud$' -v -count=1 -timeout=35m`。此测试会创建临时资源并清理镜像产物；默认不连接 SSH，设置 `ZSVIRT_ACC_COMMUNICATOR=ssh` 与 `ZSVIRT_ACC_SSH_USERNAME` 可验证 SSH。环境须具备对应资源权限及可用容量。

URL 导入验收可设置 `ZSVIRT_ACC_IMAGE_URL` 替代 `ZSVIRT_ACC_IMAGE_UUID`，两者必须且只能选一个。SSH 验收设置 `ZSVIRT_ACC_SSH_PASSWORD` 使用密码认证，留空则测试自动生成密钥。镜像地址与凭证保留在被忽略的本机 `.env`；测试会核对实际 SSH 用户、通过 provision hook 执行系统命令，并确认临时导入镜像被清理。

自动生成密钥认证要求客户机能处理云平台的公钥注入（例如启用且正确配置的 cloud-init），并使用对应 SSH 用户。密码登录成功不代表密钥注入可用；cloud-init 禁用的镜像应使用已有密码或私钥认证。

可先用密码登录临时 VM，挂载 Guest Tools ISO，并通过 SSH 执行其中的 `zs-tools-install.sh` 来准备基础镜像。Guest Tools 安装和 cloud-init 配置是两个步骤：还需检查 `cloud-init.disabled` 及将数据源固定为 `None` 的安装器配置。例如 Ubuntu 24.04.2 / cloud-init 24.4 使用 `DataSourceEc2` 识别 ZSphere；准备镜像时配置 `datasource_list: [ Ec2, None ]`，默认用户与 `ssh_username` 一致。移走安装器覆盖配置、启用 cloud-init、执行 `cloud-init clean --logs --machine-id` 后关机，制作新模板，再用新 VM 验证。只修改临时副本，保留原始源镜像。参考 [cloud-init 数据源实现](https://github.com/canonical/cloud-init/blob/24.4/cloudinit/sources/DataSourceEc2.py) 和[镜像缓存清理命令](https://docs.cloud-init.io/en/latest/reference/cli.html#clean)。

客户机还必须能够访问云端元数据服务；Guest Tools 安装或 cloud-init 启用本身不能证明密钥注入成功，应在新 VM 上验证首次启动和 cloud-init 正常完成。

设置 `ZSVIRT_ACC_SCENARIO=key-cleanup`，启用 SSH 并留空密码，可验证临时密钥注入和删除。测试镜像须具备 cloud-init、Python 3 和免密 sudo。测试等待 cloud-init 完成，先断言 SSH 用户的公钥文件包含临时密钥，再于清理步骤之后、关机和制镜像之前核对该用户与 root 的 `authorized_keys` 均已移除该密钥。

## ISO 安装验收

使用 [ISO 示例](https://github.com/ZSvirt/packer-plugin-zsvirty/tree/main/examples/iso)准备对应系统的私有配置。必须包含安装检查命令及应答文件设置的 SSH/WinRM 凭证。

```sh
ZSVIRT_ACC_ISO=1 \
ZSVIRT_ACC_ISO_CONFIG_FILE=/path/to/private-config.json \
go test ./builder/zsvirt -run '^TestAcceptanceISO$' -v -count=1 -timeout=120m
```

测试覆盖系统安装、卸载介质后的硬盘重启、文件传输、制镜像与导出、产物新 VM 启动及资源清理。应使用独立且容量充足的测试环境。已有源镜像保留，临时导入介质会清理。设置 `ZSVIRT_ACC_KEEP_OUTPUTS=1` 保留生成的镜像及导出文件供人工验证；未设置时默认清理输出。ISO 测试内部总期限为 90 分钟，嵌套环境可将 `winrm_timeout` 设为 `75m`。

应答文件通过 `cd_content` 挂载到第二张光盘，安装器读取它不依赖 Packer HTTP 服务。后续 SSH/WinRM、文件上传和安装验证仍需要运行器能够访问来宾 IP，无需公网 Internet。

Windows ISO 的按键启动提示有短暂的输入窗口。示例设置 `boot_menu_timeout = "20s"`、`boot_wait = "0s"`、`boot_key_interval = "100ms"` 并重复发送空格。启动 API 确认 Running 后插件立即发送按键，避免额外状态轮询消耗输入窗口。

`cpu_mode = "host-model"` 对应兼容 CPU；`hyperv_enlightenments = true` 对应平台 Windows 优化设置。`platform = "WindowsVirtio"` 显式请求 VirtIO 设备，应答光盘需提供匹配系统版本的存储和网卡驱动。Windows 11 通过已有 `tpm_key_provider_uuid` 随 VM 创建 TPM，插件不会删除密钥提供者。CPU、固件或单次安装速度差异不能直接作为已证明的根因。

阶段耗时来自 `ZSVIRT_ACC_TIMING_FILE`，使用 `scripts/report-iso-timings.py` 生成并列表格。仅完整通过的运行标为通过；超时、人工干预诊断以及缺少 ARM64 宿主机的场景分别记录。

Windows 首次登录脚本将记录写入 `C:\ProgramData\Packer\setup.log`。在本次 Win11 Public 网络测试中，重复写入已为 false 的 `AllowUnencrypted` 触发了 WinRM 防火墙配置错误；示例改为断言其保持 false，再配置 HTTPS 和限定运行器地址的规则，不改变网络类别。

从 Windows 快照生成镜像时，显式传入与源系统一致的 `platform`（如 `WindowsVirtio`）和 `architecture`，避免沿用 Linux 默认标签。验收同时检查快照产物的这两个字段。

## 网络验收矩阵

端口组为首选，普通 L3 网络同样需要验收。两类网络分别运行名称查询、UUID、多网卡与默认网卡场景；端口组与普通网络同名时验证优先选择端口组，使用 UUID 时验证选择指定网络。端口组名称重复应失败并提示 UUID。

分别验证平台 IPAM 和外部 DHCP/静态地址：创建 VM 时无 IP 应等待，超时或取消应进入原有资源清理流程。平台不能报告地址时显式指定 `ssh_host` / `winrm_host`。ISO 安装验收还需验证硬盘重启后的地址刷新。普通构建、ISO、快照和失败清理均需在真实环境运行；本地 mock 测试通过不代表云端验收通过。

离线插件加载验证：`make plugin-smoke PACKER=/path/to/packer`，测试三个 Builder 的 HCL 入口并保留现有配置字段。

Windows 示例使用 `shutdown_command = "shutdown.exe /s /t 10 /f"` 和 `shutdown_timeout = "10m"`，在卸载安装光驱及制作镜像前通过来宾系统正常关机，并等待平台状态变为 Stopped。必须验证重启后 HTTPS WinRM 仍然可用；Windows 系统日志中的事件 41/6008 表示未正常关机，即使平台已报告 Stopped。来宾关机超时会使构建失败，不会退回 API 强制关机。

## 提交范围与文档生成

单次运行的报告、截图、控制台会话、渲染后的应答文件和私有测试配置统一放入已忽略的 `.acceptance/`；凭证使用 `.env` 或已忽略的 `*.pkrvars.hcl` / `*.pkrvars.json`。`.env.example` 以及 example/sample 变量文件只能包含空值或占位符。`docs/plans/`、`docs/tasks/`、`docs/reports/` 中的本地过程记录也不提交，不要使用强制添加绕过忽略规则。

可复用的测试代码、脚本、示例模板和测试指南需要提交；单次验收结果及其私有资源清单留在本机。

修改配置字段或公开文档后执行 `make generate`：先通过 `go generate ./...` 生成 HCL schema，再将 `docs/` 渲染到临时 `.docs/`，编译到 `.web-docs/` 后删除临时目录。生成的 `*.hcl2spec.go` 和 `.web-docs/` 随源文件一起提交。测试 CI 会重新生成并检查差异；手动和 tag 触发的集成发布流程也会重新生成，要求工作区无变化后再通知发布。CI 不会替开发者自动提交生成结果。
