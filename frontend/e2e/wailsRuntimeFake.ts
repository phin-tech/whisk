type Listener = (event: { name: string; data: unknown }) => void;
type WorkItem = {
  id: string;
  projectId: string;
  workflowId: string;
  workflowVersion: number;
  number: number;
  title: string;
  bodyMarkdown: string;
  stageId: string;
  runState: string;
  worktree?: { branch: string; worktreePath: string } | null;
  attachments: unknown[];
  history: unknown[];
  createdAt: string;
  updatedAt: string;
};

type WorkItemLink = {
  id: string;
  projectId: string;
  sourceWorkItemId: string;
  targetWorkItemId: string;
  type: string;
  createdBy: string;
  createdAt: string;
};

const methodPrefix = "github.com/phin-tech/whisk/internal/wailsapp.Service.";
const listeners = new Map<string, Set<Listener>>();
const nextEventWaiters: Array<(event: unknown) => void> = [];
const seedActivePTY =
  typeof window !== "undefined" && new URLSearchParams(window.location.search).has("e2ePty");

let state = seedState();
let calls: Array<{ method: string; args: unknown[] }> = [];
let openedURLs: string[] = [];
let agentHookLogStatus = {
  enabled: true,
  clearAfterSession: false,
  path: "/tmp/whisk-e2e/agent-hooks.jsonl",
  sizeBytes: 0,
};

function seedState() {
  const now = "2026-06-25T12:00:00Z";
  const workflowDefinition = {
    id: "plan-execute-review",
    version: 1,
    stages: ["backlog", "planning", "ready", "execution", "blocked", "review", "done"],
    actions: [
      {
        id: "start_planning",
        from: ["backlog"],
        to: "planning",
        createsRun: {
          phase: "planning",
          preset: "reader",
          promptTemplateId: "plan",
          workingDir: "projectRoot",
        },
      },
      {
        id: "submit_draft_plan",
        from: ["planning"],
        to: "planning",
        createsArtifact: { kind: "plan", status: "draft" },
      },
      {
        id: "approve_plan",
        from: ["planning"],
        to: "ready",
        requires: [{ kind: "plan", status: "draft" }],
        updatesArtifact: { kind: "plan", status: "approved" },
        requiresHuman: true,
      },
      {
        id: "start_execution",
        from: ["ready"],
        to: "execution",
        requires: [{ kind: "plan", status: "approved" }],
        createsRun: {
          phase: "execution",
          preset: "writer",
          promptTemplateId: "implement",
          workingDir: "worktree",
          autoProvisionWorktree: true,
        },
      },
      {
        id: "complete_execution",
        from: ["execution"],
        to: "review",
        completesRun: true,
        createsGates: ["review"],
      },
      {
        id: "submit_review_feedback",
        from: ["review"],
        to: "execution",
        createsArtifact: { kind: "feedback", status: "approved" },
        resumesRun: "existing_execution",
      },
      {
        id: "approve_done",
        from: ["review"],
        to: "done",
        requiresPassingBlockingGates: true,
        requiresHuman: true,
      },
      {
        id: "report_blocked",
        from: ["planning", "execution", "review"],
        to: "blocked",
        sideStage: true,
      },
      {
        id: "unblock",
        from: ["blocked"],
        to: "$previousStage",
      },
    ],
    questions: {
      enabled: true,
      moveToBlocked: true,
      setsRunState: "awaiting_input",
      answerClearsAwaitingInputWhenNoOpenQuestionsRemain: true,
    },
    gates: [
      { id: "review", phase: "review", blocking: true },
    ],
  };
  const leanWorkflowDefinition = {
    id: "lean-review",
    version: 1,
    stages: ["backlog", "ready", "done"],
    actions: [
      { id: "ship", from: ["backlog", "ready"], to: "done", requiresHuman: true },
    ],
    questions: {
      enabled: false,
      moveToBlocked: false,
      setsRunState: "",
      answerClearsAwaitingInputWhenNoOpenQuestionsRemain: false,
    },
    gates: [],
  };
  const workflow = {
    id: "default",
    templateId: "default",
    definitionId: workflowDefinition.id,
    definitionVersion: workflowDefinition.version,
    definitionHash: "e2e-workflow",
    name: "Plan Execute Review",
    stages: [
      { id: "backlog", name: "Backlog", kind: "backlog" },
      { id: "planning", name: "Planning", kind: "planning" },
      { id: "ready", name: "Ready", kind: "ready", provisionWorktree: true },
      { id: "execution", name: "Execution", kind: "execution", provisionWorktree: true },
      { id: "blocked", name: "Blocked", kind: "blocked" },
      { id: "review", name: "Review", kind: "review" },
      { id: "done", name: "Done", kind: "done" },
    ],
    transitionRules: [],
  };
  const project = {
    id: "proj_01",
    name: "Whisk E2E",
    slug: "whisk-e2e",
    rootDir: "/tmp/whisk-e2e",
    description: "Seeded project for Playwright",
    workflow,
    preferences: {
      defaultPhaseAgents: {},
      interactiveAgentShell: true,
    },
    attachments: [],
    nextWorkItemNumber: 7,
    createdAt: now,
    updatedAt: now,
  };
  const workItems: WorkItem[] = [
    workItem({ id: "wi_backlog", number: 1, title: "Capture app launch smoke", stageId: "backlog", runState: "idle" }),
    workItem({ id: "wi_ready", number: 2, title: "Polish WorkBoard cards", stageId: "ready", runState: "queued" }),
    workItem({ id: "wi_exec", number: 3, title: "Validate terminal reconnect", stageId: "execution", runState: "running" }),
    workItem({ id: "wi_review", number: 4, title: "Review design token sweep", stageId: "review", runState: "awaiting_input" }),
    workItem({ id: "wi_done", number: 5, title: "Ship Wails bridge contract", stageId: "done", runState: "completed" }),
    workItem({ id: "wi_dependency", number: 6, title: "Map dependency graph", stageId: "ready", runState: "idle" }),
  ];
  return {
    now,
    project,
    projects: [project],
    workflowDefinitions: [
      {
        id: workflowDefinition.id,
        version: workflowDefinition.version,
        source: "builtin",
        sourcePath: "",
        contentHash: "e2e-workflow",
        definition: workflowDefinition,
        createdAt: now,
        updatedAt: now,
      },
      {
        id: leanWorkflowDefinition.id,
        version: leanWorkflowDefinition.version,
        source: "file",
        sourcePath: "/tmp/whisk-e2e/lean-review.json",
        contentHash: "e2e-lean-workflow",
        definition: leanWorkflowDefinition,
        createdAt: now,
        updatedAt: now,
      },
    ],
    sessions: [
      {
        id: "sess_01",
        projectId: "proj_01",
        name: "Seeded Session",
        rootDir: "/tmp/whisk-e2e",
        windows: {
          win_01: {
            id: "win_01",
            sessionId: "sess_01",
            name: "Main",
            layout: { kind: "leaf", paneId: "pane_01" },
          },
        },
        panes: {
          pane_01: {
            id: "pane_01",
            windowId: "win_01",
            currentPtyId: seedActivePTY ? "pty_01" : null,
            workingDir: "/tmp/whisk-e2e",
          },
        },
      },
    ],
    ptys: [
      {
        id: "pty_01",
        sessionId: "sess_01",
        paneId: "pane_01",
        title: "Seeded shell",
        command: "zsh",
        cwd: "/tmp/whisk-e2e",
        status: "running",
        createdAt: now,
      },
    ],
    ptyHistory: [],
    workItems,
    workItemLinks: [] as WorkItemLink[],
    runs: [
      {
        id: "run_queued",
        workItemId: "wi_ready",
        projectId: "proj_01",
        preset: "writer",
        promptTemplateId: "execution",
        promptSnapshot: "Execute the work item",
        status: "queued",
        createdAt: now,
        updatedAt: now,
        history: [],
      },
      {
        id: "run_exec",
        workItemId: "wi_exec",
        projectId: "proj_01",
        preset: "writer",
        promptTemplateId: "execution",
        promptSnapshot: "Run the agent",
        sessionId: "sess_01",
        ptyId: "pty_01",
        status: "running",
        createdAt: now,
        updatedAt: now,
        history: [],
      },
    ],
    artifacts: [
      {
        id: "artifact_plan",
        workItemId: "wi_ready",
        kind: "plan",
        title: "Plan",
        body: "Use Playwright WebKit and Chromium against the Wails dev server.",
        status: "approved",
        createdAt: now,
        updatedAt: now,
      },
      {
        id: "artifact_exec_plan",
        workItemId: "wi_exec",
        kind: "plan",
        title: "Plan",
        body: "Reconnect terminals after reload.",
        status: "approved",
        createdAt: now,
        updatedAt: now,
      },
      {
        id: "artifact_review_plan",
        workItemId: "wi_review",
        kind: "plan",
        title: "Plan",
        body: "Review token usage and rendered states.",
        status: "approved",
        createdAt: now,
        updatedAt: now,
      },
    ],
    questions: [
      {
        id: "question_01",
        workItemId: "wi_review",
        runId: "run_review",
        prompt: "Should WebKit be required in CI?",
        answer: "",
        status: "open",
        createdAt: now,
        updatedAt: now,
      },
    ],
    gates: [
      {
        id: "gate_01",
        workItemId: "wi_review",
        name: "Design QA",
        blocking: true,
        status: "pending",
        message: "Needs rendered verification.",
        createdAt: now,
        updatedAt: now,
      },
    ],
    workflowEvents: [
      {
        id: "event_01",
        workItemId: "wi_ready",
        type: "planning_started",
        message: "Planning started",
        createdAt: now,
      },
    ],
    statusEvents: [
      {
        id: "status_01",
        type: "agent.status",
        title: "Seeded approval requested",
        message: "The E2E fake has a notification.",
        status: "unread",
        createdAt: now,
      },
    ],
    agentPrompts: [
      {
        id: "prompt_01",
        runId: "run_exec",
        title: "Choose next step",
        message: "Pick a seeded option.",
        status: "pending",
        options: [{ label: "Continue", value: "continue" }],
        createdAt: now,
      },
    ],
    agentBridgeApprovals: [],
    agentBridgeEvents: [],
  };

  function workItem(input: { id: string; number: number; title: string; stageId: string; runState: string }): WorkItem {
    return {
      projectId: "proj_01",
      workflowId: workflowDefinition.id,
      workflowVersion: 1,
      bodyMarkdown: `Seeded body for ${input.title}.`,
      worktree:
        input.stageId === "execution"
          ? { branch: "whisk/e2e-execution", worktreePath: "/tmp/whisk-e2e/.worktrees/execution" }
          : null,
      attachments: [],
      history: [],
      createdAt: now,
      updatedAt: now,
      ...input,
    };
  }
}

function clone<T>(value: T): T {
  return structuredClone(value);
}

function createdWorkItem(req: any): WorkItem {
  const next = workItemFromRequest({
    id: `wi_${state.workItems.length + 1}`,
    number: state.project.nextWorkItemNumber++,
    title: req.title,
    bodyMarkdown: req.bodyMarkdown || "",
    stageId: "backlog",
    runState: "idle",
  });
  state.workItems = [...state.workItems, next];
  return next;
}

function workItemFromRequest(input: Partial<WorkItem> & { id: string; number: number; title: string }): WorkItem {
  return {
    projectId: "proj_01",
    workflowId: state.project.workflow.definitionId,
    workflowVersion: 1,
    bodyMarkdown: "",
    stageId: "backlog",
    runState: "idle",
    worktree: null,
    attachments: [],
    history: [],
    createdAt: state.now,
    updatedAt: state.now,
    ...input,
  };
}

function listForWorkItem<T extends { workItemId: string }>(records: T[], workItemID: unknown) {
  return workItemID ? records.filter((record) => record.workItemId === workItemID) : records;
}

function readyWorkExplanation() {
  const itemsByID = new Map(state.workItems.map((item) => [item.id, item]));
  const ready = [];
  const blocked = [];

  for (const item of state.workItems.filter((candidate) => candidate.projectId === state.project.id && candidate.stageId === "ready")) {
    const outgoing = state.workItemLinks.filter((link) => link.sourceWorkItemId === item.id);
    const incoming = state.workItemLinks.filter((link) => link.targetWorkItemId === item.id);
    const blockingLinks = outgoing.filter((link) => link.type === "blocks");
    const unresolved = blockingLinks.filter((link) => itemsByID.get(link.targetWorkItemId)?.stageId !== "done");

    if (unresolved.length > 0) {
      blocked.push({
        workItem: clone(item),
        blockedBy: unresolved.map((link) => {
          const blocker = itemsByID.get(link.targetWorkItemId);
          return {
            id: link.targetWorkItemId,
            number: blocker?.number ?? 0,
            title: blocker?.title ?? "",
            stageId: blocker?.stageId ?? "",
            runState: blocker?.runState ?? "",
          };
        }),
        blockedByCount: unresolved.length,
      });
      continue;
    }

    const resolvedBlockers = blockingLinks.map((link) => link.targetWorkItemId);
    ready.push({
      workItem: clone(item),
      reason: resolvedBlockers.length > 0 ? `${resolvedBlockers.length} blocker(s) resolved` : "no blocking dependencies",
      resolvedBlockers,
      dependencyCount: outgoing.length,
      dependentCount: incoming.length,
    });
  }

  return {
    ready,
    blocked,
    summary: { totalReady: ready.length, totalBlocked: blocked.length, cycleCount: 0 },
  };
}

function workflowDefinitionRecord(id: string, version: number) {
  return state.workflowDefinitions.find(
    (candidate) => candidate.id === id && candidate.version === version,
  );
}

function workflowTemplateFromDefinition(record: any) {
  const label = (stage: string) => stage.replace(/_/g, " ").replace(/\b\w/g, (value) => value.toUpperCase());
  return {
    id: record.id,
    templateId: record.id,
    definitionId: record.id,
    definitionVersion: record.version,
    definitionHash: record.contentHash,
    name: label(record.id),
    stages: record.definition.stages.map((stage: string) => ({
      id: stage,
      name: label(stage),
      kind: stage,
      provisionWorktree: stage === "ready" || stage === "execution",
    })),
    transitionRules: [],
  };
}

function workflowDefinitionForItem(item: WorkItem) {
  return workflowDefinitionRecord(item.workflowId, item.workflowVersion) ??
    workflowDefinitionRecord(state.project.workflow.definitionId, state.project.workflow.definitionVersion);
}

function artifactMatches(itemID: string, requirement: { kind: string; status: string }) {
  return state.artifacts.some(
    (artifact: any) =>
      artifact.workItemId === itemID &&
      artifact.kind === requirement.kind &&
      artifact.status === requirement.status,
  );
}

function workflowRequirementReason(requirement: { kind: string; status: string }) {
  if (requirement.kind === "plan" && requirement.status === "draft") return "plan draft required";
  if (requirement.kind === "plan" && requirement.status === "approved") return "approved plan required";
  return `${requirement.status} ${requirement.kind} artifact required`;
}

function workflowActionInputKind(action: any) {
  if (action.createsRun || action.completesRun) return "run";
  if (action.createsArtifact) return "artifact";
  if (action.updatesArtifact) return "artifact_selection";
  if (action.requiresPassingBlockingGates) return "gate";
  return "none";
}

function listWorkItemWorkflowActions(workItemID: string) {
  const item = state.workItems.find((candidate) => candidate.id === workItemID);
  if (!item) throw new Error("work item not found");
  const record = workflowDefinitionForItem(item);
  if (!record) throw new Error("workflow definition not found");

  return record.definition.actions
    .filter((action: any) => (action.from ?? []).includes(item.stageId))
    .map((action: any) => {
      const missing = (action.requires ?? []).find((requirement: any) => !artifactMatches(item.id, requirement));
      let enabled = !missing;
      let reason = missing ? workflowRequirementReason(missing) : "";
      if (enabled && action.requiresPassingBlockingGates) {
        const blockingGate = state.gates.find(
          (gate: any) =>
            gate.workItemId === item.id &&
            gate.blocking &&
            gate.status !== "passed" &&
            gate.status !== "overridden",
        );
        if (blockingGate) {
          enabled = false;
          reason = "blocking gates must pass or be overridden";
        }
      }
      return {
        action: clone(action),
        enabled,
        reason,
        inputKind: workflowActionInputKind(action),
      };
    });
}

function validationReportForDefinition(definition: any) {
  const errors = [];
  if (!definition?.id) errors.push({ path: "id", message: "workflow id required" });
  if (!definition?.version || definition.version <= 0) errors.push({ path: "version", message: "workflow version must be positive" });
  const stages = new Set(definition?.stages ?? []);
  for (const [index, action] of (definition?.actions ?? []).entries()) {
    if (!action.id) errors.push({ path: `actions[${index}].id`, message: "workflow action id required" });
    for (const [fromIndex, stage] of (action.from ?? []).entries()) {
      if (!stages.has(stage)) errors.push({ path: `actions[${index}].from[${fromIndex}]`, message: `unknown stage ${stage}` });
    }
    if (action.to && !stages.has(action.to) && action.to !== "$previousStage") {
      errors.push({ path: `actions[${index}].to`, message: `unknown stage ${action.to}` });
    }
  }
  return {
    valid: errors.length === 0,
    identity: definition?.id && definition?.version ? `${definition.id}@${definition.version}` : "",
    errors,
  };
}

function workflowDefinitionFromPath(path: string) {
  const slug = path
    .split("/")
    .pop()
    ?.replace(/\.json$/i, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "") || "imported-workflow";
  return {
    id: slug,
    version: 1,
    stages: ["backlog", "planning", "ready", "done"],
    actions: [
      { id: "start_planning", from: ["backlog"], to: "planning", createsRun: { phase: "planning", preset: "reader", promptTemplateId: "plan", workingDir: "projectRoot" } },
      { id: "approve_plan", from: ["planning"], to: "ready", requiresHuman: true },
      { id: "approve_done", from: ["ready"], to: "done", requiresHuman: true },
    ],
    questions: {
      enabled: true,
      moveToBlocked: false,
      setsRunState: "awaiting_input",
      answerClearsAwaitingInputWhenNoOpenQuestionsRemain: true,
    },
    gates: [],
  };
}

function planWorkflowMigration(req: { id: string; version: number }) {
  const target = workflowDefinitionRecord(req.id, req.version);
  if (!target) throw new Error("workflow definition not found");
  const targetStages = new Set(target.definition.stages);
  const items = state.workItems.map((item) => {
    const compatible = targetStages.has(item.stageId);
    return {
      workItemId: item.id,
      number: item.number,
      title: item.title,
      currentWorkflowId: item.workflowId,
      currentWorkflowVersion: item.workflowVersion,
      currentStageId: item.stageId,
      targetStageId: compatible ? item.stageId : "",
      compatible,
      reason: compatible ? "stage exists in target workflow" : `stage ${item.stageId} not present in target workflow`,
    };
  });
  return {
    projectId: state.project.id,
    currentId: state.project.workflow.definitionId,
    currentVersion: state.project.workflow.definitionVersion,
    targetId: target.id,
    targetVersion: target.version,
    existingItems: items.length,
    itemsPinnedToCurrentVersion: items.length,
    compatibleItems: items.filter((item) => item.compatible).length,
    incompatibleItems: items.filter((item) => !item.compatible).length,
    items,
  };
}

function dispatch(methodName: string, args: unknown[]) {
  const method = methodName.startsWith(methodPrefix) ? methodName.slice(methodPrefix.length) : methodName;

  switch (method) {
    case "PTYTraceEnabled":
      return false;
    case "LoadAppSettings":
      return {
        startupView: "sessions",
        railSide: "left",
        terminalFontSize: 13,
        terminalCursorBlink: true,
        keepDaemonAlive: true,
        hookLogEnabled: true,
        clearHookLogAfterSession: false,
        worktrunkPath: "/opt/homebrew/bin/wt",
        keybindings: {},
      };
    case "SaveAppSettings":
      return args[0];
    case "DaemonStatus":
      return {
        running: true,
        address: "http://127.0.0.1:8877",
        managed: false,
        apiVersion: 1,
        gitSha: "e2e",
        version: "e2e",
        dirty: true,
        error: "",
      };
    case "SyncSessionMenu":
      return undefined;
    case "ListSessions":
      return clone(state.sessions);
    case "ListPTYs":
      return clone(state.ptys);
    case "ListPTYHistory":
      return clone(state.ptyHistory);
    case "ReadPTYHistory":
      return { ptyId: args[0], title: "Seeded PTY history", entries: [] };
    case "ListPlugins":
    case "ListRegistryPlugins":
    case "ListAgentHookIntegrations":
    case "ListAgentBridgeApprovals":
    case "ListAgentBridgeEvents":
      return [];
    case "AgentHookLogStatus":
      return clone(agentHookLogStatus);
    case "SetAgentHookLogSettings": {
      const request = (args[0] ?? {}) as { enabled?: boolean; clearAfterSession?: boolean };
      agentHookLogStatus = {
        ...agentHookLogStatus,
        enabled: request.enabled ?? agentHookLogStatus.enabled,
        clearAfterSession: request.clearAfterSession ?? agentHookLogStatus.clearAfterSession,
      };
      return clone(agentHookLogStatus);
    }
    case "ClearAgentHookLog":
      agentHookLogStatus = { ...agentHookLogStatus, sizeBytes: 0 };
      return clone(agentHookLogStatus);
    case "OpenAgentHookLog":
      return clone(agentHookLogStatus);
    case "OnboardingStatus":
      return { localDaemon: true, items: [] };
    case "ListAgentProfiles":
      return [
        { id: "agent_codex", name: "Codex", provider: "codex", command: "codex" },
        { id: "agent_claude", name: "Claude", provider: "claude", command: "claude" },
      ];
    case "ListProjects":
      return clone(state.projects);
    case "ListWorkflowDefinitions":
      return clone(state.workflowDefinitions);
    case "ListWorkItemWorkflowActions":
      return clone(listWorkItemWorkflowActions(args[0] as string));
    case "ValidateWorkflowDefinition":
      return validationReportForDefinition((args[0] as any)?.definition);
    case "ValidateWorkflowDefinitionFile": {
      const path = String(((args[0] as any)?.path ?? ""));
      return validationReportForDefinition(workflowDefinitionFromPath(path));
    }
    case "ImportWorkflowDefinitionFile": {
      const path = String(((args[0] as any)?.path ?? ""));
      const definition = workflowDefinitionFromPath(path);
      const report = validationReportForDefinition(definition);
      if (!report.valid) throw new Error(report.errors[0]?.message || "invalid workflow definition");
      const record = {
        id: definition.id,
        version: definition.version,
        source: "file",
        sourcePath: path,
        contentHash: `e2e-${definition.id}`,
        definition,
        createdAt: state.now,
        updatedAt: state.now,
      };
      state.workflowDefinitions = [
        ...state.workflowDefinitions.filter((candidate) => candidate.id !== record.id || candidate.version !== record.version),
        record,
      ];
      return clone(record);
    }
    case "ExportWorkflowDefinitionFile":
      return undefined;
    case "DeleteWorkflowDefinition": {
      const id = args[0] as string;
      const version = args[1] as number;
      if (state.project.workflow.definitionId === id && state.project.workflow.definitionVersion === version) {
        throw new Error("workflow definition is active for project");
      }
      if (state.workItems.some((item) => item.workflowId === id && item.workflowVersion === version)) {
        throw new Error("workflow definition is used by work item");
      }
      const record = workflowDefinitionRecord(id, version);
      if (!record) throw new Error("workflow definition not found");
      state.workflowDefinitions = state.workflowDefinitions.filter(
        (candidate) => candidate.id !== id || candidate.version !== version,
      );
      return clone(record);
    }
    case "PlanProjectWorkflowMigration":
      return planWorkflowMigration(args[1] as { id: string; version: number });
    case "SetProjectWorkflowDefinition": {
      const projectID = args[0] as string;
      const req = args[1] as { id: string; version: number };
      const definition = state.workflowDefinitions.find(
        (candidate) => candidate.id === req.id && candidate.version === req.version,
      );
      if (!definition || projectID !== state.project.id) {
        throw new Error("workflow definition not found");
      }
      state.project = {
        ...state.project,
        workflow: workflowTemplateFromDefinition(definition),
      };
      state.projects = state.projects.map((project) => (project.id === state.project.id ? state.project : project));
      return clone(state.project);
    }
    case "ProjectDetail":
      return {
        project: clone(state.project),
        workItems: clone(state.workItems),
        sessions: clone(state.sessions),
        runs: clone(state.runs),
      };
    case "ListWorkItems":
      return clone(state.workItems);
    case "ListWorkItemLinks":
      return clone(state.workItemLinks);
    case "ReadyWork":
      return readyWorkExplanation();
    case "ListWorkItemRuns":
      return listForWorkItem(clone(state.runs), args[0]);
    case "ListArtifacts":
      return listForWorkItem(clone(state.artifacts), args[0]);
    case "ListQuestions":
      return listForWorkItem(clone(state.questions), args[0]);
    case "ListGateReports":
      return listForWorkItem(clone(state.gates), args[0]);
    case "ListWorkflowEvents":
      return listForWorkItem(clone(state.workflowEvents), args[0]);
    case "ListStatusEvents":
      return clone(state.statusEvents);
    case "ListAgentPrompts":
      return clone(state.agentPrompts);
    case "Output":
      return { ptyId: (args[0] as any)?.ptyId, fromOffset: (args[0] as any)?.fromOffset ?? 0, output: "", outputBase64: "", nextOffset: 0 };
    case "NextEvent":
      return new CancellablePromise((resolve) => {
        nextEventWaiters.push(resolve);
      });
    case "CreateWorkItem":
      return createdWorkItem(args[0]);
    case "UpdateWorkItem": {
      const req = args[0] as any;
      state.workItems = state.workItems.map((item) =>
        item.id === req.id ? { ...item, title: req.title ?? item.title, bodyMarkdown: req.bodyMarkdown ?? item.bodyMarkdown } : item,
      );
      return clone(state.workItems.find((item) => item.id === req.id));
    }
    case "MoveWorkItem": {
      const req = args[0] as any;
      state.workItems = state.workItems.map((item) =>
        item.id === req.id || item.id === req.workItemId ? { ...item, stageId: req.stageId } : item,
      );
      return clone(state.workItems.find((item) => item.id === req.id || item.id === req.workItemId));
    }
    case "StartPlanning": {
      const req = args[0] as any;
      const item = state.workItems.find((candidate) => candidate.id === req.workItemId);
      if (!item) throw new Error("work item not found");
      const run = {
        id: `run_${state.runs.length + 1}`,
        workItemId: item.id,
        projectId: item.projectId,
        preset: "reader",
        promptTemplateId: "plan",
        promptSnapshot: "Plan the work item",
        status: "running",
        createdAt: state.now,
        updatedAt: state.now,
        history: [],
      };
      state.runs = [run, ...state.runs];
      state.workItems = state.workItems.map((candidate) =>
        candidate.id === item.id ? { ...candidate, stageId: "planning", runState: "running" } : candidate,
      );
      return clone(run);
    }
    case "LaunchExecution":
    case "StartExecution":
    case "QueueExecution": {
      const req = args[0] as any;
      const item = state.workItems.find((candidate) => candidate.id === req.workItemId);
      if (!item) throw new Error("work item not found");
      const status = method === "QueueExecution" ? "queued" : "running";
      const run = {
        id: `run_${state.runs.length + 1}`,
        workItemId: item.id,
        projectId: item.projectId,
        preset: "writer",
        promptTemplateId: "implement",
        promptSnapshot: "Execute the work item",
        status,
        createdAt: state.now,
        updatedAt: state.now,
        history: [],
      };
      state.runs = [run, ...state.runs];
      state.workItems = state.workItems.map((candidate) =>
        candidate.id === item.id ? { ...candidate, stageId: "execution", runState: status } : candidate,
      );
      return clone(run);
    }
    case "CompleteExecution": {
      const req = args[0] as any;
      const item = state.workItems.find((candidate) => candidate.id === req.workItemId);
      if (!item) throw new Error("work item not found");
      state.workItems = state.workItems.map((candidate) =>
        candidate.id === item.id ? { ...candidate, stageId: "review", runState: "idle" } : candidate,
      );
      state.gates = [
        ...state.gates,
        {
          id: `gate_${state.gates.length + 1}`,
          workItemId: item.id,
          name: "Review",
          blocking: true,
          status: "pending",
          message: "",
          createdAt: state.now,
          updatedAt: state.now,
        },
      ];
      return clone(state.workItems.find((candidate) => candidate.id === item.id));
    }
    case "ApproveDone": {
      const req = args[0] as any;
      state.workItems = state.workItems.map((item) =>
        item.id === req.workItemId ? { ...item, stageId: "done", runState: "completed" } : item,
      );
      return clone(state.workItems.find((item) => item.id === req.workItemId));
    }
    case "AddWorkItemLink": {
      const req = args[0] as any;
      const source = state.workItems.find((item) => item.id === req.sourceWorkItemId);
      const target = state.workItems.find((item) => item.id === req.targetWorkItemId);
      const link = {
        id: `link_${state.workItemLinks.length + 1}`,
        projectId: source?.projectId || target?.projectId || state.project.id,
        sourceWorkItemId: req.sourceWorkItemId,
        targetWorkItemId: req.targetWorkItemId,
        type: req.type,
        createdBy: req.actor || "e2e",
        createdAt: state.now,
      };
      state.workItemLinks = [...state.workItemLinks, link];
      return clone(link);
    }
    case "CreateSession": {
      const req = args[0] as any;
      const session = {
        id: `sess_${state.sessions.length + 1}`,
        projectId: req.projectId || "",
        name: req.name || "Created Session",
        rootDir: req.rootDir || "/tmp/whisk-e2e",
        windows: {},
        panes: {},
      };
      state.sessions = [...state.sessions, session];
      return { session, paneId: "", ptyId: "" };
    }
    case "CloseSession": {
      const req = args[0] as any;
      state.sessions = state.sessions.filter((session) => session.id !== req.sessionId);
      return clone(state.sessions);
    }
    case "CreateProject": {
      const req = args[0] as any;
      const project = {
        ...state.project,
        id: `proj_${state.projects.length + 1}`,
        name: req.name,
        rootDir: req.rootDir,
        slug: String(req.name || "project").toLowerCase().replace(/[^a-z0-9]+/g, "-"),
      };
      state.projects = [...state.projects, project];
      return clone(project);
    }
    case "MarkStatusEventRead": {
      const req = args[0] as any;
      state.statusEvents = state.statusEvents.filter((event) => event.id !== req.id);
      return { id: req.id, status: "read" };
    }
    case "ResolveAgentPrompt": {
      const id = args[0] as string;
      state.agentPrompts = state.agentPrompts.filter((prompt) => prompt.id !== id);
      return { id, status: "resolved" };
    }
    case "MarkAgentBridgeEventRead":
    case "LogPTYTrace":
    case "ResizePTY":
    case "WritePTY":
      return undefined;
    default:
      throw new Error(`Unhandled Wails service call in E2E fake: ${methodName}`);
  }
}

export class CancellablePromise<T> extends Promise<T> {
  cancel(): CancellablePromise<void> {
    return CancellablePromise.resolve();
  }

  cancelOn(): CancellablePromise<T> {
    return this;
  }

  static resolve(): CancellablePromise<void>;
  static resolve<T>(value: T | PromiseLike<T>): CancellablePromise<Awaited<T>>;
  static resolve<T>(value?: T | PromiseLike<T>) {
    return new CancellablePromise<Awaited<T>>((resolve) => resolve(value as Awaited<T>));
  }

  static reject<T = never>(reason?: unknown) {
    return new CancellablePromise<T>((_, reject) => reject(reason));
  }
}

export const Call = {
  ByName(methodName: string, ...args: unknown[]) {
    calls.push({ method: methodName, args: clone(args) });
    try {
      return CancellablePromise.resolve(dispatch(methodName, args));
    } catch (error) {
      return CancellablePromise.reject(error);
    }
  },
};

export const Create = {
  Any(value: unknown) {
    return value;
  },
  Array<T>(createItem: (source: unknown) => T) {
    return (source: unknown = []) => (Array.isArray(source) ? source.map(createItem) : []);
  },
  Nullable<T>(createValue: (source: unknown) => T) {
    return (source: unknown) => (source == null ? null : createValue(source));
  },
  Map<T>(_: (source: unknown) => unknown, createValue: (source: unknown) => T) {
    return (source: unknown = {}) =>
      Object.fromEntries(Object.entries((source ?? {}) as Record<string, unknown>).map(([key, value]) => [key, createValue(value)]));
  },
};

export const Events = {
  On(eventName: string, callback: Listener) {
    const eventListeners = listeners.get(eventName) ?? new Set<Listener>();
    eventListeners.add(callback);
    listeners.set(eventName, eventListeners);
    return () => eventListeners.delete(callback);
  },
};

export const Dialogs = {
  async OpenFile() {
    return "/tmp/whisk-e2e";
  },
};

export const Browser = {
  async OpenURL(url: string | URL) {
    openedURLs.push(url.toString());
  },
};

function emitWailsEvent(name: string, data: unknown) {
  for (const listener of listeners.get(name) ?? []) {
    listener({ name, data });
  }
}

function emitRuntimeEvent(event: unknown) {
  const waiter = nextEventWaiters.shift();
  if (waiter) waiter(event);
}

function reset() {
  state = seedState();
  calls = [];
  openedURLs = [];
}

declare global {
  interface Window {
    __WHISK_E2E__: {
      calls: () => Array<{ method: string; args: unknown[] }>;
      openedURLs: () => string[];
      reset: () => void;
      emit: (event: unknown) => void;
      emitCommand: (commandID: string) => void;
    };
  }
}

if (typeof window !== "undefined") {
  window.__WHISK_E2E__ = {
    calls: () => clone(calls),
    openedURLs: () => clone(openedURLs),
    reset,
    emit: emitRuntimeEvent,
    emitCommand: (commandID: string) => emitWailsEvent("command:run", commandID),
  };
}
