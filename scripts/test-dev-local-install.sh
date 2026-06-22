#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
OUT=$(cd "$ROOT" && task --dry dev:local-install INSTALL_DIR=/tmp/whisk-local-install-test 2>&1 || true)

printf '%s\n' "$OUT" | grep -Fq 'ditto "bin/Whisk.app" "/tmp/whisk-local-install-test/Whisk.app"' || {
  printf '%s\n' "$OUT"
  exit 1
}

printf '%s\n' "$OUT" | grep -Fq 'cp "skills/whisk/SKILL.md" "bin/Whisk.app/Contents/Resources/skills/whisk/SKILL.md"' || {
  printf '%s\n' "$OUT"
  exit 1
}
