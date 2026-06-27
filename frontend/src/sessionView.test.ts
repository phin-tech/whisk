import { describe, expect, it } from "vitest";
import {
  closePaneTarget,
  closePaneRequest,
  isStalePTYError,
  killPTYRequest,
  paneIds,
  ptyHistoryRows,
  ptyRowsFromInventory,
  runtimeRefreshTargets,
  sessionGroups,
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

describe("closePaneRequest", () => {
  const session = {
    id: "sess_01",
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
      pane_02: { id: "pane_02" },
    },
  };

  it("builds the daemon close request for empty split panes", () => {
    expect(closePaneRequest(session, "win_01", "pane_02")).toEqual({
      sessionId: "sess_01",
      windowId: "win_01",
      paneId: "pane_02",
    });
  });

  it("builds the daemon close request for split panes that still own PTYs", () => {
    expect(closePaneRequest(session, "win_01", "pane_01")).toEqual({
      sessionId: "sess_01",
      windowId: "win_01",
      paneId: "pane_01",
    });
  });

  it("targets session close for the last pane in a window", () => {
    expect(
      closePaneTarget(
        {
          id: "sess_02",
          windows: { win_01: { id: "win_01", layout: { kind: "leaf", paneId: "pane_01" } } },
          panes: { pane_01: { id: "pane_01", currentPtyId: "pty_01" } },
        },
        "win_01",
        "pane_01",
      ),
    ).toEqual({ kind: "session", sessionId: "sess_02", ptyId: "pty_01" });
  });
});

describe("killPTYRequest", () => {
  it("builds the daemon kill request for panes with PTYs", () => {
    expect(killPTYRequest({ id: "pane_01", currentPtyId: "pty_01" })).toEqual({
      ptyId: "pty_01",
    });
  });

  it("does not build a kill request for empty panes", () => {
    expect(killPTYRequest({ id: "pane_02" })).toBeNull();
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
        {
          id: "pty_02",
          workingDir: "/repo",
          cols: 80,
          rows: 24,
          running: false,
          status: "killed",
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
        status: "running",
        canDelete: false,
      },
      {
        id: "pty_02",
        title: "pty_02",
        subtitle: "sess_01 / pane_01",
        detail: "/repo / 80x24",
        running: false,
        status: "killed",
        canDelete: true,
      },
    ]);
  });
});

describe("ptyHistoryRows", () => {
  it("summarizes persisted PTYs for the sidebar", () => {
    expect(
      ptyHistoryRows([
        {
          ptyId: "pty_01",
          sessionId: "sess_01",
          paneId: "pane_01",
          workingDir: "/repo",
          createdAt: "2026-06-19T12:00:00Z",
          exitCode: 0,
        },
      ]),
    ).toEqual([
      {
        id: "pty_01",
        title: "pty_01",
        subtitle: "sess_01 / pane_01",
        detail: "/repo",
        createdAt: "2026-06-19T12:00:00Z",
        exitCode: 0,
      },
    ]);
  });
});

describe("sessionGroups", () => {
  const sessions = [
    { id: "sess_01", name: "api", projectId: "proj_01", rootDir: "/repo/api", windows: {}, panes: {} },
    { id: "sess_02", name: "web", projectId: "proj_01", rootDir: "/repo/web", windows: {}, panes: {} },
    { id: "sess_03", name: "scratch", rootDir: "/tmp", windows: {}, panes: {} },
  ];

  it("groups sessions by project, folder, or recent order", () => {
    expect(sessionGroups(sessions, [{ id: "proj_01", name: "Whisk" }], "project", "")).toEqual([
      { id: "proj_01", title: "Whisk", sessions: [sessions[0], sessions[1]] },
      { id: "none", title: "No project", sessions: [sessions[2]] },
    ]);
    expect(sessionGroups(sessions, [], "folder", "repo")).toEqual([
      { id: "/repo/api", title: "/repo/api", sessions: [sessions[0]] },
      { id: "/repo/web", title: "/repo/web", sessions: [sessions[1]] },
    ]);
    expect(sessionGroups(sessions, [], "recent", "scratch")).toEqual([
      { id: "recent", title: "Recent", sessions: [sessions[2]] },
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

describe("isStalePTYError", () => {
  it("detects stale missing PTY errors after daemon restart", () => {
    expect(isStalePTYError(new Error(`{"message":"pty whisk_000959 not found","cause":{},"kind":"RuntimeError"}`))).toBe(true);
    expect(isStalePTYError(new Error("pty pty_01 not found"))).toBe(true);
    expect(isStalePTYError(new Error("permission denied"))).toBe(false);
  });
});
