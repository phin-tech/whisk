# Agent Interface

Whisk agents talk to `whiskd`. The app is only a client.

There are two supported agent paths:

- CLI callbacks: agents call `whisk question`, `whisk status`, `whisk workflow`, etc.
- Provider hooks: Claude Code and Codex call `whisk agent-bridge hook` from their native hook systems.
- Mailbox callbacks: agents exchange durable daemon-owned lifecycle mail with `whisk mail`.

## Runtime Contract

Daemon-launched agent PTYs get these env vars when available:

| Variable | Purpose |
| --- | --- |
| `WHISKD_URL` | daemon base URL |
| `WHISK_CLI` | exact CLI path to call back into Whisk |
| `WHISK_SESSION_ID` / `WHISK_PTY_ID` | current daemon-owned terminal |
| `WHISK_PROJECT_ID` / `WHISK_PROJECT_ROOT` | project context |
| `WHISK_WORK_ITEM_ID` / `WHISK_RUN_ID` | work item run context |
| `WHISK_ACTOR` | actor name for audit trail |
| `WHISK_AGENT_BRIDGE_ID` / `WHISK_AGENT_BRIDGE_TOKEN` | provider hook auth |
| `WHISK_AGENT_BRIDGE_PROVIDER` | `claude` or `codex` |

Hook callbacks read JSON on stdin and call:

```sh
whisk agent-bridge hook
```

If bridge auth is present, Whisk can block the hook while waiting for a human decision or answer. Without bridge auth, Whisk logs the event only.

## Provider Hooks

### Claude Code

Installed in `~/.claude/settings.json`.

| Event | Mode | Whisk use |
| --- | --- | --- |
| `PreToolUse` | decision | tool approval; `AskUserQuestion` becomes an answerable prompt |
| `PermissionRequest` | decision | permission approval |
| `Elicitation` | decision | answerable structured question |
| `PostToolUse` | passive | tool result logging |
| `Notification` | passive | permission and elicitation UI lifecycle logging |
| `ElicitationResult` | passive | question result logging |
| `PostToolUseFailure` | passive | tool failure logging |
| `Stop` / `StopFailure` | passive | run lifecycle logging |
| `SessionEnd` | passive | session lifecycle logging |
| `PreCompact` / `PostCompact` | passive | compaction logging |

Answer return shapes:

- `AskUserQuestion`: returns `hookSpecificOutput.permissionDecision = "allow"` plus `updatedInput.answers`.
- `Elicitation`: returns `hookSpecificOutput.decision` and `elicitationId` when present.

### Codex

Installed in `~/.codex/hooks.json`.

| Event | Mode | Whisk use |
| --- | --- | --- |
| `PreToolUse` | decision | tool approval |
| `PermissionRequest` | decision | permission approval |
| `PostToolUse` | passive | tool result logging |
| `SessionStart` | passive | session lifecycle logging |
| `UserPromptSubmit` | passive | prompt logging |
| `PreCompact` / `PostCompact` | passive | compaction logging |
| `SubagentStart` / `SubagentStop` | passive | subagent lifecycle logging |
| `Stop` | passive | run lifecycle logging |

Codex does not currently expose a Whisk-supported structured question hook. Use CLI callbacks for agent questions.

## CLI Question Path

Agents that do not use provider hooks should call:

```sh
whisk question ask -prompt "Question?" -json
```

This records a work-item question. It is separate from provider-native prompts shown by `whisk prompt list`.

## Mailbox Path

Agents can send and receive durable lifecycle messages through the daemon mailbox:

```sh
whisk mail send -to run:run_01 -type dispatch -subject "Implement task" -body "..."
whisk mail check -wait -ack -json
whisk mail reply mail_01 -body "Done" -json
```

Supported concrete address forms are `pty:<id>`, `run:<id>`, `session:<id>`, `work-item:<id>`, and `project:<id>`. `mail send` also accepts `@project:<id>` and `@work-item:<id>` selectors, which the daemon expands to current concrete session, PTY, and run recipients from its read models before storing the message. `mail send` and `mail reply` default `-from` to the current `WHISK_PTY_ID`, then `WHISK_RUN_ID`, then `WHISK_SESSION_ID`; `mail check` defaults `-to` to all current PTY, run, session, work-item, and project addresses from the environment.

Supported message types are `status`, `dispatch`, `worker_done`, `escalation`, `handoff`, `decision_gate`, and `heartbeat`. The mailbox is only the durable communication/read model in this foundation slice; dispatch authority, `@idle` selector routing, prompt injection, and automatic run completion are layered on later slices.

## Profiles

| Profile | Provider | Notes |
| --- | --- | --- |
| `claude` | Claude Code | native Claude Code, hook-capable |
| `claude-plan` | Claude Code | native Claude Code plan mode |
| `claude-openrouter` | Claude Code | Claude Code through OpenRouter auth/model env |
| `codex` | Codex | native Codex hook-capable |
| `plain-shell` | shell | no provider hooks |
| `prompt-capture` | shell | smoke-test profile |

OpenRouter is not a separate hook provider. It uses Claude Code hooks through the `claude-openrouter` profile.

## Tests

```sh
task test:remote:docker
task test:agent:claude:ask-user-question
```

`test:remote:docker` is the deterministic guard. It fakes the Claude binary but uses the real `whisk agent-bridge hook` command and asserts Whisk sends the selected answer back in Claude's expected hook response.

`test:agent:claude:ask-user-question` is live and opt-in. It runs real Claude Code through OpenRouter and confirms a native `AskUserQuestion` prompt reaches Whisk.
