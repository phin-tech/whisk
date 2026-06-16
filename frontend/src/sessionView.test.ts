import { describe, expect, it } from "vitest";
import {
  paneIds,
  ptyRowsFromInventory,
  runtimeRefreshTargets,
  visiblePtyIds,
} from "./sessionView";

describe("paneIds", () => {
  it("walks nested split layouts in render order", () => {
    expect(
      paneIds({
        kind: "split",
        direction: "horizontal",
        children: [
          {
            kind: "split",
            direction: "vertical",
            children: [
              { kind: "leaf", paneId: "pane_01" },
              { kind: "leaf", paneId: "pane_02" },
            ],
          },
          { kind: "leaf", paneId: "pane_03" },
        ],
      }),
    ).toEqual(["pane_01", "pane_02", "pane_03"]);
  });
});

describe("visiblePtyIds", () => {
  it("puts the active pane first and removes duplicates", () => {
    const sessions = [
      {
        id: "sess_01",
        name: "one",
        rootDir: ".",
        windows: {
          win_01: {
            id: "win_01",
            layout: {
              kind: "split",
              direction: "horizontal",
              children: [
                { kind: "leaf", paneId: "pane_01" },
                { kind: "leaf", paneId: "pane_02" },
              ],
            },
          },
        },
        panes: {
          pane_01: { id: "pane_01", currentPtyId: "pty_01" },
          pane_02: { id: "pane_02", currentPtyId: "pty_02" },
        },
      },
      {
        id: "sess_02",
        name: "two",
        rootDir: ".",
        windows: {
          win_02: {
            id: "win_02",
            layout: { kind: "leaf", paneId: "pane_03" },
          },
        },
        panes: {
          pane_03: { id: "pane_03", currentPtyId: "pty_02" },
        },
      },
    ];

    expect(visiblePtyIds(sessions, "sess_01", "pane_02")).toEqual(["pty_02", "pty_01"]);
  });
});

describe("ptyRowsFromInventory", () => {
  it("formats daemon PTY inventory without deriving ownership locally", () => {
    expect(
      ptyRowsFromInventory([
        {
          id: "pty_01",
          workingDir: "/repo",
          cols: 80,
          rows: 24,
          running: true,
          sessionId: "sess_01",
          paneId: "pane_01",
        },
      ]),
    ).toEqual([
      {
        id: "pty_01",
        title: "pty_01",
        subtitle: "sess_01 / pane_01",
        detail: "/repo / 80x24",
        running: true,
      },
    ]);
  });
});

describe("runtimeRefreshTargets", () => {
  it("keeps runtime events as invalidation hints", () => {
    expect(runtimeRefreshTargets({ type: "session.changed" })).toEqual({
      sessions: true,
      ptys: false,
      outputPtyId: null,
      work: false,
      statusEvents: false,
      agentBridgeApprovals: false,
      agentHookEvents: false,
    });
    expect(runtimeRefreshTargets({ type: "pty.changed", ptyId: "pty_01" })).toEqual({
      sessions: false,
      ptys: true,
      outputPtyId: null,
      work: false,
      statusEvents: false,
      agentBridgeApprovals: false,
      agentHookEvents: false,
    });
    expect(runtimeRefreshTargets({ type: "pty.output", ptyId: "pty_01", offset: 12 })).toEqual({
      sessions: false,
      ptys: false,
      outputPtyId: "pty_01",
      work: false,
      statusEvents: false,
      agentBridgeApprovals: false,
      agentHookEvents: false,
    });
    expect(runtimeRefreshTargets({ type: "workitems.changed" })).toEqual({
      sessions: false,
      ptys: false,
      outputPtyId: null,
      work: true,
      statusEvents: false,
      agentBridgeApprovals: false,
      agentHookEvents: false,
    });
    expect(runtimeRefreshTargets({ type: "status.changed" })).toEqual({
      sessions: false,
      ptys: false,
      outputPtyId: null,
      work: true,
      statusEvents: true,
      agentBridgeApprovals: false,
      agentHookEvents: false,
    });
    expect(runtimeRefreshTargets({ type: "agent_bridge_approvals.changed" })).toEqual({
      sessions: false,
      ptys: false,
      outputPtyId: null,
      work: false,
      statusEvents: false,
      agentBridgeApprovals: true,
      agentHookEvents: false,
    });
    expect(runtimeRefreshTargets({ type: "agent_hook_events.changed" })).toEqual({
      sessions: false,
      ptys: false,
      outputPtyId: null,
      work: false,
      statusEvents: false,
      agentBridgeApprovals: false,
      agentHookEvents: true,
    });
  });
});
