import { describe, expect, it } from "vitest";
import {
  dropPtyState,
  retainPtyState,
  retainPtyStateFromReadModel,
  terminalStreamRemovedPtyIds,
  terminalStreamSizes,
  type TerminalStreamState,
} from "./terminalStreams";

type SocketEntry = { id: string };

function populatedState(): TerminalStreamState<SocketEntry> {
  return {
    outputChunks: {
      pty_live: ["bGl2ZQ=="],
      pty_aux: ["YXV4"],
      pty_stale: ["c3RhbGU="],
    },
    outputChunkStartOffsets: {
      pty_live: [0],
      pty_aux: [5],
      pty_stale: [10],
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

describe("terminal stream state pruning", () => {
  it("prunes every PTY-keyed terminal stream container together", () => {
    const pruned = retainPtyState(populatedState(), ["pty_live", "pty_aux"]);

    expect(Object.keys(pruned.outputChunks).sort()).toEqual(["pty_aux", "pty_live"]);
    expect(Object.keys(pruned.outputChunkStartOffsets).sort()).toEqual(["pty_aux", "pty_live"]);
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

    expect(pruned.outputChunks.pty_live).toEqual(["bGl2ZQ=="]);
    expect(pruned.outputChunks.pty_aux).toEqual(["YXV4"]);
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
});
