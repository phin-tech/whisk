<script lang="ts">
  import { onDestroy } from "svelte";
  import type { Session } from "../bindings/github.com/phin-tech/whisk/internal/domain/session/models";
  import type { AgentBridgeApproval, AgentBridgeEvent, AgentPrompt, Project, PTYHistory, PTYHistorySummary, PTYInfo, StatusEvent, WorkItem } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import NotificationsPanel from "./NotificationsPanel.svelte";
  import ProjectsPanel from "./ProjectsPanel.svelte";
  import PtysPanel from "./PtysPanel.svelte";
  import SessionsPanel from "./SessionsPanel.svelte";
  import ResizeHandle from "./ui/ResizeHandle.svelte";
  import WorkItemsPanel from "./WorkItemsPanel.svelte";

  export let activePanel: "sessions" | "ptys" | "work" | "projects" | "notifications" | null = "sessions";
  export let sessions: Session[] = [];
  export let ptys: PTYInfo[] = [];
  export let ptyHistory: PTYHistorySummary[] = [];
  export let selectedPTYHistory: PTYHistory | null = null;
  export let projects: Project[] = [];
  export let workItems: WorkItem[] = [];
  export let statusEvents: StatusEvent[] = [];
  export let agentBridgeApprovals: AgentBridgeApproval[] = [];
  export let agentPrompts: AgentPrompt[] = [];
  export let agentBridgeEvents: AgentBridgeEvent[] = [];
  export let activeSessionId = "";
  export let activeProjectId = "";
  export let workFilterQuery = "";
  export let workFilterStageId = "";
  export let workFilterRunState = "";
  export let loadingSession = false;
  export let loadingPtys = false;
  export let loadingPTYHistory = false;
  export let loadingWork = false;
  export let loadingStatusEvents = false;
  export let railSide: "left" | "right" = "right";
  export let onClose: () => void;
  export let onNewSession: () => void;
  export let onSelectSession: (session: Session) => void;
  export let onCloseSession: (session: Session) => void;
  export let onSetSessionProject: (sessionId: string, projectId: string) => void;
  export let onRefreshPtys: () => void;
  export let onKillPTY: (ptyId: string) => void;
  export let onDeletePTY: (ptyId: string) => void;
  export let onSelectPTYHistory: (ptyId: string) => void;
  export let onRefreshStatusEvents: () => void;
  export let onClearNotifications: () => void;
  export let onSelectStatusEvent: (event: StatusEvent) => void;
  export let onSelectAgentPrompt: (prompt: AgentPrompt) => void;
  export let onSelectAgentBridgeEvent: (event: AgentBridgeEvent) => void;
  export let onResolveAgentBridgeApproval: (id: string, action: "allow" | "deny") => void;
  export let onResolveAgentPrompt: (prompt: AgentPrompt, answer: string, tuiInput?: string) => void;
  export let onRefreshWork: () => void;
  export let onNewProject: () => void;
  export let onSelectProject: (projectId: string) => void;
  export let onCreateWorkItem: (request: {
    projectId: string;
    title: string;
    bodyMarkdown: string;
  }) => void;

  const minWidth = 240;
  const maxWidth = 800;
  let width = 320;
  let dragging = false;
  let teardown: (() => void) | null = null;

  function clamp(value: number) {
    return Math.max(minWidth, Math.min(maxWidth, value));
  }

  function endDrag() {
    teardown?.();
    teardown = null;
    dragging = false;
  }

  function startDrag(event: MouseEvent) {
    event.preventDefault();
    endDrag();
    dragging = true;
    const startX = event.clientX;
    const startWidth = width;
    const sign = railSide === "left" ? 1 : -1;
    const onMove = (moveEvent: MouseEvent) => {
      width = clamp(startWidth + sign * (moveEvent.clientX - startX));
    };
    const onKey = (keyEvent: KeyboardEvent) => {
      if (keyEvent.key === "Escape") endDrag();
    };
    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", endDrag);
    window.addEventListener("blur", endDrag);
    window.addEventListener("keydown", onKey);
    teardown = () => {
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", endDrag);
      window.removeEventListener("blur", endDrag);
      window.removeEventListener("keydown", onKey);
    };
  }

  onDestroy(endDrag);
</script>

{#if activePanel}
  <div class="relative flex shrink-0">
    {#if railSide === "right"}
      <ResizeHandle {dragging} onmousedown={startDrag} />
    {/if}
    <div
      class="dock-panel relative h-full shrink-0 bg-sidebar-surface"
      style="--dock-width: {width}px"
    >
      {#if activePanel === "sessions"}
        <SessionsPanel
          {sessions}
          {projects}
          {activeSessionId}
          loading={loadingSession}
          onclose={onClose}
          {onNewSession}
          {onSelectSession}
          {onCloseSession}
          {onSetSessionProject}
        />
      {:else}
        {#if activePanel === "notifications"}
          <NotificationsPanel
            {sessions}
            {statusEvents}
            {agentBridgeApprovals}
            {agentPrompts}
            {agentBridgeEvents}
            loading={loadingStatusEvents}
            onclose={onClose}
            onRefresh={onRefreshStatusEvents}
            {onClearNotifications}
            {onSelectStatusEvent}
            {onSelectAgentPrompt}
            {onSelectAgentBridgeEvent}
            {onResolveAgentBridgeApproval}
            {onResolveAgentPrompt}
          />
        {:else if activePanel === "work"}
          <WorkItemsPanel
            {projects}
            {workItems}
            {activeProjectId}
            bind:filterQuery={workFilterQuery}
            bind:filterStageId={workFilterStageId}
            bind:filterRunState={workFilterRunState}
            loading={loadingWork}
            onclose={onClose}
            onRefresh={onRefreshWork}
            {onSelectProject}
            {onCreateWorkItem}
          />
        {:else if activePanel === "projects"}
          <ProjectsPanel
            {projects}
            {activeProjectId}
            loading={loadingWork}
            onclose={onClose}
            onRefresh={onRefreshWork}
            {onNewProject}
            {onSelectProject}
          />
        {:else if activePanel === "ptys"}
          <PtysPanel
            {ptys}
            {ptyHistory}
            {selectedPTYHistory}
            loading={loadingPtys}
            loadingHistory={loadingPTYHistory}
            onclose={onClose}
            onRefresh={onRefreshPtys}
            onKill={onKillPTY}
            onDelete={onDeletePTY}
            onSelectHistory={onSelectPTYHistory}
          />
        {/if}
      {/if}
    </div>
    {#if railSide === "left"}
      <ResizeHandle {dragging} onmousedown={startDrag} />
    {/if}
  </div>
{/if}
