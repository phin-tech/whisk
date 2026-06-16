# Whisk Agent Instructions

## Core Invariant

Whisk has one runtime owner: `whiskd`.

The desktop app is always a client. It must never own, persist, or directly
mutate runtime state. If the daemon is not available, the desktop starts it,
waits for it, reconnects to it, or reports that it cannot connect. It must not
fall back to a desktop-local runtime.

No split brain is allowed.

## Runtime Ownership

Daemon-owned state:

- sessions
- PTYs
- agent processes
- projects
- kanban/work items
- mailbox/events, if added
- runtime process state
- durable runtime storage

Client-owned state:

- native window state
- selected/focused UI state
- xterm instances and rendering lifecycle
- client-specific visual preferences
- pane layout only if explicitly treated as per-client view state

If pane layout must reconnect consistently across machines, it is daemon-owned.
Do not partially own layout on both sides.

## Required Architecture

Use this dependency direction:

```text
internal/domain      pure state transitions and validation
internal/runtime     daemon-side imperative shell around PTYs/processes/storage
internal/protocol    command, response, and event DTOs
internal/server      exposes runtime through the protocol
internal/client      typed client for local or remote daemon
internal/wailsapp    Wails adapter over the client only
cmd/whiskd           daemon entrypoint
```

Forbidden dependency:

```text
internal/wailsapp -> internal/runtime
internal/wailsapp -> internal/adapters/pty
```

The Wails app may spawn or supervise `whiskd`, but all runtime commands must go
through `internal/client`.

## Protocol Boundary

Runtime mutations happen only through protocol commands.

Frontend and Wails code render daemon read models and send protocol commands.
They do not directly construct PTYs, update session stores, or persist runtime
state.

When changing the protocol, update in this order:

1. Protocol DTOs and protocol API route metadata/catalog.
2. Daemon runtime/server handler.
3. Typed client.
4. In-process daemon/client integration tests.
5. CLI command contract, when the behavior is agent/script-facing.
6. CLI smoke or black-box integration tests, when a CLI path exists.
7. OpenAPI spec generation.
8. Generated SDKs.
9. SDK smoke tests for generated Python/TypeScript clients, when SDK paths exist.
10. Wails adapter.
11. Generated Wails bindings.
12. Frontend behavior, when the GUI should expose the behavior.

Protocol changes are not complete until the generated artifacts are refreshed
and the relevant smoke tests prove the exposed route works against a real
daemon. Do not merge a feature with only domain coverage if it adds or changes a
daemon API surface.

Prefer a small stable protocol over capability negotiation. Avoid Roux-style
"who owns this right now?" branches.

## Streaming And Events

PTY output should use a streaming path for interactive latency. Snapshot/replay
APIs are still required for attach, reconnect, and recovery.

Recommended PTY shape:

- `AttachPTY` streams output/events.
- `Output(fromOffset)` returns retained replay.
- `WritePTY` writes input.
- `ResizePTY` updates the daemon-owned PTY size.

Do not use frontend polling as the primary terminal transport after streaming
exists.

## Internal Bus

If NATS or another bus is added, keep it narrow.

Allowed:

- ephemeral fanout inside `whiskd`
- decoupling daemon services
- broadcasting daemon events to subscribed clients

Forbidden:

- source of truth
- persistence layer
- client protocol
- cross-machine federation by default

Durability belongs in storage. Client API belongs in the protocol. The bus is
only internal fanout.

## TDD And Testing

Use functional-core, imperative-shell design.

Functional core tests:

- pure input/output tests
- no mocks
- state-based assertions

Imperative shell tests:

- in-memory fakes where useful
- real integration tests for PTY/server/client paths
- no mocking frameworks by default

For daemon/client work, prefer tests that exercise the typed client against a
real in-process server. This catches protocol drift without booting the full
desktop app.

## Feature TDD Workflow

For new runtime features, start at the daemon boundary and establish the CLI
contract before building GUI behavior. This matters especially for
agent-management features such as projects, workflows, work items, runs, and
kanban boards: agents must be able to drive the same daemon-owned behavior
through `cmd/whisk` that the GUI later renders.

Required order:

1. Write pure domain tests for state transitions and validation.
2. Write daemon protocol/client integration tests against a real in-process
   server.
3. Implement the smallest runtime/server/client slice that passes.
4. Add CLI commands and CLI contract tests for agent-facing behavior, including
   `--json` output for commands agents are expected to consume.
5. Add Wails adapter tests only after the daemon and CLI contracts exist.
6. Add frontend behavior last, as a projection of daemon state.

For example, project management must start with tests for:

- project domain state transitions
- daemon create/list/update/delete project commands
- typed client behavior against the daemon server
- `cmd/whisk` commands that expose those project operations, with stable JSON
  output where agents need to consume the result
- persistence or replay behavior, if the feature is durable

Do not start project work by adding Svelte state, Wails-only commands, or
desktop-local persistence.

For work item / board / agent-run features, the required implementation shape is:

1. `internal/domain/...` first: pure project/workflow/work-item/run state
   transitions and validation.
2. Runtime storage and protocol next: daemon-owned durable state, HTTP handlers,
   typed client methods, and in-process server/client tests.
3. CLI next: agent-usable commands over the typed client, table output for
   humans and `--json` for agents, with tests locking request and response
   shapes.
4. Wails bindings/service after that.
5. GUI last: render daemon read models and invoke protocol/CLI-equivalent
   actions; never invent frontend-only board state.

## Current Technical Debt

The initial prototype currently has Wails directly constructing `app.Runtime`.
That is temporary scaffolding and violates the target invariant. Remove this
before adding substantial new runtime features.

The next architectural move should be:

1. Add `internal/protocol`.
2. Add `internal/client`.
3. Make `internal/wailsapp` depend on the client interface only.
4. Add `internal/server`.
5. Add `cmd/whiskd`.
6. Switch Wails to spawn/connect to `whiskd`.
7. Delete direct Wails-to-runtime ownership.
