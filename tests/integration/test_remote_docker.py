import os
import pathlib
import shutil
import subprocess
import json

import pytest


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
IMAGE = os.environ.get("WHISK_REMOTE_DOCKER_IMAGE", "ubuntu:24.04")


def run(cmd: list[str], **kwargs) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        cmd,
        cwd=kwargs.pop("cwd", REPO_ROOT),
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        check=True,
        timeout=kwargs.pop("timeout", 60),
        **kwargs,
    )


@pytest.fixture(scope="module")
def docker_available():
    if shutil.which("docker") is None:
        pytest.skip("docker is not installed")
    try:
        result = subprocess.run(
            ["docker", "ps"],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            check=False,
            timeout=10,
        )
    except subprocess.TimeoutExpired:
        pytest.skip("docker ps timed out")
    if result.returncode != 0:
        pytest.skip(f"docker is not running: {result.stderr.strip()}")


@pytest.fixture()
def linux_whisk(tmp_path, docker_available):
    binary = tmp_path / "whisk"
    run(
        [
            "go",
            "build",
            "-o",
            str(binary),
            "./cmd/whisk",
        ],
        env={**os.environ, "GOOS": "linux", "GOARCH": "amd64", "CGO_ENABLED": "0"},
    )
    return binary


def test_cli_only_install_runs_daemon_in_clean_linux_container(linux_whisk):
    script = r"""
set -eu
export XDG_CONFIG_HOME=/tmp/whisk-config
whisk version
whisk daemon run -addr 127.0.0.1:8787 >/tmp/whiskd.log 2>&1 &
daemon_pid=$!
cleanup() {
  whisk daemon stop >/dev/null 2>&1 || true
  kill "$daemon_pid" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM
i=0
until whisk daemon status >/dev/null 2>&1; do
  i=$((i + 1))
  if [ "$i" -gt 100 ]; then
    cat /tmp/whiskd.log >&2
    exit 1
  fi
  sleep 0.05
done
whisk onboarding status -json >/tmp/onboarding.json
test -s /tmp/onboarding.json
cleanup
"""
    result = run(
        [
            "docker",
            "run",
            "--rm",
            "-v",
            f"{linux_whisk}:/usr/local/bin/whisk:ro",
            IMAGE,
            "sh",
            "-eu",
            "-c",
            script,
        ]
    )
    assert "whisk " in result.stdout


def test_agent_can_ask_question_from_clean_linux_container(linux_whisk):
    script = r"""
set -eu
export XDG_CONFIG_HOME=/tmp/whisk-config
whisk daemon run -addr 127.0.0.1:8787 >/tmp/whiskd.log 2>&1 &
daemon_pid=$!
cleanup() {
  whisk daemon stop >/dev/null 2>&1 || true
  kill "$daemon_pid" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM
i=0
until whisk daemon status >/dev/null 2>&1; do
  i=$((i + 1))
  if [ "$i" -gt 100 ]; then
    cat /tmp/whiskd.log >&2
    exit 1
  fi
  sleep 0.05
done
project_dir=$(mktemp -d /tmp/whisk-project.XXXXXX)
project_id=$(whisk project create -name Remote -root "$project_dir")
item_line=$(whisk work-item create -project "$project_id" -title "Question smoke")
item_id=$(printf "%s\n" "$item_line" | awk "{print \$1}")
cat >/usr/local/bin/ask-question-agent <<'EOF'
#!/bin/sh
set -eu
whisk question ask -prompt "Which branch should I use?" -json >/tmp/asked-question.json
EOF
chmod +x /usr/local/bin/ask-question-agent
run_line=$(whisk run start -work-item "$item_id" -preset writer -template implement -agent-profile plain-shell -actor smoke -launch=false)
run_id=$(printf "%s\n" "$run_line" | awk "{print \$1}")
WHISK_WORK_ITEM_ID="$item_id" WHISK_RUN_ID="$run_id" WHISK_ACTOR=smoke ask-question-agent
questions=$(whisk question list -work-item "$item_id" -json)
printf "%s\n" "$questions" | grep -F "Which branch should I use?"
cleanup
"""
    result = run(
        [
            "docker",
            "run",
            "--rm",
            "-v",
            f"{linux_whisk}:/usr/local/bin/whisk:ro",
            IMAGE,
            "sh",
            "-eu",
            "-c",
            script,
        ]
    )
    assert "Which branch should I use?" in result.stdout


def test_claude_native_ask_user_question_hook_creates_structured_prompt(linux_whisk):
    script = r"""
set -eu
export XDG_CONFIG_HOME=/tmp/whisk-config
whisk daemon run -addr 127.0.0.1:8787 >/tmp/whiskd.log 2>&1 &
daemon_pid=$!
cleanup() {
  whisk daemon stop >/dev/null 2>&1 || true
  kill "$daemon_pid" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM
i=0
until whisk daemon status >/dev/null 2>&1; do
  i=$((i + 1))
  if [ "$i" -gt 100 ]; then
    cat /tmp/whiskd.log >&2
    exit 1
  fi
  sleep 0.05
done
cat >/usr/local/bin/claude <<'EOF'
#!/bin/sh
set -eu
printf 'fake claude started\n'
printf 'bridge=%s pty=%s\n' "${WHISK_AGENT_BRIDGE_ID:-}" "${WHISK_PTY_ID:-}"
cat <<'JSON' | whisk agent-bridge hook
{
  "hook_event_name": "PreToolUse",
  "tool_name": "AskUserQuestion",
  "tool_input": {
    "questions": [
      {
        "question": "What programming language was created by Guido van Rossum and named after a British comedy group?",
        "options": [
          {"label": "1. Ruby", "value": "Ruby"},
          {"label": "2. Python", "value": "Python"},
          {"label": "3. Perl", "value": "Perl"},
          {"label": "4. Cobra", "value": "Cobra"}
        ]
      }
    ]
  }
}
JSON
printf 'fake claude hook completed\n'
EOF
chmod +x /usr/local/bin/claude
project_dir=$(mktemp -d /tmp/whisk-project.XXXXXX)
project_id=$(whisk project create -name Remote -root "$project_dir")
item_line=$(whisk work-item create -project "$project_id" -title "Native Claude hook smoke")
item_id=$(printf "%s\n" "$item_line" | awk "{print \$1}")
run_json=$(whisk run start -work-item "$item_id" -preset writer -template implement -agent-profile claude -actor smoke -json)
printf "%s\n" "$run_json" >/tmp/run.json
pty_id=$(sed -n 's/.*"ptyId": "\([^"]*\)".*/\1/p' /tmp/run.json | head -n 1)
test -n "$pty_id"
i=0
while :; do
  prompts=$(whisk prompt list -json)
  printf "%s\n" "$prompts" >/tmp/prompts.json
  if grep -F "What programming language was created by Guido van Rossum" /tmp/prompts.json >/dev/null; then
    break
  fi
  i=$((i + 1))
  if [ "$i" -gt 100 ]; then
    cat /tmp/prompts.json >&2 || true
    cat /tmp/whiskd.log >&2 || true
    whisk session pty output -plain "$pty_id" >&2 || true
    exit 1
  fi
  sleep 0.05
done
prompt_id=$(sed -n 's/.*"id": "\([^"]*\)".*/\1/p' /tmp/prompts.json | head -n 1)
test -n "$prompt_id"
whisk prompt resolve "$prompt_id" -answer Python -json >/tmp/resolved-prompt.json
i=0
while :; do
  pty_output=$(whisk session pty output -plain "$pty_id")
  if printf "%s\n" "$pty_output" | grep -F "fake claude hook completed" >/dev/null; then
    break
  fi
  i=$((i + 1))
  if [ "$i" -gt 100 ]; then
    printf "%s\n" "$pty_output" >&2
    exit 1
  fi
  sleep 0.05
done
printf '%s\n' '--- PTY OUTPUT START ---'
printf '%s\n' "$pty_output"
printf '%s\n' '--- PTY OUTPUT END ---'
printf '%s\n' '--- PROMPTS JSON START ---'
cat /tmp/prompts.json
printf '%s\n' '--- PROMPTS JSON END ---'
printf '%s\n' '--- RESOLVED PROMPT JSON START ---'
cat /tmp/resolved-prompt.json
printf '%s\n' '--- RESOLVED PROMPT JSON END ---'
cleanup
"""
    result = run(
        [
            "docker",
            "run",
            "--rm",
            "-v",
            f"{linux_whisk}:/usr/local/bin/whisk:ro",
            IMAGE,
            "sh",
            "-eu",
            "-c",
            script,
        ]
    )
    print(result.stdout)
    assert "fake claude started" in result.stdout
    assert "fake claude hook completed" in result.stdout
    assert '"permissionDecision":"allow"' in result.stdout
    assert '"answers":{"What programming language was created by Guido van Rossum and named after a British comedy group?":"Python"}' in result.stdout
    prompts = json.loads(
        result.stdout.split("--- PROMPTS JSON START ---\n", 1)[1].split(
            "\n--- PROMPTS JSON END ---", 1
        )[0]
    )
    resolved = json.loads(
        result.stdout.split("--- RESOLVED PROMPT JSON START ---\n", 1)[1].split(
            "\n--- RESOLVED PROMPT JSON END ---", 1
        )[0]
    )
    assert len(prompts) == 1
    prompt = prompts[0]
    assert prompt["provider"] == "claude"
    assert prompt["eventName"] == "PreToolUse"
    assert prompt["toolName"] == "AskUserQuestion"
    assert prompt["message"].startswith("What programming language was created by Guido")
    assert prompt["options"] == [
        {"label": "1. Ruby", "value": "Ruby"},
        {"label": "2. Python", "value": "Python"},
        {"label": "3. Perl", "value": "Perl"},
        {"label": "4. Cobra", "value": "Cobra"},
    ]
    assert resolved["status"] == "resolved"
    assert resolved["answer"] == "Python"
