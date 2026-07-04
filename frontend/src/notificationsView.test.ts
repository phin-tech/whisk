import { describe, expect, it } from "vitest";
import {
  notificationDetailRows,
  notificationBadgeCount,
  notificationClearEnabled,
  notificationRows,
  notificationSurfaceCount,
  targetForStatusEvent,
} from "./notificationsView";
import { idString, parsePaneId, parsePtyId, type PaneId, type PtyId } from "./ids";

const sessions = [
  {
    id: "sess_01",
    name: "One",
    rootDir: "/repo",
    windows: {
      win_01: { id: "win_01", layout: { kind: "leaf", paneId: "pane_01" } },
    },
    panes: {
      pane_01: { id: "pane_01", currentPtyId: "pty_01" },
    },
  },
];

describe("notificationsView", () => {
  it("counts unread attention events for the rail badge", () => {
    expect(
      notificationBadgeCount([
        { id: "a", kind: "question", requiresAttention: true },
        { id: "b", kind: "blocked", requiresAttention: true, readAt: "now" },
        { id: "c", kind: "done", requiresAttention: false },
      ]),
    ).toBe(1);
  });

  it("counts useful hook notifications without counting debug hook events", () => {
    expect(
      notificationSurfaceCount(
        [{ id: "question", kind: "question", requiresAttention: true }],
        [{ id: "approval", status: "pending" }],
        [
          { id: "task", eventName: "Notification" },
          { id: "question", eventName: "PermissionRequest", toolName: "AskUserQuestion" },
          { id: "hook", eventName: "PostToolUse" },
        ],
      ),
    ).toBe(4);
  });

  it("enables clear when only hook question notifications are visible", () => {
    expect(
      notificationClearEnabled(
        [],
        [{ id: "question", eventName: "PermissionRequest", toolName: "AskUserQuestion" }],
      ),
    ).toBe(true);
  });

  it("formats rows with attention first and newest within each group", () => {
    expect(
      notificationRows([
        {
          id: "done",
          kind: "done",
          message: "Tests pass",
          requiresAttention: false,
          createdAt: "2026-06-11T12:02:00Z",
        },
        {
          id: "question",
          kind: "question",
          message: "Need API key",
          requiresAttention: true,
          sessionId: "sess_01",
          ptyId: "pty_01",
          createdAt: "2026-06-11T12:01:00Z",
        },
        {
          id: "blocked",
          kind: "blocked",
          message: "Waiting on credentials",
          requiresAttention: true,
          createdAt: "2026-06-11T12:03:00Z",
        },
      ]),
    ).toEqual([
      {
        id: "blocked",
        title: "Blocked",
        message: "Waiting on credentials",
        meta: "No terminal",
        tone: "warning",
      },
      {
        id: "question",
        title: "Question",
        message: "Need API key",
        meta: "sess_01 / pty_01",
        tone: "attention",
      },
      {
        id: "done",
        title: "Done",
        message: "Tests pass",
        meta: "No terminal",
        tone: "done",
      },
    ]);
  });

  it("builds notification details from status event and session context", () => {
    expect(
      notificationDetailRows(
        {
          id: "status_01",
          scope: "run",
          kind: "question",
          message: "Need API key",
          actor: "codex",
          projectId: "proj_01",
          workItemId: "wi_01",
          runId: "run_01",
          sessionId: "sess_01",
          paneId: "pane_01",
          ptyId: "pty_01",
          requiresAttention: true,
          createdAt: "2026-06-11T12:01:00Z",
        },
        sessions,
      ),
    ).toEqual([
      { label: "Agent", value: "codex" },
      { label: "Session", value: "sess_01" },
      { label: "Pane", value: "pane_01" },
      { label: "PTY", value: "pty_01" },
      { label: "CWD", value: "/repo" },
      { label: "Project", value: "proj_01" },
      { label: "Work item", value: "wi_01" },
      { label: "Run", value: "run_01" },
      { label: "Kind", value: "question" },
      { label: "Scope", value: "run" },
      { label: "Created", value: "2026-06-11T12:01:00Z" },
    ]);
  });

  it("resolves terminal targets from session and pty ids", () => {
    expect(
      targetForStatusEvent(
        {
          id: "status_01",
          kind: "question",
          sessionId: "sess_01",
          paneId: "pane_01",
          requiresAttention: true,
        },
        sessions,
      ),
    ).toEqual({ main: "session", sessionId: "sess_01", paneId: "pane_01" });

    expect(
      targetForStatusEvent(
        {
          id: "status_03",
          kind: "question",
          sessionId: "sess_01",
          ptyId: "pty_01",
          requiresAttention: true,
        },
        sessions,
      ),
    ).toEqual({ main: "session", sessionId: "sess_01", paneId: "pane_01" });

    expect(
      targetForStatusEvent(
        { id: "status_02", kind: "done", workItemId: "wi_01", requiresAttention: false },
        sessions,
      ),
    ).toEqual({ main: "work", sessionId: "", paneId: "" });
  });

  it("resolves a PTY-backed event to the pane that owns that PTY even when another pane has the same text", () => {
    expect(
      targetForStatusEvent(
        {
          id: "status_04",
          kind: "question",
          sessionId: "sess_01",
          ptyId: "pane_02",
          requiresAttention: true,
        },
        [
          {
            id: "sess_01",
            rootDir: "/repo",
            windows: {
              win_01: {
                id: "win_01",
                layout: {
                  kind: "split",
                  children: [
                    { kind: "leaf", paneId: "pane_01" },
                    { kind: "leaf", paneId: "pane_02" },
                  ],
                },
              },
            },
            panes: {
              pane_01: { id: "pane_01", currentPtyId: "pane_02" },
              pane_02: { id: "pane_02", currentPtyId: "pty_other" },
            },
          },
        ],
      ),
    ).toEqual({ main: "session", sessionId: "sess_01", paneId: "pane_01" });
  });

  it("keeps notification pane and PTY IDs distinct at compile time while preserving runtime strings", () => {
    function expectsPtyId(ptyId: PtyId) {
      return idString(ptyId);
    }

    function expectsPaneId(paneId: PaneId) {
      return idString(paneId);
    }

    const paneId = parsePaneId("same_text")!;
    const ptyId = parsePtyId("same_text")!;

    expect(expectsPaneId(paneId)).toBe("same_text");
    expect(expectsPtyId(ptyId)).toBe("same_text");

    // @ts-expect-error notification target resolution must not join panes through PTY IDs.
    expectsPaneId(ptyId);
    // @ts-expect-error notification target resolution must not join PTYs through pane IDs.
    expectsPtyId(paneId);
  });
});
