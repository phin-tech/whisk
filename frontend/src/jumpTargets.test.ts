import { describe, expect, it } from "vitest";
import { deriveJumpTargets, type JumpTargetsInput } from "./jumpTargets";

function fixtureInput(overrides: Partial<JumpTargetsInput> = {}): JumpTargetsInput {
  return {
    activeProjectId: "proj_whisk",
    activeSessionId: "sess_api",
    activePaneId: "pane_api",
    openWorkItemId: "item_40",
    projects: [
      {
        id: "proj_docs",
        name: "Docs",
        slug: "docs",
        rootDir: "/repo/docs",
        description: "Documentation site",
      },
      {
        id: "proj_whisk",
        name: "Whisk",
        slug: "whisk",
        rootDir: "/repo/whisk",
      },
    ],
    sessions: [
      {
        id: "sess_docs",
        projectId: "proj_docs",
        name: "Docs shell",
        rootDir: "/repo/docs",
        windows: {
          win_docs: {
            id: "win_docs",
            name: "main",
            layout: { kind: "leaf", paneId: "pane_docs" },
          },
        },
        panes: {
          pane_docs: {
            id: "pane_docs",
            windowId: "win_docs",
            currentPtyId: "pty_docs",
            workingDir: "/repo/docs",
          },
        },
      },
      {
        id: "sess_api",
        projectId: "proj_whisk",
        name: "API",
        rootDir: "/repo/whisk",
        windows: {
          win_api: {
            id: "win_api",
            name: "main",
            layout: { kind: "leaf", paneId: "pane_api" },
          },
        },
        panes: {
          pane_api: {
            id: "pane_api",
            windowId: "win_api",
            currentPtyId: "pty_api",
            workingDir: "/repo/whisk/frontend",
          },
        },
      },
    ],
    ptys: [
      {
        id: "pty_docs",
        sessionId: "sess_docs",
        windowId: "win_docs",
        paneId: "pane_docs",
        workingDir: "/repo/docs",
        running: true,
        status: "running",
      },
      {
        id: "pty_api",
        sessionId: "sess_api",
        windowId: "win_api",
        paneId: "pane_api",
        workingDir: "/repo/whisk/frontend",
        running: true,
        status: "running",
      },
    ],
    workItems: [
      {
        id: "item_other_project",
        projectId: "proj_docs",
        number: 7,
        title: "Docs task",
        stageId: "ready",
        runState: "idle",
      },
      {
        id: "item_40",
        projectId: "proj_whisk",
        number: 40,
        title: "Jump palette foundation",
        stageId: "execution",
        runState: "running",
      },
      {
        id: "item_41",
        projectId: "proj_whisk",
        number: 41,
        title: "Plugin commands",
        stageId: "backlog",
        runState: "idle",
      },
    ],
    workItemRuns: [
      {
        id: "run_no_terminal",
        projectId: "proj_whisk",
        workItemId: "item_41",
        status: "completed",
        preset: "codex",
        createdAt: "2026-01-01T00:00:00Z",
      },
      {
        id: "run_40",
        projectId: "proj_whisk",
        workItemId: "item_40",
        sessionId: "sess_api",
        ptyId: "pty_api",
        status: "running",
        preset: "codex",
        createdAt: "2026-01-02T00:00:00Z",
      },
      {
        id: "run_docs",
        projectId: "proj_docs",
        workItemId: "item_other_project",
        sessionId: "sess_docs",
        status: "running",
        preset: "codex",
        createdAt: "2026-01-03T00:00:00Z",
      },
    ],
    ...overrides,
  };
}

describe("jumpTargets", () => {
  it("derives current session, pane, pty, project, and open work item first", () => {
    const targets = deriveJumpTargets(fixtureInput());

    expect(targets.slice(0, 5).map((target) => target.id)).toEqual([
      "session:sess_api",
      "pane:sess_api:pane_api",
      "pty:pty_api",
      "project:proj_whisk",
      "work-item:item_40",
    ]);
    expect(targets.slice(0, 5).every((target) => target.current)).toBe(true);
  });

  it("includes structured payloads and searchable labels for sessions, panes, PTYs, and projects", () => {
    const targets = deriveJumpTargets(fixtureInput());

    expect(targets.find((target) => target.id === "session:sess_api")).toMatchObject({
      kind: "session",
      title: "API",
      subtitle: "Whisk",
      detail: "/repo/whisk",
      payload: { kind: "session", sessionId: "sess_api", projectId: "proj_whisk" },
    });
    expect(targets.find((target) => target.id === "pane:sess_api:pane_api")).toMatchObject({
      kind: "pane",
      payload: {
        kind: "pane",
        sessionId: "sess_api",
        windowId: "win_api",
        paneId: "pane_api",
        ptyId: "pty_api",
      },
    });
    expect(targets.find((target) => target.id === "pty:pty_api")).toMatchObject({
      kind: "pty",
      subtitle: "/repo/whisk/frontend",
      payload: {
        kind: "pty",
        ptyId: "pty_api",
        sessionId: "sess_api",
        windowId: "win_api",
        paneId: "pane_api",
      },
    });
    expect(targets.find((target) => target.id === "project:proj_whisk")).toMatchObject({
      kind: "project",
      title: "Whisk",
      subtitle: "whisk",
      detail: "/repo/whisk",
      payload: { kind: "project", projectId: "proj_whisk" },
    });
  });

  it("omits detached PTY targets that cannot resolve to a loaded session pane", () => {
    const input = fixtureInput();
    const targets = deriveJumpTargets({
      ...input,
      ptys: [
        ...(input.ptys ?? []),
        {
          id: "pty_detached",
          sessionId: "sess_api",
          windowId: "",
          paneId: "",
          workingDir: "/repo/whisk",
          running: true,
          status: "running",
        },
      ],
    });

    expect(targets.filter((target) => target.kind === "pty").map((target) => target.id)).toEqual([
      "pty:pty_api",
      "pty:pty_docs",
    ]);
    expect(targets.map((target) => target.id)).not.toContain("pty:pty_detached");
  });

  it("indexes attached PTYs by resolving pane ownership from the session model", () => {
    const input = fixtureInput();
    const targets = deriveJumpTargets({
      ...input,
      ptys: (input.ptys ?? []).map((pty) =>
        pty.id === "pty_api" ? { ...pty, windowId: "", paneId: "" } : pty,
      ),
    });

    expect(targets.find((target) => target.id === "pty:pty_api")).toMatchObject({
      kind: "pty",
      payload: {
        kind: "pty",
        ptyId: "pty_api",
        sessionId: "sess_api",
        windowId: "win_api",
        paneId: "pane_api",
      },
    });
  });

  it("indexes only active-project work items when an active project is selected", () => {
    const targets = deriveJumpTargets(fixtureInput());
    const workTargets = targets.filter((target) => target.kind === "work-item");

    expect(workTargets.map((target) => target.id)).toEqual([
      "work-item:item_40",
      "work-item:item_41",
    ]);
    expect(workTargets[0]).toMatchObject({
      title: "#40 Jump palette foundation",
      subtitle: "execution / running",
      payload: { kind: "work-item", projectId: "proj_whisk", workItemId: "item_40" },
    });
    expect(workTargets[0].keywords).toEqual(
      expect.arrayContaining(["40", "#40", "execution", "running", "proj_whisk"]),
    );
  });

  it("derives terminal-backed work item run targets without promising non-terminal jumps", () => {
    const targets = deriveJumpTargets(fixtureInput());

    expect(targets.filter((target) => target.kind === "work-item-run").map((target) => target.id)).toEqual([
      "work-item-run:run_40",
    ]);
    expect(targets.find((target) => target.id === "work-item-run:run_40")).toMatchObject({
      title: "Run for #40 Jump palette foundation",
      subtitle: "running / codex",
      detail: "API / pty_api",
      payload: {
        kind: "work-item-run",
        projectId: "proj_whisk",
        workItemId: "item_40",
        runId: "run_40",
        sessionId: "sess_api",
        ptyId: "pty_api",
      },
    });
  });

  it("keeps all work items and runs available when no active project is selected", () => {
    const targets = deriveJumpTargets(fixtureInput({ activeProjectId: "" }));

    expect(targets.filter((target) => target.kind === "work-item").map((target) => target.id)).toEqual([
      "work-item:item_40",
      "work-item:item_other_project",
      "work-item:item_41",
    ]);
    expect(targets.filter((target) => target.kind === "work-item-run").map((target) => target.id)).toEqual([
      "work-item-run:run_40",
      "work-item-run:run_docs",
    ]);
  });
});
