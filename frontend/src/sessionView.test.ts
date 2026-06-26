import { describe, expect, it } from "vitest";
import {
  bookmarkJumpTarget,
  ptyBookmarkRowsByPty,
  closePaneTarget,
  closePaneRequest,
  isStalePTYError,
  killPTYRequest,
  latestPromptJumpPointTarget,
  paneIds,
  ptyHistoryRows,
  ptyRowsFromInventory,
  nextBookmarkTarget,
  resetOutputReplayForBookmark,
  runtimeRefreshTargets,
  safeBookmarksByPty,
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

describe("ptyBookmarkRowsByPty", () => {
  it("groups bookmarks by pty and gives unlabeled bookmarks stable labels", () => {
    expect(
      ptyBookmarkRowsByPty([
        { id: "bm_02", ptyId: "pty_01", sessionId: "sess_01", paneId: "pane_01", offset: 30, kind: "manual", label: "" },
        { id: "bm_01", ptyId: "pty_01", sessionId: "sess_01", paneId: "pane_01", offset: 10, kind: "prompt", label: "Planning prompt" },
        { id: "bm_04", ptyId: "pty_01", sessionId: "sess_01", paneId: "pane_01", offset: 40, kind: "jump_point", label: "" },
        { id: "bm_03", ptyId: "pty_02", sessionId: "", paneId: "", offset: 5, kind: "", label: "" },
      ]),
    ).toEqual({
      pty_01: [
        {
          id: "bm_01",
          ptyId: "pty_01",
          label: "Planning prompt",
          offset: 10,
          offsetLabel: "@10",
          detail: "sess_01 / pane_01",
          kind: "prompt",
        },
        {
          id: "bm_02",
          ptyId: "pty_01",
          label: "manual",
          offset: 30,
          offsetLabel: "@30",
          detail: "sess_01 / pane_01",
          kind: "manual",
        },
        {
          id: "bm_04",
          ptyId: "pty_01",
          label: "jump point",
          offset: 40,
          offsetLabel: "@40",
          detail: "sess_01 / pane_01",
          kind: "jump_point",
        },
      ],
      pty_02: [
        {
          id: "bm_03",
          ptyId: "pty_02",
          label: "offset 5",
          offset: 5,
          offsetLabel: "@5",
          detail: "unowned / detached",
          kind: "",
        },
      ],
    });
  });
});

describe("safeBookmarksByPty", () => {
  it("keeps other PTYs renderable when one bookmark load fails", async () => {
    const bookmarks = await safeBookmarksByPty(
      [{ id: "whisk_000265" }, { id: "pty_ok" }],
      async (ptyId) => {
        if (ptyId === "whisk_000265") throw new Error("bookmark store failed");
        return [
          {
            id: "bm_01",
            ptyId,
            sessionId: "sess_01",
            paneId: "pane_01",
            offset: 12,
            kind: "manual",
            label: "Agent handoff",
          },
        ];
      },
    );

    expect(bookmarks).toEqual({
      whisk_000265: [],
      pty_ok: [
        {
          id: "bm_01",
          ptyId: "pty_ok",
          sessionId: "sess_01",
          paneId: "pane_01",
          offset: 12,
          kind: "manual",
          label: "Agent handoff",
        },
      ],
    });
  });
});

describe("nextBookmarkTarget", () => {
  const bookmarks = [
    { id: "bm_30", ptyId: "pty_01", offset: 30 },
    { id: "bm_10", ptyId: "pty_01", offset: 10 },
    { id: "bm_20", ptyId: "pty_01", offset: 20 },
  ];

  it("steps through sorted bookmarks by active bookmark id", () => {
    expect(nextBookmarkTarget(bookmarks, "bm_10", 0, "next")?.id).toBe("bm_20");
    expect(nextBookmarkTarget(bookmarks, "bm_10", 0, "previous")?.id).toBe("bm_30");
  });

  it("uses the replay offset when no active bookmark is selected", () => {
    expect(nextBookmarkTarget(bookmarks, "", 15, "next")?.id).toBe("bm_20");
    expect(nextBookmarkTarget(bookmarks, "", 15, "previous")?.id).toBe("bm_10");
  });

  it("wraps at the ends", () => {
    expect(nextBookmarkTarget(bookmarks, "bm_30", 0, "next")?.id).toBe("bm_10");
    expect(nextBookmarkTarget(bookmarks, "", 35, "next")?.id).toBe("bm_10");
    expect(nextBookmarkTarget(bookmarks, "", 5, "previous")?.id).toBe("bm_30");
  });
});

describe("latestPromptJumpPointTarget", () => {
  it("returns the highest-offset jump point and ignores manual bookmarks", () => {
    const bookmarks = [
      { id: "manual", ptyId: "pty_01", offset: 80, kind: "manual" },
      { id: "older", ptyId: "pty_01", offset: 10, kind: "jump_point" },
      { id: "latest", ptyId: "pty_01", offset: 40, kind: "jump_point" },
    ];

    expect(latestPromptJumpPointTarget(bookmarks)?.id).toBe("latest");
  });

  it("uses id as a stable tie-breaker", () => {
    const bookmarks = [
      { id: "jump_a", ptyId: "pty_01", offset: 40, kind: "jump_point" },
      { id: "jump_b", ptyId: "pty_01", offset: 40, kind: "jump_point" },
    ];

    expect(latestPromptJumpPointTarget(bookmarks)?.id).toBe("jump_b");
  });
});

describe("bookmarkJumpTarget", () => {
  const sessions = [
    {
      id: "sess_01",
      name: "one",
      rootDir: ".",
      windows: {
        win_01: { id: "win_01", layout: { kind: "leaf", paneId: "pane_01" } },
      },
      panes: {
        pane_01: { id: "pane_01", currentPtyId: "pty_01" },
      },
    },
    {
      id: "sess_02",
      name: "two",
      rootDir: ".",
      windows: {
        win_02: { id: "win_02", layout: { kind: "leaf", paneId: "pane_02" } },
      },
      panes: {
        pane_02: { id: "pane_02", currentPtyId: "pty_02" },
      },
    },
  ];

  it("resolves the owning session and pane for a bookmark", () => {
    expect(
      bookmarkJumpTarget(sessions, {
        id: "bm_01",
        ptyId: "pty_01",
        sessionId: "sess_01",
        paneId: "pane_01",
        offset: 12,
      }),
    ).toEqual({ sessionId: "sess_01", paneId: "pane_01", ptyId: "pty_01", offset: 12 });
  });

  it("falls back to the live pty owner when stored pane metadata is stale", () => {
    expect(
      bookmarkJumpTarget(sessions, {
        id: "bm_02",
        ptyId: "pty_02",
        sessionId: "old",
        paneId: "old",
        offset: 20,
      }),
    ).toEqual({ sessionId: "sess_02", paneId: "pane_02", ptyId: "pty_02", offset: 20 });
  });

  it("returns null for bookmarks whose pty is not live", () => {
    expect(
      bookmarkJumpTarget(sessions, {
        id: "bm_stale",
        ptyId: "pty_missing",
        sessionId: "sess_01",
        paneId: "pane_01",
        offset: 0,
      }),
    ).toBeNull();
  });
});

describe("resetOutputReplayForBookmark", () => {
  it("clears rendered chunks, seeks to the bookmark offset, and bumps the jump revision", () => {
    expect(
      resetOutputReplayForBookmark(
        {
          outputChunks: { pty_01: ["old"], pty_02: ["keep"] },
          offsets: { pty_01: 99, pty_02: 4 },
          jumpRevisions: { pty_01: 2 },
        },
        { ptyId: "pty_01", offset: 12 },
      ),
    ).toEqual({
      outputChunks: { pty_01: [], pty_02: ["keep"] },
      offsets: { pty_01: 12, pty_02: 4 },
      jumpRevisions: { pty_01: 3 },
    });
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
