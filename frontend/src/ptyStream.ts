export type PTYStreamFrame = {
  type: string;
  ptyId?: string;
  data?: string;
  offset?: number;
  output?: string;
  outputBase64?: string;
  terminalSnapshot?: TerminalSnapshot | null;
  code?: number | null;
  message?: string;
};

export type TerminalSnapshot = {
  offset: number;
  cols: number;
  rows: number;
  cursor?: { x: number; y: number };
  title?: string;
  workingDirectory?: string;
  scrollbackAnsi: string;
  rehydrateBeforeViewport?: string;
  viewportAnsi: string;
  rehydrateSequences: string;
  modes?: unknown;
  mouseTrackingModes?: string[];
  mouseEncodingModes?: string[];
  truncated?: boolean;
};

export type PTYOutputChunk = {
  startOffset: number;
  nextOffset: number;
  bytes: Uint8Array;
};

export type PTYStreamOutputChunk = PTYOutputChunk & {
  ptyId: string;
};

export type PTYStreamSnapshot = {
  ptyId: string;
  snapshot: TerminalSnapshot;
};

export const PTY_BINARY_OUTPUT_FRAME_KIND = 0x01;
export const PTY_BINARY_OUTPUT_FRAME_HEADER_BYTES = 9;

export function ptyAttachWebSocketURL(
  daemonAddress: string,
  ptyId: string,
  fromOffset: number,
  controlToken = "",
  binaryOutput = true,
  snapshot = false,
) {
  const url = new URL(`/v1/ptys/${encodeURIComponent(ptyId)}/attach`, daemonAddress);
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  url.searchParams.set("from", String(Math.max(0, fromOffset)));
  if (binaryOutput) url.searchParams.set("binary", "1");
  if (snapshot) url.searchParams.set("snapshot", "1");
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
  const snapshot = ptySnapshotFromTextFrame(frame);
  if (snapshot) return Math.max(currentOffset, snapshot.snapshot.offset);
  const chunk = ptyOutputChunkFromTextFrame(frame);
  const nextChunk = chunk && ptyOutputChunkAfterOffset(chunk, currentOffset);
  return nextChunk?.nextOffset ?? currentOffset;
}

export type PTYOutputSnapshotLike = {
  offset: number;
  output?: string;
  outputBase64?: string;
};

export type PTYOutputSnapshotChunk = {
  startOffset: number;
  nextOffset: number;
  bytes: Uint8Array;
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

function base64Bytes(outputBase64: string) {
  const binary = atob(outputBase64);
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index);
  }
  return bytes;
}

function outputSnapshotBytes(snapshot: PTYOutputSnapshotLike) {
  if (snapshot.outputBase64) {
    return base64Bytes(snapshot.outputBase64);
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
    bytes: remaining,
  };
}

export function ptyOutputChunkFromTextFrame(frame: PTYStreamFrame): PTYStreamOutputChunk | null {
  if (frame.type !== "output" || !frame.ptyId || typeof frame.offset !== "number" || !Number.isFinite(frame.offset)) {
    return null;
  }
  const bytes = ptyTextOutputBytes(frame);
  const startOffset = Math.max(0, Math.trunc(frame.offset));
  return {
    ptyId: frame.ptyId,
    startOffset,
    nextOffset: startOffset + bytes.length,
    bytes,
  };
}

export function ptySnapshotFromTextFrame(frame: PTYStreamFrame): PTYStreamSnapshot | null {
  if (frame.type !== "snapshot" || !frame.ptyId || !frame.terminalSnapshot) return null;
  const offset = frame.terminalSnapshot.offset;
  if (typeof offset !== "number" || !Number.isFinite(offset)) return null;
  return {
    ptyId: frame.ptyId,
    snapshot: frame.terminalSnapshot,
  };
}

export function ptyOutputChunkAfterOffset(chunk: PTYOutputChunk, currentOffset: number) {
  if (chunk.nextOffset <= currentOffset) return null;
  if (chunk.startOffset < currentOffset) {
    const trimBytes = currentOffset - chunk.startOffset;
    return {
      startOffset: currentOffset,
      nextOffset: chunk.nextOffset,
      bytes: chunk.bytes.slice(trimBytes),
    };
  }
  return chunk;
}

export function decodePTYBinaryOutputFrame(data: ArrayBuffer | ArrayBufferView): PTYOutputChunk {
  const frame = binaryFrameBytes(data);
  if (frame.length < PTY_BINARY_OUTPUT_FRAME_HEADER_BYTES) {
    throw new Error(
      `malformed PTY binary frame: got ${frame.length} bytes, want at least ${PTY_BINARY_OUTPUT_FRAME_HEADER_BYTES}`,
    );
  }
  if (frame[0] !== PTY_BINARY_OUTPUT_FRAME_KIND) {
    throw new Error(`unknown PTY binary frame kind 0x${frame[0].toString(16).padStart(2, "0")}`);
  }
  const view = new DataView(frame.buffer, frame.byteOffset + 1, 8);
  const high = view.getUint32(0, false);
  const low = view.getUint32(4, false);
  const startOffset = high * 2 ** 32 + low;
  if (!Number.isSafeInteger(startOffset)) {
    throw new Error(`PTY binary frame offset ${startOffset} exceeds JavaScript safe integer range`);
  }
  const bytes = frame.slice(PTY_BINARY_OUTPUT_FRAME_HEADER_BYTES);
  const nextOffset = startOffset + bytes.length;
  if (!Number.isSafeInteger(nextOffset)) {
    throw new Error(`PTY binary frame next offset ${nextOffset} exceeds JavaScript safe integer range`);
  }
  return { startOffset, nextOffset, bytes };
}

function ptyTextOutputBytes(frame: PTYStreamFrame) {
  if (frame.outputBase64) return base64Bytes(frame.outputBase64);
  if (frame.output) return new TextEncoder().encode(frame.output);
  return new Uint8Array();
}

function binaryFrameBytes(data: ArrayBuffer | ArrayBufferView) {
  if (data instanceof ArrayBuffer) return new Uint8Array(data);
  return new Uint8Array(data.buffer, data.byteOffset, data.byteLength);
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
