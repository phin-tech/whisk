import { describe, expect, it } from "vitest";
import type { WorkflowStage } from "../../bindings/github.com/phin-tech/whisk/internal/domain/workitem/models";
import type {
  Artifact,
  GateReport,
  Project,
  Question,
  WorkItem,
  WorkItemRun,
  WorkflowDefinitionRecord,
} from "../../bindings/github.com/phin-tech/whisk/internal/protocol/models";
import {
  canOpenRunTerminal,
  defaultWorktreeBranch,
  deriveWorkBoardView,
  filterWorkItems,
  groupRunsByItem,
  workflowDefinitionLabel,
} from "./work-board-state";

const stages: WorkflowStage[] = [
  { id: "backlog", name: "Backlog", kind: "backlog" },
  { id: "ready", name: "Ready", kind: "ready", provisionWorktree: true },
  { id: "execution", name: "Execution", kind: "execution", provisionWorktree: true },
  { id: "review", name: "Review", kind: "review" },
  { id: "done", name: "Done", kind: "done" },
] as WorkflowStage[];

function project(overrides: Partial<Project> = {}): Project {
  return {
    id: "project-1",
    name: "Whisk UI",
    slug: "whisk-ui",
    rootDir: "/tmp/whisk-ui",
    workflow: {
      id: "workflow-1",
      templateId: "template-1",
      definitionId: "plan-execute-review",
      definitionVersion: 2,
      name: "Plan Execute Review",
      stages,
      transitionRules: [],
    },
    preferences: { autoRun: "", autoWorktree: false },
    attachments: [],
    nextWorkItemNumber: 1,
    createdAt: null,
    updatedAt: null,
    ...overrides,
  } as Project;
}

function workItem(overrides: Partial<WorkItem>): WorkItem {
  return {
    id: "item-1",
    projectId: "project-1",
    workflowId: "workflow-1",
    workflowVersion: 1,
    number: 1,
    title: "Build the work board",
    bodyMarkdown: "Body",
    stageId: "backlog",
    runState: "idle",
    attachments: [],
    history: [],
    createdAt: null,
    updatedAt: null,
    ...overrides,
  } as WorkItem;
}

function run(overrides: Partial<WorkItemRun>): WorkItemRun {
  return {
    id: "run-1",
    workItemId: "item-1",
    projectId: "project-1",
    preset: "writer",
    promptTemplateId: "prompt-1",
    promptSnapshot: "",
    status: "completed",
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
    history: [],
    ...overrides,
  } as WorkItemRun;
}

function artifact(overrides: Partial<Artifact>): Artifact {
  return {
    id: "artifact-1",
    projectId: "project-1",
    workItemId: "item-1",
    kind: "plan",
    status: "draft",
    createdAt: null,
    updatedAt: null,
    ...overrides,
  } as Artifact;
}

function question(overrides: Partial<Question>): Question {
  return {
    id: "question-1",
    projectId: "project-1",
    workItemId: "item-1",
    prompt: "Clarify?",
    status: "open",
    createdAt: null,
    updatedAt: null,
    ...overrides,
  } as Question;
}

function gate(overrides: Partial<GateReport>): GateReport {
  return {
    id: "gate-1",
    projectId: "project-1",
    workItemId: "item-1",
    name: "Review",
    blocking: true,
    status: "pending",
    createdAt: null,
    updatedAt: null,
    ...overrides,
  } as GateReport;
}

function workflowDefinition(overrides: Partial<WorkflowDefinitionRecord> = {}): WorkflowDefinitionRecord {
  return {
    id: "plan-execute-review",
    version: 2,
    source: "builtin",
    contentHash: "hash-1",
    definition: {
      id: "plan-execute-review",
      version: 2,
      stages: [],
      actions: [],
      questions: {
        enabled: false,
        moveToBlocked: false,
        setsRunState: "",
        answerClearsAwaitingInputWhenNoOpenQuestionsRemain: false,
      },
      gates: [],
    },
    createdAt: null,
    updatedAt: null,
    ...overrides,
  } as WorkflowDefinitionRecord;
}

describe("work-board-state", () => {
  it("derives active project labels, filtered stage views, stable keys, and collapsed flags", () => {
    const backlog = workItem({ id: "item-backlog", number: 1, title: "Backlog item", stageId: "backlog" });
    const ready = workItem({
      id: "item-ready",
      number: 2,
      title: "Polish WorkBoard cards",
      bodyMarkdown: "Queued card",
      stageId: "ready",
      runState: "queued",
      worktree: { branch: "feature", base: "main", worktreePath: "/tmp/work", createdAt: null },
    });

    const view = deriveWorkBoardView({
      projects: [project()],
      activeProjectId: "project-1",
      workItems: [backlog, ready],
      workItemRuns: [run({ id: "ready-run", workItemId: "item-ready", status: "queued" })],
      artifacts: [],
      questions: [],
      gateReports: [],
      workflowDefinitions: [workflowDefinition()],
      filters: { query: "workboard", stageId: "ready", runState: "queued" },
      collapsedStageIds: new Set(["ready"]),
      detailItemId: "item-ready",
    });

    expect(view.activeProject?.id).toBe("project-1");
    expect(view.workflowLabel).toBe("plan-execute-review@2");
    expect(view.stages.map((stage) => stage.id)).toEqual([
      "backlog",
      "ready",
      "execution",
      "review",
      "done",
    ]);
    expect(view.stageViews).toHaveLength(1);
    expect(view.stageViews[0]).toMatchObject({
      key: "stage:ready",
      collapsed: true,
      count: 1,
      hasAttention: true,
      attentionClass: "bg-blue",
    });
    expect(view.stageViews[0].cards[0]).toMatchObject({
      key: "work-item:item-ready",
      item: ready,
      latestRun: { id: "ready-run" },
      canExecute: false,
    });
    expect(view.detailItem).toBe(ready);
  });

  it("derives per-card targets, terminal run, indicators, and execution eligibility", () => {
    const item = workItem({
      id: "item-ready",
      number: 2,
      stageId: "ready",
      runState: "running",
      worktree: { branch: "feature", base: "main", worktreePath: "/tmp/work", createdAt: null },
    });

    const view = deriveWorkBoardView({
      projects: [project()],
      activeProjectId: "project-1",
      workItems: [item],
      workItemRuns: [
        run({
          id: "older-run",
          workItemId: "item-ready",
          status: "failed",
          createdAt: "2026-01-01T00:00:00Z",
        }),
        run({
          id: "latest-run",
          workItemId: "item-ready",
          status: "running",
          sessionId: "session-1",
          createdAt: "2026-01-02T00:00:00Z",
        }),
      ],
      artifacts: [artifact({ id: "plan-1", workItemId: "item-ready", status: "approved" })],
      questions: [],
      gateReports: [],
      workflowDefinitions: [],
      collapsedStageIds: new Set(),
    });

    const card = view.stageViews.find((stage) => stage.stage.id === "ready")?.cards[0];
    expect(card?.targets.previous?.id).toBe("backlog");
    expect(card?.targets.next?.id).toBe("execution");
    expect(card?.latestRun?.id).toBe("latest-run");
    expect(card?.terminalRun?.id).toBe("latest-run");
    expect(card?.attention.terminalRunId).toBe("latest-run");
    expect(card?.indicators).toEqual([
      { id: "plan-approved", label: "Plan approved", tone: "success" },
      { id: "run-running", label: "Running", tone: "success" },
    ]);
    expect(card?.canExecute).toBe(false);
  });

  it("allows execution only for ready items with approved plans and no active run", () => {
    const item = workItem({ id: "item-ready", stageId: "ready" });

    const view = deriveWorkBoardView({
      projects: [project()],
      activeProjectId: "project-1",
      workItems: [item],
      workItemRuns: [run({ id: "complete-run", workItemId: "item-ready", status: "completed" })],
      artifacts: [artifact({ id: "plan-1", workItemId: "item-ready", status: "approved" })],
      questions: [],
      gateReports: [],
      workflowDefinitions: [],
    });

    expect(view.stageViews.find((stage) => stage.stage.id === "ready")?.cards[0].canExecute).toBe(
      true,
    );
  });

  it("prioritizes danger stage attention across card signals", () => {
    const reviewItem = workItem({ id: "item-review", stageId: "review" });
    const view = deriveWorkBoardView({
      projects: [project()],
      activeProjectId: "project-1",
      workItems: [reviewItem],
      workItemRuns: [run({ id: "review-run", workItemId: "item-review", status: "completed" })],
      artifacts: [],
      questions: [question({ workItemId: "item-review" })],
      gateReports: [gate({ workItemId: "item-review" })],
      workflowDefinitions: [],
    });

    const review = view.stageViews.find((stage) => stage.stage.id === "review");
    expect(review?.hasAttention).toBe(true);
    expect(review?.attentionClass).toBe("bg-red");
    expect(review?.cards[0].attention.signals.map((signal) => signal.id)).toEqual([
      "open-questions",
      "blocking-gates",
      "missing-plan",
    ]);
  });

  it("filters work items by query, stage, and run state", () => {
    const items = [
      workItem({ id: "a", title: "Alpha", bodyMarkdown: "needs docs", stageId: "ready", runState: "idle" }),
      workItem({ id: "b", title: "Beta", bodyMarkdown: "ship", stageId: "review", runState: "queued" }),
    ];

    expect(filterWorkItems(items, { query: "docs", stageId: "ready", runState: "idle" })).toEqual([
      items[0],
    ]);
    expect(filterWorkItems(items, { query: "docs", stageId: "review" })).toEqual([]);
  });

  it("groups runs by item with newest runs first", () => {
    const grouped = groupRunsByItem([
      run({ id: "old", workItemId: "item-1", createdAt: "2026-01-01T00:00:00Z" }),
      run({ id: "new", workItemId: "item-1", createdAt: "2026-01-02T00:00:00Z" }),
      run({ id: "other", workItemId: "item-2", createdAt: "2026-01-03T00:00:00Z" }),
    ]);

    expect(grouped["item-1"].map((itemRun) => itemRun.id)).toEqual(["new", "old"]);
    expect(grouped["item-2"].map((itemRun) => itemRun.id)).toEqual(["other"]);
  });

  it("formats workflow labels and default branches without component state", () => {
    expect(workflowDefinitionLabel(project(), [workflowDefinition()])).toBe("plan-execute-review@2");
    expect(workflowDefinitionLabel(project({ workflow: { ...project().workflow, definitionId: "", definitionVersion: 0 } }), [])).toBe(
      "template-1",
    );
    expect(defaultWorktreeBranch(workItem({ number: 42, title: "Ship UI polish!" }), project())).toBe(
      "whisk/whisk-ui-42-ship-ui-polish",
    );
    expect(defaultWorktreeBranch(workItem({ number: 1, title: "..." }), null)).toBe(
      "whisk/work-1-item",
    );
  });

  it("keeps terminal opening gated by run terminal bindings", () => {
    expect(canOpenRunTerminal(run({ sessionId: "session-1" }))).toBe(true);
    expect(canOpenRunTerminal(run({ ptyId: "pty-1" }))).toBe(true);
    expect(canOpenRunTerminal(run({}))).toBe(false);
    expect(canOpenRunTerminal(null)).toBe(false);
  });
});
