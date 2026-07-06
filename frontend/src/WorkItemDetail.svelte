<script lang="ts">
  import { Dialogs } from "@wailsio/runtime";
  import ChevronDown from "@lucide/svelte/icons/chevron-down";
  import ClipboardCheck from "@lucide/svelte/icons/clipboard-check";
  import Clock3 from "@lucide/svelte/icons/clock-3";
  import Ellipsis from "@lucide/svelte/icons/ellipsis";
  import FileText from "@lucide/svelte/icons/file-text";
  import GitBranch from "@lucide/svelte/icons/git-branch";
  import History from "@lucide/svelte/icons/history";
  import Paperclip from "@lucide/svelte/icons/paperclip";
  import Play from "@lucide/svelte/icons/play";
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
    WorkItemLink,
    WorkItemRun,
    WorkflowActionAvailability,
    WorkflowEvent,
    ReadyWorkExplanation,
  } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import { canMoveToStage, deriveNextStep, selectDetailRun } from "./workView";
  import type { NextStepView } from "./workView";
  import Badge from "./ui/Badge.svelte";
  import Button from "./ui/Button.svelte";
  import Checkbox from "./ui/Checkbox.svelte";
  import DetailLayout from "./ui/DetailLayout.svelte";
  import IconButton from "./ui/IconButton.svelte";
  import Menu from "./ui/Menu.svelte";
  import MenuItem from "./ui/MenuItem.svelte";
  import ModalShell from "./ui/ModalShell.svelte";
  import NextActionBar from "./ui/NextActionBar.svelte";
  import Popover from "./ui/Popover.svelte";
  import PropertyRow from "./ui/PropertyRow.svelte";
  import SectionHeader from "./ui/SectionHeader.svelte";
  import SelectField from "./ui/SelectField.svelte";
  import StatusDot from "./ui/StatusDot.svelte";
  import TextArea from "./ui/TextArea.svelte";
  import TextField from "./ui/TextField.svelte";

  export let item: WorkItem;
  export let project: Project;
  export let stages: WorkflowStage[] = [];
  export let agentProfiles: AgentProfile[] = [];
  export let workItems: WorkItem[] = [];
  export let workItemLinks: WorkItemLink[] = [];
  export let readyWork: ReadyWorkExplanation = { ready: [], blocked: [], summary: { totalReady: 0, totalBlocked: 0, cycleCount: 0 } };
  export let workItemRuns: WorkItemRun[] = [];
  export let artifacts: Artifact[] = [];
  export let questions: Question[] = [];
  export let gateReports: GateReport[] = [];
  export let workflowActions: WorkflowActionAvailability[] = [];
  export let workflowEvents: WorkflowEvent[] = [];
  export let loading = false;

  export let onClose: () => void;
  export let onUpdateWorkItem: (request: { id: string; title: string; bodyMarkdown: string }) => void;
  export let onMoveWorkItem: (workItemId: string, stageId: string) => void;
  export let onAddWorkItemLink: (request: {
    sourceWorkItemId: string;
    targetWorkItemId: string;
    type: string;
  }) => void;
  export let onGenerateWorktree: (request: { workItemId: string; branch: string }) => void;
  export let onAttachFile: (workItemId: string, path: string) => void;
  export let onDeleteWorkItem: (workItemId: string) => void;
  export let onCancelRun: (runId: string) => void;
  export let onLaunchRun: (runId: string, agentProfileId?: string) => void;
  export let onOpenRunTerminal: (run: WorkItemRun) => void;
  export let onStartPlanning: (workItemId: string) => void;
  export let onSubmitPlan: (request: { workItemId: string; runId: string; title: string; body: string }) => void;
  export let onApprovePlan: (workItemId: string, artifactId: string) => void;
  export let onLaunchExecution: (workItemId: string, agentProfileId?: string) => void;
  export let onSetPhaseAgent: (projectId: string, preset: string, agentProfileId: string) => void;
  export let onSetInteractiveAgentShell: (projectId: string, enabled: boolean) => void;
  export let onCompleteExecution: (request: { workItemId: string; runId: string; message: string }) => void;
  export let onSubmitReviewFeedback: (request: { workItemId: string; runId: string; body: string }) => void;
  export let onAskQuestion: (request: { workItemId: string; runId: string; prompt: string }) => void;
  export let onAnswerQuestion: (questionId: string, answer: string) => void;
  export let onCompleteGate: (request: { id: string; status: string; overrideReason: string }) => void;
  export let onRunWorkflowAction: (request: {
    workItemId: string;
    actionId: string;
    runId?: string;
    reason?: string;
  }) => void;
  export let onApproveDone: (workItemId: string, reason: string) => void;

  // Modal-local input state, keyed per item so switching items resets cleanly.
  let worktreeBranches: Record<string, string> = {};
  let planBodies: Record<string, string> = {};
  let feedbackBodies: Record<string, string> = {};
  let questionPrompts: Record<string, string> = {};
  let questionAnswers: Record<string, string> = {};
  let gateOverrideReasons: Record<string, string> = {};
  let doneReasons: Record<string, string> = {};
  let agentSelections: Record<string, string> = {};
  let blockerSelections: Record<string, string> = {};

  // One popover open at a time in the properties rail.
  let openMenu = "";
  let runHistoryOpen = false;

  // Title/description draft, reset when the open item changes.
  let draftItemId = "";
  let title = "";
  let body = "";
  $: if (item.id !== draftItemId) {
    draftItemId = item.id;
    title = item.title;
    body = item.bodyMarkdown;
    openMenu = "";
    runHistoryOpen = false;
  }
  $: dirty = title !== item.title || body !== item.bodyMarkdown;

  $: detailRuns = [...workItemRuns.filter((run) => run.workItemId === item.id)].sort(
    (a, b) => timestamp(b.createdAt) - timestamp(a.createdAt),
  );
  $: detailArtifacts = artifacts.filter((artifact) => artifact.workItemId === item.id);
  $: detailQuestions = questions.filter((question) => question.workItemId === item.id);
  $: detailGates = gateReports.filter((gate) => gate.workItemId === item.id);
  $: detailWorkflowEvents = workflowEvents.filter((event) => event.workItemId === item.id);
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
  $: feedbackArtifacts = detailArtifacts.filter((artifact) => artifact.kind === "feedback");
  $: blockingLinks = workItemLinks.filter(
    (link) => link.sourceWorkItemId === item.id && link.type === "blocks",
  );
  $: dependentLinks = workItemLinks.filter(
    (link) => link.targetWorkItemId === item.id && link.type === "blocks",
  );
  $: blockedExplanation = readyWork.blocked?.find((entry) => entry.workItem?.id === item.id) ?? null;
  $: readyExplanation = readyWork.ready?.find((entry) => entry.workItem?.id === item.id) ?? null;
  $: availableBlockers = workItems.filter(
    (candidate) =>
      candidate.projectId === item.projectId &&
      candidate.id !== item.id &&
      !blockingLinks.some((link) => link.targetWorkItemId === candidate.id),
  );
  $: selectedBlockerId = blockerSelections[item.id] || availableBlockers[0]?.id || "";
  $: blockerOptions = availableBlockers.map((candidate) => ({
    value: candidate.id,
    label: `#${candidate.number} ${candidate.title}`,
  }));

  // Agent selection: the preset a launch runs under and the project's remembered default.
  $: executionStage = stages.find((stage) => stage.kind === "execution" || stage.id === "execution") ?? null;
  $: launchPreset = detailCurrentRun?.preset || executionStage?.defaultRunPreset || "writer";
  $: phaseAgentDefault = project?.preferences?.defaultPhaseAgents?.[launchPreset] ?? "";
  $: selectedAgentId = agentSelections[launchPreset] ?? phaseAgentDefault;
  $: selectedAgentLabel =
    agentProfiles.find((profile) => profile.id === selectedAgentId)?.label ?? "Default agent";
  $: agentOptions = [
    { value: "", label: "Default agent" },
    ...agentProfiles.map((profile) => ({ value: profile.id, label: profile.label })),
  ];

  $: nextStep = computeNextStep(item, detailCurrentRun, detailLatestRun, approvedPlan, latestDraftPlan, workflowActions);

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

  function slugify(value: string) {
    return value
      .toLowerCase()
      .trim()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-+|-+$/g, "");
  }

  function defaultWorktreeBranch(target: WorkItem) {
    const projectSlug = project?.slug || "work";
    const itemSlug = slugify(target.title) || "item";
    return `whisk/${projectSlug}-${target.number}-${itemSlug}`;
  }

  function canCancelRun(run: WorkItemRun | null) {
    return run?.status === "queued" || run?.status === "running" || run?.status === "awaiting_input";
  }

  function canOpenRunTerminal(run: WorkItemRun | null) {
    return Boolean(run?.sessionId || run?.ptyId);
  }

  function stageLabel(stageId: string) {
    return stages.find((stage) => stage.id === stageId)?.name ?? stageId;
  }

  function workItemLabel(workItemId: string) {
    const linked = workItems.find((candidate) => candidate.id === workItemId);
    if (!linked) return workItemId;
    return `#${linked.number} ${linked.title}`;
  }

  function blockerStageLabel(workItemId: string) {
    const linked = workItems.find((candidate) => candidate.id === workItemId);
    return linked ? stageLabel(linked.stageId) : "";
  }

  function selectBlocker(value: string) {
    blockerSelections = { ...blockerSelections, [item.id]: value };
  }

  function addBlocker() {
    if (!selectedBlockerId || loading) return;
    onAddWorkItemLink({
      sourceWorkItemId: item.id,
      targetWorkItemId: selectedBlockerId,
      type: "blocks",
    });
    blockerSelections = { ...blockerSelections, [item.id]: "" };
  }

  function openRunTerminal(run: WorkItemRun | null) {
    if (!run || !canOpenRunTerminal(run)) return;
    onOpenRunTerminal(run);
  }

  function setMenuOpen(name: string, open: boolean) {
    openMenu = open ? name : openMenu === name ? "" : openMenu;
  }

  function moveToStage(stageId: string) {
    openMenu = "";
    if (stageId === item.stageId) return;
    onMoveWorkItem(item.id, stageId);
  }

  function generateWorktree() {
    const branch = (worktreeBranches[item.id] || defaultWorktreeBranch(item)).trim();
    if (!branch || loading) return;
    openMenu = "";
    onGenerateWorktree({ workItemId: item.id, branch });
  }

  function selectAgent(value: string) {
    if (!project) return;
    agentSelections = { ...agentSelections, [launchPreset]: value };
    onSetPhaseAgent(project.id, launchPreset, value);
  }

  function setInteractiveAgentShell(enabled: boolean) {
    if (!project) return;
    onSetInteractiveAgentShell(project.id, enabled);
  }

  function submitPlan() {
    const value = (planBodies[item.id] ?? "").trim();
    if (!value || loading) return;
    onSubmitPlan({ workItemId: item.id, runId: detailLatestRun?.id ?? "", title: "Plan", body: value });
    planBodies = { ...planBodies, [item.id]: "" };
  }

  function submitFeedback() {
    const value = (feedbackBodies[item.id] ?? "").trim();
    if (!value || loading) return;
    onSubmitReviewFeedback({ workItemId: item.id, runId: detailLatestRun?.id ?? "", body: value });
    feedbackBodies = { ...feedbackBodies, [item.id]: "" };
  }

  function askWorkflowQuestion() {
    const prompt = (questionPrompts[item.id] ?? "").trim();
    if (!prompt || loading) return;
    onAskQuestion({ workItemId: item.id, runId: detailLatestRun?.id ?? "", prompt });
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

  function approveDone() {
    const reason = (doneReasons[item.id] ?? "").trim();
    if (loading) return;
    openMenu = "";
    onApproveDone(item.id, reason);
  }

  function sendToReview() {
    if (loading || !detailLatestRun) return;
    openMenu = "";
    onCompleteExecution({ workItemId: item.id, runId: detailLatestRun?.id ?? "", message: "ready for review" });
  }

  function workflowActionLabel(actionId: string) {
    return actionId
      .split("_")
      .filter(Boolean)
      .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
      .join(" ");
  }

  function workflowActionEffect(availability: WorkflowActionAvailability) {
    const action = availability.action;
    const parts = [];
    if (action.to) parts.push(`to ${stageLabel(action.to)}`);
    if (action.createsRun?.preset) parts.push(`run ${action.createsRun.preset}`);
    if (action.createsArtifact) parts.push(`${action.createsArtifact.kind} ${action.createsArtifact.status}`);
    if (action.updatesArtifact) parts.push(`${action.updatesArtifact.kind} ${action.updatesArtifact.status}`);
    if ((action.createsGates ?? []).length > 0) parts.push(`${action.createsGates?.length} gate(s)`);
    if (action.requiresPassingBlockingGates) parts.push("gates required");
    return parts.join(" · ");
  }

  function supportedWorkflowAction(actionId: string) {
    return actionId === "start_planning" ||
      actionId === "start_execution" ||
      actionId === "complete_execution" ||
      actionId === "approve_done";
  }

  function workflowActionReason(availability: WorkflowActionAvailability) {
    if (!availability.enabled) return availability.reason || "not available";
    if (!supportedWorkflowAction(availability.action.id)) {
      if (availability.inputKind === "artifact" || availability.inputKind === "artifact_selection") return "use the artifact form";
      if (availability.inputKind === "gate") return "use the gate form";
      if (availability.inputKind !== "none") return "not wired in the UI";
    }
    if (availability.action.id === "complete_execution" && !detailLatestRun) return "no run";
    return workflowActionEffect(availability);
  }

  function canRunWorkflowAction(availability: WorkflowActionAvailability) {
    if (!availability.enabled) return false;
    if (supportedWorkflowAction(availability.action.id)) {
      return !(availability.action.id === "complete_execution" && !detailLatestRun);
    }
    return availability.inputKind === "none";
  }

  function runWorkflowAction(availability: WorkflowActionAvailability) {
    if (!canRunWorkflowAction(availability) || loading) return;
    openMenu = "";
    switch (availability.action.id) {
      case "start_planning":
        onStartPlanning(item.id);
        break;
      case "start_execution":
        onLaunchExecution(item.id, selectedAgentId);
        break;
      case "complete_execution":
        sendToReview();
        break;
      case "approve_done":
        approveDone();
        break;
      default:
        onRunWorkflowAction({
          workItemId: item.id,
          actionId: availability.action.id,
          runId: detailLatestRun?.id ?? "",
          reason: "",
        });
        break;
    }
  }

  function setDoneReason(value: string) {
    doneReasons = { ...doneReasons, [item.id]: value };
  }

  function setWorktreeBranch(value: string) {
    worktreeBranches = { ...worktreeBranches, [item.id]: value };
  }

  async function attachFile() {
    const selected = await Dialogs.OpenFile({
      Title: "Attach file",
      ButtonText: "Attach",
      Directory: project?.rootDir || undefined,
      CanChooseDirectories: false,
      CanChooseFiles: true,
      AllowsMultipleSelection: false,
    });
    if (typeof selected === "string" && selected.length > 0) {
      onAttachFile(item.id, selected);
    }
  }

  function deleteWorkItem() {
    if (loading) return;
    if (window.confirm(`Delete #${item.number} ${item.title}?`)) {
      onDeleteWorkItem(item.id);
      onClose();
    }
  }

  function resetDraft() {
    title = item.title;
    body = item.bodyMarkdown;
  }

  function saveDetail() {
    if (loading || !title.trim()) return;
    const trimmed = title.trim();
    title = trimmed;
    onUpdateWorkItem({ id: item.id, title: trimmed, bodyMarkdown: body });
  }

  type NextStep = NextStepView & { run: () => void };

  // computeNextStep derives the recommended action (pure logic in deriveNextStep) and wires
  // the matching handler closure for the primary button.
  function computeNextStep(
    target: WorkItem,
    currentRun: WorkItemRun | null,
    latestRun: WorkItemRun | null,
    approved: Artifact | undefined,
    draft: Artifact | undefined,
    workflowActions: WorkflowActionAvailability[],
  ): NextStep {
    const view = deriveNextStep({
      stageId: target.stageId,
      runStatus: currentRun?.status ?? "",
      hasTerminal: canOpenRunTerminal(currentRun),
      hasApprovedPlan: Boolean(approved),
      hasDraftPlan: Boolean(draft),
      hasLatestRun: Boolean(latestRun),
      workflowActions,
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
          onLaunchExecution(target.id, selectedAgentId);
          break;
        case "approve-plan":
          if (draft?.id) onApprovePlan(target.id, draft.id);
          break;
        case "start-planning":
        case "retry-planning":
          onStartPlanning(target.id);
          break;
        case "send-to-review":
          onCompleteExecution({ workItemId: target.id, runId: latestRun?.id ?? "", message: "ready for review" });
          break;
        case "mark-done":
          approveDone();
          break;
        default:
          break;
      }
    };

    return { ...view, run };
  }

  function handleKey(event: KeyboardEvent) {
    if (event.key !== "Escape") return;
    if (openMenu) {
      event.preventDefault();
      openMenu = "";
      return;
    }
    event.preventDefault();
    onClose();
  }

  const subHeader = "text-[11px] font-semibold uppercase text-text-muted";
</script>

<ModalShell
  open
  titleId="work-item-detail-title"
  titleClass="sr-only"
  class="flex max-h-[92vh] max-w-[calc(100vw-2rem)] flex-col overflow-hidden shadow-[0_28px_90px_rgba(0,0,0,0.7)] xl:max-w-[1180px]"
  interactOutsideBehavior="ignore"
  escapeKeydownBehavior="ignore"
  onkeydown={handleKey}
>
  {#snippet heading()}
    Work item editor
  {/snippet}

    <!-- Header -->
    <div class="shrink-0 border-b border-hairline bg-bg-base px-5 py-3">
      <div class="flex items-start justify-between gap-4">
        <div class="min-w-0 flex-1">
          <div class="flex min-w-0 flex-wrap items-center gap-2 text-[12px]">
            <span class="truncate text-text-muted">{project.name}</span>
            <span class="opacity-40">·</span>
            <span class="font-mono text-text-muted">#{item.number}</span>
            <span class="opacity-40">·</span>
            <Badge class="bg-bg-surface/60 text-text-secondary">{stageLabel(item.stageId)}</Badge>
            <StatusDot status={detailCurrentRun?.status || item.runState || "idle"} showLabel />
          </div>
          <TextField
            variant="seamless"
            bind:value={title}
            disabled={loading}
            aria-label="Work item title"
            class="mt-1.5 w-full px-0 text-[18px] font-semibold leading-7 focus:text-accent"
          />
        </div>
        <div class="flex shrink-0 items-center gap-1">
          {#if dirty}
            <Button type="button" variant="outline" size="sm" disabled={loading} onclick={resetDraft}>
              Cancel
            </Button>
            <Button
              type="button"
              variant="primary"
              disabled={loading || !title.trim()}
              onclick={saveDetail}
            >
              Save
            </Button>
          {/if}
          <IconButton label="Close" onclick={onClose}>
            <X size={16} />
          </IconButton>
        </div>
      </div>
    </div>

    {#if nextStep}
      <NextActionBar step={nextStep} disabled={loading}>
        {#snippet controls()}
          {#if nextStep.isLaunch}
            <SelectField
              value={selectedAgentId}
              label="Agent profile"
              options={agentOptions}
              disabled={loading}
              class="h-7 text-[11px]"
              onValueChange={selectAgent}
            />
            <Checkbox
              checked={Boolean(project.preferences?.useInteractiveAgentShell)}
              disabled={loading}
              class="h-7 rounded border border-border bg-bg-deep px-2 text-[11px]"
              onCheckedChange={setInteractiveAgentShell}
            >
              Shell
            </Checkbox>
          {/if}
        {/snippet}
      </NextActionBar>
    {/if}

    <div class="app-scrollbar min-h-0 flex-1 overflow-y-auto px-5 py-5">
      <DetailLayout>
        {#snippet main()}
          <!-- Main column -->
          <!-- Description -->
          <section class="grid gap-2">
            <SectionHeader title="Description" />
            <TextArea
              variant="seamless"
              bind:value={body}
              disabled={loading}
              aria-label="Work item description"
              placeholder="Add a description…"
              class="min-h-28 text-[14px] leading-6"
            />
          </section>

          <!-- Plan -->
          <section class="grid gap-2">
            <SectionHeader title="Plan">
              {#if approvedPlan}
                <span class="inline-flex items-center gap-1 text-[11px] font-medium text-green">
                  <span>●</span> Approved
                </span>
              {:else if latestDraftPlan}
                <span class="inline-flex items-center gap-1 text-[11px] font-medium text-amber">
                  <span>●</span> Draft ready
                </span>
              {/if}
            </SectionHeader>

            {#each planArtifacts as artifact (artifact.id)}
              <article class="grid gap-2 rounded border border-border-subtle bg-bg-surface/30 p-3">
                <div class="flex min-w-0 items-start justify-between gap-3">
                  <div class="min-w-0">
                    <div class="flex min-w-0 items-center gap-2 text-[13px] font-medium text-text-primary">
                      <FileText size={14} class="shrink-0 text-text-muted" />
                      <span class="truncate">{artifact.title || "Plan"}</span>
                      <span class="shrink-0 font-mono text-[10px] uppercase text-text-muted">{artifact.status}</span>
                    </div>
                    <div class="mt-0.5 truncate text-[11px] text-text-muted">
                      {formattedTime(artifact.updatedAt || artifact.createdAt)}
                    </div>
                  </div>
                  {#if artifact.status === "draft"}
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      disabled={loading}
                      onclick={() => onApprovePlan(item.id, artifact.id)}
                    >
                      Approve
                    </Button>
                  {/if}
                </div>
                {#if artifact.body}
                  <pre class="app-scrollbar max-h-72 overflow-auto whitespace-pre-wrap rounded border border-hairline bg-bg-deep p-3 font-mono text-[12px] leading-6 text-text-secondary">{artifact.body}</pre>
                {/if}
              </article>
            {:else}
              <div class="px-2 py-2 text-[12px] text-text-muted">No plan submitted.</div>
            {/each}

            <details class="rounded border border-border-subtle bg-bg-surface/30">
              <summary class="cursor-pointer px-3 py-2 text-[12px] font-medium text-text-secondary">
                New draft plan
              </summary>
              <div class="grid gap-2 border-t border-hairline p-3">
                <TextArea
                  value={planBodies[item.id] ?? ""}
                  disabled={loading}
                  placeholder="Draft plan"
                  class="min-h-28 py-1.5 text-[13px] leading-6"
                  oninput={(event: Event) =>
                    (planBodies = { ...planBodies, [item.id]: (event.currentTarget as HTMLTextAreaElement).value })}
                />
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={loading || !(planBodies[item.id] ?? "").trim()}
                  onclick={submitPlan}
                >
                  Submit plan
                </Button>
              </div>
            </details>
          </section>

          <!-- Dependencies -->
          <section class="grid gap-2">
            <SectionHeader title="Dependencies">
              <span class="text-[11px] text-text-muted">
                Ready {readyWork.summary?.totalReady ?? 0} · Blocked {readyWork.summary?.totalBlocked ?? 0}
              </span>
            </SectionHeader>

            <div class="grid gap-2 rounded border border-border-subtle bg-bg-surface/30 p-3">
              <div class="grid gap-1.5">
                <div class="text-[11px] font-medium uppercase tracking-wide text-text-muted">Blocked by</div>
                {#if blockedExplanation?.blockedBy?.length}
                  {#each blockedExplanation.blockedBy as blocker (blocker.id)}
                    <div class="flex min-w-0 items-center gap-2 text-[13px] text-text-secondary">
                      <span class="h-1.5 w-1.5 shrink-0 rounded-full bg-amber"></span>
                      <span class="min-w-0 flex-1 truncate">
                        {blocker.number ? `#${blocker.number} ` : ""}{blocker.title || blocker.id}
                      </span>
                      <span class="shrink-0 rounded border border-border-subtle px-1.5 py-0.5 text-[11px] text-text-muted">
                        {blocker.stageId || "unknown"}
                      </span>
                    </div>
                  {/each}
                {:else if blockingLinks.length > 0}
                  {#each blockingLinks as link (link.id)}
                    <div class="flex min-w-0 items-center gap-2 text-[13px] text-text-secondary">
                      <span class="h-1.5 w-1.5 shrink-0 rounded-full bg-green"></span>
                      <span class="min-w-0 flex-1 truncate">{workItemLabel(link.targetWorkItemId)}</span>
                      <span class="shrink-0 rounded border border-border-subtle px-1.5 py-0.5 text-[11px] text-text-muted">
                        {blockerStageLabel(link.targetWorkItemId) || "resolved"}
                      </span>
                    </div>
                  {/each}
                {:else}
                  <div class="text-[13px] text-text-muted">No blocking dependencies</div>
                {/if}
              </div>

              <div class="grid gap-1.5 border-t border-hairline pt-2">
                <div class="text-[11px] font-medium uppercase tracking-wide text-text-muted">Ready because</div>
                <div class="text-[13px] leading-5 text-text-secondary">
                  {readyExplanation?.reason || (blockedExplanation ? "blocked by unfinished dependency" : "No blocking dependencies")}
                </div>
              </div>

              {#if dependentLinks.length > 0}
                <div class="grid gap-1.5 border-t border-hairline pt-2">
                  <div class="text-[11px] font-medium uppercase tracking-wide text-text-muted">Blocks</div>
                  {#each dependentLinks as link (link.id)}
                    <div class="truncate text-[13px] text-text-secondary">{workItemLabel(link.sourceWorkItemId)}</div>
                  {/each}
                </div>
              {/if}

              <div class="grid grid-cols-[minmax(0,1fr)_auto] gap-2 border-t border-hairline pt-2">
                <SelectField
                  value={selectedBlockerId}
                  label="Blocker work item"
                  options={blockerOptions}
                  placeholder="Blocker work item"
                  disabled={loading || availableBlockers.length === 0}
                  onValueChange={selectBlocker}
                />
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={loading || !selectedBlockerId}
                  onclick={addBlocker}
                >
                  Add blocker
                </Button>
              </div>
            </div>
          </section>

          <!-- Activity (feedback, questions, gates, history) -->
          <section class="grid gap-4">
            <SectionHeader title="Activity" />

            <!-- Questions -->
            <div class="grid gap-2">
              <div class="text-[11px] font-medium uppercase tracking-wide text-text-muted">Questions</div>
              <div class="grid grid-cols-[minmax(0,1fr)_auto] gap-2">
                <TextField
                  value={questionPrompts[item.id] ?? ""}
                  disabled={loading}
                  placeholder="Question for user"
                  oninput={(event: Event) =>
                    (questionPrompts = { ...questionPrompts, [item.id]: (event.currentTarget as HTMLInputElement).value })}
                />
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={loading || !(questionPrompts[item.id] ?? "").trim()}
                  onclick={askWorkflowQuestion}
                >
                  Ask
                </Button>
              </div>
              {#each detailQuestions as question (question.id)}
                <div class="grid gap-2 border-l border-hairline py-1 pl-3">
                  <div class="text-[13px] leading-5 text-text-secondary">
                    <StatusDot status={question.status === "open" ? "queued" : ""} />
                    {question.prompt}
                  </div>
                  {#if question.status === "open"}
                    <div class="grid grid-cols-[minmax(0,1fr)_auto] gap-2">
                      <TextField
                        value={questionAnswers[question.id] ?? ""}
                        disabled={loading}
                        placeholder="Answer"
                        oninput={(event: Event) =>
                          (questionAnswers = { ...questionAnswers, [question.id]: (event.currentTarget as HTMLInputElement).value })}
                      />
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        disabled={loading || !(questionAnswers[question.id] ?? "").trim()}
                        onclick={() => answerWorkflowQuestion(question)}
                      >
                        Answer
                      </Button>
                    </div>
                  {:else if question.answer}
                    <div class="text-[12px] leading-5 text-text-muted">{question.answer}</div>
                  {/if}
                </div>
              {/each}
            </div>

            <!-- Feedback -->
            <div class="grid gap-2">
              <div class="text-[11px] font-medium uppercase tracking-wide text-text-muted">Feedback</div>
              <TextArea
                value={feedbackBodies[item.id] ?? ""}
                disabled={loading}
                placeholder="Review feedback"
                class="min-h-20 py-1.5 text-[13px] leading-6"
                oninput={(event: Event) =>
                  (feedbackBodies = { ...feedbackBodies, [item.id]: (event.currentTarget as HTMLTextAreaElement).value })}
              />
              <div class="flex justify-end">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={loading || !(feedbackBodies[item.id] ?? "").trim()}
                  onclick={submitFeedback}
                >
                  Send feedback
                </Button>
              </div>
              {#each feedbackArtifacts as artifact (artifact.id)}
                <div class="border-l border-hairline py-1 pl-3 text-[13px] leading-5 text-text-secondary">
                  <span class="font-mono text-[11px] text-text-muted">{artifact.status}</span>
                  {#if artifact.body}<span> {artifact.body}</span>{/if}
                </div>
              {/each}
            </div>

            <!-- Gates -->
            {#if detailGates.length > 0}
              <div class="grid gap-2">
                <div class="text-[11px] font-medium uppercase tracking-wide text-text-muted">Gates</div>
                {#each detailGates as gate (gate.id)}
                  <div class="grid gap-2 rounded border border-border-subtle bg-bg-surface/30 px-3 py-2">
                    <div class="flex items-center justify-between gap-2 text-[13px] text-text-secondary">
                      <span>{gate.name}</span>
                      <span class="font-mono text-[11px] text-text-muted">{gate.status}</span>
                    </div>
                    <TextField
                      value={gateOverrideReasons[gate.id] ?? ""}
                      disabled={loading}
                      placeholder="Override reason"
                      oninput={(event: Event) =>
                        (gateOverrideReasons = { ...gateOverrideReasons, [gate.id]: (event.currentTarget as HTMLInputElement).value })}
                    />
                    <div class="grid grid-cols-3 gap-2">
                      <Button type="button" variant="outline" size="sm" disabled={loading} onclick={() => completeGate(gate, "passed")}>
                        Pass
                      </Button>
                      <Button
                        type="button"
                        variant="danger"
                        size="sm"
                        disabled={loading}
                        onclick={() => completeGate(gate, "failed")}
                      >
                        Fail
                      </Button>
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        disabled={loading || !(gateOverrideReasons[gate.id] ?? "").trim()}
                        onclick={() => completeGate(gate, "overridden")}
                      >
                        Override
                      </Button>
                    </div>
                  </div>
                {/each}
              </div>
            {/if}

            <!-- History -->
            <details class="rounded border border-border-subtle bg-bg-surface/30">
              <summary class="cursor-pointer px-3 py-2 text-[11px] font-semibold uppercase text-text-muted">
                History
              </summary>
              <div class="grid gap-3 border-t border-hairline p-3">
                {#if detailPastRuns.length > 0}
                  <div class="grid gap-1.5">
                    <div class="text-[12px] font-medium text-text-primary">Run history</div>
                    {#each detailPastRuns as run (run.id)}
                      <div class="flex min-w-0 items-center gap-2 text-[12px]">
                        <StatusDot status={run.status} showLabel />
                        <span class="truncate text-text-secondary">{run.preset || "agent"} · {run.promptTemplateId || "prompt"}</span>
                        <span class="ml-auto shrink-0 text-[11px] text-text-muted">{formattedTime(run.createdAt)}</span>
                      </div>
                    {/each}
                  </div>
                {/if}
                {#if detailWorkflowEvents.length > 0}
                  <div class="grid gap-1.5">
                    <div class="text-[12px] font-medium text-text-primary">Workflow events</div>
                    {#each detailWorkflowEvents as event (event.id)}
                      <div class="flex min-w-0 items-center gap-2 text-[12px] text-text-secondary">
                        <span>{event.type}</span>
                        {#if event.actor}<span class="font-mono text-text-muted">{event.actor}</span>{/if}
                        <span class="ml-auto shrink-0 text-[11px] text-text-muted">{formattedTime(event.at)}</span>
                      </div>
                    {/each}
                  </div>
                {/if}
                <div class="grid gap-1.5">
                  <div class="text-[12px] font-medium text-text-primary">Item history</div>
                  {#each item.history as event (event.id)}
                    <div class="flex min-w-0 items-center gap-2 text-[12px] text-text-secondary">
                      <span>{event.type}</span>
                      {#if event.stageId}<span class="font-mono text-text-muted">{event.stageId}</span>{/if}
                      <span class="ml-auto shrink-0 text-[11px] text-text-muted">
                        {event.at?.toLocaleString?.() ?? event.at}
                      </span>
                    </div>
                  {/each}
                </div>
              </div>
            </details>
          </section>
        {/snippet}

        {#snippet aside()}
          <!-- Properties rail -->
          <div class="flex items-center justify-between gap-2 pb-1">
            <h3 class={subHeader}>Properties</h3>
            <Popover
              open={openMenu === "actions"}
              onOpenChange={(open) => setMenuOpen("actions", open)}
            >
              {#snippet trigger({ props })}
                <IconButton {...props} label="More actions" title="More actions">
                  <Ellipsis size={14} />
                </IconButton>
              {/snippet}
              <Menu class="min-w-60">
                {#each workflowActions as availability (availability.action.id)}
                  {@const actionId = availability.action.id}
                  {@const reason = workflowActionReason(availability)}
                  <MenuItem
                    disabled={loading || !canRunWorkflowAction(availability)}
                    title={reason}
                    onclick={() => runWorkflowAction(availability)}
                    class="items-start gap-2"
                  >
                    {#if actionId === "start_planning" || actionId === "approve_done"}
                      <ClipboardCheck size={13} class="mt-0.5 shrink-0" />
                    {:else if actionId === "start_execution"}
                      <Play size={13} class="mt-0.5 shrink-0" />
                    {:else if actionId === "complete_execution"}
                      <Search size={13} class="mt-0.5 shrink-0" />
                    {:else}
                      <Clock3 size={13} class="mt-0.5 shrink-0" />
                    {/if}
                    <span class="grid min-w-0 gap-0.5">
                      <span class="truncate">{workflowActionLabel(actionId)}</span>
                      {#if reason}
                        <span class="truncate font-normal text-[10px] text-text-muted">{reason}</span>
                      {/if}
                    </span>
                  </MenuItem>
                {:else}
                  <div class="px-3 py-2 text-[11px] text-text-muted">No workflow actions.</div>
                {/each}
                <div class="my-1 border-t border-hairline"></div>
                <div class="px-3 py-1.5">
                  <TextField
                    value={doneReasons[item.id] ?? ""}
                    disabled={loading}
                    placeholder="Done reason (optional)"
                    oninput={(event: Event) => setDoneReason((event.currentTarget as HTMLInputElement).value)}
                  />
                </div>
              </Menu>
            </Popover>
          </div>

          <div class="divide-y divide-hairline rounded border border-border-subtle bg-bg-surface/20">
            <PropertyRow label="Status">
              <div class="flex items-center gap-2">
                <span class="inline-flex items-center gap-1 text-[12px]">
                  <StatusDot status={detailCurrentRun?.status || item.runState || "idle"} showLabel />
                </span>
                {#if canCancelRun(detailCurrentRun)}
                  <Button
                    type="button"
                    variant="danger"
                    size="sm"
                    class="h-6 px-1.5 text-[11px]"
                    disabled={loading}
                    onclick={() => detailCurrentRun && onCancelRun(detailCurrentRun.id)}
                  >
                    <Square size={11} /> Cancel
                  </Button>
                {/if}
              </div>
            </PropertyRow>

            <PropertyRow label="Stage">
              <Popover
                open={openMenu === "stage"}
                onOpenChange={(open) => setMenuOpen("stage", open)}
              >
                {#snippet trigger({ props })}
                  <Button
                    {...props}
                    type="button"
                    variant="ghost"
                    size="sm"
                    class="max-w-[60%] border-transparent px-1.5 py-1 text-[12px] text-text-primary hover:border-border-subtle hover:bg-bg-surface/60 hover:text-text-primary"
                    disabled={loading}
                  >
                    <span class="truncate">{stageLabel(item.stageId)}</span>
                    <ChevronDown size={12} class="shrink-0 text-text-muted" />
                  </Button>
                {/snippet}
                <Menu class="min-w-44">
                  {#each stages as targetStage (targetStage.id)}
                    {@const allowed = canMoveToStage(item, targetStage)}
                    <MenuItem
                      active={targetStage.id === item.stageId}
                      disabled={loading || !allowed}
                      onclick={() => moveToStage(targetStage.id)}
                    >
                      <span class="truncate">{targetStage.name}</span>
                      {#if !allowed}<span class="ml-auto text-[10px] text-text-muted">worktree first</span>{/if}
                    </MenuItem>
                  {/each}
                </Menu>
              </Popover>
            </PropertyRow>

            <!-- Branch / worktree -->
            <div class="relative grid gap-1 px-3 py-2">
              <div class="flex items-center justify-between gap-2">
                <span class={subHeader}>Branch</span>
                {#if !item.worktree}
                  <Popover
                    open={openMenu === "branch"}
                    onOpenChange={(open) => setMenuOpen("branch", open)}
                    class="w-64 p-2"
                  >
                    {#snippet trigger({ props })}
                      <Button
                        {...props}
                        type="button"
                        variant="ghost"
                        size="sm"
                        class="border-transparent px-1.5 py-1 text-[12px] text-text-primary hover:border-border-subtle hover:bg-bg-surface/60 hover:text-text-primary"
                        disabled={loading}
                      >
                        <GitBranch size={12} class="text-text-muted" />
                        <span class="truncate">Generate</span>
                      </Button>
                    {/snippet}
                    <div class="grid gap-2">
                      <TextField
                        value={worktreeBranches[item.id] || defaultWorktreeBranch(item)}
                        disabled={loading}
                        aria-label="Worktree branch"
                        class="font-mono"
                        oninput={(event: Event) => setWorktreeBranch((event.currentTarget as HTMLInputElement).value)}
                      />
                      <Button type="button" variant="primary" disabled={loading} onclick={generateWorktree}>
                        <GitBranch size={13} /> Generate worktree
                      </Button>
                    </div>
                  </Popover>
                {/if}
              </div>
              {#if item.worktree}
                <div class="min-w-0">
                  <div class="truncate font-mono text-[12px] text-text-secondary">{item.worktree.branch}</div>
                  <div class="truncate font-mono text-[11px] text-text-muted">{item.worktree.worktreePath}</div>
                </div>
              {/if}
            </div>

            <PropertyRow label="Agent">
              <Popover
                open={openMenu === "agent"}
                onOpenChange={(open) => setMenuOpen("agent", open)}
              >
                {#snippet trigger({ props })}
                  <Button
                    {...props}
                    type="button"
                    variant="ghost"
                    size="sm"
                    class="max-w-[60%] border-transparent px-1.5 py-1 text-[12px] text-text-primary hover:border-border-subtle hover:bg-bg-surface/60 hover:text-text-primary"
                    disabled={loading}
                  >
                    <span class="truncate">{selectedAgentLabel}</span>
                    <ChevronDown size={12} class="shrink-0 text-text-muted" />
                  </Button>
                {/snippet}
                <Menu class="min-w-48">
                  <MenuItem
                    active={selectedAgentId === ""}
                    disabled={loading}
                    onclick={() => { openMenu = ""; selectAgent(""); }}
                  >
                    Default agent
                  </MenuItem>
                  {#each agentProfiles as profile (profile.id)}
                    <MenuItem
                      active={selectedAgentId === profile.id}
                      disabled={loading}
                      onclick={() => { openMenu = ""; selectAgent(profile.id); }}
                    >
                      {profile.label}
                    </MenuItem>
                  {/each}
                  <div class="my-1 border-t border-hairline"></div>
                  <Checkbox
                    class="px-3 py-1.5 text-[12px]"
                    checked={Boolean(project.preferences?.useInteractiveAgentShell)}
                    disabled={loading}
                    onCheckedChange={setInteractiveAgentShell}
                  >
                    Interactive shell
                  </Checkbox>
                </Menu>
              </Popover>
            </PropertyRow>

            <!-- Run -->
            {#if detailCurrentRun}
              <div class="grid gap-1.5 px-3 py-2">
                <div class="flex items-center justify-between gap-2">
                  <span class={subHeader}>Run</span>
                  {#if detailPastRuns.length > 0}
                    <Button
                      variant="ghost"
                      size="sm"
                      class="inline-flex items-center gap-1 text-[11px] text-text-muted transition-colors hover:text-text-primary"
                      onclick={() => (runHistoryOpen = !runHistoryOpen)}
                    >
                      <History size={12} /> {detailPastRuns.length}
                    </Button>
                  {/if}
                </div>
                <Button
                  variant="ghost"
                  align="start"
                  class="grid w-full min-w-0 gap-1 rounded border-border-subtle bg-bg-surface/40 p-2 text-left hover:border-accent/40 hover:bg-bg-surface/70 disabled:cursor-default disabled:hover:border-border-subtle disabled:hover:bg-bg-surface/40"
                  disabled={!canOpenRunTerminal(detailCurrentRun)}
                  title={canOpenRunTerminal(detailCurrentRun) ? "Open terminal" : "No terminal linked"}
                  onclick={() => openRunTerminal(detailCurrentRun)}
                >
                  <div class="flex min-w-0 items-center gap-2">
                    <StatusDot status={detailCurrentRun.status} />
                    <span class="truncate text-[13px] font-medium text-text-primary">{detailCurrentRun.preset || "agent"}</span>
                    {#if canOpenRunTerminal(detailCurrentRun)}
                      <SquareTerminal size={13} class="ml-auto shrink-0 text-text-muted" />
                    {/if}
                  </div>
                  <div class="truncate text-[11px] text-text-muted">{detailCurrentRun.promptTemplateId || "prompt"}</div>
                  <div class="inline-flex items-center gap-1 text-[11px] text-text-muted">
                    <Clock3 size={11} /> {formattedTime(detailCurrentRun.createdAt)}
                  </div>
                </Button>
                {#if runHistoryOpen}
                  <div class="grid gap-1 border-l border-hairline pl-2">
                    {#each detailPastRuns as run (run.id)}
                      <div class="flex min-w-0 items-center gap-2 text-[11px]">
                        <StatusDot status={run.status} />
                        <span class="truncate text-text-muted">{run.preset || "agent"}</span>
                        <span class="ml-auto shrink-0 text-text-muted">{formattedTime(run.createdAt)}</span>
                      </div>
                    {/each}
                  </div>
                {/if}
              </div>
            {/if}

            <div class="grid gap-1.5">
              <PropertyRow label="Attachments">
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  class="max-w-[60%] border-transparent px-1.5 py-1 text-[12px] text-text-primary hover:border-border-subtle hover:bg-bg-surface/60 hover:text-text-primary"
                  disabled={loading}
                  onclick={attachFile}
                >
                  <Paperclip size={12} class="text-text-muted" />
                  <span>Attach</span>
                </Button>
              </PropertyRow>
              {#if item.attachments.length > 0}
                <div class="grid gap-1 px-3 pb-2">
                  {#each item.attachments as attachment (attachment.id)}
                    <div class="truncate font-mono text-[11px] text-text-muted">
                      {attachment.path || attachment.url || attachment.note}
                    </div>
                  {/each}
                </div>
              {/if}
            </div>
          </div>
        {/snippet}
      </DetailLayout>
    </div>

    <!-- Footer -->
    <div class="flex shrink-0 items-center justify-between gap-2 border-t border-hairline bg-bg-base px-5 py-3">
      <Button
        type="button"
        variant="danger-ghost"
        disabled={loading}
        onclick={deleteWorkItem}
      >
        <Trash2 size={14} /> Delete
      </Button>
      <Button type="button" variant="outline" onclick={onClose}>
        Close
      </Button>
    </div>
</ModalShell>
