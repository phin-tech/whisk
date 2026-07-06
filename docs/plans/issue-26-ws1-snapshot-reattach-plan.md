# Issue #26 WS1 Snapshot Reattach Plan

Issue: https://github.com/phin-tech/whisk/issues/26

## Current State

The original issue text is partly implemented on `main`.

Already present:

- `internal/domain/terminal` is the pure terminal-state core. It wraps `charmbracelet/x/vt`, tracks DEC private modes, and serializes `terminal.Snapshot` with `scrollbackAnsi`, `viewportAnsi`, `rehydrateBeforeViewport`, `rehydrateSequences`, cursor, size, title, working directory, and mode metadata.
- `internal/adapters/terminalhistory.FileStore` persists PTY metadata, append-only output logs, JSON checkpoints, generation numbers, and restorable PTYs.
- `internal/app.Runtime` registers PTYs with `TerminalHistoryStore`, appends output, and marks exits.
- The frontend uses binary PTY attach output and no longer polls output as the steady-state transport.

Still missing:

- Live runtime output is not fed into `internal/domain/terminal.State`.
- `TerminalHistoryAppendResult.CheckpointNeeded` is ignored.
- `PTYAttach` and `/v1/ptys/{ptyID}/attach` still replay raw bytes from the native PTY ring.
- `GET /v1/ptys/{ptyID}/output` only returns raw output bytes.
- The frontend still stores `outputChunks` and `outputChunkStartOffsets` per PTY and replays all retained chunks through xterm on pane switches.
- Cold restore from `TerminalHistoryStore.ListRestorable` is not wired into daemon startup.

## Target Shape

The daemon owns terminal reconstruction.

Runtime maintains a per-PTY terminal state beside the PTY lifecycle. Output bytes update both:

- durable terminal history output log
- live terminal emulator state

Attach/reconnect should start from a daemon-produced terminal snapshot and then stream only deltas after the snapshot offset.

The frontend should apply snapshots to xterm and retain only live offset/state needed for the currently connected stream. It should not keep unbounded per-PTY output arrays as the durable replay model.

## PR Slices

### PR 1: Runtime Snapshot Fan-In

Goal: keep live terminal snapshots current without changing protocol yet.

Touch points:

- `internal/app/runtime.go`
- `internal/app/runtime_test.go`
- `internal/domain/terminal` only if current tests expose serializer gaps

Implementation:

- Add runtime-owned terminal state storage keyed by PTY ID.
- Create terminal state when a PTY is registered, using PTY cols/rows and working dir metadata.
- Feed output from the existing `appendPTYHistoryOutput` path or the `watchPTY` fan-in path into the terminal state at the same offset.
- Resize terminal state from `ResizePTY`.
- On `TerminalHistoryAppendResult.CheckpointNeeded`, marshal the current `terminal.Snapshot` and call `TerminalHistoryStore.WriteCheckpoint`.
- Expose an internal runtime method for reading the current snapshot by PTY ID.

Tests:

- Functional core: keep existing `internal/domain/terminal` state/mode/serializer tests pure.
- Imperative shell: in-memory PTY backend plus in-memory terminal history store. Assert output bytes produce a terminal snapshot with the expected offset/content/modes, and assert checkpoint writes happen when `CheckpointNeeded` is returned.

Do not:

- Add HTTP/SDK/frontend changes in this PR.
- Move terminal state into the native PTY adapter. Runtime owns the durable restore contract.

### PR 2: Protocol And Server Snapshot Transport

Goal: expose snapshots without removing raw fallback yet.

Touch points:

- `internal/protocol/protocol.go`
- `internal/protocol/pty_stream.go`
- `internal/protocol/pty_stream_test.go`
- `internal/protocol/routes.go`
- `internal/server/http.go`
- `internal/server/http_test.go`
- `internal/client/http.go`
- `internal/client/http_test.go`
- `sdk/openapi.json`
- generated Go/TS/Python SDKs if the HTTP response shape changes

Implementation:

- Add `TerminalSnapshot` DTO mirroring `terminal.Snapshot`.
- Extend `OutputSnapshot` with optional `terminalSnapshot`.
- Add `snapshot=true` query support to `GET /v1/ptys/{ptyID}/output`.
- Extend `PTYStreamFrame` with `type:"snapshot"` and `terminalSnapshot`.
- In `attachPTY`, send a snapshot frame first when a snapshot is available, then stream output deltas starting after `snapshot.offset`.
- Keep raw replay as fallback when no snapshot exists.

Tests:

- Protocol unit tests for JSON shape and binary output compatibility.
- Server/client in-process tests proving:
  - snapshot output query returns a snapshot and preserves raw output fallback fields
  - attach sends a snapshot frame before output frames
  - old from-offset behavior still works when `snapshot=false`

Required checks:

- `go test ./internal/protocol ./internal/server ./internal/client`
- `task sdk:check`
- `task sdk:test`

### PR 3: Frontend Snapshot Application

Goal: consume daemon snapshots and stop using retained output arrays as the reattach model.

Touch points:

- `frontend/src/ptyStream.ts`
- `frontend/src/ptyStream.test.ts`
- `frontend/src/TerminalPane.svelte`
- `frontend/src/TerminalPane.test.ts`
- `frontend/src/App.svelte`
- `frontend/src/App.test.ts`
- `frontend/src/terminalStreams.ts`
- `frontend/src/terminalStreams.test.ts`
- `frontend/e2e/jump-palette.spec.ts` or a focused terminal e2e if needed

Implementation:

- Decode `snapshot` frames in `ptyStream.ts`.
- Add a `TerminalSnapshot` frontend type from generated bindings when available.
- In `TerminalPane`, apply snapshot in order:
  1. `terminal.reset()`
  2. write `rehydrateBeforeViewport`
  3. write `scrollbackAnsi`
  4. write `viewportAnsi`
  5. write `rehydrateSequences`
- Track the current PTY offset after snapshot application.
- Remove or sharply bound `outputChunks` / `outputChunkStartOffsets` as durable pane-switch state.
- Keep `terminal.bottom` behavior as an explicit scroll action, not replay.

Tests:

- Pure frontend tests for snapshot frame decoding and offset advancement.
- Component/source tests proving `TerminalPane` applies snapshot order and no longer replays all historical chunks on every PTY switch.
- Existing frontend unit suite.

Required checks:

- `npm --prefix frontend test -- ptyStream.test.ts TerminalPane.test.ts terminalStreams.test.ts App.test.ts`
- `npm --prefix frontend test`
- `npm --prefix frontend run check`

### PR 4: Cold Restore From Terminal History

Goal: restart/reconnect can restore PTY read models from durable history.

Touch points:

- `internal/app/runtime.go`
- `internal/app/runtime_test.go`
- `internal/adapters/terminalhistory/files_test.go` if restore ordering or validation needs tightening
- `internal/server/http_test.go`
- frontend PTY/history panels only if read-model display changes

Implementation:

- During runtime startup, call `TerminalHistoryStore.ListRestorable`.
- Rehydrate terminal state from checkpoint snapshot plus matching log bytes.
- Recreate daemon read models for restorable PTYs with status `exited` or a distinct restorable status; do not pretend dead OS PTYs are live.
- Define whether restored PTYs can be attached as read-only history snapshots or only shown through PTY history. Prefer read-only attach only if the protocol can make that explicit.
- Ensure stale/missing/corrupt checkpoints are skipped without blocking daemon startup.

Tests:

- Runtime integration with file store: create PTY, write output, checkpoint, restart runtime, assert restorable record and snapshot are visible.
- Corrupt checkpoint/log tests remain in the adapter, not the runtime.

### PR 5: Raw Ring Reduction And Cleanup

Goal: make the payoff explicit after snapshot transport is proven.

Touch points:

- `internal/adapters/pty/native/native.go`
- `internal/server/http.go`
- `frontend/src/App.svelte`
- `frontend/src/TerminalPane.svelte`
- docs/comments as needed

Implementation:

- Reduce dependence on the 256 KB native raw ring for normal attach.
- Keep enough raw buffering for clients that explicitly request raw fallback.
- Remove obsolete frontend replay machinery once no path depends on it.

Tests:

- Attach/reconnect integration after output exceeds the old 256 KB ring.
- Frontend memory-state tests proving retained per-PTY output arrays are gone or bounded.

## Risks

- xterm snapshot write order is fragile. Rehydrate modes before/after viewport intentionally; tests must lock the order.
- Alt-screen restore can look correct but leave cursor/mouse/bracketed-paste wrong. Mode tests need to include `?1049`, `?2004`, `?1000/1002/1003`, `?1006/1016`, `?25`, and `?1`.
- Cold restore must not create split brain. Durable history can restore display/read models, but not a live OS process.
- Generated artifacts are required when protocol shape changes.
- Frontend cleanup should wait until snapshot attach is proven; deleting raw replay too early will make reconnect worse.

## Done Criteria

- Reattach after more than 256 KB of PTY output restores the visible terminal state.
- Bracketed paste, mouse tracking/encoding, cursor visibility, application cursor, and alt-screen state survive reconnect.
- Pane switching is not O(total PTY output).
- Frontend no longer owns durable terminal replay state.
- Runtime can list and safely restore durable terminal history records after daemon restart.
