import { describe, expect, it } from "vitest";
import {
  claimSingleFlight,
  daemonConnectionState,
  daemonGenerationKey,
  initialDaemonLinkSnapshot,
  isCurrentDaemonGeneration,
  nextDaemonLinkSnapshot,
  reconnectBackoffDelayMs,
  releaseSingleFlight,
} from "./daemonLink";

describe("daemonLink", () => {
  it("derives stable generation keys from meaningful daemon status fields", () => {
    const key = daemonGenerationKey({
      running: true,
      address: "http://127.0.0.1:8787",
      apiVersion: 24,
      gitSha: "abc",
      version: "1.0.0",
      dirty: false,
      error: "",
    });

    expect(key).toBe(
      daemonGenerationKey({
        running: true,
        address: "http://127.0.0.1:8787",
        apiVersion: 24,
        gitSha: "abc",
        version: "1.0.0",
        dirty: false,
        error: "",
      }),
    );
    expect(key).not.toBe(daemonGenerationKey({ running: true, address: "http://127.0.0.1:8788", apiVersion: 24 }));
    expect(key).not.toBe(daemonGenerationKey({ running: true, address: "http://127.0.0.1:8787", apiVersion: 25 }));
  });

  it("calculates reconnect states without treating the frontend as runtime owner", () => {
    expect(daemonConnectionState(null)).toBe("stopped");
    expect(daemonConnectionState({ running: false, error: "lost" }, true)).toBe("reconnecting");
    expect(daemonConnectionState({ running: true, address: "http://127.0.0.1:8787", error: "" })).toBe("connected");
    expect(daemonConnectionState({ running: true, address: "http://127.0.0.1:8787", error: "protocol mismatch" })).toBe(
      "incompatible",
    );
  });

  it("increments generation only when daemon identity changes", () => {
    const initial = initialDaemonLinkSnapshot();
    const first = nextDaemonLinkSnapshot(initial, {
      running: true,
      address: "http://127.0.0.1:8787",
      apiVersion: 24,
    });
    const same = nextDaemonLinkSnapshot(first.snapshot, {
      running: true,
      address: "http://127.0.0.1:8787",
      apiVersion: 24,
    });
    const changed = nextDaemonLinkSnapshot(same.snapshot, {
      running: true,
      address: "http://127.0.0.1:8788",
      apiVersion: 24,
    });

    expect(first.changed).toBe(true);
    expect(first.snapshot.generation).toBe(1);
    expect(first.shouldResetEventCursor).toBe(true);
    expect(first.shouldReconcile).toBe(true);
    expect(same.changed).toBe(false);
    expect(same.snapshot.generation).toBe(1);
    expect(changed.changed).toBe(true);
    expect(changed.snapshot.generation).toBe(2);
    expect(isCurrentDaemonGeneration(changed.snapshot, 1)).toBe(false);
    expect(isCurrentDaemonGeneration(changed.snapshot, 2)).toBe(true);
  });

  it("backs reconnect attempts off with an upper cap", () => {
    const opts = { jitterRatio: 0 };
    expect(reconnectBackoffDelayMs(0, opts)).toBe(250);
    expect(reconnectBackoffDelayMs(1, opts)).toBe(500);
    expect(reconnectBackoffDelayMs(2, opts)).toBe(1000);
    expect(reconnectBackoffDelayMs(10, opts)).toBe(5000);
  });

  it("applies deterministic jitter when requested", () => {
    expect(reconnectBackoffDelayMs(0, { jitterRatio: 0.2, random: () => 0 })).toBe(200);
    expect(reconnectBackoffDelayMs(0, { jitterRatio: 0.2, random: () => 1 })).toBe(300);
  });

  it("claims single-flight work until released", () => {
    const claims = new Set<string>();
    expect(claimSingleFlight(claims, "pty_01")).toBe(true);
    expect(claimSingleFlight(claims, "pty_01")).toBe(false);
    releaseSingleFlight(claims, "pty_01");
    expect(claimSingleFlight(claims, "pty_01")).toBe(true);
  });
});
