import { describe, expect, it } from "vitest";
import { projectDetailCounts, selectedProjectDetail } from "./projectView";

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
});
