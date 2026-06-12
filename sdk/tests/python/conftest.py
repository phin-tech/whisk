"""Pytest fixtures that boot a real whiskd daemon for the SDK integration suite.

The daemon is launched on an ephemeral loopback port with an isolated XDG state
directory so the suite never touches a developer's real session state. Requires
the daemon binary path in WHISKD_BIN (the Taskfile builds it before running).
"""

import os
import pathlib
import socket
import subprocess
import sys
import time

import httpx
import pytest

# Make the generated `whiskd_client` package importable without installation.
PKG_ROOT = pathlib.Path(__file__).resolve().parents[2] / "python"
sys.path.insert(0, str(PKG_ROOT))


def _free_port() -> int:
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.bind(("127.0.0.1", 0))
    port = sock.getsockname()[1]
    sock.close()
    return port


@pytest.fixture(scope="session")
def base_url(tmp_path_factory) -> str:
    binary = os.environ.get("WHISKD_BIN")
    if not binary or not pathlib.Path(binary).exists():
        pytest.skip("WHISKD_BIN not set to a built daemon binary (run via `task sdk:test:python`)")

    addr = f"127.0.0.1:{_free_port()}"
    state = tmp_path_factory.mktemp("whiskd-state")
    env = {
        **os.environ,
        "WHISKD_ADDR": addr,
        "XDG_CONFIG_HOME": str(state / "config"),
        "XDG_DATA_HOME": str(state / "data"),
        "XDG_STATE_HOME": str(state / "state"),
        "XDG_CACHE_HOME": str(state / "cache"),
    }
    proc = subprocess.Popen(
        [binary, "-addr", addr],
        env=env,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
    )
    url = f"http://{addr}"
    try:
        deadline = time.monotonic() + 15
        while time.monotonic() < deadline:
            if proc.poll() is not None:
                output = proc.stdout.read().decode() if proc.stdout else ""
                raise RuntimeError(f"daemon exited early (code {proc.returncode}):\n{output}")
            try:
                if httpx.get(url + "/v1/compat", timeout=0.5).status_code == 200:
                    break
            except httpx.HTTPError:
                time.sleep(0.1)
        else:
            raise RuntimeError("daemon did not become ready within 15s")
        yield url
    finally:
        proc.terminate()
        try:
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            proc.kill()
