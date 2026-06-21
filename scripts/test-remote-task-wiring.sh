#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

assert_contains() {
  file="$1"
  needle="$2"
  if ! grep -Fq "$needle" "$ROOT/$file"; then
    printf '%s missing expected text: %s\n' "$file" "$needle" >&2
    exit 1
  fi
}

assert_contains "Taskfile.yml" "test:remote:docker"
assert_contains "Taskfile.yml" "tests/integration/test_remote_docker.py"
assert_contains "Taskfile.yml" "test:remote:agent:openrouter"
assert_contains "Taskfile.yml" "scripts/test-remote-agent-openrouter.sh"
assert_contains "Taskfile.yml" "test:agent:claude:ask-user-question"
assert_contains "Taskfile.yml" "scripts/test-real-claude-ask-user-question.sh"
assert_contains ".gitignore" ".env.openrouter"
