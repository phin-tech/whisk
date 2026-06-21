# Source this file to run Claude Code through OpenRouter for Whisk smoke tests.

if ! (return 0 2>/dev/null); then
  printf 'source this file instead: . scripts/source-claude-openrouter-env.sh\n' >&2
  exit 1
fi

if [ -z "${OPENROUTER_API_KEY:-}" ]; then
  if command -v op >/dev/null 2>&1; then
    OPENROUTER_API_KEY=$(op read 'op://Development/whisk-openrouter-key/credential')
  else
    printf 'OPENROUTER_API_KEY is unset and op is not installed\n' >&2
    return 1
  fi
fi

export OPENROUTER_API_KEY
export ANTHROPIC_BASE_URL="${ANTHROPIC_BASE_URL:-https://openrouter.ai/api}"
export ANTHROPIC_AUTH_TOKEN="$OPENROUTER_API_KEY"
export ANTHROPIC_API_KEY=""
export ANTHROPIC_DEFAULT_OPUS_MODEL="${ANTHROPIC_DEFAULT_OPUS_MODEL:-~anthropic/claude-haiku-latest}"
export ANTHROPIC_DEFAULT_SONNET_MODEL="${ANTHROPIC_DEFAULT_SONNET_MODEL:-~anthropic/claude-haiku-latest}"
export ANTHROPIC_DEFAULT_HAIKU_MODEL="${ANTHROPIC_DEFAULT_HAIKU_MODEL:-~anthropic/claude-haiku-latest}"
export CLAUDE_CODE_SUBAGENT_MODEL="${CLAUDE_CODE_SUBAGENT_MODEL:-~anthropic/claude-haiku-latest}"

printf 'Claude/OpenRouter env loaded for %s\n' "$ANTHROPIC_DEFAULT_SONNET_MODEL"
