export type DaemonStatusLike = {
  running?: boolean;
  address?: string;
  apiVersion?: number;
  gitSha?: string;
  version?: string;
  dirty?: boolean;
  error?: string;
  restarting?: boolean;
};

export type DaemonConnectionState = "connected" | "reconnecting" | "incompatible" | "stopped";

export type DaemonLinkSnapshot = {
  generation: number;
  key: string;
  state: DaemonConnectionState;
  canUseDaemon: boolean;
  wasConnected: boolean;
};

export function daemonGenerationKey(status: DaemonStatusLike | null | undefined) {
  if (!status) return "none";
  return JSON.stringify({
    address: status.address ?? "",
    running: Boolean(status.running),
    apiVersion: status.apiVersion ?? 0,
    gitSha: status.gitSha ?? "",
    version: status.version ?? "",
    dirty: Boolean(status.dirty),
    error: status.error ?? "",
  });
}

export function daemonConnectionState(
  status: DaemonStatusLike | null | undefined,
  wasConnected = false,
): DaemonConnectionState {
  if (!status) return wasConnected ? "reconnecting" : "stopped";
  if (status.running && status.address && !status.error) return "connected";
  if (status.running && status.error) return "incompatible";
  if (status.restarting || wasConnected || status.error) return "reconnecting";
  return "stopped";
}

export function initialDaemonLinkSnapshot(): DaemonLinkSnapshot {
  return {
    generation: 0,
    key: "none",
    state: "stopped",
    canUseDaemon: false,
    wasConnected: false,
  };
}

export function nextDaemonLinkSnapshot(
  previous: DaemonLinkSnapshot | null | undefined,
  status: DaemonStatusLike | null | undefined,
) {
  const prior = previous ?? initialDaemonLinkSnapshot();
  const key = daemonGenerationKey(status);
  const wasConnected = prior.wasConnected || prior.state === "connected";
  const state = daemonConnectionState(status, wasConnected);
  const changed = key !== prior.key;
  const snapshot: DaemonLinkSnapshot = {
    generation: changed ? prior.generation + 1 : prior.generation,
    key,
    state,
    canUseDaemon: state === "connected",
    wasConnected: wasConnected || state === "connected",
  };
  return {
    snapshot,
    changed,
    shouldResetEventCursor: changed,
    shouldReconcile: changed && snapshot.canUseDaemon,
    shouldReconnectStreams: changed && snapshot.canUseDaemon,
  };
}

export function isCurrentDaemonGeneration(snapshot: DaemonLinkSnapshot, generation: number) {
  return snapshot.generation === generation;
}

export type ReconnectBackoffOptions = {
  baseMs?: number;
  maxMs?: number;
  jitterRatio?: number;
  random?: () => number;
};

export function reconnectBackoffDelayMs(attempt: number, options: ReconnectBackoffOptions = {}) {
  const baseMs = options.baseMs ?? 250;
  const maxMs = options.maxMs ?? 5000;
  const jitterRatio = options.jitterRatio ?? 0.2;
  const random = options.random ?? Math.random;
  const rawDelay = Math.min(maxMs, baseMs * 2 ** Math.max(0, attempt));
  if (jitterRatio <= 0) return rawDelay;
  const jitter = rawDelay * jitterRatio;
  return Math.round(rawDelay - jitter + random() * jitter * 2);
}

export function claimSingleFlight(claims: Set<string>, key: string) {
  if (claims.has(key)) return false;
  claims.add(key);
  return true;
}

export function releaseSingleFlight(claims: Set<string>, key: string) {
  claims.delete(key);
}
