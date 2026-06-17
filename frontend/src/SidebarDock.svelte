<script lang="ts">
  import { onDestroy } from "svelte";
  import type { Session } from "../bindings/github.com/phin-tech/whisk/internal/domain/session/models";
  import type { AgentBridgeApproval, AgentBridgeEvent, Project, StatusEvent, WorkItem } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import type { PTYInfo } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import NotificationsPanel from "./NotificationsPanel.svelte";
  import ProjectsPanel from "./ProjectsPanel.svelte";
  import PtysPanel from "./PtysPanel.svelte";
  import SessionsPanel from "./SessionsPanel.svelte";
  import WorkItemsPanel from "./WorkItemsPanel.svelte";

  export let activePanel: "sessions" | "ptys" | "work" | "projects" | "notifications" | null = "sessions";
  export let sessions: Session[] = [];
  export let ptys: PTYInfo[] = [];
  export let projects: Project[] = [];
  export let workItems: WorkItem[] = [];
  export let statusEvents: StatusEvent[] = [];
  export let agentBridgeApprovals: AgentBridgeApproval[] = [];
  export let agentBridgeEvents: AgentBridgeEvent[] = [];
  export let activeSessionId = "";
  export let activeProjectId = "";
  export let loadingSession = false;
  export let loadingPtys = false;
  export let loadingWork = false;
  export let loadingStatusEvents = false;
  export let railSide: "left" | "right" = "right";
  export let onClose: () => void;
  export let onNewSession: () => void;
  export let onSelectSession: (session: Session) => void;
  export let onCloseSession: (session: Session) => void;
  export let onRefreshPtys: () => void;
  export let onRefreshStatusEvents: () => void;
  export let onClearNotifications: () => void;
  export let onSelectStatusEvent: (event: StatusEvent) => void;
  export let onResolveAgentBridgeApproval: (id: string, action: "allow" | "deny") => void;
  export let onRefreshWork: () => void;
  export let onNewProject: () => void;
  export let onSelectProject: (projectId: string) => void;
  export let onCreateWorkItem: (request: {
    projectId: string;
    title: string;
    bodyMarkdown: string;
  }) => void;
  export let onMoveWorkItem: (workItemId: string, stageId: string) => void;
  export let onGenerateWorktree: (request: {
    workItemId: string;
    branch: string;
  }) => void;
  export let onAttachFile: (workItemId: string, path: string) => void;
  export let onDeleteWorkItem: (workItemId: string) => void;

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
      <button
        type="button"
        aria-label="Resize sidebar"
        class="group relative flex w-1 shrink-0 cursor-col-resize self-stretch flex-col items-center border-0 bg-transparent p-0"
        on:mousedown={startDrag}
      >
        <div
          class="min-h-0 max-w-[0.5px] min-w-[0.5px] flex-1 transition-all duration-150 {dragging
            ? 'bg-white/30'
            : 'bg-white/20 group-hover:bg-white/40'}"
        ></div>
      </button>
    {/if}
    <div
      class="dock-panel relative h-full shrink-0 bg-bg-deep"
      style="--dock-width: {width}px"
    >
      {#if activePanel === "sessions"}
        <SessionsPanel
          {sessions}
          {activeSessionId}
          loading={loadingSession}
          onclose={onClose}
          {onNewSession}
          {onSelectSession}
          {onCloseSession}
        />
      {:else}
        {#if activePanel === "notifications"}
          <NotificationsPanel
            {statusEvents}
            {agentBridgeApprovals}
            {agentBridgeEvents}
            loading={loadingStatusEvents}
            onclose={onClose}
            onRefresh={onRefreshStatusEvents}
            {onClearNotifications}
            {onSelectStatusEvent}
            {onResolveAgentBridgeApproval}
          />
        {:else if activePanel === "work"}
          <WorkItemsPanel
            {projects}
            {workItems}
            {activeProjectId}
            loading={loadingWork}
            onclose={onClose}
            onRefresh={onRefreshWork}
            {onSelectProject}
            {onCreateWorkItem}
            {onMoveWorkItem}
            {onGenerateWorktree}
            {onAttachFile}
            {onDeleteWorkItem}
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
          <PtysPanel ptys={ptys} loading={loadingPtys} onclose={onClose} onRefresh={onRefreshPtys} />
        {/if}
      {/if}
    </div>
    {#if railSide === "left"}
      <button
        type="button"
        aria-label="Resize sidebar"
        class="group relative flex w-1 shrink-0 cursor-col-resize self-stretch flex-col items-center border-0 bg-transparent p-0"
        on:mousedown={startDrag}
      >
        <div
          class="min-h-0 max-w-[0.5px] min-w-[0.5px] flex-1 transition-all duration-150 {dragging
            ? 'bg-white/30'
            : 'bg-white/20 group-hover:bg-white/40'}"
        ></div>
      </button>
    {/if}
  </div>
{/if}
