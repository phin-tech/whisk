import { describe, expect, it } from "vitest";
import {
  consumeOptimisticEcho,
  nextPTYStreamOffset,
  ptyInputFrame,
  ptyInputTraceLine,
  optimisticTerminalEcho,
  ptyAttachWebSocketURL,
  terminalInputRefreshDelays,
  terminalInputShouldRefreshOutput,
  writePTYInputOverSocket,
} from "./ptyStream";

describe("ptyStream", () => {
  it("builds websocket attach URLs from daemon HTTP URLs", () => {
    expect(ptyAttachWebSocketURL("http://127.0.0.1:8787", "pty_01", 7)).toBe(
      "ws://127.0.0.1:8787/v1/ptys/pty_01/attach?from=7",
    );
    expect(ptyAttachWebSocketURL("http://127.0.0.1:8787", "pty_01", 7, "secret token")).toBe(
      "ws://127.0.0.1:8787/v1/ptys/pty_01/attach?from=7&access_token=secret+token",
    );
    expect(ptyAttachWebSocketURL("https://daemon.local", "pty/a", -1)).toBe(
      "wss://daemon.local/v1/ptys/pty%2Fa/attach?from=0",
    );
  });

  it("advances output offsets and ignores stale frames", () => {
    expect(
      nextPTYStreamOffset(10, { type: "output", ptyId: "pty_01", offset: 10, outputBase64: "aGk=" }),
    ).toBe(12);
    expect(
      nextPTYStreamOffset(12, { type: "output", ptyId: "pty_01", offset: 10, outputBase64: "aGk=" }),
    ).toBe(12);
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
