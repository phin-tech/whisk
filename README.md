# Whisk

## Docs

- [Agent Interface](docs/agent-interface.md)

## Agent Bridge Testing

Whisk has two tests for Claude structured questions.

Run the default Docker integration suite:

```sh
task test:remote:docker
```

This builds a Linux `whisk` binary, starts `whiskd` in `ubuntu:24.04`, launches a fake `claude` PTY, sends a real `whisk agent-bridge hook` payload shaped like Claude Code `AskUserQuestion`, and verifies `whisk prompt list` exposes the numbered options. To see the PTY transcript on a passing run:

```sh
uv run --project tests/integration pytest -s tests/integration/test_remote_docker.py::test_claude_native_ask_user_question_hook_creates_structured_prompt
```

Run the opt-in real Claude Code smoke:

```sh
task test:agent:claude:ask-user-question
```

This reads `op://Development/whisk-openrouter-key/credential` when Claude auth env is not already set, uses a temp `HOME`, installs only Whisk's Claude hook there, starts a fresh daemon on an isolated port, runs Claude Code through the `claude-openrouter` profile, waits for a native `AskUserQuestion`, resolves it with `Python`, and prints PTY snapshots plus prompt JSON. The Docker hook test asserts the answer is sent back in Claude's provider hook response. Keep the real Claude smoke out of default CI: it needs Claude Code, network auth, and spends OpenRouter budget.
