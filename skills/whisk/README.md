# Whisk CLI Reference

Use `${WHISK_CLI:-whisk}` in examples. Most commands default `-url` from `WHISKD_URL`, then `http://127.0.0.1:8787`. Prefer `-json` when another tool or agent will parse output.

## Environment

- `WHISK_SESSION=1`: running inside a daemon-owned Whisk PTY.
- `WHISK_SESSION_ID`, `WHISK_PTY_ID`: current runtime location.
- `WHISK_PROJECT_ID`, `WHISK_PROJECT`, `WHISK_PROJECT_ROOT`: project context.
- `WHISK_WORK_ITEM_ID`, `WHISK_RUN_ID`: work item/run context.
- `WHISK_ACTOR`: actor for audit trails.
- `WHISK_CLI`: exact CLI path injected by the daemon.
- `WHISKD_URL`: daemon URL.

## whisk daemon

- `whisk daemon status [-url URL]`: health check.
- `whisk daemon start [-url URL]`: ensure daemon is running.
- `whisk daemon stop [-url URL]`: stop daemon.
- `whisk daemon clear -yes [-url URL]`: clear daemon-owned runtime state.
- `whisk daemon run [-addr HOST:PORT]`: foreground daemon process.

## whisk forward

- `whisk forward create <target-url> [-name name] [-url URL]`: expose a local target through daemon-managed forwarding. Runs until interrupted.

## whisk session

- `whisk session list [-project id] [-json] [-url URL]`: list sessions.
- `whisk session create -root path [-working-dir path] [-project id] [-name name] [-command command] [-pty=false] [-url URL]`: create session, optionally with initial PTY.
- `whisk session update <session-id> (-project id | -clear-project) [-json] [-url URL]`: assign/clear project.
- `whisk session set-root <session-id> <path> [-url URL]`: update session root.
- `whisk session close <session-id> [-url URL]`: close session.

## whisk session pty

- `whisk session pty list [session-id] [-url URL]`: list PTYs, optionally by session.
- `whisk session pty output <pty-id> [-from offset|end] [-plain] [-json] [-url URL]`: replay retained output.
- `whisk session pty tail <pty-id> [-from offset|end] [-poll 500ms] [-plain] [-once] [-url URL]`: follow output.
- `whisk session pty write <pty-id> (-data text | -stdin) [-url URL]`: write input.
- `whisk session pty resize <pty-id> -cols n -rows n [-url URL]`: resize terminal.
- `whisk session pty kill <pty-id> [-url URL]`: kill PTY.

## whisk project

- `whisk project list [-json] [-url URL]`: list projects.
- `whisk project create -name name -root path [-description text] [-slug slug] [-workflow id] [-json] [-url URL]`: create project.
- `whisk project show <project-id> [-json] [-url URL]`: show project detail.
- `whisk project update <project-id> [-name name] [-description text] [-slug slug] [-json] [-url URL]`: update project metadata.
- `whisk project attach <project-id> -kind file|url|note|external [-scope scope] [-title title] [-path path] [-attachment-url url] [-note text] [-provider name] [-target id] [-context] [-json] [-url URL]`: add attachment.
- `whisk project context <project-id> [-json] [-url URL]`: list context attachments.

## whisk work-item

- `whisk work-item list [-project id] [-json] [-url URL]`: list items.
- `whisk work-item create -project id -title title [-body markdown] [-stage stage] [-actor actor] [-json] [-url URL]`: create item.
- `whisk work-item move <work-item-id> -stage stage [-actor actor] [-json] [-url URL]`: move stage.
- `whisk work-item bind-worktree <work-item-id> -branch branch -path path [-base base] [-actor actor] [-json] [-url URL]`: bind worktree.
- `whisk work-item attach-file <work-item-id> <path> [-scope project|worktree|external] [-actor actor] [-json] [-url URL]`: attach file.
- `whisk work-item delete <work-item-id> [-actor actor] [-json] [-url URL]`: delete item.

## whisk run

- `whisk run list [-work-item id] [-json] [-url URL]`: list runs.
- `whisk run start -work-item id [-preset writer] [-template id] [-launch=false] [-agent-profile codex] [-system-prompt text] [-session id] [-pty id] [-actor actor] [-json] [-url URL]`: start run.
- `whisk run cancel <run-id> [-actor actor] [-json] [-url URL]`: cancel run.

## whisk workflow

- `whisk workflow start-planning [-work-item id] [-actor actor] [-launch] [-json] [-url URL]`
- `whisk workflow submit-plan -body text [-work-item id] [-run id] [-title title] [-actor actor] [-json] [-url URL]`
- `whisk workflow approve-plan -artifact id [-work-item id] [-actor actor] [-json] [-url URL]`
- `whisk workflow start-execution [-work-item id] [-actor actor] [-launch] [-json] [-url URL]`
- `whisk workflow complete-execution [-run id] [-message text] [-actor actor] [-json] [-url URL]`
- `whisk workflow feedback -body text [-work-item id] [-run id] [-actor actor] [-json] [-url URL]`
- `whisk workflow approve-done [-work-item id] [-reason text] [-actor actor] [-json] [-url URL]`
- `whisk workflow artifacts [-work-item id] [-json] [-url URL]`
- `whisk workflow events [-work-item id] [-json] [-url URL]`

## whisk question

- `whisk question list [-work-item id] [-json] [-url URL]`
- `whisk question ask -prompt text [-work-item id] [-run id] [-actor actor] [-json] [-url URL]`
- `whisk question answer <question-id> -answer text [-actor actor] [-json] [-url URL]`

## whisk gate

- `whisk gate list [-work-item id] [-json] [-url URL]`
- `whisk gate complete <gate-report-id> -status passed|failed|overridden [-override-reason text] [-actor actor] [-json] [-url URL]`

## whisk status

- `whisk status question -message text [-run id] [-session id] [-pty id] [-actor actor] [-json] [-url URL]`
- `whisk status done -message text [-run id] [-session id] [-pty id] [-actor actor] [-json] [-url URL]`
- `whisk status blocked -message text [-run id] [-session id] [-pty id] [-actor actor] [-json] [-url URL]`

Use status commands inside Whisk sessions to leave daemon-visible progress instead of only printing local terminal output.

## whisk agent-bridge

- `whisk agent-bridge hook [-url URL] [-bridge id] [-token token] [-provider claude|codex] [-event name]`: provider hook callback. Usually invoked by generated hook scripts, not manually.

## whisk plugin

- `whisk plugin list [-json] [-url URL]`: list discovered plugins.
- `whisk plugin rescan [-json] [-url URL]`: rescan plugin directories.
- `whisk plugin trust <plugin-id> [-json] [-url URL]`: trust plugin.
- `whisk plugin untrust <plugin-id> [-json] [-url URL]`: untrust plugin.
- `whisk plugin attach <plugin-id> <template-id> -project <project-id> [-field key=value ...] [-json] [-url URL]`: run trusted project attachment template.
