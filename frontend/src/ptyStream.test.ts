import { describe, expect, it } from "vitest";
import {
  consumeOptimisticEcho,
  base64DecodedByteLength,
  decodePTYBinaryOutputFrame,
  nextPTYStreamOffset,
  outputSnapshotChunkAfterOffset,
  outputSnapshotStartOffset,
  ptySnapshotFromTextFrame,
  ptyOutputChunkAfterOffset,
  ptyOutputChunkFromTextFrame,
  ptyInputFrame,
  ptyInputTraceLine,
  optimisticTerminalEcho,
  ptyAttachWebSocketURL,
  PTY_BINARY_OUTPUT_FRAME_KIND,
  terminalInputRefreshDelays,
  terminalInputShouldRefreshOutput,
  writePTYInputOverSocket,
} from "./ptyStream";

describe("ptyStream", () => {
  function expectChunk(
    chunk: ReturnType<typeof outputSnapshotChunkAfterOffset>,
    expected: { startOffset: number; nextOffset: number; text: string },
  ) {
    expect(chunk && { ...chunk, bytes: new TextDecoder().decode(chunk.bytes) }).toEqual({
      startOffset: expected.startOffset,
      nextOffset: expected.nextOffset,
      bytes: expected.text,
    });
  }

  it("builds websocket attach URLs from daemon HTTP URLs", () => {
    expect(ptyAttachWebSocketURL("http://127.0.0.1:8787", "pty_01", 7)).toBe(
      "ws://127.0.0.1:8787/v1/ptys/pty_01/attach?from=7&binary=1",
    );
    expect(ptyAttachWebSocketURL("http://127.0.0.1:8787", "pty_01", 7, "secret token")).toBe(
      "ws://127.0.0.1:8787/v1/ptys/pty_01/attach?from=7&binary=1&access_token=secret+token",
    );
    expect(ptyAttachWebSocketURL("https://daemon.local", "pty/a", -1)).toBe(
      "wss://daemon.local/v1/ptys/pty%2Fa/attach?from=0&binary=1",
    );
    expect(ptyAttachWebSocketURL("http://127.0.0.1:8787", "pty_01", 7, "", false)).toBe(
      "ws://127.0.0.1:8787/v1/ptys/pty_01/attach?from=7",
    );
    expect(ptyAttachWebSocketURL("http://127.0.0.1:8787", "pty_01", 7, "", true, true)).toBe(
      "ws://127.0.0.1:8787/v1/ptys/pty_01/attach?from=7&binary=1&snapshot=1",
    );
  });

  it("advances output and snapshot offsets and ignores stale frames", () => {
    expect(
      nextPTYStreamOffset(10, { type: "output", ptyId: "pty_01", offset: 10, outputBase64: "aGk=" }),
    ).toBe(12);
    expect(
      nextPTYStreamOffset(12, { type: "output", ptyId: "pty_01", offset: 10, outputBase64: "aGk=" }),
    ).toBe(12);
    expect(
      nextPTYStreamOffset(12, {
        type: "snapshot",
        ptyId: "pty_01",
        terminalSnapshot: {
          offset: 40,
          cols: 80,
          rows: 24,
          scrollbackAnsi: "",
          viewportAnsi: "ready",
          rehydrateSequences: "",
        },
      }),
    ).toBe(40);
  });

  it("infers output snapshot starts from decoded byte length", () => {
    expect(base64DecodedByteLength("aGk=")).toBe(2);
    expect(outputSnapshotStartOffset({ offset: 12, outputBase64: "aGk=" })).toBe(10);
    expect(outputSnapshotStartOffset({ offset: 12, output: "é" })).toBe(10);
    expect(outputSnapshotStartOffset({ offset: 2, outputBase64: "aGVsbG8=" })).toBe(0);
    expect(outputSnapshotStartOffset({ offset: 12 })).toBe(12);
  });

  it("trims output snapshots that overlap already rendered bytes", () => {
    expectChunk(outputSnapshotChunkAfterOffset({ offset: 16, outputBase64: "YWJjZGVm" }, 13), {
      startOffset: 13,
      nextOffset: 16,
      text: "def",
    });
    expect(outputSnapshotChunkAfterOffset({ offset: 16, outputBase64: "YWJjZGVm" }, 16)).toBeNull();
    expect(outputSnapshotChunkAfterOffset({ offset: 16, outputBase64: "YWJjZGVm" }, 20)).toBeNull();
  });

  it("keeps clamped output snapshots at the inferred daemon start", () => {
    expectChunk(outputSnapshotChunkAfterOffset({ offset: 16, outputBase64: "YWJjZGVm" }, 4), {
      startOffset: 10,
      nextOffset: 16,
      text: "abcdef",
    });
  });

  it("trims text output snapshots by encoded byte offsets", () => {
    expectChunk(outputSnapshotChunkAfterOffset({ offset: 5, output: "éabc" }, 2), {
      startOffset: 2,
      nextOffset: 5,
      text: "abc",
    });
  });

  it("decodes legacy JSON output frames into byte chunks", () => {
    const chunk = ptyOutputChunkFromTextFrame({
      type: "output",
      ptyId: "pty_01",
      offset: 7,
      outputBase64: "aGk=",
    });
    expect(chunk && { ...chunk, bytes: new TextDecoder().decode(chunk.bytes) }).toEqual({
      ptyId: "pty_01",
      startOffset: 7,
      nextOffset: 9,
      bytes: "hi",
    });
  });

  it("decodes snapshot frames into terminal snapshots", () => {
    const snapshot = ptySnapshotFromTextFrame({
      type: "snapshot",
      ptyId: "pty_01",
      terminalSnapshot: {
        offset: 33,
        cols: 100,
        rows: 30,
        scrollbackAnsi: "old",
        rehydrateBeforeViewport: "\x1b[?1049h",
        viewportAnsi: "\x1b[H\x1b[2Jready",
        rehydrateSequences: "\x1b[1;6H",
      },
    });

    expect(snapshot).toEqual({
      ptyId: "pty_01",
      snapshot: {
        offset: 33,
        cols: 100,
        rows: 30,
        scrollbackAnsi: "old",
        rehydrateBeforeViewport: "\x1b[?1049h",
        viewportAnsi: "\x1b[H\x1b[2Jready",
        rehydrateSequences: "\x1b[1;6H",
      },
    });
    expect(ptySnapshotFromTextFrame({ type: "snapshot", ptyId: "pty_01" })).toBeNull();
    expect(ptySnapshotFromTextFrame({ type: "output", ptyId: "pty_01" })).toBeNull();
  });

  it("keeps output chunk offset propagation behavior", () => {
    const chunk = { startOffset: 10, nextOffset: 12, bytes: new Uint8Array([1, 2]) };
    expect(ptyOutputChunkAfterOffset(chunk, 10)).toBe(chunk);
    expect(ptyOutputChunkAfterOffset(chunk, 12)).toBeNull();
    expect(ptyOutputChunkAfterOffset(chunk, 11)).toEqual({
      startOffset: 11,
      nextOffset: 12,
      bytes: new Uint8Array([2]),
    });
  });

  it("trims overlapping legacy JSON output frames to the new suffix", () => {
    const chunk = ptyOutputChunkFromTextFrame({
      type: "output",
      ptyId: "pty_01",
      offset: 10,
      outputBase64: "YWJjZGVm",
    });
    const trimmed = chunk && ptyOutputChunkAfterOffset(chunk, 13);

    expect(trimmed && { ...trimmed, bytes: new TextDecoder().decode(trimmed.bytes) }).toEqual({
      startOffset: 13,
      nextOffset: 16,
      bytes: "def",
    });
    expect(
      nextPTYStreamOffset(13, {
        type: "output",
        ptyId: "pty_01",
        offset: 10,
        outputBase64: "YWJjZGVm",
      }),
    ).toBe(16);
  });

  it("decodes binary output frames with a 1-byte kind and 8-byte big-endian offset header", () => {
    const frame = new Uint8Array([
      PTY_BINARY_OUTPUT_FRAME_KIND,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      7,
      104,
      105,
    ]);

    const chunk = decodePTYBinaryOutputFrame(frame);

    expect(chunk.startOffset).toBe(7);
    expect(chunk.nextOffset).toBe(9);
    expect(new TextDecoder().decode(chunk.bytes)).toBe("hi");
  });

  it("trims overlapping binary output frames to the new suffix", () => {
    const frame = new Uint8Array([
      PTY_BINARY_OUTPUT_FRAME_KIND,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      10,
      97,
      98,
      99,
      100,
      101,
      102,
    ]);

    const trimmed = ptyOutputChunkAfterOffset(decodePTYBinaryOutputFrame(frame), 13);

    expect(trimmed && { ...trimmed, bytes: new TextDecoder().decode(trimmed.bytes) }).toEqual({
      startOffset: 13,
      nextOffset: 16,
      bytes: "def",
    });
  });

  it("rejects malformed binary output frames", () => {
    expect(() => decodePTYBinaryOutputFrame(new Uint8Array([PTY_BINARY_OUTPUT_FRAME_KIND, 0]))).toThrow(
      "malformed PTY binary frame",
    );
    expect(() => decodePTYBinaryOutputFrame(new Uint8Array([0xff, 0, 0, 0, 0, 0, 0, 0, 0]))).toThrow(
      "unknown PTY binary frame kind 0xff",
    );
  });

  it("copies binary output bytes out of the websocket frame buffer", () => {
    const frame = new Uint8Array([
      PTY_BINARY_OUTPUT_FRAME_KIND,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      1,
      65,
    ]);
    const chunk = decodePTYBinaryOutputFrame(frame);

    frame[9] = 66;

    expect(new TextDecoder().decode(chunk.bytes)).toBe("A");
  });

  it("builds websocket input frames", () => {
    expect(ptyInputFrame("pty_01", "x")).toEqual({ type: "input", ptyId: "pty_01", data: "x" });
  });

  it("formats PTY input trace lines for frontend logging", () => {
    expect(ptyInputTraceLine("frontend.websocket", "pty_01", "abc", 123.4)).toBe(
      "pty.input channel=frontend.websocket pty=pty_01 bytes=3 at=123.400",
    );
    expect(ptyInputTraceLine("frontend.websocket", "pty_01", "é", 123.4)).toContain("bytes=2");
  });

  it("writes PTY input only when the websocket is open", () => {
    const sent: string[] = [];
    expect(writePTYInputOverSocket({ readyState: WebSocket.OPEN, send: (data: string) => sent.push(data) }, "pty_01", "x")).toBe(true);
    expect(JSON.parse(sent[0])).toEqual({ type: "input", ptyId: "pty_01", data: "x" });
    expect(writePTYInputOverSocket({ readyState: WebSocket.CONNECTING, send: () => sent.push("bad") }, "pty_01", "x")).toBe(false);
    expect(writePTYInputOverSocket(null, "pty_01", "x")).toBe(false);
  });

  it("does not poll output snapshots after terminal input when streaming is active", () => {
    expect(terminalInputRefreshDelays()).toEqual([]);
    expect(terminalInputShouldRefreshOutput()).toBe(false);
  });

  it("does not optimistically echo terminal input locally", () => {
    expect(optimisticTerminalEcho("a")).toBe("");
    expect(optimisticTerminalEcho("hello")).toBe("");
    expect(optimisticTerminalEcho("\r")).toBe("");
    expect(optimisticTerminalEcho("\u007f")).toBe("");
    expect(optimisticTerminalEcho("\x1b[A")).toBe("");
  });

  it("passes daemon output through unchanged", () => {
    expect(consumeOptimisticEcho("", "abc")).toEqual({ pending: "", output: "abc" });
    expect(consumeOptimisticEcho("abc", "abc")).toEqual({ pending: "abc", output: "abc" });
    expect(consumeOptimisticEcho("abc", "ab")).toEqual({ pending: "abc", output: "ab" });
    expect(consumeOptimisticEcho("abc", "abcd")).toEqual({ pending: "abc", output: "abcd" });
  });
});
