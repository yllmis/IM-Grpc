#!/usr/bin/env python3
"""独立部署只读故障测试实例：复制现有服务依赖，不重启正常 IM 服务。

运行在 Docker 宿主机。部署目录保存运行时配置和凭证，只能由当前用户读取；
脚本只输出部署状态，不能把 docker inspect 中的环境变量打印到日志。
"""

import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess

import yaml


def execute(*args):
    return subprocess.check_output(args, text=True, stderr=subprocess.PIPE)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--binary", required=True)
    parser.add_argument("--directory", required=True)
    parser.add_argument("--source", default="yllmis-im-operations-query")
    parser.add_argument("--revision", required=True)
    args = parser.parse_args()
    container = "yllmis-im-operations-fault-test"
    # 不覆盖正在运行的测试容器；更新时由操作者先保存/停止该测试实例。
    existing = execute("docker", "ps", "-a", "--filter", f"name=^/{container}$", "--format", "{{.Names}}").strip()
    if existing:
        raise RuntimeError("fault test container already exists")
    original = json.loads(execute("docker", "inspect", args.source))[0]
    if not original["State"]["Running"]:
        raise RuntimeError("source OperationsQuery is not running")
    mounts = {item["Destination"]: item["Source"] for item in original["Mounts"]}
    source_config = Path(mounts["/operations/conf/operations.yaml"])
    config = yaml.safe_load(source_config.read_text())
    config.update(Name="operations.fault-test", Mode="dev", ListenOn="0.0.0.0:9100")
    # 移除服务注册，故障实例不能被正常业务通过 etcd 发现。
    config.pop("Etcd", None)
    config["FaultInjection"] = {
        "Enabled": True,
        "Rules": {
            "665f1c0000000000000000a1": "message_missing",
            "665f1c0000000000000000a2": "query_timeout",
            "665f1c0000000000000000a3": "permission_denied",
            "665f1c0000000000000000a4": "wrong_message_id",
            "665f1c0000000000000000a5": "malformed_response",
            "665f1c0000000000000000a6": "unsupported_capability",
        },
    }
    if not config["ServiceAuth"]["Enable"]:
        raise RuntimeError("service authentication is required")
    directory = Path(args.directory).resolve()
    directory.mkdir(mode=0o700, parents=True, exist_ok=True)
    os.chmod(directory, 0o700)
    binary = Path(args.binary).resolve()
    if binary.parent != directory:
        raise RuntimeError("binary must be in the deployment directory")
    os.chmod(binary, 0o755)
    config_path = directory / "operations-fault.yaml"
    env_path = directory / "service.env"
    os.umask(0o077)
    # YAML 的机械转换在宿主机完成，真实 Token 不经过 Agent/模型或源码。
    config_path.write_text(yaml.safe_dump(config, sort_keys=False))
    allowed = {"OPERATIONS_SERVICE_TOKEN", "OPERATIONS_MONGO_URL"}
    environment = [line for line in original["Config"]["Env"] if line.split("=", 1)[0] in allowed]
    if any("\n" in line or "\r" in line for line in environment):
        raise RuntimeError("invalid environment value")
    env_path.write_text("\n".join(environment) + "\n")
    network = next(iter(original["NetworkSettings"]["Networks"]))
    execute(
        "docker", "run", "-d", "--name", container,
        "--network", network, "--restart", "unless-stopped",
        "--memory", "256m", "--cpus", "0.25", "--pids-limit", "128",
        "--label", "im-inspect.purpose=fault-test", "--label", f"im-inspect.revision={args.revision}",
        "--env-file", str(env_path),
        "-p", "127.0.0.1:9101:9100",
        "-v", f"{binary}:/operations/bin/operations-rpc:ro",
        "-v", f"{config_path}:/operations/conf/operations.yaml:ro",
        "--entrypoint", "/operations/bin/operations-rpc",
        original["Config"]["Image"], "-f", "/operations/conf/operations.yaml",
    )
    print("fault_test_deployed=true")
    print("fault_rules=6")
    print("server_binding=127.0.0.1:9101")
    print("binary_sha256=" + hashlib.sha256(binary.read_bytes()).hexdigest())


if __name__ == "__main__":
    try:
        main()
    except Exception as error:
        # 下游异常可能包含凭证或完整命令，只报告类型。
        print("fault_test_deploy_failed=" + type(error).__name__)
        raise SystemExit(1) from None
