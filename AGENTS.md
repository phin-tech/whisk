# Whisk Agent Instructions

## Core Invariant

Whisk has one runtime owner: the daemon started by `whisk daemon run`.

The desktop app is always a client. It may start, wait for, reconnect to, or
report failure to connect to the daemon. It must not own, persist, or directly
mutate runtime state, and must not fall back to a desktop-local runtime.

No split brain.

## Ownership

Daemon-owned:

- sessions, PTYs, agent processes, process state
- projects, work items, boards, workflows, runs
- mailbox/events, if added
- durable runtime storage
- pane layout, if it must reconnect consistently across machines

Client-owned:

- native window state
- selected/focused UI state
- xterm instances and rendering lifecycle
- client-specific visual preferences
- pane layout only when explicitly per-client view state

Do not partially own the same state on both sides.

## Required Architecture

Dependency direction:

```text
internal/domain      pure state transitions and validation
internal/runtime     daemon-side imperative shell around PTYs/processes/storage
internal/protocol    command, response, and event DTOs
internal/server      exposes runtime through the protocol
internal/client      typed client for local or remote daemon
internal/wailsapp    Wails adapter over the client only
cmd/whisk daemon run daemon entrypoint
```

Forbidden:

```text
internal/wailsapp -> internal/runtime
internal/wailsapp -> internal/adapters/pty
```

Runtime mutations happen only through protocol commands. Frontend and Wails code
render daemon read models and send protocol commands; they do not construct
PTYs, update runtime stores, or persist runtime state.

## Daemon/API Workflow

For runtime features and protocol-facing changes, work from the daemon boundary
outward:

1. `internal/domain/...`: pure state transitions and validation.
2. Protocol DTOs and route metadata/catalog.
3. Runtime storage/server handler.
4. Typed client.
5. In-process daemon/client integration tests.
6. CLI contract, when agent/script-facing: table output for humans, `--json`
   for agents, and tests locking request/response shapes.
7. CLI smoke or black-box integration tests, when a CLI path exists.
8. OpenAPI spec generation.
9. Generated SDKs.
10. SDK smoke tests, when SDK paths exist.
11. Wails adapter.
12. Generated Wails bindings.
13. GUI: render daemon read models and invoke protocol/CLI-equivalent actions.

Protocol work is incomplete until generated artifacts are refreshed and relevant
smoke tests prove the route against a real daemon.

Prefer a small stable protocol over capability negotiation. Object-level daemon
commands should complete the expected user action end to end; add separate
commands only for genuinely distinct outcomes.

## Streaming And Events

Current PTY shape:

- `GET /v1/ptys/{ptyID}/attach?from=` is the interactive WebSocket stream.
  It sends `output`, `exit`, and `error` frames and accepts `input` frames.
- `GET /v1/ptys/{ptyID}/output?from=` returns retained replay snapshots for
  attach, reconnect, CLI tailing, and WebSocket fallback.
- `POST /v1/ptys/{ptyID}/write` is the HTTP write path for CLI and fallback.
- `POST /v1/ptys/{ptyID}/resize` updates daemon-owned PTY size.
- Runtime publishes `pty.output` and `pty.changed` events; the frontend uses
  `/v1/events/next` to refresh read models and only polls output when no PTY
  WebSocket is active for that PTY.

Do not reintroduce frontend output polling as the primary terminal transport.

## Internal Bus

Whisk uses embedded NATS in the daemon as an internal runtime event fanout.

Allowed:

- `internal/events.NATSBus` owns the embedded loopback NATS server.
- Runtime publishes `app.RuntimeEvent` values through `EventSink`.
- Event consumers read through `EventSource`; HTTP exposes this as
  `GET /v1/events/next`.
- Subjects are implementation details: `whisk.session.changed`,
  `whisk.pty.changed`, `whisk.pty.output`, `whisk.workitems.changed`, and
  `whisk.status.changed`.

Forbidden:

- source of truth
- persistence layer
- client protocol
- cross-machine federation by default

Durability belongs in storage. Client API belongs in the protocol. The bus is
only internal fanout, not a public client API.

## Testing

Use functional-core, imperative-shell design.

Functional core tests:

- pure input/output
- no mocks
- state-based assertions

Imperative shell tests:

- in-memory fakes where useful
- real integration tests for PTY/server/client paths
- no mocking frameworks by default

For daemon/client work, prefer typed-client tests against a real in-process
server. This catches protocol drift without booting the desktop app.

Do not start runtime features with Svelte state, Wails-only commands, or
desktop-local persistence.
