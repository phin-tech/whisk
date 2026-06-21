#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
IMAGE="${WHISK_REMOTE_DOCKER_IMAGE:-ubuntu:24.04}"
TMP=$(mktemp -d /tmp/whisk-remote-agent.XXXXXX)

cleanup() {
  rm -rf "$TMP"
}
trap cleanup EXIT INT TERM

command -v docker >/dev/null 2>&1 || {
  printf 'docker is required\n' >&2
  exit 1
}

if [ -z "${OPENROUTER_API_KEY:-}" ]; then
  printf 'OPENROUTER_API_KEY is required; use op run --env-file=.env.openrouter -- task test:remote:agent:openrouter\n' >&2
  exit 1
fi

(cd "$ROOT" && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o "$TMP/whisk" ./cmd/whisk)

docker run --rm \
  -e OPENROUTER_API_KEY \
  -v "$TMP/whisk:/usr/local/bin/whisk:ro" \
  "$IMAGE" \
  sh -eu -c '
    export DEBIAN_FRONTEND=noninteractive
    apt-get update >/dev/null
    apt-get install -y curl ca-certificates >/dev/null
    curl -fsSL https://claude.ai/install.sh | bash
    export PATH="$HOME/.local/bin:$PATH"
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
    item_line=$(whisk work-item create -project "$project_id" -title "OpenRouter question smoke" -body "Use the Whisk CLI to ask this exact question and then stop: Which branch should I use?")
    item_id=$(printf "%s\n" "$item_line" | awk "{print \$1}")
    run_line=$(whisk run start -work-item "$item_id" -preset writer -template implement -agent-profile claude-openrouter -actor smoke -system-prompt "You are testing Whisk. Ask exactly one question by running: whisk question ask -prompt \"Which branch should I use?\". Do not answer the question yourself.")
    pty_id=$(printf "%s\n" "$run_line" | awk "{print \$5}")
    i=0
    until whisk question list -work-item "$item_id" -json | grep -F "Which branch should I use?" >/tmp/questions.json; do
      i=$((i + 1))
      if [ "$i" -gt 120 ]; then
        whisk session pty output "$pty_id" >&2 || true
        exit 1
      fi
      sleep 1
    done
    whisk session pty output "$pty_id" >/tmp/pty-output.txt
    test -s /tmp/pty-output.txt
    cleanup
  '
