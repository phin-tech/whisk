export type PTYStreamFrame =
  | { type: "output"; ptyId: string; offset: number; outputBase64: string }
  | { type: "input"; ptyId: string; data: string }
  | { type: "exit"; ptyId: string; code?: number | null }
  | { type: "error"; ptyId: string; message: string };

export function ptyAttachWebSocketURL(daemonAddress: string, ptyId: string, fromOffset: number, controlToken = "") {
  const url = new URL(`/v1/ptys/${encodeURIComponent(ptyId)}/attach`, daemonAddress);
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  url.searchParams.set("from", String(Math.max(0, fromOffset)));
  if (controlToken) url.searchParams.set("access_token", controlToken);
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

export type PTYOutputSnapshotLike = {
  offset: number;
  output?: string;
  outputBase64?: string;
};

export type PTYOutputSnapshotChunk = {
  startOffset: number;
  nextOffset: number;
  outputBase64: string;
};

export function base64DecodedByteLength(outputBase64: string) {
  if (!outputBase64) return 0;
  return atob(outputBase64).length;
}

export function outputSnapshotByteLength(snapshot: PTYOutputSnapshotLike) {
  if (snapshot.outputBase64) return base64DecodedByteLength(snapshot.outputBase64);
  if (snapshot.output) return new TextEncoder().encode(snapshot.output).length;
  return 0;
}

export function outputSnapshotStartOffset(snapshot: PTYOutputSnapshotLike) {
  return Math.max(0, snapshot.offset - outputSnapshotByteLength(snapshot));
}

function bytesToBase64(bytes: Uint8Array) {
  let binary = "";
  for (const byte of bytes) binary += String.fromCharCode(byte);
  return btoa(binary);
}

function outputSnapshotBytes(snapshot: PTYOutputSnapshotLike) {
  if (snapshot.outputBase64) {
    const binary = atob(snapshot.outputBase64);
    const bytes = new Uint8Array(binary.length);
    for (let index = 0; index < binary.length; index += 1) {
      bytes[index] = binary.charCodeAt(index);
    }
    return bytes;
  }
  if (snapshot.output) return new TextEncoder().encode(snapshot.output);
  return new Uint8Array();
}

export function outputSnapshotChunkAfterOffset(
  snapshot: PTYOutputSnapshotLike,
  currentOffset: number,
): PTYOutputSnapshotChunk | null {
  if (snapshot.offset <= currentOffset) return null;
  const snapshotStartOffset = outputSnapshotStartOffset(snapshot);
  const trimBytes = Math.max(0, currentOffset - snapshotStartOffset);
  const bytes = outputSnapshotBytes(snapshot);
  const remaining = trimBytes > 0 ? bytes.slice(trimBytes) : bytes;
  if (remaining.length === 0) return null;
  return {
    startOffset: Math.max(snapshotStartOffset, currentOffset),
    nextOffset: snapshot.offset,
    outputBase64: bytesToBase64(remaining),
  };
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
