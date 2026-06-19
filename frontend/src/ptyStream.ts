export type PTYStreamFrame =
  | { type: "output"; ptyId: string; offset: number; outputBase64: string }
  | { type: "input"; ptyId: string; data: string }
  | { type: "exit"; ptyId: string; code?: number | null }
  | { type: "error"; ptyId: string; message: string };

export function ptyAttachWebSocketURL(daemonAddress: string, ptyId: string, fromOffset: number) {
  const url = new URL(`/v1/ptys/${encodeURIComponent(ptyId)}/attach`, daemonAddress);
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  url.searchParams.set("from", String(Math.max(0, fromOffset)));
  return url.toString();
}

export function ptyInputFrame(ptyId: string, data: string) {
  return { type: "input", ptyId, data };
}

export function ptyInputTraceLine(channel: string, ptyId: string, data: string, at: number) {
  return `pty.input channel=${channel} pty=${ptyId} bytes=${new TextEncoder().encode(data).length} at=${at.toFixed(3)}`;
}

export function writePTYInputOverSocket(
  socket: { readyState: number; send: (data: string) => void } | null | undefined,
  ptyId: string,
  data: string,
) {
  if (socket?.readyState !== WebSocket.OPEN) return false;
  socket.send(JSON.stringify(ptyInputFrame(ptyId, data)));
  return true;
}

export function nextPTYStreamOffset(currentOffset: number, frame: PTYStreamFrame) {
  if (frame.type !== "output" || frame.offset < currentOffset) return currentOffset;
  return frame.offset + atob(frame.outputBase64).length;
}

export function terminalInputRefreshDelays() {
  return [];
}

export function terminalInputShouldRefreshOutput() {
  return false;
}

export function optimisticTerminalEcho(data: string) {
  return "";
}

export function consumeOptimisticEcho(pending: string, output: string) {
  return { pending, output };
}
