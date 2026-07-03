package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	out := flag.String("o", "sdk/python/whiskd_client/local.py", "output file")
	flag.Parse()
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "mkdir:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, []byte(localClientPython), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", *out)
}

const localClientPython = `"""Same-user helpers for authenticating to the local whiskd daemon."""

from __future__ import annotations

import os
from pathlib import Path

from .client import AuthenticatedClient

TOKEN_FILE_NAME = "control-token"


def state_dir() -> Path:
    xdg_state_home = os.environ.get("XDG_STATE_HOME")
    if xdg_state_home:
        return Path(xdg_state_home) / "whisk"
    if os.name == "nt":
        local_app_data = os.environ.get("LOCALAPPDATA")
        if local_app_data:
            return Path(local_app_data) / "whisk" / "state"
    return Path.home() / ".local" / "state" / "whisk"


def control_token_path() -> Path:
    return state_dir() / TOKEN_FILE_NAME


def read_control_token() -> str:
    return control_token_path().read_text(encoding="utf-8").strip()


def control_auth_headers() -> dict[str, str]:
    return {"Authorization": f"Bearer {read_control_token()}"}


def local_client(base_url: str | None = None, **kwargs: object) -> AuthenticatedClient:
    """Return an authenticated client for a same-user whiskd daemon."""
    return AuthenticatedClient(
        base_url=base_url or os.environ.get("WHISKD_URL", "http://127.0.0.1:8787"),
        token=read_control_token(),
        **kwargs,
    )
`
