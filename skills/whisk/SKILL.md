---
name: whisk
description: Use Whisk's daemon-backed CLI for Whisk sessions, daemon-managed PTYs, projects, work items, runs, workflows, questions, gates, status updates, plugins, forwards, and agent bridge hooks. Trigger when Codex is running inside a Whisk session or needs to manage Whisk/whiskd runtime state through the CLI.
---

# Whisk

Use `${WHISK_CLI:-whisk}` for all commands. Whisk runtime state belongs to `whiskd`; do not create desktop-local fallbacks or mutate runtime files directly.

## Session Context

Detect an active Whisk PTY with `WHISK_SESSION=1`. Prefer these environment values over rediscovery:

- `WHISKD_URL`: daemon URL
- `WHISK_SESSION_ID`: current session
- `WHISK_PTY_ID`: current PTY
- `WHISK_PROJECT_ID` / `WHISK_PROJECT`: current project
- `WHISK_PROJECT_ROOT`: project root
- `WHISK_WORK_ITEM_ID`: current work item
- `WHISK_RUN_ID`: current run
- `WHISK_ACTOR`: actor name

When parsing output, pass `-json` if the command supports it.

## Common Flow

- Check daemon availability with `${WHISK_CLI:-whisk} daemon status`; outside Whisk, start it with `${WHISK_CLI:-whisk} daemon start` when needed.
- Report agent-visible progress with `${WHISK_CLI:-whisk} status done|blocked|question -message "..."`.
- Inspect current PTY output with `${WHISK_CLI:-whisk} session pty output -plain "${WHISK_PTY_ID}"`.
- Use workflow commands for plan/execution lifecycle instead of ad hoc comments when `WHISK_WORK_ITEM_ID` or `WHISK_RUN_ID` is set.

## CLI Reference

Read `README.md` in this skill for the command reference before using less common Whisk commands.
