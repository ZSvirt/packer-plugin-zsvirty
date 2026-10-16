#!/usr/bin/env python3
"""Run opt-in acceptance with private dotenv settings; never print credentials."""
import argparse
import os
from pathlib import Path
import shlex
import subprocess

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("scenario", choices=["cloud", "snapshot", "cancel", "export", "aksk", "iso", "ssh", "key-cleanup", "winrm", "offering", "data-disk", "network"])
parser.add_argument("--env-file", default=".env")
parser.add_argument("--log-name", help="File stem within .acceptance")
args = parser.parse_args()
root = Path(__file__).resolve().parent.parent
env = {k: v for k, v in os.environ.items() if not k.startswith(("ZSVIRT_", "ZSPHERE_", "ZSTACK_"))}
for line in Path(args.env_file).read_text().splitlines():
    if not line.strip() or line.lstrip().startswith("#"):
        continue
    key, value = line.split("=", 1)
    values = shlex.split(value, comments=True)
    if len(values) > 1:
        parser.error("dotenv values containing spaces must be quoted")
    env[key.strip()] = values[0] if values else ""
env.update({k: v for k, v in os.environ.items() if k.startswith(("ZSVIRT_", "ZSPHERE_", "ZSTACK_"))})
env.update(ZSVIRT_ACC="1", ZSVIRT_ACC_ISO="0", ZSVIRT_ACC_AKSK="0", ZSVIRT_ACC_OFFERING="0", ZSVIRT_ACC_DATA_DISK="0", ZSVIRT_ACC_NETWORK="0",
           ZSVIRT_ACC_SCENARIO="", ZSVIRT_ACC_CHECK_EXPORT="0")
test = "TestAcceptanceCloud"
if args.scenario in ("snapshot", "cancel", "key-cleanup"):
    env["ZSVIRT_ACC_SCENARIO"] = args.scenario
if args.scenario == "export":
    env.update(ZSVIRT_ACC_CHECK_EXPORT="1", ZSVIRT_ACC_SKIP_EXPORT="0")
if args.scenario in ("ssh", "key-cleanup", "winrm"):
    env["ZSVIRT_ACC_COMMUNICATOR"] = "winrm" if args.scenario == "winrm" else "ssh"
if args.scenario == "iso":
    if not env.get("ZSVIRT_ACC_ISO_CONFIG_FILE"):
        parser.error("ISO requires ZSVIRT_ACC_ISO_CONFIG_FILE")
    env["ZSVIRT_ACC_ISO"] = "1"
    test = "TestAcceptanceISO"
    env["ZSVIRT_ACC_ISO_CONFIG_FILE"] = str(Path(env["ZSVIRT_ACC_ISO_CONFIG_FILE"]).resolve())
if args.scenario == "network":
    env["ZSVIRT_ACC_NETWORK"] = "1"
    test = "TestAcceptanceSDKNetworkCompatibility"
if args.scenario == "data-disk":
    env["ZSVIRT_ACC_DATA_DISK"] = "1"
    test = "TestAcceptanceDataDisk"
if args.scenario == "offering":
    env["ZSVIRT_ACC_OFFERING"] = "1"
    test = "TestAcceptanceOffering"
if args.scenario == "aksk":
    env["ZSVIRT_ACC_AKSK"] = "1"
    test = "TestAcceptanceAccessKey"
output = root / ".acceptance"
output.mkdir(mode=0o700, exist_ok=True)
log_name = args.log_name or args.scenario
if Path(log_name).name != log_name:
    parser.error("log-name must be a file stem")
log = output / (log_name + ".log")
if args.scenario == "iso":
    env["ZSVIRT_ACC_TIMING_FILE"] = str(output / (log_name + "-timing.json"))
with log.open("w") as stream:
    log.chmod(0o600)
    result = subprocess.run(["go", "test", "./builder/zsvirt", "-run", "^" + test + "$",
                             "-count=1", "-v", "-timeout=" + ("120m" if args.scenario == "iso" else "60m")], cwd=root, env=env,
                            stdout=stream, stderr=subprocess.STDOUT)
print(f"{args.scenario}: {'PASS' if result.returncode == 0 else 'FAIL'}; log: {log}")
raise SystemExit(result.returncode)
