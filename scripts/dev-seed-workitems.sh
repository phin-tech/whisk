#!/bin/sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
BIN_DIR="${BIN_DIR:-$ROOT_DIR/bin}"
TARGET="${1:-${WHISKD_URL:-${WHISKD_DEV_ADDR:-127.0.0.1:8877}}}"
case "$TARGET" in
  http://*|https://*)
    URL="$TARGET"
    ADDR=${TARGET#http://}
    ADDR=${ADDR#https://}
    ;;
  *:*)
    ADDR="$TARGET"
    URL="http://$ADDR"
    ;;
  *)
    ADDR="127.0.0.1:$TARGET"
    URL="http://$ADDR"
    ;;
esac
PROJECT_NAME="${WHISK_SEED_PROJECT_NAME:-Whisk UAT Seed}"
PROJECT_ROOT="${WHISK_SEED_PROJECT_ROOT:-$ROOT_DIR}"
ACTOR="${WHISK_SEED_ACTOR:-uat-seed}"
LOG_FILE="${TMPDIR:-/tmp}/whisk-seed-daemon.log"
DAEMON_PID=""
STARTED_DAEMON=0

cleanup() {
  if [ "$STARTED_DAEMON" = "1" ]; then
    "$BIN_DIR/whisk" daemon stop -url "$URL" >/dev/null 2>&1 || true
    if [ -n "$DAEMON_PID" ]; then
      wait "$DAEMON_PID" 2>/dev/null || true
    fi
  fi
}
trap cleanup EXIT INT TERM

if [ ! -x "$BIN_DIR/whisk" ]; then
  printf 'missing CLI: %s\n' "$BIN_DIR/whisk" >&2
  printf 'run: task build:cli\n' >&2
  exit 1
fi

if ! curl -fsS "$URL/v1/health" >/dev/null 2>&1; then
  printf 'starting seed daemon: %s\n' "$URL"
  (
    cd "$ROOT_DIR"
    "$BIN_DIR/whisk" daemon run -addr "$ADDR"
  ) >"$LOG_FILE" 2>&1 &
  DAEMON_PID=$!
  STARTED_DAEMON=1
  i=0
  while ! curl -fsS "$URL/v1/health" >/dev/null 2>&1; do
    i=$((i + 1))
    if [ "$i" -gt 120 ]; then
      printf 'daemon did not become ready; log follows:\n' >&2
      cat "$LOG_FILE" >&2
      exit 1
    fi
    sleep 0.05
  done
fi

WHISK="$BIN_DIR/whisk"
COMMON="-url $URL"

PROJECTS_JSON=$("$WHISK" project list $COMMON -json)
PROJECT_ID=$(printf '%s\n' "$PROJECTS_JSON" | jq -r --arg name "$PROJECT_NAME" '.[] | select(.name == $name) | .id' | head -n 1)
DEDICATED_PROJECT=1
if [ -z "$PROJECT_ID" ]; then
  DEDICATED_PROJECT=0
  PROJECT_ID=$(printf '%s\n' "$PROJECTS_JSON" | jq -r '.[0].id // empty')
fi
if [ -z "$PROJECT_ID" ]; then
  PROJECT_ID=$("$WHISK" project create $COMMON -name "$PROJECT_NAME" -root "$PROJECT_ROOT" -json | jq -r '.id')
  DEDICATED_PROJECT=1
fi

printf 'resetting seed project: %s (%s)\n' "$PROJECT_NAME" "$PROJECT_ID"
if [ "$DEDICATED_PROJECT" = "1" ]; then
  DELETE_FILTER='.[].id'
else
  DELETE_FILTER='.[] | select(.title | startswith("UAT seed:")) | .id'
fi
"$WHISK" work-item list $COMMON -project "$PROJECT_ID" -json \
  | jq -r "$DELETE_FILTER" \
  | while IFS= read -r item_id; do
      [ -n "$item_id" ] || continue
      "$WHISK" work-item delete $COMMON -actor "$ACTOR" "$item_id" >/dev/null
    done

create_item() {
  title=$1
  body=$2
  stage=${3:-}
  if [ -n "$stage" ]; then
    "$WHISK" work-item create $COMMON -project "$PROJECT_ID" -title "$title" -body "$body" -stage "$stage" -actor "$ACTOR" -json | jq -r '.id'
  else
    "$WHISK" work-item create $COMMON -project "$PROJECT_ID" -title "$title" -body "$body" -actor "$ACTOR" -json | jq -r '.id'
  fi
}

seed_ready() {
  item=$1
  run=$("$WHISK" workflow start-planning $COMMON -work-item "$item" -actor "$ACTOR" -json | jq -r '.id')
  draft=$("$WHISK" workflow submit-plan $COMMON -work-item "$item" -run "$run" -body "Seed plan: validate the workflow controls and read models." -actor "$ACTOR" -json | jq -r '.id')
  "$WHISK" workflow approve-plan $COMMON -work-item "$item" -artifact "$draft" -actor "$ACTOR" -json >/dev/null
}

seed_execution() {
  item=$1
  seed_ready "$item"
  "$WHISK" workflow start-execution $COMMON -work-item "$item" -actor "$ACTOR" -json | jq -r '.id'
}

seed_review() {
  item=$1
  run=$(seed_execution "$item")
  "$WHISK" workflow complete-execution $COMMON -run "$run" -message "Seed execution completed for review." -actor "$ACTOR" -json >/dev/null
}

seed_done() {
  item=$1
  seed_review "$item"
  gate=$("$WHISK" gate list $COMMON -work-item "$item" -json | jq -r '.[] | select(.id != null and .id != "") | .id' | head -n 1)
  if [ -n "$gate" ]; then
    "$WHISK" gate complete "$gate" $COMMON -status passed -actor "$ACTOR" -json >/dev/null
  fi
  "$WHISK" workflow approve-done $COMMON -work-item "$item" -reason "Seed review gate passed." -actor "$ACTOR" -json >/dev/null
}

BACKLOG=$(create_item "UAT seed: backlog candidate" "Backlog item for board scanning." backlog)
PLANNING=$(create_item "UAT seed: planning draft needed" "Planning item with an active planning run.")
"$WHISK" workflow start-planning $COMMON -work-item "$PLANNING" -actor "$ACTOR" -json >/dev/null
READY=$(create_item "UAT seed: approved plan ready" "Ready item with an approved plan artifact.")
seed_ready "$READY"
EXECUTION=$(create_item "UAT seed: implementation running" "Execution item with approved plan and execution run.")
seed_execution "$EXECUTION" >/dev/null
REVIEW=$(create_item "UAT seed: awaiting review" "Review item with a pending blocking gate.")
seed_review "$REVIEW"
DONE=$(create_item "UAT seed: completed workflow" "Done item with passed gate and done approval.")
seed_done "$DONE"
BLOCKED=$(create_item "UAT seed: technical blocker" "Blocked side-stage item for technical blocker display." blocked)

printf '\nseeded work items:\n'
"$WHISK" work-item list $COMMON -project "$PROJECT_ID" -json | jq -r '.[] | "#\(.number)\t\(.stageId)\t\(.title)"'
