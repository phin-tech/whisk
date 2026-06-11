import { describe, expect, it } from "vitest";
import { adjacentStageTargets, canMoveToStage, groupWorkItemsByStage } from "./workView";

const stages = [
  { id: "backlog" },
  { id: "ready", provisionWorktree: true },
  { id: "in_progress" },
];

describe("workView", () => {
  it("groups work items by workflow stage and keeps unknown stages visible", () => {
    const grouped = groupWorkItemsByStage(
      [
        { id: "a", stageId: "backlog" },
        { id: "b", stageId: "in_progress" },
        { id: "c", stageId: "custom" },
      ],
      stages,
    );

    expect(grouped.backlog.map((item) => item.id)).toEqual(["a"]);
    expect(grouped.ready).toEqual([]);
    expect(grouped.in_progress.map((item) => item.id)).toEqual(["b"]);
    expect(grouped.custom.map((item) => item.id)).toEqual(["c"]);
  });

  it("blocks movement into provisioned stages until a worktree exists", () => {
    const item = { id: "a", stageId: "backlog" };

    expect(canMoveToStage(item, stages[1])).toBe(false);
    expect(adjacentStageTargets(item, stages)).toEqual({
      previous: null,
      next: null,
      blockedNext: stages[1],
    });
  });

  it("allows movement into provisioned stages after worktree binding", () => {
    const item = { id: "a", stageId: "backlog", worktree: { branch: "feature" } };

    expect(canMoveToStage(item, stages[1])).toBe(true);
    expect(adjacentStageTargets(item, stages)).toEqual({
      previous: null,
      next: stages[1],
      blockedNext: null,
    });
  });
});
