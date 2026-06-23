<script lang="ts">
  import { Dialogs } from "@wailsio/runtime";
  import ArrowLeft from "@lucide/svelte/icons/arrow-left";
  import ArrowRight from "@lucide/svelte/icons/arrow-right";
  import ClipboardCheck from "@lucide/svelte/icons/clipboard-check";
  import Clock3 from "@lucide/svelte/icons/clock-3";
  import FileText from "@lucide/svelte/icons/file-text";
  import GitBranch from "@lucide/svelte/icons/git-branch";
  import History from "@lucide/svelte/icons/history";
  import MoreHorizontal from "@lucide/svelte/icons/ellipsis";
  import Paperclip from "@lucide/svelte/icons/paperclip";
  import Play from "@lucide/svelte/icons/play";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import Search from "@lucide/svelte/icons/search";
  import Square from "@lucide/svelte/icons/square";
  import SquareTerminal from "@lucide/svelte/icons/square-terminal";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import X from "@lucide/svelte/icons/x";
  import type { WorkflowStage } from "../bindings/github.com/phin-tech/whisk/internal/domain/workitem/models";
  import type {
    AgentProfile,
    Artifact,
    GateReport,
    Project,
    Question,
    WorkItem,
    WorkItemRun,
    WorkflowEvent,
  } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import {
    adjacentStageTargets,
    canMoveToStage,
    collapsedStageStorageKey,
    deriveNextStep,
    deriveWorkItemAttention,
    groupWorkItemsByStage,
    parseCollapsedStages,
    selectDetailRun,
    serializeCollapsedStages,
  } from "./workView";
  import type { NextStepView } from "./workView";

  export let projects: Project[] = [];
  export let workItems: WorkItem[] = [];
  export let workItemRuns: WorkItemRun[] = [];
  export let artifacts: Artifact[] = [];
  export let questions: Question[] = [];
  export let gateReports: GateReport[] = [];
  export let workflowEvents: WorkflowEvent[] = [];
  export let agentProfiles: AgentProfile[] = [];
  export let activeProjectId = "";
  export let filterQuery = "";
  export let filterStageId = "";
  export let filterRunState = "";
  export let loading = false;
  export let onRefresh: () => void;
  export let onCreateWorkItem: (request: {
    projectId: string;
    title: string;
    bodyMarkdown: string;
  }) => void;
  export let onUpdateWorkItem: (request: {
    id: string;
    title: string;
    bodyMarkdown: string;
  }) => void;
  export let onMoveWorkItem: (workItemId: string, stageId: string) => void;
  export let onGenerateWorktree: (request: { workItemId: string; branch: string }) => void;
  export let onAttachFile: (workItemId: string, path: string) => void;
  export let onDeleteWorkItem: (workItemId: string) => void;
  export let onCancelRun: (runId: string) => void;
  export let onLaunchRun: (runId: string, agentProfileId?: string) => void;
  export let onOpenRunTerminal: (run: WorkItemRun) => void;
  export let onStartPlanning: (workItemId: string) => void;
  export let onSubmitPlan: (request: {
    workItemId: string;
    runId: string;
    title: string;
    body: string;
  }) => void;
  export let onApprovePlan: (workItemId: string, artifactId: string) => void;
  export let onQueueExecution: (workItemId: string) => void;
  export let onLaunchExecution: (workItemId: string, agentProfileId?: string) => void;
  export let onSetPhaseAgent: (projectId: string, preset: string, agentProfileId: string) => void;
  export let onSetInteractiveAgentShell: (projectId: string, enabled: boolean) => void;
  export let onCompleteExecution: (request: {
    workItemId: string;
    runId: string;
    message: string;
  }) => void;
  export let onSubmitReviewFeedback: (request: {
    workItemId: string;
    runId: string;
    body: string;
  }) => void;
  export let onAskQuestion: (request: { workItemId: string; runId: string; prompt: string }) => void;
  export let onAnswerQuestion: (questionId: string, answer: string) => void;
  export let onCompleteGate: (request: { id: string; status: string; overrideReason: string }) => void;
  export let onApproveDone: (workItemId: string, reason: string) => void;

  let newItemTitle = "";
  let newItemBody = "";
  let worktreeBranches: Record<string, string> = {};
  let planBodies: Record<string, string> = {};
  let feedbackBodies: Record<string, string> = {};
  let questionPrompts: Record<string, string> = {};
  let questionAnswers: Record<string, string> = {};
  let gateOverrideReasons: Record<string, string> = {};
  let doneReasons: Record<string, string> = {};
  let agentSelections: Record<string, string> = {};
  let detailItemId = "";
  let detailDraftItemId = "";
  let detailTitle = "";
  let detailBody = "";
  let createBodyOpen = false;
  let collapsedStageIds = new Set<string>();
  let collapsedProjectId = "";

  $: activeProject = projects.find((project) => project.id === activeProjectId) ?? null;
  $: stages = activeProject?.workflow?.stages ?? [];
  $: filteredWorkItems = filterWorkItems(workItems);
  $: boardStages = filterStageId ? stages.filter((stage) => stage.id === filterStageId) : stages;
  $: itemsByStage = groupWorkItemsByStage(filteredWorkItems, stages);
  $: detailItem = workItems.find((item) => item.id === detailItemId) ?? null;
  $: detailDirty = Boolean(
    detailItem && (detailTitle !== detailItem.title || detailBody !== detailItem.bodyMarkdown),
  );
  $: runsByItem = groupRunsByItem(workItemRuns);
  $: detailRuns = detailItem ? (runsByItem[detailItem.id] ?? []) : [];
  $: detailArtifacts = detailItem ? artifacts.filter((artifact) => artifact.workItemId === detailItem.id) : [];
  $: detailQuestions = detailItem ? questions.filter((question) => question.workItemId === detailItem.id) : [];
  $: detailGates = detailItem ? gateReports.filter((gate) => gate.workItemId === detailItem.id) : [];
  $: detailWorkflowEvents = detailItem ? workflowEvents.filter((event) => event.workItemId === detailItem.id) : [];
  $: detailLatestRun = detailRuns[0] ?? null;
  $: detailCurrentRun = selectDetailRun(detailRuns);
  $: detailPastRuns = detailCurrentRun
    ? detailRuns.filter((run) => run.id !== detailCurrentRun.id)
    : detailRuns;
  $: latestDraftPlan = [...detailArtifacts]
    .reverse()
    .find((artifact) => artifact.kind === "plan" && artifact.status === "draft");
  $: approvedPlan = [...detailArtifacts]
    .reverse()
    .find((artifact) => artifact.kind === "plan" && artifact.status === "approved");
  $: planArtifacts = [...detailArtifacts]
    .filter((artifact) => artifact.kind === "plan")
    .sort((a, b) => timestamp(b.updatedAt || b.createdAt) - timestamp(a.updatedAt || a.createdAt));

  // Agent selection: the preset a launch will run under (current run's preset, else the
  // execution stage default), and the project's remembered default agent for that preset.
  $: executionStage = stages.find((stage) => stage.kind === "execution" || stage.id === "execution") ?? null;
  $: launchPreset = detailCurrentRun?.preset || executionStage?.defaultRunPreset || "writer";
  $: phaseAgentDefault = activeProject?.preferences?.defaultPhaseAgents?.[launchPreset] ?? "";
  $: selectedAgentId = agentSelections[launchPreset] ?? phaseAgentDefault;

  // Contextual "what's next" for the open item: a single sentence plus the primary action.
  $: nextStep = detailItem
    ? computeNextStep(detailItem, detailCurrentRun, detailLatestRun, approvedPlan, latestDraftPlan)
    : null;
  $: if (activeProjectId !== collapsedProjectId) {
    collapsedProjectId = activeProjectId;
    collapsedStageIds =
      typeof localStorage === "undefined"
        ? new Set<string>()
        : parseCollapsedStages(localStorage.getItem(collapsedStageStorageKey(activeProjectId)));
  }

  function slugify(value: string) {
    return value
      .toLowerCase()
      .trim()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-+|-+$/g, "");
  }

  function defaultWorktreeBranch(item: WorkItem) {
    const projectSlug = activeProject?.slug || "work";
    const itemSlug = slugify(item.title) || "item";
    return `whisk/${projectSlug}-${item.number}-${itemSlug}`;
  }

  function groupRunsByItem(runs: WorkItemRun[]) {
    const result: Record<string, WorkItemRun[]> = {};
    for (const run of runs) {
      if (!result[run.workItemId]) result[run.workItemId] = [];
      result[run.workItemId].push(run);
    }
    for (const itemRuns of Object.values(result)) {
      itemRuns.sort((a, b) => timestamp(b.createdAt) - timestamp(a.createdAt));
    }
    return result;
  }

  function filterWorkItems(items: WorkItem[]) {
    const query = filterQuery.trim().toLowerCase();
    return items.filter((item) => {
      if (filterStageId && item.stageId !== filterStageId) return false;
      if (filterRunState && (item.runState || "idle") !== filterRunState) return false;
      if (!query) return true;
      return `#${item.number} ${item.title} ${item.bodyMarkdown} ${item.stageId} ${item.runState}`
        .toLowerCase()
        .includes(query);
    });
  }

  function timestamp(value: unknown) {
    if (!value) return 0;
    if (value instanceof Date) return value.getTime();
    return new Date(String(value)).getTime() || 0;
  }

  function formattedTime(value: unknown) {
    if (!value) return "";
    if (value instanceof Date) return value.toLocaleString();
    const parsed = new Date(String(value));
    return Number.isNaN(parsed.getTime()) ? String(value) : parsed.toLocaleString();
  }

  function canCancelRun(run: WorkItemRun) {
    return run.status === "queued" || run.status === "running" || run.status === "awaiting_input";
  }

  function canOpenRunTerminal(run: WorkItemRun | null) {
    return Boolean(run?.sessionId || run?.ptyId);
  }

  function hasApprovedPlan(itemId: string) {
    return artifacts.some(
      (artifact) =>
        artifact.workItemId === itemId && artifact.kind === "plan" && artifact.status === "approved",
    );
  }

  function hasActiveRun(run: WorkItemRun | null) {
    return run?.status === "queued" || run?.status === "running" || run?.status === "awaiting_input";
  }

  function canQueueOrLaunchExecution(item: WorkItem, latestRun: WorkItemRun | null) {
    return item.stageId === "ready" && hasApprovedPlan(item.id) && !hasActiveRun(latestRun);
  }

  function openRunTerminal(run: WorkItemRun) {
    if (!canOpenRunTerminal(run)) return;
    closeDetail();
    onOpenRunTerminal(run);
  }

  function runStatusClass(status: string) {
    if (status === "running" || status === "awaiting_input") {
      return "border-green/35 bg-green/10 text-green";
    }
    if (status === "queued") return "border-blue/35 bg-blue/10 text-blue";
    if (status === "failed" || status === "cancelled") return "border-red/35 bg-red/10 text-red";
    return "border-border-subtle bg-bg-surface/50 text-text-secondary";
  }

  function stageRequiresPlan(stage: WorkflowStage) {
    const value = `${stage.id} ${stage.name}`.toLowerCase();
    return value.includes("execution") || value.includes("review") || value.includes("done");
  }

  function attentionFor(item: WorkItem, stage: WorkflowStage) {
    return deriveWorkItemAttention(item, {
      runs: runsByItem[item.id] ?? [],
      questions: detailRecordsForItem(questions, item.id),
      gates: detailRecordsForItem(gateReports, item.id),
      artifacts: detailRecordsForItem(artifacts, item.id),
      stageRequiresWorktree: Boolean(stage.provisionWorktree),
      stageRequiresPlan: stageRequiresPlan(stage),
    });
  }

  function detailRecordsForItem<T extends { workItemId: string }>(records: T[], itemId: string) {
    return records.filter((record) => record.workItemId === itemId);
  }

  function attentionToneClass(tone: string) {
    if (tone === "danger") return "border-red/40 bg-red/12 text-red";
    if (tone === "warning") return "border-amber/45 bg-amber/12 text-amber";
    if (tone === "success") return "border-green/40 bg-green/12 text-green";
    return "border-blue/40 bg-blue/12 text-blue";
  }

  function cardRailClass(severity: string) {
    if (severity === "danger") return "bg-red";
    if (severity === "warning") return "bg-amber";
    if (severity === "info") return "bg-blue";
    return "bg-border";
  }

  function hasStageAttention(stage: WorkflowStage) {
    return (itemsByStage[stage.id] ?? []).some(
      (item) => attentionFor(item, stage).severity !== "none",
    );
  }

  function stageAttentionClass(stage: WorkflowStage) {
    const severities = (itemsByStage[stage.id] ?? []).map((item) => attentionFor(item, stage).severity);
    if (severities.includes("danger")) return "bg-red";
    if (severities.includes("warning")) return "bg-amber";
    if (severities.includes("info")) return "bg-blue";
    return "bg-border";
  }

  function toggleStageCollapsed(stageId: string) {
    const next = new Set(collapsedStageIds);
    if (next.has(stageId)) {
      next.delete(stageId);
    } else {
      next.add(stageId);
    }
    collapsedStageIds = next;
    if (typeof localStorage !== "undefined" && activeProjectId) {
      localStorage.setItem(collapsedStageStorageKey(activeProjectId), serializeCollapsedStages(next));
    }
  }

  function setAllColumnsCollapsed(collapsed: boolean) {
    const next = collapsed ? new Set(boardStages.map((stage) => stage.id)) : new Set<string>();
    collapsedStageIds = next;
    if (typeof localStorage !== "undefined" && activeProjectId) {
      localStorage.setItem(collapsedStageStorageKey(activeProjectId), serializeCollapsedStages(next));
    }
  }

  function stageLabel(stageId: string) {
    return stages.find((stage) => stage.id === stageId)?.name ?? stageId;
  }

  function submitPlan(item: WorkItem) {
    const body = (planBodies[item.id] ?? "").trim();
    if (!body || loading) return;
    onSubmitPlan({
      workItemId: item.id,
      runId: detailLatestRun?.id ?? "",
      title: "Plan",
      body,
    });
    planBodies = { ...planBodies, [item.id]: "" };
  }

  function submitFeedback(item: WorkItem) {
    const body = (feedbackBodies[item.id] ?? "").trim();
    if (!body || loading) return;
    onSubmitReviewFeedback({
      workItemId: item.id,
      runId: detailLatestRun?.id ?? "",
      body,
    });
    feedbackBodies = { ...feedbackBodies, [item.id]: "" };
  }

  function askWorkflowQuestion(item: WorkItem) {
    const prompt = (questionPrompts[item.id] ?? "").trim();
    if (!prompt || loading) return;
    onAskQuestion({
      workItemId: item.id,
      runId: detailLatestRun?.id ?? "",
      prompt,
    });
    questionPrompts = { ...questionPrompts, [item.id]: "" };
  }

  function answerWorkflowQuestion(question: Question) {
    const answer = (questionAnswers[question.id] ?? "").trim();
    if (!answer || loading) return;
    onAnswerQuestion(question.id, answer);
    questionAnswers = { ...questionAnswers, [question.id]: "" };
  }

  function completeGate(gate: GateReport, status: string) {
    const overrideReason = (gateOverrideReasons[gate.id] ?? "").trim();
    if (loading) return;
    onCompleteGate({ id: gate.id, status, overrideReason });
    if (status === "overridden") gateOverrideReasons = { ...gateOverrideReasons, [gate.id]: "" };
  }

  function approveDone(item: WorkItem) {
    const reason = (doneReasons[item.id] ?? "").trim();
    if (loading) return;
    onApproveDone(item.id, reason);
  }

  function createWorkItem() {
    if (!activeProject || !newItemTitle.trim() || loading) return;
    onCreateWorkItem({
      projectId: activeProject.id,
      title: newItemTitle.trim(),
      bodyMarkdown: newItemBody.trim(),
    });
    newItemTitle = "";
    newItemBody = "";
  }

  function generateWorktree(item: WorkItem) {
    const branch = (worktreeBranches[item.id] || defaultWorktreeBranch(item)).trim();
    if (!branch || loading) return;
    onGenerateWorktree({ workItemId: item.id, branch });
  }

  async function attachFile(item: WorkItem) {
    const selected = await Dialogs.OpenFile({
      Title: "Attach file",
      ButtonText: "Attach",
      Directory: activeProject?.rootDir || undefined,
      CanChooseDirectories: false,
      CanChooseFiles: true,
      AllowsMultipleSelection: false,
    });
    if (typeof selected === "string" && selected.length > 0) {
      onAttachFile(item.id, selected);
    }
  }

  function deleteWorkItem(item: WorkItem) {
    if (loading) return;
    if (window.confirm(`Delete #${item.number} ${item.title}?`)) {
      onDeleteWorkItem(item.id);
      if (detailItemId === item.id) detailItemId = "";
    }
  }

  function openDetail(item: WorkItem) {
    detailItemId = item.id;
    detailDraftItemId = item.id;
    detailTitle = item.title;
    detailBody = item.bodyMarkdown;
  }

  function resetDetailDraft() {
    if (!detailItem) return;
    detailDraftItemId = detailItem.id;
    detailTitle = detailItem.title;
    detailBody = detailItem.bodyMarkdown;
  }

  function saveDetail() {
    if (!detailItem || loading || !detailTitle.trim()) return;
    detailTitle = detailTitle.trim();
    onUpdateWorkItem({
      id: detailItem.id,
      title: detailTitle,
      bodyMarkdown: detailBody,
    });
    detailDraftItemId = detailItem.id;
  }

  function movePrevious(item: WorkItem) {
    const { previous } = adjacentStageTargets(item, stages);
    if (previous) onMoveWorkItem(item.id, previous.id);
  }

  function moveNext(item: WorkItem) {
    const { next } = adjacentStageTargets(item, stages);
    if (next) onMoveWorkItem(item.id, next.id);
  }

  type NextStep = NextStepView & { run: () => void };

  // computeNextStep derives the recommended action (pure logic in deriveNextStep) and wires
  // the matching handler closure for the detail view's primary button.
  function computeNextStep(
    item: WorkItem,
    currentRun: WorkItemRun | null,
    latestRun: WorkItemRun | null,
    approved: Artifact | undefined,
    draft: Artifact | undefined,
  ): NextStep {
    const view = deriveNextStep({
      stageId: item.stageId,
      runStatus: currentRun?.status ?? "",
      hasTerminal: canOpenRunTerminal(currentRun),
      hasApprovedPlan: Boolean(approved),
      hasDraftPlan: Boolean(draft),
      hasLatestRun: Boolean(latestRun),
    });

    const run = () => {
      switch (view.kind) {
        case "open-terminal":
          if (currentRun) openRunTerminal(currentRun);
          break;
        case "launch-run":
          if (currentRun) onLaunchRun(currentRun.id, selectedAgentId);
          break;
        case "launch-execution":
          onLaunchExecution(item.id, selectedAgentId);
          break;
        case "approve-plan":
          if (draft?.id) onApprovePlan(item.id, draft.id);
          break;
        case "start-planning":
        case "retry-planning":
          onStartPlanning(item.id);
          break;
        case "send-to-review":
          onCompleteExecution({ workItemId: item.id, runId: latestRun?.id ?? "", message: "ready for review" });
          break;
        case "mark-done":
          approveDone(item);
          break;
        default:
          break;
      }
    };

    return { ...view, run };
  }

  function nextStepToneClass(tone: NextStep["tone"]) {
    if (tone === "accent") return "border-green/40 bg-green/15 text-green hover:border-green";
    if (tone === "primary") return "border-amber/45 bg-amber/12 text-amber hover:border-amber";
    return "border-white/14 bg-white/8 text-text-primary hover:border-accent hover:text-accent";
  }

  function selectAgent(value: string) {
    if (!detailItem || !activeProject) return;
    agentSelections = { ...agentSelections, [launchPreset]: value };
    onSetPhaseAgent(activeProject.id, launchPreset, value);
  }

  function setInteractiveAgentShell(enabled: boolean) {
    if (!activeProject) return;
    onSetInteractiveAgentShell(activeProject.id, enabled);
  }

  function closeDetail() {
    detailItemId = "";
    detailDraftItemId = "";
    detailTitle = "";
    detailBody = "";
  }

  function handleKey(event: KeyboardEvent) {
    if (event.key === "Escape" && detailItemId) {
      event.preventDefault();
      closeDetail();
    }
  }
</script>

<svelte:window on:keydown={handleKey} />

<div class="flex min-h-0 flex-1 flex-col bg-[#050506]">
  <div class="flex min-h-14 shrink-0 flex-wrap items-center justify-between gap-2 border-b border-white/12 bg-[#0a0a0c] px-3 py-2">
    <div class="flex min-w-0 flex-1 items-center gap-2">
      {#if activeProject}
        <div class="min-w-0">
          <div class="truncate text-[13px] font-semibold text-text-primary">{activeProject.name}</div>
          <div class="truncate text-[12px] text-text-secondary">{activeProject.rootDir}</div>
        </div>
      {:else}
        <div class="truncate text-[13px] text-text-muted">No project selected</div>
      {/if}
    </div>

    <div class="flex min-w-0 flex-1 items-center justify-end gap-2">
      {#if activeProject}
        <div class="flex min-w-[260px] max-w-[520px] flex-1 items-center overflow-hidden rounded-md border border-white/16 bg-[#0d0d10] focus-within:border-accent-dim">
          <input
            class="h-9 min-w-0 flex-1 bg-transparent px-3 text-[14px] text-text-primary outline-none placeholder:text-text-muted"
            type="text"
            bind:value={newItemTitle}
            placeholder="Create work item"
            disabled={loading}
          />
          <button
            type="button"
            class="inline-flex h-8 w-8 shrink-0 items-center justify-center text-text-muted transition-colors hover:text-text-primary"
            aria-label={createBodyOpen ? "Hide work item body" : "Add work item body"}
            title={createBodyOpen ? "Hide body" : "Add body"}
            on:click={() => (createBodyOpen = !createBodyOpen)}
          >
            <MoreHorizontal size={16} />
          </button>
          <button
            type="button"
            class="mr-1 inline-flex h-7 shrink-0 items-center justify-center rounded border border-white/14 bg-white/8 px-2.5 text-[12px] font-semibold text-text-primary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed"
            disabled={loading || !newItemTitle.trim()}
            on:click={createWorkItem}
          >
            Create
          </button>
        </div>
      {/if}
      <button
        type="button"
        class="inline-flex h-9 w-9 items-center justify-center rounded-md border border-white/14 bg-white/6 text-text-secondary transition-colors hover:bg-white/10 hover:text-text-primary disabled:cursor-wait disabled:opacity-60"
        disabled={loading}
        aria-label="Refresh work board"
        title="Refresh work board"
        on:click={onRefresh}
      >
        <RefreshCw size={15} class={loading ? "animate-spin" : ""} />
      </button>
      <button
        type="button"
        class="hidden h-9 items-center justify-center rounded-md border border-white/14 bg-white/6 px-2.5 text-[12px] font-medium text-text-secondary transition-colors hover:bg-white/10 hover:text-text-primary sm:inline-flex"
        disabled={loading || boardStages.length === 0}
        on:click={() => setAllColumnsCollapsed(false)}
      >
        Expand
      </button>
      <button
        type="button"
        class="hidden h-9 items-center justify-center rounded-md border border-white/14 bg-white/6 px-2.5 text-[12px] font-medium text-text-secondary transition-colors hover:bg-white/10 hover:text-text-primary sm:inline-flex"
        disabled={loading || boardStages.length === 0}
        on:click={() => setAllColumnsCollapsed(true)}
      >
        Collapse
      </button>
    </div>
  </div>

  {#if activeProject}
    {#if createBodyOpen}
      <div class="shrink-0 border-b border-white/12 bg-[#080809] px-3 py-2">
        <textarea
          class="h-20 w-full resize-none rounded-md border border-white/14 bg-[#0d0d10] px-3 py-2 text-[13px] leading-5 text-text-primary outline-none placeholder:text-text-muted focus:border-accent-dim"
          bind:value={newItemBody}
          placeholder="Markdown body"
          disabled={loading}
        ></textarea>
      </div>
    {/if}

    <div class="app-scrollbar min-h-0 flex-1 overflow-auto p-3">
      <div class="flex min-h-full min-w-max items-stretch gap-3">
        {#each boardStages as stage (stage.id)}
          {@const stageItems = itemsByStage[stage.id] ?? []}
          {@const collapsed = collapsedStageIds.has(stage.id)}
          {@const stageHasAttention = hasStageAttention(stage)}
          {#if collapsed}
            <button
              type="button"
              class="flex min-h-[420px] w-12 shrink-0 flex-col items-center justify-between rounded-md border border-white/14 bg-[#0b0b0d] px-2 py-3 text-text-secondary transition-colors hover:border-white/28 hover:bg-[#101014] hover:text-text-primary"
              aria-label={`Expand ${stage.name}`}
              title={`Expand ${stage.name}`}
              on:click={() => toggleStageCollapsed(stage.id)}
            >
              <span class="writing-vertical truncate text-[13px] font-semibold">{stage.name}</span>
              <span class="flex flex-col items-center gap-2">
                {#if stageHasAttention}
                  <span class="h-2 w-2 rounded-full {stageAttentionClass(stage)}"></span>
                {/if}
                <span class="rounded border border-white/14 bg-black/40 px-1.5 py-1 font-mono text-[12px]">
                  {stageItems.length}
                </span>
              </span>
            </button>
          {:else}
            <section class="flex min-h-[420px] w-[320px] shrink-0 flex-col overflow-hidden rounded-md border border-white/14 bg-[#0b0b0d]">
              <div class="flex h-12 min-w-0 shrink-0 items-center justify-between gap-2 border-b border-white/12 px-3">
                <div class="flex min-w-0 items-center gap-2">
                  {#if stageHasAttention}
                    <span class="h-2 w-2 shrink-0 rounded-full {stageAttentionClass(stage)}"></span>
                  {/if}
                  <button
                    type="button"
                    class="min-w-0 truncate text-left text-[13px] font-semibold text-text-primary outline-none focus-visible:text-accent"
                    aria-label={`Collapse ${stage.name}`}
                    title="Double-click to collapse"
                    on:dblclick={() => toggleStageCollapsed(stage.id)}
                  >
                    {stage.name}
                  </button>
                  <div class="rounded border border-white/12 bg-black/35 px-1.5 py-0.5 font-mono text-[12px] text-text-secondary">
                    {stageItems.length}
                  </div>
                </div>
                <button
                  type="button"
                  class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded border border-transparent text-text-muted transition-colors hover:border-white/14 hover:bg-white/8 hover:text-text-primary"
                  aria-label={`Collapse ${stage.name}`}
                  title={`Collapse ${stage.name}`}
                  on:click={() => toggleStageCollapsed(stage.id)}
                >
                  <ArrowLeft size={13} />
                </button>
              </div>

              <div class="grid min-w-0 content-start gap-2 p-2">
                {#if stageItems.length === 0}
                  <div class="min-h-24 rounded-md border border-dashed border-white/16 bg-black/20 px-3 py-7 text-center text-[13px] text-text-muted">
                    Empty
                  </div>
                {:else}
                  {#each stageItems as item (item.id)}
                    {@const targets = adjacentStageTargets(item, stages)}
                    {@const latestRun = (runsByItem[item.id] ?? [])[0] ?? null}
                    {@const canExecute = canQueueOrLaunchExecution(item, latestRun)}
                    {@const attention = attentionFor(item, stage)}
                    {@const terminalRun = attention.terminalRunId
                      ? (runsByItem[item.id] ?? []).find((run) => run.id === attention.terminalRunId)
                      : null}
                    <article class="group relative min-h-[76px] overflow-hidden rounded-md border border-white/12 bg-[#111114] shadow-[0_1px_0_rgba(255,255,255,0.03)] transition-colors hover:border-white/28 hover:bg-[#151519] focus-within:border-white/28">
                      <div class="absolute inset-y-0 left-0 w-1 {cardRailClass(attention.severity)}"></div>
                      <div class="grid gap-2 px-3 py-2 pl-4">
                        <div class="flex min-w-0 items-start gap-2">
                          <button
                            type="button"
                            class="work-card-title min-w-0 flex-1 text-left text-[14px] font-semibold leading-5 text-text-primary outline-none transition-colors hover:text-accent focus-visible:text-accent"
                            on:click={() => openDetail(item)}
                          >
                            <span class="font-mono text-[12px] font-medium text-text-muted">#{item.number}</span>
                            {item.title}
                          </button>
                          {#if terminalRun}
                            <button
                              type="button"
                              aria-label="Open running terminal"
                              title="Open running terminal"
                              class="inline-flex h-7 w-7 shrink-0 animate-pulse items-center justify-center rounded border border-green/45 bg-green/12 text-green shadow-[0_0_0_1px_rgba(56,211,159,0.08),0_0_18px_rgba(56,211,159,0.16)] transition-colors hover:border-green disabled:cursor-not-allowed"
                              disabled={loading}
                              on:click={() => openRunTerminal(terminalRun)}
                            >
                              <SquareTerminal size={14} />
                            </button>
                          {/if}
                          <div class="flex shrink-0 gap-1 opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100">
                            {#if latestRun?.status === "queued"}
                              <button
                                type="button"
                                aria-label="Launch queued run"
                                title="Launch queued run"
                                class="inline-flex h-7 w-7 items-center justify-center rounded border border-blue/35 bg-blue/10 text-blue transition-colors hover:border-blue disabled:cursor-not-allowed"
                                disabled={loading}
                                on:click={() => onLaunchRun(latestRun.id)}
                              >
                                <Play size={13} />
                              </button>
                            {/if}
                            {#if canExecute}
                              <button
                                type="button"
                                aria-label="Queue execution"
                                title="Queue execution"
                                class="inline-flex h-7 w-7 items-center justify-center rounded border border-blue/35 bg-blue/10 text-blue transition-colors hover:border-blue disabled:cursor-not-allowed"
                                disabled={loading}
                                on:click={() => onQueueExecution(item.id)}
                              >
                                <Clock3 size={13} />
                              </button>
                              <button
                                type="button"
                                aria-label="Launch execution"
                                title="Launch execution"
                                class="inline-flex h-7 w-7 items-center justify-center rounded border border-green/35 bg-green/10 text-green transition-colors hover:border-green disabled:cursor-not-allowed"
                                disabled={loading}
                                on:click={() => onLaunchExecution(item.id)}
                              >
                                <Play size={13} />
                              </button>
                            {/if}
                            {#if targets.blockedNext && !item.worktree}
                              <button
                                type="button"
                                aria-label="Generate worktree"
                                title="Generate worktree"
                                class="inline-flex h-7 w-7 items-center justify-center rounded border border-white/14 bg-white/6 text-text-secondary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed"
                                disabled={loading}
                                on:click={() => generateWorktree(item)}
                              >
                                <GitBranch size={13} />
                              </button>
                            {/if}
                            <button
                              type="button"
                              aria-label="Move previous"
                              title="Move previous"
                              class="inline-flex h-7 w-7 items-center justify-center rounded border border-white/14 bg-white/6 text-text-secondary transition-colors hover:bg-white/10 hover:text-text-primary disabled:cursor-not-allowed disabled:opacity-40"
                              disabled={loading || !targets.previous}
                              on:click={() => movePrevious(item)}
                            >
                              <ArrowLeft size={13} />
                            </button>
                            <button
                              type="button"
                              aria-label={targets.blockedNext ? "Generate worktree before moving" : "Move next"}
                              title={targets.blockedNext ? "Generate worktree before moving" : "Move next"}
                              class="inline-flex h-7 w-7 items-center justify-center rounded border border-white/14 bg-white/6 text-text-secondary transition-colors hover:bg-white/10 hover:text-text-primary disabled:cursor-not-allowed disabled:opacity-40"
                              disabled={loading || !targets.next}
                              on:click={() => moveNext(item)}
                            >
                              <ArrowRight size={13} />
                            </button>
                          </div>
                        </div>

                        <div class="flex min-w-0 flex-wrap items-center gap-1.5">
                          {#if attention.signals.length > 0}
                            {#each attention.signals as signal (signal.id)}
                              <span class="rounded border px-1.5 py-0.5 text-[12px] font-medium {attentionToneClass(signal.tone)}">
                                {signal.label}
                              </span>
                            {/each}
                          {:else}
                            <span class="text-[12px] text-text-muted">
                              {item.runState || "Idle"}
                            </span>
                          {/if}
                        </div>
                      </div>
                    </article>
                  {/each}
                {/if}
              </div>
            </section>
          {/if}
        {/each}
      </div>
    </div>
  {:else}
    <div class="flex min-h-0 flex-1 items-center justify-center p-6">
      <div class="grid max-w-sm gap-3 text-center">
        <div class="text-base font-semibold text-text-primary">No project selected</div>
        <div class="text-[13px] text-text-muted">Select or create a project from the sidebar.</div>
      </div>
    </div>
  {/if}
</div>

{#if detailItem && activeProject}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 px-4 py-6 backdrop-blur-sm"
    role="dialog"
    aria-modal="true"
    aria-label="Work item editor"
  >
    <div class="flex max-h-[92vh] w-full max-w-[1180px] flex-col overflow-hidden rounded-md border border-white/16 bg-[#08080a] shadow-[0_28px_90px_rgba(0,0,0,0.7)]">
      <div class="shrink-0 border-b border-white/12 bg-[#0b0b0d] px-5 py-4">
        <div class="flex items-start justify-between gap-4">
          <div class="min-w-0">
            <div class="mb-2 flex min-w-0 flex-wrap items-center gap-2">
              <span class="rounded border border-white/14 bg-white/6 px-2 py-1 font-mono text-[13px] text-text-secondary">
                #{detailItem.number}
              </span>
              <span class="rounded border border-white/14 bg-white/6 px-2 py-1 text-[13px] font-medium text-text-secondary">
                {stageLabel(detailItem.stageId)}
              </span>
              <span class="rounded border px-2 py-1 font-mono text-[12px] font-semibold {runStatusClass(detailCurrentRun?.status || detailItem.runState || 'idle')}">
                {detailCurrentRun?.status || detailItem.runState || "idle"}
              </span>
            </div>
            <input
              class="max-w-[860px] bg-transparent text-[20px] font-semibold leading-7 text-text-primary outline-none focus:text-accent"
              type="text"
              value={detailDraftItemId === detailItem.id ? detailTitle : detailItem.title}
              disabled={loading}
              aria-label="Work item title"
              on:input={(event) => (detailTitle = event.currentTarget.value)}
            />
            <div class="mt-2 truncate text-[13px] text-text-muted">
              {activeProject.name}
            </div>
          </div>
          <button
            type="button"
            aria-label="Close"
            class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-white/14 bg-white/6 text-text-secondary transition-colors hover:bg-white/10 hover:text-text-primary focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50"
            on:click={closeDetail}
          >
            <X size={16} />
          </button>
        </div>
      </div>

      <div class="app-scrollbar min-h-0 flex-1 overflow-y-auto px-5 py-5">
        <div class="grid gap-5 xl:grid-cols-[minmax(0,1fr)_360px] xl:items-start">
          {#if nextStep}
            <div class="flex flex-wrap items-center justify-between gap-3 rounded-md border border-white/12 bg-[#0d0d10] px-4 py-3 xl:col-span-2">
              <div class="flex min-w-0 items-center gap-3">
                <span class="shrink-0 text-[11px] font-semibold uppercase tracking-widest text-text-muted">Next</span>
                <span class="min-w-0 text-[14px] leading-5 text-text-primary">{nextStep.message}</span>
              </div>
              <div class="flex shrink-0 flex-wrap items-center gap-2">
                {#if nextStep.isLaunch}
                  <label class="flex items-center gap-1.5 text-[12px] text-text-muted">
                    <span>Agent</span>
                    <select
                      class="h-9 rounded-md border border-white/14 bg-black/30 px-2 text-[13px] text-text-primary outline-none focus:border-accent-dim disabled:opacity-60"
                      value={selectedAgentId}
                      disabled={loading}
                      aria-label="Agent profile"
                      on:change={(event) => selectAgent(event.currentTarget.value)}
                    >
                      <option value="">Default agent</option>
                      {#each agentProfiles as profile (profile.id)}
                        <option value={profile.id}>{profile.label}</option>
                      {/each}
                    </select>
                  </label>
                  <label class="flex h-9 items-center gap-2 rounded-md border border-white/14 bg-black/20 px-2 text-[12px] text-text-muted">
                    <input
                      type="checkbox"
                      class="h-4 w-4 accent-accent"
                      checked={Boolean(activeProject.preferences?.useInteractiveAgentShell)}
                      disabled={loading}
                      aria-label="Use interactive shell for agent runs"
                      on:change={(event) => setInteractiveAgentShell(event.currentTarget.checked)}
                    />
                    <span>Shell</span>
                  </label>
                {/if}
                {#if nextStep.label}
                  <button
                    type="button"
                    class="inline-flex h-9 items-center justify-center gap-2 rounded-md border px-4 text-[13px] font-semibold transition-colors disabled:cursor-not-allowed disabled:opacity-60 {nextStepToneClass(nextStep.tone)}"
                    disabled={loading}
                    on:click={() => nextStep?.run()}
                  >
                    {nextStep.label}
                  </button>
                {/if}
              </div>
            </div>
          {/if}

          <main class="grid min-w-0 gap-5">
            <section class="grid gap-2">
              <div class="flex items-center justify-between gap-2">
                <h3 class="text-[13px] font-semibold uppercase tracking-wide text-text-muted">
                  Description
                </h3>
                <div class="flex items-center gap-2">
                  <button
                    type="button"
                    class="h-8 rounded-md border border-white/14 bg-white/6 px-3 text-[12px] font-medium text-text-secondary transition-colors hover:bg-white/10 hover:text-text-primary disabled:cursor-not-allowed disabled:opacity-50"
                    disabled={loading || !detailDirty}
                    on:click={resetDetailDraft}
                  >
                    Cancel
                  </button>
                  <button
                    type="button"
                    class="h-8 rounded-md border border-accent-dim bg-accent/15 px-3 text-[12px] font-semibold text-accent transition-colors hover:border-accent disabled:cursor-not-allowed disabled:opacity-50"
                    disabled={loading || !detailDirty || !detailTitle.trim()}
                    on:click={saveDetail}
                  >
                    Save
                  </button>
                </div>
              </div>
              <textarea
                class="min-h-36 resize-y rounded-md border border-white/12 bg-[#0d0d10] px-4 py-3 text-[14px] leading-6 text-text-primary outline-none placeholder:text-text-muted focus:border-accent-dim"
                value={detailDraftItemId === detailItem.id ? detailBody : detailItem.bodyMarkdown}
                disabled={loading}
                aria-label="Work item description"
                placeholder="No body."
                on:input={(event) => (detailBody = event.currentTarget.value)}
              ></textarea>
            </section>

            <section class="grid gap-3">
              <div class="flex min-w-0 items-center justify-between gap-3">
                <h3 class="text-[13px] font-semibold uppercase tracking-wide text-text-muted">
                  Plan
                </h3>
                {#if approvedPlan}
                  <span class="rounded border border-green/35 bg-green/10 px-2 py-1 text-[12px] font-semibold text-green">
                    Approved
                  </span>
                {:else if latestDraftPlan}
                  <span class="rounded border border-amber/40 bg-amber/10 px-2 py-1 text-[12px] font-semibold text-amber">
                    Draft ready
                  </span>
                {/if}
              </div>

              <div class="grid gap-2">
                {#each planArtifacts as artifact (artifact.id)}
                  <article
                    class="grid gap-3 rounded-md border p-3 {artifact.status === 'draft'
                      ? 'border-amber/35 bg-amber/8'
                      : 'border-green/30 bg-green/8'}"
                  >
                    <div class="flex min-w-0 items-start justify-between gap-3">
                      <div class="min-w-0">
                        <div class="flex min-w-0 items-center gap-2 text-[14px] font-semibold text-text-primary">
                          <FileText size={16} class="shrink-0" />
                          <span class="truncate">{artifact.title || "Plan"}</span>
                          <span class="shrink-0 rounded border border-white/16 bg-black/25 px-1.5 py-0.5 font-mono text-[11px] uppercase text-text-secondary">
                            {artifact.status}
                          </span>
                        </div>
                        <div class="mt-1 truncate text-[12px] text-text-muted">
                          {formattedTime(artifact.updatedAt || artifact.createdAt)}
                        </div>
                      </div>
                      {#if artifact.status === "draft"}
                        <button
                          type="button"
                          class="shrink-0 rounded-md border border-amber/40 bg-black/30 px-3 py-1.5 text-[13px] font-medium text-amber transition-colors hover:border-amber hover:bg-amber/15 disabled:cursor-not-allowed disabled:opacity-60"
                          disabled={loading}
                          on:click={() => onApprovePlan(detailItem.id, artifact.id)}
                        >
                          Approve
                        </button>
                      {/if}
                    </div>
                    {#if artifact.body}
                      <pre class="app-scrollbar max-h-72 overflow-auto whitespace-pre-wrap rounded-md border border-white/12 bg-black/30 p-3 font-mono text-[13px] leading-6 text-text-primary">{artifact.body}</pre>
                    {/if}
                  </article>
                {:else}
                  <div class="rounded-md border border-white/12 bg-[#0d0d10] px-4 py-3 text-[14px] text-text-muted">
                    No plan submitted.
                  </div>
                {/each}
              </div>

              <details class="rounded-md border border-white/12 bg-[#0d0d10]">
                <summary class="cursor-pointer px-3 py-2 text-[13px] font-medium text-text-secondary">
                  New draft plan
                </summary>
                <div class="grid gap-3 border-t border-white/12 p-3">
                  <textarea
                    class="min-h-32 resize-y rounded-md border border-white/14 bg-black/30 px-3 py-2 text-[14px] leading-6 text-text-primary outline-none placeholder:text-text-muted focus:border-accent-dim"
                    value={planBodies[detailItem.id] ?? ""}
                    disabled={loading}
                    placeholder="Draft plan"
                    on:input={(event) =>
                      (planBodies = {
                        ...planBodies,
                        [detailItem.id]: event.currentTarget.value,
                      })}
                  ></textarea>
                  <button
                    type="button"
                    class="inline-flex h-9 items-center justify-center rounded-md border border-white/14 bg-white/8 px-3 text-[13px] font-medium text-text-primary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-60"
                    disabled={loading || !(planBodies[detailItem.id] ?? "").trim()}
                    on:click={() => submitPlan(detailItem)}
                  >
                    Submit plan
                  </button>
                </div>
              </details>
            </section>

            <section class="grid gap-3">
              <h3 class="text-[13px] font-semibold uppercase tracking-wide text-text-muted">
                Feedback
              </h3>
              <textarea
                class="min-h-32 resize-y rounded-md border border-white/14 bg-[#0d0d10] px-3 py-2 text-[14px] leading-6 text-text-primary outline-none placeholder:text-text-muted focus:border-accent-dim"
                value={feedbackBodies[detailItem.id] ?? ""}
                disabled={loading}
                placeholder="Review feedback"
                on:input={(event) =>
                  (feedbackBodies = {
                    ...feedbackBodies,
                    [detailItem.id]: event.currentTarget.value,
                  })}
              ></textarea>
              <button
                type="button"
                class="inline-flex h-9 items-center justify-center rounded-md border border-white/14 bg-white/8 px-3 text-[13px] font-medium text-text-primary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-60"
                disabled={loading || !(feedbackBodies[detailItem.id] ?? "").trim()}
                on:click={() => submitFeedback(detailItem)}
              >
                Send feedback
              </button>
              {#if detailArtifacts.filter((artifact) => artifact.kind === "feedback").length > 0}
                <div class="grid gap-2">
                  {#each detailArtifacts.filter((artifact) => artifact.kind === "feedback") as artifact (artifact.id)}
                    <div class="rounded-md border border-white/12 bg-[#0d0d10] px-3 py-2 text-[13px] leading-5 text-text-secondary">
                      <span class="font-mono text-text-muted">{artifact.status}</span>
                      {#if artifact.body}
                        <span> {artifact.body}</span>
                      {/if}
                    </div>
                  {/each}
                </div>
              {/if}
            </section>
          </main>

          <aside class="grid min-w-0 content-start gap-4">
            <section class="grid gap-3 rounded-md border border-white/12 bg-[#0d0d10] p-3">
              <h3 class="text-[13px] font-semibold uppercase tracking-wide text-text-muted">
                State
              </h3>
              <select
                class="h-10 rounded-md border border-white/14 bg-black/30 px-3 text-[14px] text-text-primary outline-none focus:border-accent-dim"
                value={detailItem.stageId}
                disabled={loading}
                on:change={(event) => onMoveWorkItem(detailItem.id, event.currentTarget.value)}
              >
                {#each stages as targetStage (targetStage.id)}
                  <option value={targetStage.id} disabled={!canMoveToStage(detailItem, targetStage)}>
                    {targetStage.name}{!canMoveToStage(detailItem, targetStage)
                      ? " - generate worktree first"
                      : ""}
                  </option>
                {/each}
              </select>

              {#if detailItem.worktree}
                <div class="min-w-0 rounded-md border border-white/12 bg-black/25 px-3 py-2">
                  <div class="truncate font-mono text-[13px] text-text-primary">
                    {detailItem.worktree.branch}
                  </div>
                  <div class="mt-1 truncate font-mono text-[12px] text-text-muted">
                    {detailItem.worktree.worktreePath}
                  </div>
                </div>
              {:else}
                <div class="grid grid-cols-[minmax(0,1fr)_40px] gap-2">
                  <input
                    class="h-10 min-w-0 rounded-md border border-white/14 bg-black/30 px-3 font-mono text-[12px] text-text-primary outline-none placeholder:text-text-muted focus:border-accent-dim"
                    type="text"
                    value={worktreeBranches[detailItem.id] || defaultWorktreeBranch(detailItem)}
                    disabled={loading}
                    aria-label="Worktree branch"
                    on:input={(event) =>
                      (worktreeBranches = {
                        ...worktreeBranches,
                        [detailItem.id]: event.currentTarget.value,
                      })}
                  />
                  <button
                    type="button"
                    aria-label="Generate worktree"
                    title="Generate worktree"
                    class="inline-flex h-10 w-10 items-center justify-center rounded-md border border-white/14 bg-white/8 text-text-secondary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed"
                    disabled={loading}
                    on:click={() => generateWorktree(detailItem)}
                  >
                    <GitBranch size={15} />
                  </button>
                </div>
              {/if}
            </section>

            {#if detailCurrentRun}
              <section class="grid gap-3 rounded-md border border-white/12 bg-[#0d0d10] p-3">
                <div class="flex items-center justify-between gap-2">
                  <h3 class="text-[13px] font-semibold uppercase tracking-wide text-text-muted">
                    Current Run
                  </h3>
                  {#if detailPastRuns.length > 0}
                    <span class="inline-flex items-center gap-1 rounded border border-white/12 bg-white/6 px-2 py-1 text-[12px] text-text-muted">
                      <History size={13} />
                      {detailPastRuns.length}
                    </span>
                  {/if}
                </div>
                <button
                  type="button"
                  class="grid min-w-0 gap-2 rounded-md border border-white/12 bg-black/25 p-3 text-left transition-colors hover:border-white/24 hover:bg-white/6 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50 disabled:cursor-default"
                  disabled={!canOpenRunTerminal(detailCurrentRun)}
                  title={canOpenRunTerminal(detailCurrentRun) ? "Open terminal" : "No terminal linked"}
                  on:click={() => detailCurrentRun && openRunTerminal(detailCurrentRun)}
                >
                  <div class="flex min-w-0 flex-wrap items-center gap-2">
                    <span class="rounded border px-2 py-1 font-mono text-[12px] font-semibold {runStatusClass(detailCurrentRun.status)}">
                      {detailCurrentRun.status}
                    </span>
                    <span class="truncate text-[14px] font-semibold text-text-primary">
                      {detailCurrentRun.preset || "agent"}
                    </span>
                  </div>
                  <div class="truncate text-[13px] text-text-secondary">
                    {detailCurrentRun.promptTemplateId || "prompt"}
                  </div>
                  <div class="inline-flex items-center gap-1 text-[12px] text-text-muted">
                    <Clock3 size={13} />
                    {formattedTime(detailCurrentRun.createdAt)}
                  </div>
                </button>
                {#if canCancelRun(detailCurrentRun)}
                  <button
                    type="button"
                    class="inline-flex h-9 items-center justify-center gap-2 rounded-md border border-red/35 bg-red/10 px-3 text-[13px] font-medium text-red transition-colors hover:border-red disabled:cursor-not-allowed disabled:opacity-60"
                    disabled={loading}
                    on:click={() => onCancelRun(detailCurrentRun.id)}
                  >
                    <Square size={13} />
                    <span>Cancel run</span>
                  </button>
                {/if}
              </section>
            {/if}

            <details class="rounded-md border border-white/12 bg-[#0d0d10]">
              <summary class="cursor-pointer px-3 py-2 text-[13px] font-medium text-text-secondary">
                More actions
              </summary>
              <div class="grid gap-2 border-t border-white/12 p-3">
                <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
                  <button
                    type="button"
                    class="inline-flex h-9 items-center justify-center gap-2 rounded-md border border-white/14 bg-white/6 px-2 text-[13px] font-medium text-text-secondary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-60"
                    disabled={loading}
                    on:click={() => onStartPlanning(detailItem.id)}
                  >
                    <ClipboardCheck size={14} />
                    <span>Planning</span>
                  </button>
                  <button
                    type="button"
                    class="inline-flex h-9 items-center justify-center gap-2 rounded-md border border-white/14 bg-white/6 px-2 text-[13px] font-medium text-text-secondary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-60"
                    disabled={loading || detailItem.stageId !== "ready" || !approvedPlan || hasActiveRun(detailCurrentRun)}
                    on:click={() => onQueueExecution(detailItem.id)}
                  >
                    <Clock3 size={14} />
                    <span>Queue execution</span>
                  </button>
                  <button
                    type="button"
                    class="inline-flex h-9 items-center justify-center gap-2 rounded-md border border-white/14 bg-white/6 px-2 text-[13px] font-medium text-text-secondary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-60"
                    disabled={loading || detailItem.stageId !== "ready" || !approvedPlan || hasActiveRun(detailCurrentRun)}
                    on:click={() => onLaunchExecution(detailItem.id, selectedAgentId)}
                  >
                    <Play size={14} />
                    <span>Launch execution</span>
                  </button>
                  <button
                    type="button"
                    class="inline-flex h-9 items-center justify-center gap-2 rounded-md border border-white/14 bg-white/6 px-2 text-[13px] font-medium text-text-secondary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-60"
                    disabled={loading || !detailLatestRun}
                    on:click={() =>
                      onCompleteExecution({
                        workItemId: detailItem.id,
                        runId: detailLatestRun?.id ?? "",
                        message: "ready for review",
                      })}
                  >
                    <Search size={14} />
                    <span>Review</span>
                  </button>
                  <button
                    type="button"
                    class="inline-flex h-9 items-center justify-center gap-2 rounded-md border border-white/14 bg-white/6 px-2 text-[13px] font-medium text-text-secondary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-60"
                    disabled={loading}
                    on:click={() => approveDone(detailItem)}
                  >
                    <ClipboardCheck size={14} />
                    <span>Done</span>
                  </button>
                </div>
                <input
                  class="h-9 rounded-md border border-white/14 bg-black/30 px-3 text-[13px] text-text-primary outline-none placeholder:text-text-muted focus:border-accent-dim"
                  value={doneReasons[detailItem.id] ?? ""}
                  disabled={loading}
                  placeholder="Done approval reason (optional)"
                  on:input={(event) =>
                    (doneReasons = {
                      ...doneReasons,
                      [detailItem.id]: event.currentTarget.value,
                    })}
                />
              </div>
            </details>

            <section class="grid gap-3 rounded-md border border-white/12 bg-[#0d0d10] p-3">
              <h3 class="text-[13px] font-semibold uppercase tracking-wide text-text-muted">
                Questions
              </h3>
              <div class="grid grid-cols-[minmax(0,1fr)_auto] gap-2">
                <input
                  class="h-10 rounded-md border border-white/14 bg-black/30 px-3 text-[13px] text-text-primary outline-none placeholder:text-text-muted focus:border-accent-dim"
                  value={questionPrompts[detailItem.id] ?? ""}
                  disabled={loading}
                  placeholder="Question for user"
                  on:input={(event) =>
                    (questionPrompts = {
                      ...questionPrompts,
                      [detailItem.id]: event.currentTarget.value,
                    })}
                />
                <button
                  type="button"
                  class="rounded-md border border-white/14 bg-white/8 px-3 text-[13px] font-medium text-text-primary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-60"
                  disabled={loading || !(questionPrompts[detailItem.id] ?? "").trim()}
                  on:click={() => askWorkflowQuestion(detailItem)}
                >
                  Ask
                </button>
              </div>
              <div class="grid gap-2">
                {#each detailQuestions as question (question.id)}
                  <div class="grid gap-2 rounded-md border border-white/12 bg-black/25 px-3 py-2">
                    <div class="text-[13px] leading-5 text-text-secondary">
                      <span class="font-mono text-text-muted">{question.status}</span> {question.prompt}
                    </div>
                    {#if question.status === "open"}
                      <div class="grid grid-cols-[minmax(0,1fr)_auto] gap-2">
                        <input
                          class="h-9 rounded-md border border-white/14 bg-[#0d0d10] px-3 text-[13px] text-text-primary outline-none placeholder:text-text-muted focus:border-accent-dim"
                          value={questionAnswers[question.id] ?? ""}
                          disabled={loading}
                          placeholder="Answer"
                          on:input={(event) =>
                            (questionAnswers = {
                              ...questionAnswers,
                              [question.id]: event.currentTarget.value,
                            })}
                        />
                        <button
                          type="button"
                          class="rounded-md border border-white/14 bg-white/8 px-3 text-[13px] text-text-secondary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed"
                          disabled={loading || !(questionAnswers[question.id] ?? "").trim()}
                          on:click={() => answerWorkflowQuestion(question)}
                        >
                          Answer
                        </button>
                      </div>
                    {:else if question.answer}
                      <div class="text-[13px] leading-5 text-text-muted">{question.answer}</div>
                    {/if}
                  </div>
                {/each}
              </div>
            </section>

            <section class="grid gap-3 rounded-md border border-white/12 bg-[#0d0d10] p-3">
              <div class="flex items-center justify-between gap-2">
                <h3 class="text-[13px] font-semibold uppercase tracking-wide text-text-muted">
                  Attachments
                </h3>
                <button
                  type="button"
                  class="inline-flex h-8 items-center justify-center gap-1 rounded-md border border-white/14 bg-white/8 px-2 text-[13px] text-text-secondary transition-colors hover:bg-white/10 hover:text-text-primary disabled:cursor-not-allowed"
                  disabled={loading}
                  on:click={() => attachFile(detailItem)}
                >
                  <Paperclip size={13} />
                  <span>Attach</span>
                </button>
              </div>
              {#if detailItem.attachments.length > 0}
                <div class="grid gap-2">
                  {#each detailItem.attachments as attachment (attachment.id)}
                    <div class="truncate rounded-md border border-white/12 bg-black/25 px-3 py-2 font-mono text-[12px] text-text-secondary">
                      {attachment.path || attachment.url || attachment.note}
                    </div>
                  {/each}
                </div>
              {/if}
            </section>

            {#if detailGates.length > 0}
              <section class="grid gap-3 rounded-md border border-white/12 bg-[#0d0d10] p-3">
                <h3 class="text-[13px] font-semibold uppercase tracking-wide text-text-muted">
                  Gates
                </h3>
                <div class="grid gap-2">
                  {#each detailGates as gate (gate.id)}
                    <div class="grid gap-2 rounded-md border border-white/12 bg-black/25 px-3 py-2">
                      <div class="flex items-center justify-between gap-2 text-[13px] text-text-secondary">
                        <span>{gate.name}</span>
                        <span class="font-mono text-text-muted">{gate.status}</span>
                      </div>
                      <input
                        class="h-9 rounded-md border border-white/14 bg-[#0d0d10] px-3 text-[13px] text-text-primary outline-none placeholder:text-text-muted focus:border-accent-dim"
                        value={gateOverrideReasons[gate.id] ?? ""}
                        disabled={loading}
                        placeholder="Override reason"
                        on:input={(event) =>
                          (gateOverrideReasons = {
                            ...gateOverrideReasons,
                            [gate.id]: event.currentTarget.value,
                          })}
                      />
                      <div class="grid grid-cols-3 gap-2">
                        <button
                          type="button"
                          class="rounded-md border border-white/14 bg-white/8 px-2 py-1.5 text-[13px] text-text-secondary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed"
                          disabled={loading}
                          on:click={() => completeGate(gate, "passed")}
                        >
                          Pass
                        </button>
                        <button
                          type="button"
                          class="rounded-md border border-white/14 bg-white/8 px-2 py-1.5 text-[13px] text-text-secondary transition-colors hover:border-red hover:text-red disabled:cursor-not-allowed"
                          disabled={loading}
                          on:click={() => completeGate(gate, "failed")}
                        >
                          Fail
                        </button>
                        <button
                          type="button"
                          class="rounded-md border border-white/14 bg-white/8 px-2 py-1.5 text-[13px] text-text-secondary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed"
                          disabled={loading || !(gateOverrideReasons[gate.id] ?? "").trim()}
                          on:click={() => completeGate(gate, "overridden")}
                        >
                          Override
                        </button>
                      </div>
                    </div>
                  {/each}
                </div>
              </section>
            {/if}

            <details class="rounded-md border border-white/12 bg-[#0d0d10]">
              <summary class="cursor-pointer px-3 py-2 text-[13px] font-semibold uppercase tracking-wide text-text-muted">
                History
              </summary>
              <div class="grid gap-3 border-t border-white/12 p-3">
                {#if detailPastRuns.length > 0}
                  <div class="grid gap-2">
                    <div class="text-[13px] font-semibold text-text-primary">Run history</div>
                    {#each detailPastRuns as run (run.id)}
                      <div class="grid gap-1 rounded-md border border-white/12 bg-black/25 px-3 py-2">
                        <div class="flex min-w-0 flex-wrap items-center gap-2">
                          <span class="rounded border px-1.5 py-0.5 font-mono text-[11px] font-semibold {runStatusClass(run.status)}">
                            {run.status}
                          </span>
                          <span class="truncate text-[13px] text-text-secondary">
                            {run.preset || "agent"} · {run.promptTemplateId || "prompt"}
                          </span>
                        </div>
                        <div class="truncate text-[12px] text-text-muted">{formattedTime(run.createdAt)}</div>
                      </div>
                    {/each}
                  </div>
                {/if}

                {#if detailWorkflowEvents.length > 0}
                  <div class="grid gap-2">
                    <div class="text-[13px] font-semibold text-text-primary">Workflow events</div>
                    {#each detailWorkflowEvents as event (event.id)}
                      <div class="rounded-md border border-white/12 bg-black/25 px-3 py-2">
                        <div class="text-[13px] text-text-secondary">
                          {event.type}
                          {#if event.actor}
                            <span class="font-mono text-text-muted"> {event.actor}</span>
                          {/if}
                        </div>
                        <div class="text-[12px] text-text-muted">{formattedTime(event.at)}</div>
                      </div>
                    {/each}
                  </div>
                {/if}

                <div class="grid gap-2">
                  <div class="text-[13px] font-semibold text-text-primary">Item history</div>
                  {#each detailItem.history as event (event.id)}
                    <div class="rounded-md border border-white/12 bg-black/25 px-3 py-2">
                      <div class="text-[13px] text-text-secondary">
                        {event.type}
                        {#if event.stageId}
                          <span class="font-mono text-text-muted"> {event.stageId}</span>
                        {/if}
                      </div>
                      <div class="text-[12px] text-text-muted">
                        {event.at?.toLocaleString?.() ?? event.at}
                      </div>
                    </div>
                  {/each}
                </div>
              </div>
            </details>
          </aside>
        </div>
      </div>

      <div class="flex shrink-0 items-center justify-between gap-2 border-t border-white/12 bg-[#0b0b0d] px-5 py-3">
        <button
          type="button"
          class="inline-flex h-9 items-center justify-center gap-2 rounded-md border border-red/35 bg-red/10 px-3 text-[13px] font-medium text-red transition-colors hover:border-red disabled:cursor-not-allowed"
          disabled={loading}
          on:click={() => deleteWorkItem(detailItem)}
        >
          <Trash2 size={14} />
          <span>Delete</span>
        </button>
        <button
          type="button"
          class="h-9 rounded-md border border-white/14 bg-white/8 px-4 text-[13px] font-medium text-text-secondary transition-colors hover:bg-white/10 hover:text-text-primary"
          on:click={closeDetail}
        >
          Close
        </button>
      </div>
    </div>
  </div>
{/if}
