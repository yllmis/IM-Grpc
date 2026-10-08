#!/usr/bin/env python3
"""在 Docker 宿主机启动真实查询联调夹具，不写正常数据库、不注册业务服务。

先交叉编译 apps/{im,user}/rpc/queryfixture 到部署目录的
message-query-fixture 和 user-query-fixture；使用 --directory 指定受保护目录。
fixture.env 是 Git 外的服务端测试配置，经 SSH 安全复制后运行 Connector 测试。
测试后使用 --stop 停止此目录拥有的两个进程，保留独立测试数据供复查。
"""
import argparse
import json
import os
from pathlib import Path
import re
import secrets
import subprocess
import time
import yaml


def execute(*args, data=None):
    return subprocess.check_output(args, input=data, text=True, stderr=subprocess.PIPE)


def inspect(name):
    return json.loads(execute("docker", "inspect", name))[0]


def configuration(spec, destination):
    mount = next(m for m in spec["Mounts"] if m["Destination"] == destination)
    raw = Path(mount["Source"]).read_text()
    env = dict(line.split("=", 1) for line in spec["Config"]["Env"])
    # 只替换已定义的 ${NAME}，不能打印展开后的配置。
    raw = re.sub(r"\$\{([A-Z0-9_]+)\}", lambda match: env[match[1]], raw)
    return yaml.safe_load(raw)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--directory", required=True)
    parser.add_argument("--stop", action="store_true")
    parser.add_argument("--observation-address", default="127.0.0.1:9100")
    args = parser.parse_args()
    os.umask(0o077)
    directory = Path(args.directory).resolve()
    directory.mkdir(mode=0o700, parents=True, exist_ok=True)
    os.chmod(directory, 0o700)
    state_path = directory / "fixture-state.json"
    if args.stop:
        state = json.loads(state_path.read_text())
        for name in state["containers"]:
            if not name.startswith("im-query-fixture-"):
                raise RuntimeError("unexpected container ownership")
            execute("docker", "stop", "--time", "5", name)
        print("isolated_fixture_processes_stopped=true")
        return
    if state_path.exists():
        raise RuntimeError("fixture directory already owns a run; do not overwrite")
    revision = secrets.token_hex(4)
    db = "query_contract_test_" + revision
    token = secrets.token_urlsafe(48)
    observation = inspect("yllmis-im-operations-query")
    config = configuration(observation, "/operations/conf/operations.yaml")
    mysql = inspect("mysql")
    mysql_env = dict(line.split("=", 1) for line in mysql["Config"]["Env"])
    # 数据库名由脚本生成，仅含安全字符；数据库初始化只新建隔离 schema。
    execute("docker", "exec", "mysql", "sh", "-c", 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" exec mysql -uroot -e "$1"', "fixture-setup", "CREATE DATABASE `" + db + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci")
    envs = {
        "message": {"QUERY_TEST_DB": db, "QUERY_TEST_TOKEN": token, "QUERY_TEST_MONGO_URL": config["Mongo"]["Url"]},
        "user": {"QUERY_TEST_TOKEN": token, "QUERY_TEST_MYSQL_DSN": "root:" + mysql_env["MYSQL_ROOT_PASSWORD"] + "@tcp(mysql:3306)/" + db + "?parseTime=true"},
    }
    containers = []
    state_path.write_text(json.dumps({"database": db, "containers": containers}))
    try:
        for kind, port in [("message", 19112), ("user", 19113)]:
            binary = directory / (kind + "-query-fixture")
            if not binary.is_file():
                raise RuntimeError("missing fixture binary")
            os.chmod(binary, 0o755)
            env_path = directory / (kind + ".env")
            if any("\n" in v or "\r" in v for v in envs[kind].values()):
                raise RuntimeError("multiline environment unsupported")
            env_path.write_text("".join(k + "=" + v + "\n" for k, v in envs[kind].items()))
            name = "im-query-fixture-" + kind + "-" + revision
            execute("docker", "run", "-d", "--name", name, "--network", observation["HostConfig"]["NetworkMode"], "--memory", "96m", "--cpus", "0.25", "--pids-limit", "128", "--env-file", str(env_path), "-p", "127.0.0.1:" + str(port) + ":9100", "-v", str(binary) + ":/query-fixture:ro", "--entrypoint", "/query-fixture", observation["Image"])
            containers.append(name)
            state_path.write_text(json.dumps({"database": db, "containers": containers}))
            for attempt in range(30):
                live = inspect(name)
                if not live["State"]["Running"]:
                    raise RuntimeError("fixture process failed")
                logs = execute("docker", "logs", name)
                if "isolated_" + kind + "_query_ready=true" in logs:
                    break
                time.sleep(0.5)
            else:
                raise RuntimeError("fixture startup timeout")
    except Exception:
        # 保留数据和容器供诊断，但失败的初始化不能遗留运行进程。
        for name in containers:
            execute("docker", "stop", "--time", "5", name)
        raise
    profile = {
        "GO_IM_QUERY_CONTRACT": "domain", "GO_IM_INSECURE": "true",
        "GO_IM_MESSAGE_GRPC_URL": "127.0.0.1:19112", "GO_IM_USER_GRPC_URL": "127.0.0.1:19113",
        "GO_IM_OBSERVATION_GRPC_URL": args.observation_address,
        "GO_IM_MESSAGE_SERVICE_TOKEN": token, "GO_IM_USER_SERVICE_TOKEN": token,
        "GO_IM_OBSERVATION_SERVICE_TOKEN": config["ServiceAuth"]["Token"],
        "GO_IM_TEST_MESSAGE_ID": "000000000000000000000002",
        "GO_IM_TEST_MISSING_MESSAGE_ID": "000000000000000000000099",
        "GO_IM_TEST_USER_ID": "query-fixture-u1", "GO_IM_TEST_FIXTURE_SET": "query-contract-v1",
    }
    (directory / "fixture.env").write_text("".join(k + "=" + v + "\n" for k, v in profile.items()))
    print("isolated_fixture_started=true")
    print("fixture_database=" + db)
    print("fixture_profile=" + str(directory / "fixture.env"))
    print("normal_chat_database_modified=false")


if __name__ == "__main__":
    try:
        main()
    except Exception as error:
        print("fixture_setup_failed=" + type(error).__name__)
        raise SystemExit(1) from None
