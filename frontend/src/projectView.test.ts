import { describe, expect, it } from "vitest";
import {
  projectDetailCounts,
  projectDetailWithStoreSessions,
  selectedProjectDetail,
  sessionNameSuffix,
  sortRunsRecent,
} from "./projectView";

describe("projectView", () => {
  it("counts daemon-owned project detail sections", () => {
    expect(
      projectDetailCounts({
        project: { id: "proj_01" },
        workItems: [{ id: "wi_01" }],
        sessions: [{ id: "sess_01" }, { id: "sess_02" }],
        runs: [{ id: "run_01" }],
      }),
    ).toEqual({ workItems: 1, sessions: 2, runs: 1 });
  });

  it("uses an empty local projection until daemon detail arrives", () => {
    expect(
      selectedProjectDetail([{ id: "proj_01" }], null, "proj_01"),
    ).toEqual({
      project: { id: "proj_01" },
      workItems: [],
      sessions: [],
      runs: [],
    });
    expect(selectedProjectDetail([{ id: "proj_01" }], null, "missing")).toBeNull();
  });

  it("sorts runs newest first without mutating the daemon payload", () => {
    const runs = [
      { id: "old-cancelled", status: "cancelled", createdAt: "2026-01-01T10:00:00Z", updatedAt: "2026-01-01T10:01:00Z" },
      { id: "created-only", status: "completed", createdAt: "2026-01-03T10:00:00Z", updatedAt: "" },
      { id: "new-plan", status: "completed", createdAt: "2026-01-02T10:00:00Z", updatedAt: "2026-01-04T10:00:00Z" },
    ];

    expect(sortRunsRecent(runs).map((run) => run.id)).toEqual([
      "new-plan",
      "created-only",
      "old-cancelled",
    ]);
    expect(runs.map((run) => run.id)).toEqual(["old-cancelled", "created-only", "new-plan"]);
  });

  it("only disambiguates same-named sessions", () => {
    const sessions = [
      { id: "sess_abcdef123", name: "dev", rootDir: "/repo/a" },
      { id: "sess_123456789", name: "dev", rootDir: "/repo/b" },
      { id: "sess_unique", name: "ops", rootDir: "/repo/c" },
    ];

    expect(sessionNameSuffix(sessions[0], sessions)).toBe("abcdef");
    expect(sessionNameSuffix(sessions[2], sessions)).toBe("");
  });

  it("normalizes project detail sessions from the live session store", () => {
    const detail = {
      project: { id: "proj_01" },
      workItems: [],
      runs: [],
      sessions: [{ id: "stale", projectId: "proj_01" }],
    };
    const sessions = [
      { id: "live-a", projectId: "proj_01" },
      { id: "other-project", projectId: "proj_02" },
      { id: "live-b", projectId: "proj_01" },
    ];

    expect(projectDetailWithStoreSessions(detail, "proj_01", sessions)?.sessions).toEqual([
      { id: "live-a", projectId: "proj_01" },
      { id: "live-b", projectId: "proj_01" },
    ]);
    expect(projectDetailWithStoreSessions(detail, "proj_02", sessions)).toBe(detail);
    expect(projectDetailWithStoreSessions(null, "proj_01", sessions)).toBeNull();
  });
});
