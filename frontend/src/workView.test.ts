import { describe, expect, it } from "vitest";
import {
  adjacentStageTargets,
  canMoveToStage,
  collapsedStageStorageKey,
  deriveNextStep,
  deriveWorkItemCardIndicators,
  deriveWorkItemAttention,
  groupWorkItemsByStage,
  parseCollapsedStages,
  selectDetailRun,
  serializeCollapsedStages,
} from "./workView";

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

  it("derives only attention-worthy card signals", () => {
    const item = { id: "a", stageId: "planning" };

    expect(
      deriveWorkItemAttention(item, {
        runs: [{ id: "run-1", workItemId: "a", status: "awaiting_input", ptyId: "pty-1" }],
        questions: [{ id: "q-1", workItemId: "a", status: "open" }],
        gates: [{ id: "g-1", workItemId: "a", name: "Review", status: "pending", blocking: true }],
        artifacts: [{ id: "plan-1", workItemId: "a", kind: "plan", status: "draft" }],
        stageRequiresWorktree: true,
      }),
    ).toEqual({
      severity: "danger",
      terminalRunId: "run-1",
      signals: [
        { id: "awaiting-input", label: "Awaiting input", tone: "warning" },
        { id: "open-questions", label: "1 question", tone: "warning" },
        { id: "blocking-gates", label: "1 gate", tone: "danger" },
        { id: "missing-worktree", label: "Needs worktree", tone: "warning" },
      ],
    });
  });

  it("keeps healthy workflow state quiet", () => {
    expect(
      deriveWorkItemAttention(
        { id: "a", stageId: "execution", worktree: { branch: "feature" } },
        {
          runs: [{ id: "run-1", workItemId: "a", status: "completed" }],
          questions: [{ id: "q-1", workItemId: "a", status: "answered" }],
          gates: [{ id: "g-1", workItemId: "a", name: "Review", status: "passed", blocking: true }],
          artifacts: [{ id: "plan-1", workItemId: "a", kind: "plan", status: "approved" }],
          stageRequiresPlan: true,
          stageRequiresWorktree: true,
        },
      ),
    ).toEqual({ severity: "none", terminalRunId: "", signals: [] });
  });

  it("shows pending agent prompts linked through work item runs", () => {
    expect(
      deriveWorkItemAttention(
        { id: "a", stageId: "execution", worktree: { branch: "feature" } },
        {
          runs: [
            { id: "run-1", workItemId: "a", status: "running" },
            { id: "run-2", workItemId: "b", status: "running" },
          ],
          agentPrompts: [
            { id: "prompt-1", runId: "run-1", status: "pending" },
            { id: "prompt-2", runId: "run-1", status: "resolved" },
            { id: "prompt-3", runId: "run-2", status: "pending" },
          ],
        },
      ),
    ).toEqual({
      severity: "warning",
      terminalRunId: "",
      signals: [{ id: "agent-questions", label: "1 agent question", tone: "warning" }],
    });
  });

  it("treats queued runs as visible actionable info", () => {
    expect(
      deriveWorkItemAttention(
        { id: "a", stageId: "execution", worktree: { branch: "feature" } },
        {
          runs: [{ id: "run-1", workItemId: "a", status: "queued" }],
        },
      ),
    ).toEqual({
      severity: "info",
      terminalRunId: "",
      signals: [{ id: "queued", label: "Queued", tone: "info" }],
    });
  });

  it("exposes linked running runs as terminal targets", () => {
    expect(
      deriveWorkItemAttention(
        { id: "a", stageId: "execution", worktree: { branch: "feature" } },
        {
          runs: [{ id: "run-1", workItemId: "a", status: "running", sessionId: "session-1" }],
        },
      ).terminalRunId,
    ).toBe("run-1");
  });

  it("derives progress indicators for card surfaces", () => {
    const item = { id: "a", stageId: "planning" };

    expect(
      deriveWorkItemCardIndicators(item, {
        artifacts: [{ id: "plan-1", workItemId: "a", kind: "plan", status: "draft" }],
      }),
    ).toEqual([{ id: "plan-draft", label: "Plan ready", tone: "info" }]);

    expect(
      deriveWorkItemCardIndicators(
        { ...item, stageId: "ready" },
        {
          artifacts: [{ id: "plan-1", workItemId: "a", kind: "plan", status: "approved" }],
        },
      ),
    ).toEqual([{ id: "plan-approved", label: "Plan approved", tone: "success" }]);

    expect(
      deriveWorkItemCardIndicators(
        { ...item, stageId: "execution" },
        {
          artifacts: [{ id: "plan-1", workItemId: "a", kind: "plan", status: "approved" }],
          runs: [{ id: "run-1", workItemId: "a", status: "running" }],
        },
      ),
    ).toEqual([
      { id: "plan-approved", label: "Plan approved", tone: "success" },
      { id: "run-running", label: "Running", tone: "success" },
    ]);

    expect(
      deriveWorkItemCardIndicators(
        { ...item, stageId: "review" },
        {
          gates: [{ id: "gate-1", workItemId: "a", status: "pending", blocking: true }],
          runs: [{ id: "run-1", workItemId: "a", status: "completed" }],
        },
      ),
    ).toEqual([
      { id: "review", label: "Review work", tone: "info" },
      { id: "review-gate", label: "Review gate", tone: "warning" },
    ]);

    expect(deriveWorkItemCardIndicators({ ...item, stageId: "done" }, {})).toEqual([
      { id: "done", label: "Done", tone: "success" },
    ]);
  });

  it("selects the latest run for the detail modal even when an older run is cancellable", () => {
    expect(
      selectDetailRun([
        { id: "execution", workItemId: "a", status: "cancelled" },
        { id: "planning", workItemId: "a", status: "queued" },
      ])?.id,
    ).toBe("execution");
  });

  it("serializes collapsed stages as stable client-owned view state", () => {
    expect(collapsedStageStorageKey("project-1")).toBe("whisk.workBoard.collapsedStages.project-1");
    expect(serializeCollapsedStages(new Set(["done", "backlog", "done"]))).toBe(
      '["backlog","done"]',
    );
    expect(parseCollapsedStages('["done","",42,"backlog","done"]')).toEqual(
      new Set(["backlog", "done"]),
    );
    expect(parseCollapsedStages("not json")).toEqual(new Set());
  });
});

describe("deriveNextStep", () => {
  const base = {
    stageId: "planning",
    runStatus: "",
    hasTerminal: false,
    hasApprovedPlan: false,
    hasDraftPlan: false,
    hasLatestRun: false,
  };

  it("offers planning when there is no plan yet", () => {
    const step = deriveNextStep({ ...base });
    expect(step.kind).toBe("start-planning");
    expect(step.isLaunch).toBe(false);
    expect(step.label).toBe("Start planning");
  });

  it("offers retry when the run failed and no plan exists", () => {
    expect(deriveNextStep({ ...base, runStatus: "failed" }).kind).toBe("retry-planning");
  });

  it("prioritises approving a ready draft plan", () => {
    expect(deriveNextStep({ ...base, hasDraftPlan: true }).kind).toBe("approve-plan");
  });

  it("offers a launch (with agent picker) once approved in ready", () => {
    const step = deriveNextStep({ ...base, stageId: "ready", hasApprovedPlan: true });
    expect(step.kind).toBe("launch-execution");
    expect(step.isLaunch).toBe(true);
  });

  it("treats a queued run as a launchable run", () => {
    const step = deriveNextStep({ ...base, runStatus: "queued" });
    expect(step.kind).toBe("launch-run");
    expect(step.isLaunch).toBe(true);
  });

  it("surfaces the terminal while a run is active", () => {
    const running = deriveNextStep({ ...base, runStatus: "running", hasTerminal: true });
    expect(running.kind).toBe("open-terminal");
    expect(running.label).toBe("Open terminal");
    // No terminal linked -> no button label, but still not a transition.
    expect(deriveNextStep({ ...base, runStatus: "running" }).label).toBe("");
  });

  it("guides review and done stages", () => {
    expect(deriveNextStep({ ...base, stageId: "review" }).kind).toBe("mark-done");
    expect(deriveNextStep({ ...base, stageId: "done" }).kind).toBe("none");
  });

  it("sends finished execution to review", () => {
    const step = deriveNextStep({ ...base, stageId: "execution", hasApprovedPlan: true, hasLatestRun: true });
    expect(step.kind).toBe("send-to-review");
  });
});
