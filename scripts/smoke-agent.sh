#!/bin/sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
BIN_DIR="${BIN_DIR:-$ROOT_DIR/bin}"
ADDR="${WHISK_SMOKE_ADDR:-127.0.0.1:8899}"
URL="http://$ADDR"
CONFIG_DIR=$(mktemp -d /tmp/whisk-smoke-config.XXXXXX)
PROJECT_DIR=$(mktemp -d /tmp/whisk-smoke-project.XXXXXX)
LOG_FILE="$CONFIG_DIR/whiskd.log"
DAEMON_PID=""

cleanup() {
  if [ -n "$DAEMON_PID" ]; then
    curl -fsS -X POST "$URL/v1/shutdown" >/dev/null 2>&1 || true
    wait "$DAEMON_PID" 2>/dev/null || true
  fi
  if [ "${WHISK_SMOKE_KEEP:-0}" != "1" ]; then
    rm -rf "$CONFIG_DIR" "$PROJECT_DIR"
  else
    printf 'kept config dir: %s\n' "$CONFIG_DIR"
    printf 'kept project dir: %s\n' "$PROJECT_DIR"
    printf 'daemon log: %s\n' "$LOG_FILE"
  fi
}
trap cleanup EXIT INT TERM

printf 'starting smoke daemon: %s\n' "$URL"
(
  cd "$ROOT_DIR"
  XDG_CONFIG_HOME="$CONFIG_DIR" SHELL=/bin/sh "$BIN_DIR/whisk" daemon run -addr "$ADDR"
) >"$LOG_FILE" 2>&1 &
DAEMON_PID=$!

i=0
while ! curl -fsS "$URL/v1/health" >/dev/null 2>&1; do
  i=$((i + 1))
  if [ "$i" -gt 100 ]; then
    printf 'daemon did not become ready; log follows:\n' >&2
    cat "$LOG_FILE" >&2
    exit 1
  fi
  sleep 0.05
done

WHISK="$BIN_DIR/whisk"
COMMON="-url $URL"

printf 'creating project in %s\n' "$PROJECT_DIR"
PROJECT_ID=$("$WHISK" project create $COMMON -name Smoke -root "$PROJECT_DIR")
printf 'project: %s\n' "$PROJECT_ID"

ITEM_LINE=$("$WHISK" work-item create $COMMON -project "$PROJECT_ID" -title "Smoke agent run" -body "Echo this prompt through prompt-capture.")
ITEM_ID=$(printf '%s\n' "$ITEM_LINE" | awk '{print $1}')
printf 'work item: %s\n' "$ITEM_ID"

"$WHISK" work-item bind-worktree $COMMON -branch smoke/agent-run -path "$PROJECT_DIR" "$ITEM_ID" >/dev/null
printf 'bound worktree: %s\n' "$PROJECT_DIR"

RUN_LINE=$("$WHISK" run start $COMMON -work-item "$ITEM_ID" -preset writer -template implement -agent-profile prompt-capture -actor smoke)
RUN_ID=$(printf '%s\n' "$RUN_LINE" | awk '{print $1}')
STATUS=$(printf '%s\n' "$RUN_LINE" | awk '{print $2}')
SESSION_ID=$(printf '%s\n' "$RUN_LINE" | awk '{print $4}')
PTY_ID=$(printf '%s\n' "$RUN_LINE" | awk '{print $5}')
printf 'run: %s status=%s session=%s pty=%s\n' "$RUN_ID" "$STATUS" "$SESSION_ID" "$PTY_ID"

printf '\n--- PTY output ---\n'
"$WHISK" session pty output $COMMON "$PTY_ID"
printf '\n--- end PTY output ---\n'
