import { describe, expect, it } from "vitest";
import type { Session } from "../bindings/github.com/phin-tech/whisk/internal/domain/session/models";
import type { Project } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
import {
  deriveSessionsPanelRows,
  sessionRowHeight,
  SESSION_HEADER_HEIGHT,
  SESSION_ROW_HEIGHT,
  type SessionsPanelRow,
} from "./sessions-panel-state";

function session(overrides: Partial<Session> = {}): Session {
  return {
    id: "sess_01",
    name: "api",
    projectId: "proj_01",
    rootDir: "/repo/api",
    windows: {},
    panes: {},
    ...overrides,
  } as Session;
}

function project(overrides: Partial<Project> = {}): Project {
  return {
    id: "proj_01",
    name: "Whisk",
    ...overrides,
  } as Project;
}

describe("deriveSessionsPanelRows", () => {
  const defaultInput = {
    sessions: [
      session({ id: "sess_01", name: "api", projectId: "proj_01", rootDir: "/repo/api" }),
      session({ id: "sess_02", name: "web", projectId: "proj_01", rootDir: "/repo/web" }),
      session({ id: "sess_03", name: "scratch", projectId: "", rootDir: "/tmp" }),
    ],
    projects: [project({ id: "proj_01", name: "Whisk" })],
    groupMode: "project" as const,
    query: "",
    collapsedGroupIds: new Set<string>(),
    activeSessionId: "",
    confirmingSessionId: "",
  };

  it("flattens project groups into rows with stable keys", () => {
    const rows = deriveSessionsPanelRows(defaultInput);

    expect(rows).toHaveLength(5);
    expect(rows[0]).toMatchObject({
      kind: "group",
      key: "group:proj_01",
      groupId: "proj_01",
      title: "Whisk",
      count: 2,
      collapsed: false,
    });
    expect(rows[1]).toMatchObject({
      kind: "session",
      key: "session:sess_01",
      active: false,
      confirmingClose: false,
    });
    expect(rows[2]).toMatchObject({
      kind: "session",
      key: "session:sess_02",
    });
    expect(rows[3]).toMatchObject({
      kind: "group",
      key: "group:none",
      groupId: "none",
      title: "No project",
      count: 1,
    });
    expect(rows[4]).toMatchObject({
      kind: "session",
      key: "session:sess_03",
    });
  });

  it("removes session rows when group is collapsed", () => {
    const rows = deriveSessionsPanelRows({
      ...defaultInput,
      collapsedGroupIds: new Set(["proj_01"]),
    });

    expect(rows).toHaveLength(3);
    expect(rows[0].kind).toBe("group");
    expect(rows[0].key).toBe("group:proj_01");
    expect(rows[0]).toMatchObject({ collapsed: true });
    expect(rows[1].kind).toBe("group");
    expect(rows[1].key).toBe("group:none");
    expect(rows[2].kind).toBe("session");
    expect(rows[2].key).toBe("session:sess_03");
  });

  it("collapsing all groups shows only headers", () => {
    const rows = deriveSessionsPanelRows({
      ...defaultInput,
      collapsedGroupIds: new Set(["proj_01", "none"]),
    });

    expect(rows).toHaveLength(2);
    expect(rows[0].kind).toBe("group");
    expect(rows[0].key).toBe("group:proj_01");
    expect(rows[1].kind).toBe("group");
    expect(rows[1].key).toBe("group:none");
  });

  it("marks active session and confirming close", () => {
    const rows = deriveSessionsPanelRows({
      ...defaultInput,
      activeSessionId: "sess_01",
      confirmingSessionId: "sess_02",
    });

    const sess1Row = rows.find((r) => r.key === "session:sess_01") as SessionsPanelRow;
    const sess2Row = rows.find((r) => r.key === "session:sess_02") as SessionsPanelRow;

    expect(sess1Row.kind === "session" && sess1Row.active).toBe(true);
    expect(sess2Row.kind === "session" && sess2Row.confirmingClose).toBe(true);
  });

  it("returns empty rows when all sessions are filtered by search", () => {
    const rows = deriveSessionsPanelRows({
      ...defaultInput,
      query: "nonexistent",
    });

    expect(rows).toHaveLength(0);
  });

  it("returns rows matching search query", () => {
    const rows = deriveSessionsPanelRows({
      ...defaultInput,
      query: "web",
    });

    expect(rows).toHaveLength(2);
    expect(rows[0].kind).toBe("group");
    expect(rows[1].kind).toBe("session");
    expect((rows[1] as SessionsPanelRow & { kind: "session" }).session.id).toBe("sess_02");
  });

  it("handles empty sessions list", () => {
    const rows = deriveSessionsPanelRows({
      ...defaultInput,
      sessions: [],
    });

    expect(rows).toHaveLength(0);
  });

  it("handles recent group mode with single flat group", () => {
    const rows = deriveSessionsPanelRows({
      ...defaultInput,
      groupMode: "recent",
    });

    expect(rows).toHaveLength(4);
    expect(rows[0]).toMatchObject({
      kind: "group",
      key: "group:recent",
      title: "Recent",
      count: 3,
    });
    expect(rows[1].kind).toBe("session");
    expect(rows[2].kind).toBe("session");
    expect(rows[3].kind).toBe("session");
  });

  it("assigns stable keys with correct prefixes", () => {
    const rows = deriveSessionsPanelRows(defaultInput);

    for (const row of rows) {
      if (row.kind === "group") {
        expect(row.key).toMatch(/^group:/);
      } else {
        expect(row.key).toMatch(/^session:/);
      }
    }
  });
});

describe("sessionRowHeight", () => {
  it("returns header height for group rows", () => {
    const row: SessionsPanelRow = {
      kind: "group",
      key: "group:test",
      groupId: "test",
      title: "Test",
      count: 0,
      collapsed: false,
    };
    expect(sessionRowHeight(row)).toBe(SESSION_HEADER_HEIGHT);
  });

  it("returns row height for session rows", () => {
    const row: SessionsPanelRow = {
      kind: "session",
      key: "session:test",
      session: session(),
      active: false,
      confirmingClose: false,
    };
    expect(sessionRowHeight(row)).toBe(SESSION_ROW_HEIGHT);
  });
});
