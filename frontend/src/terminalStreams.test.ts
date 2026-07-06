import { describe, expect, it } from "vitest";
import {
  dropPtyState,
  retainPtyState,
  retainPtyStateFromReadModel,
  terminalStreamCleanupForLivePtys,
  terminalStreamHasLivePty,
  terminalStreamRemovedPtyIds,
  terminalStreamPtyIds,
  terminalStreamSizes,
  type TerminalStreamState,
} from "./terminalStreams";

type SocketEntry = { id: string };

function populatedState(): TerminalStreamState<SocketEntry> {
  return {
    outputChunks: {
      pty_live: [new Uint8Array([1])],
      pty_aux: [new Uint8Array([2])],
      pty_stale: [new Uint8Array([3])],
    },
    outputChunkStartOffsets: {
      pty_live: [0],
      pty_aux: [5],
      pty_stale: [10],
    },
    terminalSnapshots: {
      pty_live: { offset: 4 },
      pty_aux: { offset: 8 },
      pty_stale: { offset: 15 },
    },
    offsets: {
      pty_live: 4,
      pty_aux: 8,
      pty_stale: 15,
    },
    bottomJumpRevisions: {
      pty_live: 1,
      pty_aux: 2,
      pty_stale: 3,
    },
    ptyStreams: {
      pty_live: { id: "socket-live" },
      pty_aux: { id: "socket-aux" },
      pty_stale: { id: "socket-stale" },
    },
    ptyReconnectTimers: {
      pty_live: 11,
      pty_aux: 22,
      pty_stale: 33,
    },
    ptyReconnectAttempts: {
      pty_live: 0,
      pty_aux: 1,
      pty_stale: 4,
    },
    outputFetchInFlight: new Map([
      ["pty_live", 1],
      ["pty_aux", 1],
      ["pty_stale", 1],
    ]),
    outputFetchAgain: new Set(["pty_live", "pty_aux", "pty_stale"]),
    ptyDialInFlight: new Set(["7:pty_live", "7:pty_aux", "7:pty_stale"]),
  };
}

function addPtyState(
  state: TerminalStreamState<SocketEntry>,
  ptyId: string,
  index: number,
): TerminalStreamState<SocketEntry> {
  return {
    outputChunks: { ...state.outputChunks, [ptyId]: [new Uint8Array([index])] },
    outputChunkStartOffsets: { ...state.outputChunkStartOffsets, [ptyId]: [index * 10] },
    terminalSnapshots: { ...state.terminalSnapshots, [ptyId]: { offset: index * 10 + 1 } },
    offsets: { ...state.offsets, [ptyId]: index * 10 + 1 },
    bottomJumpRevisions: { ...state.bottomJumpRevisions, [ptyId]: index },
    ptyStreams: { ...state.ptyStreams, [ptyId]: { id: `socket-${ptyId}` } },
    ptyReconnectTimers: { ...state.ptyReconnectTimers, [ptyId]: index + 100 },
    ptyReconnectAttempts: { ...state.ptyReconnectAttempts, [ptyId]: index + 1 },
    outputFetchInFlight: new Map([...state.outputFetchInFlight, [ptyId, index]]),
    outputFetchAgain: new Set([...state.outputFetchAgain, ptyId]),
    ptyDialInFlight: new Set([...state.ptyDialInFlight, `${index}:${ptyId}`]),
  };
}

describe("terminal stream state pruning", () => {
  it("prunes every PTY-keyed terminal stream container together", () => {
    const pruned = retainPtyState(populatedState(), ["pty_live", "pty_aux"]);

    expect(Object.keys(pruned.outputChunks).sort()).toEqual(["pty_aux", "pty_live"]);
    expect(Object.keys(pruned.outputChunkStartOffsets).sort()).toEqual(["pty_aux", "pty_live"]);
    expect(Object.keys(pruned.terminalSnapshots).sort()).toEqual(["pty_aux", "pty_live"]);
    expect(Object.keys(pruned.offsets).sort()).toEqual(["pty_aux", "pty_live"]);
    expect(Object.keys(pruned.bottomJumpRevisions).sort()).toEqual(["pty_aux", "pty_live"]);
    expect(Object.keys(pruned.ptyStreams).sort()).toEqual(["pty_aux", "pty_live"]);
    expect(Object.keys(pruned.ptyReconnectTimers).sort()).toEqual(["pty_aux", "pty_live"]);
    expect(Object.keys(pruned.ptyReconnectAttempts).sort()).toEqual(["pty_aux", "pty_live"]);
    expect([...pruned.outputFetchInFlight.keys()].sort()).toEqual(["pty_aux", "pty_live"]);
    expect([...pruned.outputFetchAgain].sort()).toEqual(["pty_aux", "pty_live"]);
    expect([...pruned.ptyDialInFlight].sort()).toEqual(["7:pty_aux", "7:pty_live"]);
    expect(terminalStreamSizes(pruned)).toEqual({
      outputChunks: 2,
      outputChunkStartOffsets: 2,
      terminalSnapshots: 2,
      offsets: 2,
      bottomJumpRevisions: 2,
      ptyStreams: 2,
      ptyReconnectTimers: 2,
      ptyReconnectAttempts: 2,
      outputFetchInFlight: 2,
      outputFetchAgain: 2,
      ptyDialInFlight: 2,
    });
  });

  it("preserves unrelated live PTYs while dropping a single stale PTY", () => {
    const pruned = dropPtyState(populatedState(), "pty_stale");

    expect(pruned.outputChunks.pty_live).toEqual([new Uint8Array([1])]);
    expect(pruned.outputChunks.pty_aux).toEqual([new Uint8Array([2])]);
    expect(pruned.terminalSnapshots.pty_live).toEqual({ offset: 4 });
    expect(pruned.terminalSnapshots.pty_stale).toBeUndefined();
    expect(pruned.offsets.pty_live).toBe(4);
    expect(pruned.offsets.pty_aux).toBe(8);
    expect(pruned.ptyStreams.pty_live).toEqual({ id: "socket-live" });
    expect(pruned.ptyStreams.pty_aux).toEqual({ id: "socket-aux" });
    expect(pruned.outputChunks.pty_stale).toBeUndefined();
    expect(pruned.ptyStreams.pty_stale).toBeUndefined();
    expect([...pruned.outputFetchAgain].sort()).toEqual(["pty_aux", "pty_live"]);
  });

  it("reports stale PTYs from every current terminal stream container", () => {
    expect(terminalStreamRemovedPtyIds(populatedState(), ["pty_live"])).toEqual(["pty_aux", "pty_stale"]);
  });

  it("checks continuation PTY liveness with the same normalized IDs used for pruning", () => {
    expect(terminalStreamHasLivePty("pty_live", [" pty_live ", "pty_aux"])).toBe(true);
    expect(terminalStreamHasLivePty(" pty_aux ", ["pty_live", "pty_aux"])).toBe(true);
    expect(terminalStreamHasLivePty("", ["pty_live"])).toBe(false);
    expect(terminalStreamHasLivePty("pty_stale", ["pty_live", "pty_aux"])).toBe(false);
  });

  it("does not erase client-owned visual state when PTY read-model input is omitted or partial", () => {
    const state = populatedState();

    expect(retainPtyStateFromReadModel(state, null)).toBe(state);
    expect(retainPtyStateFromReadModel(state, {})).toBe(state);
    expect(retainPtyStateFromReadModel(state, { ptys: undefined })).toBe(state);
    expect(retainPtyStateFromReadModel(state, { ptys: [{ id: "" }] })).toBe(state);

    const pruned = retainPtyStateFromReadModel(state, { ptys: [{ id: "pty_live" }] });
    expect(Object.keys(pruned.bottomJumpRevisions)).toEqual(["pty_live"]);
    expect(pruned.bottomJumpRevisions.pty_live).toBe(1);
  });

  it("treats an explicit empty PTY list as all PTYs removed", () => {
    expect(terminalStreamSizes(retainPtyStateFromReadModel(populatedState(), { ptys: [] }))).toEqual({
      outputChunks: 0,
      outputChunkStartOffsets: 0,
      terminalSnapshots: 0,
      offsets: 0,
      bottomJumpRevisions: 0,
      ptyStreams: 0,
      ptyReconnectTimers: 0,
      ptyReconnectAttempts: 0,
      outputFetchInFlight: 0,
      outputFetchAgain: 0,
      ptyDialInFlight: 0,
    });
  });

  it("describes stream handles and timers that App must clean up for removed PTYs", () => {
    const cleanup = terminalStreamCleanupForLivePtys(populatedState(), ["pty_live", "pty_aux"]);

    expect(cleanup.removedPtyIds).toEqual(["pty_stale"]);
    expect(cleanup.streamsToClose).toEqual([{ id: "socket-stale" }]);
    expect(cleanup.reconnectTimersToClear).toEqual([33]);
    expect(terminalStreamPtyIds(cleanup.nextState)).toEqual(["pty_aux", "pty_live"]);
    expect(cleanup.nextState.outputChunks.pty_stale).toBeUndefined();
    expect(cleanup.nextState.outputChunkStartOffsets.pty_stale).toBeUndefined();
    expect(cleanup.nextState.terminalSnapshots.pty_stale).toBeUndefined();
    expect(cleanup.nextState.offsets.pty_stale).toBeUndefined();
    expect(cleanup.nextState.ptyStreams.pty_stale).toBeUndefined();
    expect(cleanup.nextState.outputFetchInFlight.has("pty_stale")).toBe(false);
    expect(cleanup.nextState.outputFetchAgain.has("pty_stale")).toBe(false);
    expect([...cleanup.nextState.ptyDialInFlight]).not.toContain("7:pty_stale");
  });

  it("purges every PTY owned by a closed session from stream and pending state", () => {
    const cleanup = terminalStreamCleanupForLivePtys(populatedState(), ["pty_aux"]);

    expect(cleanup.removedPtyIds).toEqual(["pty_live", "pty_stale"]);
    expect(cleanup.streamsToClose).toEqual([{ id: "socket-live" }, { id: "socket-stale" }]);
    expect(cleanup.reconnectTimersToClear).toEqual([11, 33]);
    expect(terminalStreamPtyIds(cleanup.nextState)).toEqual(["pty_aux"]);
    expect(terminalStreamSizes(cleanup.nextState)).toEqual({
      outputChunks: 1,
      outputChunkStartOffsets: 1,
      terminalSnapshots: 1,
      offsets: 1,
      bottomJumpRevisions: 1,
      ptyStreams: 1,
      ptyReconnectTimers: 1,
      ptyReconnectAttempts: 1,
      outputFetchInFlight: 1,
      outputFetchAgain: 1,
      ptyDialInFlight: 1,
    });
    expect([...cleanup.nextState.outputFetchInFlight.keys()]).toEqual(["pty_aux"]);
    expect([...cleanup.nextState.outputFetchAgain]).toEqual(["pty_aux"]);
    expect([...cleanup.nextState.ptyDialInFlight]).toEqual(["7:pty_aux"]);
  });

  it("returns PTY-keyed stream containers to baseline after repeated remove cascades", () => {
    let state = retainPtyState(populatedState(), ["pty_live"]);
    const baselineSizes = terminalStreamSizes(state);

    for (let index = 0; index < 25; index += 1) {
      state = addPtyState(state, `pty_closed_${index}`, index);
      state = terminalStreamCleanupForLivePtys(state, ["pty_live"]).nextState;

      expect(terminalStreamSizes(state)).toEqual(baselineSizes);
      expect(terminalStreamPtyIds(state)).toEqual(["pty_live"]);
    }
  });
});
