import json
import os
import pathlib
import socket
import subprocess
import time
import urllib.request

import pytest


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
DEFAULT_WHISK_BIN = REPO_ROOT / "bin" / "whisk"


@pytest.fixture
def whisk_bin() -> pathlib.Path:
    path = pathlib.Path(os.environ.get("WHISK_BIN", DEFAULT_WHISK_BIN))
    if not path.exists():
        raise RuntimeError(f"whisk binary not found at {path}; run `task build:cli` first")
    return path


@pytest.fixture
def daemon(tmp_path: pathlib.Path, whisk_bin: pathlib.Path):
    port = reserve_local_port()
    url = f"http://127.0.0.1:{port}"
    env = {
        **os.environ,
        "WHISKD_URL": url,
        "WHISK_CLI": str(whisk_bin),
        "XDG_CONFIG_HOME": str(tmp_path / "config"),
    }
    proc = subprocess.Popen(
        [str(whisk_bin), "daemon", "run", "-addr", f"127.0.0.1:{port}"],
        env=env,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )

    try:
        wait_for_health(url, proc)
        yield {"url": url, "env": env, "whisk": whisk_bin}
    finally:
        subprocess.run(
            [str(whisk_bin), "daemon", "stop", "-url", url],
            env=env,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            check=False,
        )
        if proc.poll() is None:
            proc.terminate()
            try:
                proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                proc.kill()
                proc.wait(timeout=5)


def wait_for_health(url: str, proc: subprocess.Popen[str]) -> None:
    deadline = time.time() + 5
    while time.time() < deadline:
        if proc.poll() is not None:
            _, stderr = proc.communicate(timeout=1)
            raise RuntimeError(f"whisk daemon exited early with {proc.returncode}: {stderr}")
        try:
            with urllib.request.urlopen(f"{url}/v1/health", timeout=0.2) as response:
                if response.status == 200:
                    return
        except Exception:
            time.sleep(0.05)
    proc.terminate()
    _, stderr = proc.communicate(timeout=5)
    raise RuntimeError(f"whisk daemon did not become healthy: {stderr}")


def reserve_local_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


def run_json(daemon, args: list[str]):
    result = subprocess.run(
        [str(daemon["whisk"]), *args],
        env=daemon["env"],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        check=True,
    )
    try:
        return json.loads(result.stdout)
    except json.JSONDecodeError as exc:
        raise AssertionError(f"expected JSON from {args}, got stdout={result.stdout!r} stderr={result.stderr!r}") from exc
