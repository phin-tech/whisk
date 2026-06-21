#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
TMP=$(mktemp -d /tmp/whisk-real-claude.XXXXXX)
WHISKD_PORT="${WHISKD_PORT:-$((18000 + ($$ % 20000)))}"
WHISKD_ADDR="${WHISKD_ADDR:-127.0.0.1:$WHISKD_PORT}"
WHISKD_URL="${WHISKD_URL:-http://$WHISKD_ADDR}"
TIMEOUT_SECONDS="${WHISK_REAL_CLAUDE_TIMEOUT_SECONDS:-180}"

cleanup() {
  WHISKD_URL="$WHISKD_URL" "$TMP/whisk" daemon stop >/dev/null 2>&1 || true
  rm -rf "$TMP"
}
trap cleanup EXIT INT TERM

command -v claude >/dev/null 2>&1 || {
  printf 'claude is required; install Claude Code first\n' >&2
  exit 1
}
CLAUDE_BIN=$(command -v claude)

if [ -z "${ANTHROPIC_AUTH_TOKEN:-}" ] && [ -z "${ANTHROPIC_API_KEY:-}" ] && [ -z "${OPENROUTER_API_KEY:-}" ]; then
  if command -v op >/dev/null 2>&1; then
    OPENROUTER_API_KEY=$(op read 'op://Development/whisk-openrouter-key/credential')
    export OPENROUTER_API_KEY
  else
    printf 'Claude auth is required; set OPENROUTER_API_KEY or install/sign in to op\n' >&2
    exit 1
  fi
fi

if [ -n "${OPENROUTER_API_KEY:-}" ]; then
  export ANTHROPIC_BASE_URL="${ANTHROPIC_BASE_URL:-https://openrouter.ai/api}"
  export ANTHROPIC_AUTH_TOKEN="${ANTHROPIC_AUTH_TOKEN:-$OPENROUTER_API_KEY}"
  export ANTHROPIC_API_KEY="${ANTHROPIC_API_KEY:-}"
  export ANTHROPIC_DEFAULT_OPUS_MODEL="${ANTHROPIC_DEFAULT_OPUS_MODEL:-~anthropic/claude-haiku-latest}"
  export ANTHROPIC_DEFAULT_SONNET_MODEL="${ANTHROPIC_DEFAULT_SONNET_MODEL:-~anthropic/claude-haiku-latest}"
  export ANTHROPIC_DEFAULT_HAIKU_MODEL="${ANTHROPIC_DEFAULT_HAIKU_MODEL:-~anthropic/claude-haiku-latest}"
  export CLAUDE_CODE_SUBAGENT_MODEL="${CLAUDE_CODE_SUBAGENT_MODEL:-~anthropic/claude-haiku-latest}"
fi

(cd "$ROOT" && go build -o "$TMP/whisk" ./cmd/whisk)

export WHISKD_URL
export WHISK_CLI="$TMP/whisk"
export XDG_CONFIG_HOME="$TMP/config"
export HOME="$TMP/home"
mkdir -p "$HOME/.local/bin"
ln -s "$CLAUDE_BIN" "$HOME/.local/bin/claude"
export PATH="$HOME/.local/bin:$PATH"

"$TMP/whisk" daemon run -addr "$WHISKD_ADDR" >"$TMP/whiskd.log" 2>&1 &
daemon_pid=$!

i=0
until "$TMP/whisk" daemon status >/dev/null 2>&1; do
  i=$((i + 1))
  if [ "$i" -gt 100 ]; then
    cat "$TMP/whiskd.log" >&2
    exit 1
  fi
  sleep 0.05
done

"$TMP/whisk" onboarding apply -items hook:claude >"$TMP/onboarding.json"

project_dir="$TMP/project"
mkdir -p "$project_dir"
project_id=$("$TMP/whisk" project create -name "Real Claude" -root "$project_dir")
item_line=$("$TMP/whisk" work-item create -project "$project_id" -title "Real Claude AskUserQuestion smoke")
item_id=$(printf "%s\n" "$item_line" | awk '{print $1}')

system_prompt='You are running a Whisk integration smoke. Immediately ask the user exactly one structured multiple-choice question using the native AskUserQuestion tool. Header: Trivia. Question: What programming language was created by Guido van Rossum and named after a British comedy group? Options: Ruby, Python, Perl, Cobra. Do not answer it yourself. Do not use Bash for the question.'
run_line=$("$TMP/whisk" run start -work-item "$item_id" -preset writer -template implement -agent-profile claude-openrouter -actor smoke -system-prompt "$system_prompt")
pty_id=$(printf "%s\n" "$run_line" | awk '{print $5}')

deadline=$(( $(date +%s) + TIMEOUT_SECONDS ))
prompt_id=""
while [ "$(date +%s)" -lt "$deadline" ]; do
  "$TMP/whisk" prompt list -json >"$TMP/prompts.json"
  if grep -F "Guido van Rossum" "$TMP/prompts.json" >/dev/null; then
    prompt_id=$(sed -n 's/.*"id": "\([^"]*\)".*/\1/p' "$TMP/prompts.json" | head -n 1)
    break
  fi
  sleep 1
done

printf '%s\n' '--- PTY OUTPUT START ---'
"$TMP/whisk" session pty output -plain "$pty_id" || true
printf '%s\n' '--- PTY OUTPUT END ---'

if [ -z "$prompt_id" ]; then
  printf 'real Claude did not emit the expected AskUserQuestion prompt within %ss\n' "$TIMEOUT_SECONDS" >&2
  cat "$TMP/prompts.json" >&2 || true
  exit 1
fi

"$TMP/whisk" prompt resolve "$prompt_id" -answer Python -json >"$TMP/resolved-prompt.json"
sleep 1

printf '%s\n' '--- PTY OUTPUT AFTER RESOLVE START ---'
"$TMP/whisk" session pty output -plain "$pty_id" || true
printf '%s\n' '--- PTY OUTPUT AFTER RESOLVE END ---'

printf '%s\n' '--- PROMPTS JSON START ---'
cat "$TMP/prompts.json"
printf '%s\n' '--- PROMPTS JSON END ---'
printf '%s\n' '--- RESOLVED PROMPT JSON START ---'
cat "$TMP/resolved-prompt.json"
printf '%s\n' '--- RESOLVED PROMPT JSON END ---'

kill "$daemon_pid" >/dev/null 2>&1 || true
