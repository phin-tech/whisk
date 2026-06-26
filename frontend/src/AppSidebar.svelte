<script lang="ts">
  import type { Session } from "../bindings/github.com/phin-tech/whisk/internal/domain/session/models";
  import type {
    AgentBridgeApproval,
    AgentBridgeEvent,
    AgentPrompt,
    Project,
    PTYBookmark,
    PTYHistory,
    PTYHistorySummary,
    PTYInfo,
    StatusEvent,
    WorkItem,
  } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import ActivityRail from "./ActivityRail.svelte";
  import SidebarDock from "./SidebarDock.svelte";
  import type { MainView } from "./navigation";

  type SidebarId = "sessions" | "ptys" | "work" | "projects" | "notifications";
  type RailSide = "left" | "right";

  export let side: RailSide = "right";
  export let activeMain: MainView = "session";
  export let activeSidebar: SidebarId | null = "sessions";
  export let settingsOpen = false;
  export let notificationCount = 0;
  export let sessions: Session[] = [];
  export let ptys: PTYInfo[] = [];
  export let bookmarksByPty: Record<string, PTYBookmark[]> = {};
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
  export let onSidebar: (id: SidebarId) => void;
  export let onSettings: () => void;
  export let onClose: () => void;
  export let onNewSession: () => void;
  export let onSelectSession: (session: Session) => void;
  export let onCloseSession: (session: Session) => void;
  export let onSetSessionProject: (sessionId: string, projectId: string) => void;
  export let onRefreshPtys: () => void;
  export let onKillPTY: (ptyId: string) => void;
  export let onDeletePTY: (ptyId: string) => void;
  export let onSelectBookmark: (bookmark: PTYBookmark) => void;
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
  export let onSelectProjectDetail: (projectId: string) => void;
  export let onCreateWorkItem: (request: {
    projectId: string;
    title: string;
    bodyMarkdown: string;
  }) => void;

  $: railActiveSidebar =
    activeSidebar ?? (activeMain === "work" ? "work" : activeMain === "projects" ? "projects" : null);
  $: selectProject = activeSidebar === "projects" ? onSelectProjectDetail : onSelectProject;
</script>

{#snippet rail()}
  <div
    class="flex h-full w-[36px] shrink-0 flex-col {side === 'left'
      ? 'border-r'
      : 'border-l'} border-hairline bg-bg-base/96"
  >
    <ActivityRail
      activeSidebar={railActiveSidebar}
      {settingsOpen}
      {notificationCount}
      onSidebar={onSidebar}
      onSettings={onSettings}
    />
  </div>
{/snippet}

{#snippet dock()}
  <SidebarDock
    activePanel={activeSidebar}
    {sessions}
    {ptys}
    {bookmarksByPty}
    {ptyHistory}
    {selectedPTYHistory}
    {projects}
    {workItems}
    {statusEvents}
    {agentBridgeApprovals}
    {agentPrompts}
    {agentBridgeEvents}
    {activeSessionId}
    {activeProjectId}
    bind:workFilterQuery
    bind:workFilterStageId
    bind:workFilterRunState
    {loadingSession}
    {loadingPtys}
    {loadingPTYHistory}
    {loadingWork}
    {loadingStatusEvents}
    railSide={side}
    {onClose}
    {onNewSession}
    {onSelectSession}
    {onCloseSession}
    {onSetSessionProject}
    {onRefreshPtys}
    {onKillPTY}
    {onDeletePTY}
    {onSelectBookmark}
    {onSelectPTYHistory}
    {onRefreshStatusEvents}
    {onClearNotifications}
    {onSelectStatusEvent}
    {onSelectAgentPrompt}
    {onSelectAgentBridgeEvent}
    {onResolveAgentBridgeApproval}
    {onResolveAgentPrompt}
    {onRefreshWork}
    {onNewProject}
    onSelectProject={selectProject}
    {onCreateWorkItem}
  />
{/snippet}

<div class="flex h-full shrink-0 {side === 'left' ? 'order-first' : 'order-last'}">
  {#if side === "left"}
    {@render rail()}
    {@render dock()}
  {:else if side === "right"}
    {@render dock()}
    {@render rail()}
  {/if}
</div>
