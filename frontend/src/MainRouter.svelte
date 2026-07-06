<script lang="ts">
  import type {
    Session,
    SessionWindow,
  } from "../bindings/github.com/phin-tech/whisk/internal/domain/session/models";
  import type {
    AgentProfile,
    Artifact,
    GateReport,
    MetadataValue,
    Project,
    ProjectAttachmentTemplate,
    ProjectDetail,
    Question,
    ReadyWorkExplanation,
    WorkItem,
    WorkItemLink,
    WorkItemRun,
    WorkflowActionAvailability,
    WorkflowDefinitionRecord,
    WorkflowEvent,
    WorkflowMigrationPlan,
    WorkflowValidationReport,
  } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import LayoutView from "./LayoutView.svelte";
  import type { MainView } from "./navigation";
  import type { TerminalSnapshot } from "./ptyStream";
  import ProjectsView from "./ProjectsView.svelte";
  import Button from "./ui/Button.svelte";
  import WorkBoard from "./WorkBoard.svelte";

  export let activeMain: MainView = "session";
  export let activeSession: Session | null = null;
  export let activeSessionWindow: SessionWindow | null = null;
  export let outputChunks: Record<string, Uint8Array[]> = {};
  export let outputChunkStartOffsets: Record<string, number[]> = {};
  export let terminalSnapshots: Record<string, TerminalSnapshot> = {};
  export let bottomJumpRevisions: Record<string, number> = {};
  export let activePaneId = "";
  export let terminalFontSize = 13;
  export let terminalCursorBlink = true;
  export let loadingSession = false;
  export let projects: Project[] = [];
  export let projectDetail: ProjectDetail | null = null;
  export let activeProjectId = "";
  export let loadingWork = false;
  export let workBoardOpenItemId = "";
  export let canNavigateBack = false;
  export let workItems: WorkItem[] = [];
  export let workItemLinks: WorkItemLink[] = [];
  export let readyWork: ReadyWorkExplanation;
  export let workItemRuns: WorkItemRun[] = [];
  export let artifacts: Artifact[] = [];
  export let questions: Question[] = [];
  export let gateReports: GateReport[] = [];
  export let workflowDefinitions: WorkflowDefinitionRecord[] = [];
  export let workflowActionsByItem: Record<string, WorkflowActionAvailability[]> = {};
  export let workflowMigrationPlan: WorkflowMigrationPlan | null = null;
  export let workflowValidationReport: WorkflowValidationReport | null = null;
  export let workflowEvents: WorkflowEvent[] = [];
  export let agentProfiles: AgentProfile[] = [];
  export let workFilterQuery = "";
  export let workFilterStageId = "";
  export let workFilterRunState = "";
  export let pluginAttachmentTemplates: (ProjectAttachmentTemplate & { pluginId: string })[] = [];
  export let onUpdateProject: (
    projectId: string,
    request: { name: string; description: string },
  ) => void;
  export let onSetProjectWorkflowDefinition: (
    projectId: string,
    id: string,
    version: number,
  ) => void;
  export let onPlanProjectWorkflowMigration: (
    projectId: string,
    id: string,
    version: number,
  ) => void;
  export let onValidateWorkflowFile: (path: string) => void;
  export let onImportWorkflowFile: (path: string) => void;
  export let onExportWorkflowFile: (id: string, version: number, path: string) => void;
  export let onDeleteWorkflowDefinition: (id: string, version: number) => void;
  export let onDeleteProject: (projectId: string) => void;
  export let onNewProjectSession: (projectId: string) => void;
  export let onOpenSession: (sessionId: string) => void;
  export let onRemoveSession: (sessionId: string) => void;
  export let onCreateWorkItem: (request: {
    projectId: string;
    title: string;
    bodyMarkdown: string;
  }) => void;
  export let onDeleteWorkItem: (workItemId: string) => void;
  export let onOpenWorkItem: (workItemId: string) => void;
  export let onOpenRunTerminal: (run: WorkItemRun) => void;
  export let onAddProjectAttachment: (request: {
    projectId: string;
    kind: string;
    title: string;
    path: string;
    url: string;
    note: string;
    provider: string;
    target: string;
    includeInContext: boolean;
  }) => void;
  export let onRunPluginProjectAttachmentTemplate: (request: {
    pluginId: string;
    templateId: string;
    projectId: string;
    values: Record<string, string>;
  }) => void;
  export let onUpdateProjectAttachment: (
    projectId: string,
    attachmentId: string,
    request: {
      title: string;
      path: string;
      url: string;
      note: string;
      provider: string;
      target: string;
      includeInContext: boolean;
      meta?: Record<string, MetadataValue>;
    },
  ) => void;
  export let onDeleteProjectAttachment: (projectId: string, attachmentId: string) => void;
  export let onDetailClose: (() => void) | null = null;
  export let onRefreshWork: () => void;
  export let onUpdateWorkItem: (request: {
    id: string;
    title: string;
    bodyMarkdown: string;
  }) => void;
  export let onMoveWorkItem: (workItemId: string, stageId: string) => void;
  export let onAddWorkItemLink: (request: {
    sourceWorkItemId: string;
    targetWorkItemId: string;
    type: string;
  }) => void;
  export let onGenerateWorktree: (request: { workItemId: string; branch: string }) => void;
  export let onAttachFile: (workItemId: string, path: string) => void;
  export let onCancelRun: (runId: string) => void;
  export let onLaunchRun: (runId: string, agentProfileId?: string) => void;
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
  export let onSetPhaseAgent: (
    projectId: string,
    preset: string,
    agentProfileId: string,
  ) => void;
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
  export let onAskQuestion: (request: {
    workItemId: string;
    runId: string;
    prompt: string;
  }) => void;
  export let onAnswerQuestion: (questionId: string, answer: string) => void;
  export let onCompleteGate: (request: { id: string; status: string; overrideReason: string }) => void;
  export let onApproveDone: (workItemId: string, reason: string) => void;
  export let onFocusPane: (paneId: string) => void;
  export let onPtyInput: (ptyId: string) => void;
  export let onWriteInput: (ptyId: string, data: string) => Promise<void>;
  export let onClosePane: (paneId: string) => void;
  export let onKillPanePTY: (paneId: string) => void;
  export let canClosePane: (paneId: string) => boolean;
  export let onNewSession: () => void;
</script>

{#if activeMain === "projects"}
  <ProjectsView
    {projects}
    {workflowDefinitions}
    {workflowMigrationPlan}
    {workflowValidationReport}
    detail={projectDetail}
    {activeProjectId}
    loading={loadingWork || loadingSession}
    {onUpdateProject}
    {onSetProjectWorkflowDefinition}
    {onPlanProjectWorkflowMigration}
    {onValidateWorkflowFile}
    {onImportWorkflowFile}
    {onExportWorkflowFile}
    {onDeleteWorkflowDefinition}
    {onDeleteProject}
    onNewSession={onNewProjectSession}
    {onOpenSession}
    onRemoveSession={onRemoveSession}
    {artifacts}
    {gateReports}
    {onCreateWorkItem}
    {onDeleteWorkItem}
    {onOpenWorkItem}
    {onOpenRunTerminal}
    {pluginAttachmentTemplates}
    onAddProjectAttachment={onAddProjectAttachment}
    {onRunPluginProjectAttachmentTemplate}
    {onUpdateProjectAttachment}
    {onDeleteProjectAttachment}
  />
{:else if activeMain === "work"}
  <WorkBoard
    openItemId={workBoardOpenItemId}
    onDetailClose={canNavigateBack ? onDetailClose : null}
    {projects}
    {workItems}
    {workItemLinks}
    {readyWork}
    {workItemRuns}
    {artifacts}
    {questions}
    {gateReports}
    workflowDefinitions={workflowDefinitions}
    {workflowActionsByItem}
    {workflowEvents}
    {agentProfiles}
    {activeProjectId}
    filterQuery={workFilterQuery}
    filterStageId={workFilterStageId}
    filterRunState={workFilterRunState}
    loading={loadingWork}
    onRefresh={onRefreshWork}
    {onCreateWorkItem}
    {onUpdateWorkItem}
    {onMoveWorkItem}
    {onAddWorkItemLink}
    {onGenerateWorktree}
    {onAttachFile}
    {onDeleteWorkItem}
    {onCancelRun}
    {onLaunchRun}
    {onOpenRunTerminal}
    {onStartPlanning}
    {onSubmitPlan}
    {onApprovePlan}
    {onQueueExecution}
    {onLaunchExecution}
    {onSetPhaseAgent}
    {onSetInteractiveAgentShell}
    {onCompleteExecution}
    {onSubmitReviewFeedback}
    {onAskQuestion}
    {onAnswerQuestion}
    {onCompleteGate}
    {onApproveDone}
  />
{:else if activeSession}
  {#if activeSessionWindow}
    <LayoutView
      node={activeSessionWindow.layout}
      panes={activeSession.panes}
      {outputChunks}
      {outputChunkStartOffsets}
      {terminalSnapshots}
      {bottomJumpRevisions}
      {activePaneId}
      {terminalFontSize}
      {terminalCursorBlink}
      onFocus={onFocusPane}
      onInput={onPtyInput}
      onWriteInput={onWriteInput}
      onClose={onClosePane}
      onKillPTY={onKillPanePTY}
      canClose={canClosePane}
    />
  {/if}
{:else}
  <div class="flex flex-1 flex-col items-center justify-center gap-4 text-center text-text-secondary">
    <div
      class="flex h-16 w-16 items-center justify-center rounded-2xl border border-border-subtle bg-bg-surface/80 text-accent shadow-[0_18px_40px_rgba(2,6,23,0.45)]"
    >
      <span class="text-[30px]">W</span>
    </div>
    <div class="space-y-1">
      <p class="text-[14px] font-semibold tracking-tight text-text-primary">
        No active sessions
      </p>
      <p class="text-[13px] text-text-secondary">
        Start a daemon-owned shell session.
      </p>
    </div>
    <Button
      variant="outline"
      size="lg"
      class="shadow-[0_18px_40px_rgba(2,6,23,0.45)]"
      disabled={loadingSession}
      onclick={onNewSession}
    >
      New Session
    </Button>
  </div>
{/if}
