import { describe, expect, it } from "vitest";
import { nextPTYStreamOffset, ptyAttachWebSocketURL } from "./ptyStream";

describe("ptyStream", () => {
  it("builds websocket attach URLs from daemon HTTP URLs", () => {
    expect(ptyAttachWebSocketURL("http://127.0.0.1:8787", "pty_01", 7)).toBe(
      "ws://127.0.0.1:8787/v1/ptys/pty_01/attach?from=7",
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
});
