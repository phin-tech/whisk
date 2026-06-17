export type PTYStreamFrame =
  | { type: "output"; ptyId: string; offset: number; outputBase64: string }
  | { type: "exit"; ptyId: string; code?: number | null }
  | { type: "error"; ptyId: string; message: string };

export function ptyAttachWebSocketURL(daemonAddress: string, ptyId: string, fromOffset: number) {
  const url = new URL(`/v1/ptys/${encodeURIComponent(ptyId)}/attach`, daemonAddress);
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  url.searchParams.set("from", String(Math.max(0, fromOffset)));
  return url.toString();
}

export function nextPTYStreamOffset(currentOffset: number, frame: PTYStreamFrame) {
  if (frame.type !== "output" || frame.offset < currentOffset) return currentOffset;
  return frame.offset + atob(frame.outputBase64).length;
}
