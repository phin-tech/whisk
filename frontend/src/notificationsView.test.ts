import { describe, expect, it } from "vitest";
import {
  notificationBadgeCount,
  notificationRows,
  targetForStatusEvent,
} from "./notificationsView";

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

  it("resolves terminal targets from session and pty ids", () => {
    expect(
      targetForStatusEvent(
        {
          id: "status_01",
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
});
