type StageLike = {
  id: string;
  provisionWorktree?: boolean;
};

type WorkItemLike = {
  id: string;
  stageId: string;
  worktree?: unknown;
};

type RunLike = {
  id: string;
  workItemId: string;
  status: string;
  ptyId?: string;
  sessionId?: string;
  createdAt?: unknown;
};

type QuestionLike = {
  id?: string;
  workItemId: string;
  status: string;
};

type AgentPromptLike = {
  id?: string;
  runId?: string;
  status: string;
};

type GateLike = {
  id?: string;
  name?: string;
  workItemId: string;
  status: string;
  blocking?: boolean;
};

type ArtifactLike = {
  id?: string;
  workItemId: string;
  kind: string;
  status: string;
};

export type WorkItemAttentionTone = "info" | "success" | "warning" | "danger";

export type WorkItemAttentionSignal = {
  id: string;
  label: string;
  tone: WorkItemAttentionTone;
};

export type WorkItemAttention = {
  severity: "none" | "info" | "warning" | "danger";
  terminalRunId: string;
  signals: WorkItemAttentionSignal[];
};

export type WorkItemCardIndicator = {
  id: string;
  label: string;
  tone: WorkItemAttentionTone;
};

export function groupWorkItemsByStage<T extends WorkItemLike, S extends StageLike>(
  items: T[],
  stages: S[],
) {
  const result: Record<string, T[]> = {};
  for (const stage of stages) result[stage.id] = [];
  for (const item of items) {
    if (!result[item.stageId]) result[item.stageId] = [];
    result[item.stageId].push(item);
  }
  return result;
}

export function adjacentStageTargets<T extends WorkItemLike, S extends StageLike>(
  item: T,
  stages: S[],
) {
  const index = stages.findIndex((stage) => stage.id === item.stageId);
  const previous = index > 0 ? stages[index - 1] : null;
  const next = index >= 0 && index < stages.length - 1 ? stages[index + 1] : null;
  return {
    previous: previous && canMoveToStage(item, previous) ? previous : null,
    next: next && canMoveToStage(item, next) ? next : null,
    blockedNext: next && !canMoveToStage(item, next) ? next : null,
  };
}

export function canMoveToStage<T extends WorkItemLike, S extends StageLike>(item: T, stage: S) {
  return !stage.provisionWorktree || Boolean(item.worktree);
}

export function selectDetailRun<T extends RunLike>(runs: T[]): T | null {
  return runs[0] ?? null;
}

export type NextStepKind =
  | "open-terminal"
  | "launch-run"
  | "launch-execution"
  | "approve-plan"
  | "start-planning"
  | "retry-planning"
  | "send-to-review"
  | "mark-done"
  | "none";

export type NextStepView = {
  kind: NextStepKind;
  message: string;
  label: string;
  tone: "accent" | "primary" | "neutral";
  isLaunch: boolean;
};

// deriveNextStep maps an item's stage + current run status + plan state to the single
// recommended next action, so the detail view can guide the user instead of showing every
// transition at once. It is pure (no component state) to keep it unit-testable; the caller
// wires each kind to the matching handler.
export function deriveNextStep(context: {
  stageId: string;
  runStatus: string;
  hasTerminal: boolean;
  hasApprovedPlan: boolean;
  hasDraftPlan: boolean;
  hasLatestRun: boolean;
}): NextStepView {
  const { stageId, runStatus, hasTerminal, hasApprovedPlan, hasDraftPlan, hasLatestRun } = context;

  if (runStatus === "running" || runStatus === "awaiting_input") {
    return {
      kind: "open-terminal",
      message: runStatus === "awaiting_input" ? "The agent is waiting for input." : "A run is in progress.",
      label: hasTerminal ? "Open terminal" : "",
      tone: "neutral",
      isLaunch: false,
    };
  }
  if (runStatus === "queued") {
    return {
      kind: "launch-run",
      message: "A run is queued — pick an agent and launch it.",
      label: "Launch run",
      tone: "accent",
      isLaunch: true,
    };
  }
  if (stageId === "done") {
    return { kind: "none", message: "This item is done.", label: "", tone: "neutral", isLaunch: false };
  }
  if (stageId === "review") {
    return {
      kind: "mark-done",
      message: "Review the work, then mark it done or send feedback below.",
      label: "Mark done",
      tone: "accent",
      isLaunch: false,
    };
  }
  if (hasDraftPlan && !hasApprovedPlan) {
    return {
      kind: "approve-plan",
      message: "A draft plan is ready — review it and approve to continue.",
      label: "Approve plan",
      tone: "accent",
      isLaunch: false,
    };
  }
  if (stageId === "ready" && hasApprovedPlan) {
    return {
      kind: "launch-execution",
      message: "Plan approved. Choose an agent and launch execution.",
      label: "Launch execution",
      tone: "accent",
      isLaunch: true,
    };
  }
  if (runStatus === "failed") {
    return {
      kind: "retry-planning",
      message: "The last run failed. Review the output, then retry planning.",
      label: "Retry planning",
      tone: "primary",
      isLaunch: false,
    };
  }
  if (!hasApprovedPlan && !hasDraftPlan) {
    return {
      kind: "start-planning",
      message: "Start planning to generate a draft plan for review.",
      label: "Start planning",
      tone: "accent",
      isLaunch: false,
    };
  }
  if (hasApprovedPlan && hasLatestRun) {
    return {
      kind: "send-to-review",
      message: "Execution is done — send it to review.",
      label: "Send to review",
      tone: "accent",
      isLaunch: false,
    };
  }
  return {
    kind: "none",
    message: "Use the actions below to move this item forward.",
    label: "",
    tone: "neutral",
    isLaunch: false,
  };
}

export function deriveWorkItemAttention<T extends WorkItemLike>(
  item: T,
  context: {
    runs?: RunLike[];
    questions?: QuestionLike[];
    agentPrompts?: AgentPromptLike[];
    gates?: GateLike[];
    artifacts?: ArtifactLike[];
    stageRequiresWorktree?: boolean;
    stageRequiresPlan?: boolean;
  },
): WorkItemAttention {
  const signals: WorkItemAttentionSignal[] = [];
  const runs = context.runs ?? [];
  const latestRun = runs[0] ?? null;

  if (latestRun?.status === "failed" || latestRun?.status === "cancelled") {
    signals.push({
      id: "run-failed",
      label: latestRun.status === "failed" ? "Run failed" : "Cancelled",
      tone: "danger",
    });
  } else if (latestRun?.status === "awaiting_input") {
    signals.push({ id: "awaiting-input", label: "Awaiting input", tone: "warning" });
  } else if (latestRun?.status === "queued") {
    signals.push({ id: "queued", label: "Queued", tone: "info" });
  }

  const openQuestions = (context.questions ?? []).filter(
    (question) => question.workItemId === item.id && question.status === "open",
  ).length;
  if (openQuestions > 0) {
    signals.push({
      id: "open-questions",
      label: `${openQuestions} ${openQuestions === 1 ? "question" : "questions"}`,
      tone: "warning",
    });
  }

  const itemRunIDs = new Set(runs.filter((run) => run.workItemId === item.id).map((run) => run.id));
  const pendingAgentQuestions = (context.agentPrompts ?? []).filter(
    (prompt) => prompt.status === "pending" && prompt.runId && itemRunIDs.has(prompt.runId),
  ).length;
  if (pendingAgentQuestions > 0) {
    signals.push({
      id: "agent-questions",
      label: `${pendingAgentQuestions} agent ${pendingAgentQuestions === 1 ? "question" : "questions"}`,
      tone: "warning",
    });
  }

  const blockingGates = (context.gates ?? []).filter(
    (gate) =>
      gate.workItemId === item.id &&
      gate.blocking &&
      gate.status !== "passed" &&
      gate.status !== "overridden",
  ).length;
  if (blockingGates > 0) {
    signals.push({
      id: "blocking-gates",
      label: `${blockingGates} ${blockingGates === 1 ? "gate" : "gates"}`,
      tone: "danger",
    });
  }

  if (context.stageRequiresWorktree && !item.worktree) {
    signals.push({ id: "missing-worktree", label: "Needs worktree", tone: "warning" });
  }

  const hasApprovedPlan = (context.artifacts ?? []).some(
    (artifact) =>
      artifact.workItemId === item.id && artifact.kind === "plan" && artifact.status === "approved",
  );
  if (context.stageRequiresPlan && !hasApprovedPlan) {
    signals.push({ id: "missing-plan", label: "Needs plan", tone: "warning" });
  }

  const severity = signals.some((signal) => signal.tone === "danger")
    ? "danger"
    : signals.some((signal) => signal.tone === "warning")
      ? "warning"
      : signals.some((signal) => signal.tone === "info")
        ? "info"
        : "none";

  return {
    severity,
    terminalRunId:
      latestRun &&
      (latestRun.status === "running" || latestRun.status === "awaiting_input") &&
      (latestRun.ptyId || latestRun.sessionId)
        ? latestRun.id
        : "",
    signals,
  };
}

export function deriveWorkItemCardIndicators<T extends WorkItemLike>(
  item: T,
  context: {
    runs?: RunLike[];
    gates?: GateLike[];
    artifacts?: ArtifactLike[];
  },
): WorkItemCardIndicator[] {
  if (item.stageId === "done") {
    return [{ id: "done", label: "Done", tone: "success" }];
  }

  const indicators: WorkItemCardIndicator[] = [];
  const itemArtifacts = (context.artifacts ?? []).filter((artifact) => artifact.workItemId === item.id);
  const hasApprovedPlan = itemArtifacts.some(
    (artifact) => artifact.kind === "plan" && artifact.status === "approved",
  );
  const hasDraftPlan = itemArtifacts.some(
    (artifact) => artifact.kind === "plan" && artifact.status === "draft",
  );
  const latestRun = (context.runs ?? []).find((run) => run.workItemId === item.id) ?? null;

  if (hasApprovedPlan) {
    indicators.push({ id: "plan-approved", label: "Plan approved", tone: "success" });
  } else if (hasDraftPlan) {
    indicators.push({ id: "plan-draft", label: "Plan ready", tone: "info" });
  }

  if (latestRun?.status === "queued") {
    indicators.push({ id: "run-queued", label: "Queued", tone: "info" });
  } else if (latestRun?.status === "running") {
    indicators.push({ id: "run-running", label: "Running", tone: "success" });
  } else if (latestRun?.status === "awaiting_input") {
    indicators.push({ id: "run-awaiting-input", label: "Needs input", tone: "warning" });
  } else if (latestRun?.status === "failed" || latestRun?.status === "cancelled") {
    indicators.push({
      id: "run-failed",
      label: latestRun.status === "failed" ? "Run failed" : "Cancelled",
      tone: "danger",
    });
  } else if (item.stageId === "execution" && latestRun?.status === "completed") {
    indicators.push({ id: "execution-complete", label: "Ready for review", tone: "success" });
  }

  if (item.stageId === "review") {
    indicators.push({ id: "review", label: "Review work", tone: "info" });
    const hasOpenBlockingGate = (context.gates ?? []).some(
      (gate) =>
        gate.workItemId === item.id &&
        gate.blocking &&
        gate.status !== "passed" &&
        gate.status !== "overridden",
    );
    if (hasOpenBlockingGate) {
      indicators.push({ id: "review-gate", label: "Review gate", tone: "warning" });
    }
  }

  return indicators;
}

export function collapsedStageStorageKey(projectID: string) {
  return `whisk.workBoard.collapsedStages.${projectID}`;
}

export function serializeCollapsedStages(collapsed: Set<string>) {
  return JSON.stringify([...collapsed].filter(Boolean).sort());
}

export function parseCollapsedStages(raw: string | null) {
  if (!raw) return new Set<string>();
  try {
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return new Set<string>();
    return new Set(
      parsed
        .filter((value): value is string => typeof value === "string" && value.length > 0)
        .sort(),
    );
  } catch {
    return new Set<string>();
  }
}
