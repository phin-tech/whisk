<script lang="ts">
  import { Events } from "@wailsio/runtime";
  import { onDestroy, onMount } from "svelte";
  import type { Session } from "../bindings/github.com/phin-tech/whisk/internal/domain/session/models";
  import type {
    AgentBridgeApproval,
    AgentBridgeEvent,
    AgentProfile,
    AgentPrompt,
    AgentHookIntegration,
    AgentHookLogStatus,
    Project,
    ProjectDetail,
    PTYHistory,
    PTYHistorySummary,
    PTYInfo,
    RuntimeEvent,
    StatusEvent,
    Artifact,
    GateReport,
    OnboardingStatus,
    PluginStatus,
    RegistryPlugin,
    ProjectAttachmentTemplate,
    MetadataValue,
    Question,
    WorkItem,
    WorkItemLink,
    WorkItemRun,
    WorkflowActionAvailability,
    WorkflowDefinitionRecord,
    WorkflowEvent,
    WorkflowMigrationPlan,
    WorkflowValidationReport,
    ReadyWorkExplanation,
  } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import type { DaemonStatus } from "../bindings/github.com/phin-tech/whisk/internal/wailsapp/models";
  import {
    AddProjectAttachment,
    AddWorkItemLink,
    AddWorkItemAttachment,
    AnswerQuestion,
    ApproveDone,
    ApprovePlan,
    AskQuestion,
    BindWorkItemWorktree,
    CancelWorkItemRun,
    ClosePane,
    CloseSession,
    CompleteExecution,
    CompleteGate,
    CreateProject,
    CreateSession,
    CreateWorkItem,
    CreateWorktree,
    DaemonStatus as LoadDaemonStatus,
    DeletePTY,
    DeleteProject,
    DeleteProjectAttachment,
    DeleteWorkItem,
    DeleteWorkflowDefinition,
    ExportWorkflowDefinitionFile,
    CheckAgentHookIntegration,
    AgentHookLogStatus as LoadAgentHookLogStatus,
    ApplyOnboarding,
    ClearAgentHookLog,
    InstallAgentHookIntegration,
    KillPTY,
    ListAgentBridgeApprovals,
    ListAgentBridgeEvents,
    ListAgentPrompts,
    ListAgentHookIntegrations,
    ListArtifacts,
    ListAgentProfiles,
    ListGateReports,
    ListPTYHistory,
    ListPTYs,
    ProjectDetail as LoadProjectDetail,
    ListProjects,
    ListQuestions,
    ListSessions,
    ListStatusEvents,
    ListWorkItemLinks,
    ListWorkItemRuns,
    ListWorkItemWorkflowActions,
    ListWorkflowDefinitions,
    ListWorkItems,
    ListWorkflowEvents,
    LoadAppSettings,
    LaunchExecution,
    LaunchWorkItemRun,
    LogPTYTrace,
    MarkAgentBridgeEventRead,
    MarkStatusEventRead,
    MoveWorkItem,
    NextEvent,
    Output,
    OpenAgentHookLog,
    PTYTraceEnabled,
    PlanProjectWorkflowMigration,
    QueueExecution,
    ReadyWork,
    RemoveAgentHookIntegration,
    RescanPlugins,
    ResolveAgentBridgeApproval,
    ResolveAgentPrompt,
    ReadPTYHistory,
    RunPluginProjectAttachmentTemplate,
    SaveAppSettings,
    SetAgentHookLogSettings,
    SetNotificationFocusContext,
    SetProjectWorkflowDefinition,
    SetSessionProject,
    SplitPane,
    StartPlanning,
    SubmitDraftPlan,
    SubmitReviewFeedback,
    SyncSessionMenu,
    TrustPlugin,
    UntrustPlugin,
    UpdateProject,
    UpdateProjectAttachment,
    UpdateWorkItem,
    ImportWorkflowDefinitionFile,
    ValidateWorkflowDefinitionFile,
    WritePTY,
    ListPlugins,
    OnboardingStatus as LoadOnboardingStatus,
    ListRegistryPlugins,
    InstallPlugin,
  } from "../bindings/github.com/phin-tech/whisk/internal/wailsapp/service";
  import AppSidebar from "./AppSidebar.svelte";
  import CommandPalette from "./CommandPalette.svelte";
  import ConfirmDialog from "./ConfirmDialog.svelte";
  import MainRouter from "./MainRouter.svelte";
  import NewProjectDialog from "./NewProjectDialog.svelte";
  import NewSessionDialog from "./NewSessionDialog.svelte";
  import OnboardingPanel from "./OnboardingPanel.svelte";
  import SettingsView from "./SettingsView.svelte";
  import { agentHookNotificationClickTarget, isAgentHookNotification, upsertAgentHookIntegration as upsertAgentHookIntegrationView } from "./agentHooksView";
  import type { Command } from "./commands";
  import { runCommand } from "./commands";
  import { notificationSurfaceCount, targetForStatusEvent } from "./notificationsView";
  import {
    clearNavigationStack as clearNavigationStackState,
    navigateBack as navigateBackState,
    navigateTo as navigateToState,
    selectMainView,
    type MainView,
    type NavigationState,
  } from "./navigation";
  import { projectDetailWithStoreSessions } from "./projectView";
  import {
    nextPTYStreamOffset,
    outputSnapshotChunkAfterOffset,
    ptyAttachWebSocketURL,
    ptyInputTraceLine,
    type PTYStreamFrame,
    writePTYInputOverSocket,
  } from "./ptyStream";
  import {
    claimSingleFlight,
    initialDaemonLinkSnapshot,
    isCurrentDaemonGeneration,
    nextDaemonLinkSnapshot,
    reconnectBackoffDelayMs,
    releaseSingleFlight,
  } from "./daemonLink";
  import { sessionSplitCommands } from "./sessionCommands";
  import { activeWindow, closePaneRequest, closePaneTarget, firstPaneId, isStalePTYError, killPTYRequest, runtimeRefreshTargets, visiblePtyIds } from "./sessionView";
  import {
    normalizeStartupView,
    startupTarget,
    type StartupView,
  } from "./startupView";
  import { nextSidebarAfterToggle } from "./sidebarCommands";

  type SidebarId = "sessions" | "ptys" | "work" | "projects" | "notifications";
  type RailSide = "left" | "right";

  const SETTINGS_KEY = "whisk.ui.settings";
  const DAEMON_STATUS_EVENT = "daemon-status:changed";
  const STATUS_NOTIFICATION_ACTIVATED_EVENT = "status-notification:activated";

  type StatusNotificationActivation = {
    event?: StatusEvent;
  };

  let sessions: Session[] = [];
  let ptys: PTYInfo[] = [];
  let ptyHistory: PTYHistorySummary[] = [];
  let selectedPTYHistory: PTYHistory | null = null;
  let projects: Project[] = [];
  let projectDetail: ProjectDetail | null = null;
  let agentProfiles: AgentProfile[] = [];
  let workItems: WorkItem[] = [];
  let workItemLinks: WorkItemLink[] = [];
  let readyWork: ReadyWorkExplanation = emptyReadyWorkExplanation();
  let workItemRuns: WorkItemRun[] = [];
  let artifacts: Artifact[] = [];
  let questions: Question[] = [];
  let gateReports: GateReport[] = [];
  let workflowDefinitions: WorkflowDefinitionRecord[] = [];
  let workflowActionsByItem: Record<string, WorkflowActionAvailability[]> = {};
  let workflowMigrationPlan: WorkflowMigrationPlan | null = null;
  let workflowValidationReport: WorkflowValidationReport | null = null;
  let workflowEvents: WorkflowEvent[] = [];
  let statusEvents: StatusEvent[] = [];
  let agentBridgeApprovals: AgentBridgeApproval[] = [];
  let agentPrompts: AgentPrompt[] = [];
  let agentBridgeEvents: AgentBridgeEvent[] = [];
  let plugins: PluginStatus[] = [];
  let registryPlugins: RegistryPlugin[] = [];
  let installingPluginId = "";
  let agentHookIntegrations: AgentHookIntegration[] = [];
  let agentHookLogStatus: AgentHookLogStatus | null = null;
  let agentHookAction = "";
  let agentHookNotice = "";
  let commands: Command[] = [];
  let activeSessionId = "";
  let activePaneId = "";
  let activeProjectId = "";
  let workFilterQuery = "";
  let workFilterStageId = "";
  let workFilterRunState = "";
  let activeMain: MainView = "session";
  let workBoardOpenItemId = "";
  let navigationStack: MainView[] = [];

  function emptyReadyWorkExplanation(): ReadyWorkExplanation {
    return { ready: [], blocked: [], summary: { totalReady: 0, totalBlocked: 0, cycleCount: 0 } };
  }

  function currentNavigationState(): NavigationState {
    return { activeMain, navigationStack, workBoardOpenItemId };
  }

  function applyNavigationState(next: NavigationState) {
    activeMain = next.activeMain;
    navigationStack = next.navigationStack;
    workBoardOpenItemId = next.workBoardOpenItemId;
  }

  function syncNotificationFocusContext() {
    const focus = {
      activeMain,
      sessionId: activeSessionId,
      paneId: activePaneId,
      windowFocused,
    };
    const key = JSON.stringify(focus);
    if (key === lastNotificationFocusKey) return;
    lastNotificationFocusKey = key;
    void SetNotificationFocusContext(focus).catch(() => undefined);
  }

  function updateWindowFocusState() {
    windowFocused = document.hasFocus() && document.visibilityState !== "hidden";
    syncNotificationFocusContext();
  }

  function navigateTo(target: MainView, opts?: { openItemId?: string }) {
    applyNavigationState(navigateToState(currentNavigationState(), target, opts));
  }

  function navigateBack() {
    applyNavigationState(navigateBackState(currentNavigationState()));
  }

  function clearNavigationStack() {
    applyNavigationState(clearNavigationStackState(currentNavigationState()));
  }

  function selectMain(target: MainView) {
    applyNavigationState(selectMainView(currentNavigationState(), target));
  }
  let activeSidebar: SidebarId | null = "sessions";
  let commandPaletteOpen = false;
  let newProjectOpen = false;
  let newSessionOpen = false;
  let pendingSessionProjectId = "";
  let pendingSessionRootDir = "";
  let pendingSessionWorkingDir = "";
  let settingsOpen = false;
  let onboardingOpen = false;
  let onboardingBusy = false;
  let onboardingStatus: OnboardingStatus | null = null;
  let railSide: RailSide = "right";
  let startupView: StartupView = "sessions";
  let terminalFontSize = 13;
  let terminalCursorBlink = true;
  let closePanePromptDisabled = false;
  let closePaneDialogOpen = false;
  let closeDialogTitle = "";
  let closeDialogMessage = "";
  let pendingClosePaneTarget: ReturnType<typeof closePaneTarget> = null;
  let keepDaemonAlive = true;
  let autoRestartManagedDaemon = false;
  let hookLogEnabled = true;
  let clearHookLogAfterSession = false;
  let worktrunkPath = "/opt/homebrew/bin/wt";
  let error = "";
  let loadingSession = false;
  let loadingPtys = false;
  let loadingPTYHistory = false;
  let loadingWork = false;
  let loadingStatusEvents = false;
  let outputChunks: Record<string, string[]> = {};
  let outputChunkStartOffsets: Record<string, number[]> = {};
  let offsets: Record<string, number> = {};
  let bottomJumpRevisions: Record<string, number> = {};
  let daemonStatus: DaemonStatus | null = null;
  let daemonAddress = "";
  let daemonControlToken = "";
  let daemonLink = initialDaemonLinkSnapshot();
  let ptyStreams: Record<string, WebSocket> = {};
  let ptyTraceEnabled = false;
  let workReconcileTimer: number | undefined;
  let stopCommandEvents: (() => void) | undefined;
  let stopDaemonStatusEvents: (() => void) | undefined;
  let stopStatusNotificationEvents: (() => void) | undefined;
  let windowFocused = true;
  let lastNotificationFocusKey = "";
  let eventLoopRunning = false;
  let lastRuntimeEventSeq = 0;
  let settingsLoaded = false;
  let runtimeReadModelsLoaded = false;
  let stopped = false;
  const outputFetchInFlight = new Map<string, number>();
  const outputFetchAgain = new Set<string>();
  const ptyDialInFlight = new Set<string>();
  let ptyReconnectTimers: Record<string, number> = {};
  let ptyReconnectAttempts: Record<string, number> = {};

  $: activeSession = sessions.find((session) => session.id === activeSessionId) ?? null;
  $: activeSessionWindow = activeWindow(activeSession, activePaneId);
  $: activePtyId = activeSession?.panes[activePaneId]?.currentPtyId ?? "";
  $: visiblePTYIds = visiblePtyIds(sessions, activeSessionId, activePaneId);
  $: notificationCount = notificationSurfaceCount(statusEvents, agentPrompts, agentBridgeEvents);
  $: projectAttachmentTemplates = plugins.flatMap((plugin) =>
    plugin.trusted && plugin.valid
      ? (plugin.projectAttachmentTemplates ?? []).map((template) => ({ ...template, pluginId: plugin.id }))
      : [],
  );
  $: commands = [
    {
      id: "palette.open",
      title: "Open Command Palette",
      shortcut: "Cmd/Ctrl K",
      run: () => {
        commandPaletteOpen = true;
      },
    },
    {
      id: "sidebar.toggle",
      title: "Show/Hide Sidebar",
      shortcut: "Cmd/Ctrl \\",
      run: toggleCurrentSidebar,
    },
    {
      id: "notifications.clear",
      title: "Clear Notifications",
      enabled: () => statusEvents.length > 0 || agentBridgeEvents.some(isAgentHookNotification),
      run: clearNotifications,
    },
    {
      id: "notifications.refresh",
      title: "Refresh Notifications",
      run: refreshStatusEvents,
    },
    {
      id: "session.new",
      title: "New Session",
      run: openNewSession,
    },
    {
      id: "work.refresh",
      title: "Refresh Work",
      run: refreshProjects,
    },
    {
      id: "preferences.open",
      title: "Open Preferences",
      shortcut: "Cmd ,",
      run: openPreferences,
    },
    ...sessionSplitCommands({
      canSplit: Boolean(activeSession && activePaneId),
      canClose: Boolean(closePaneTarget(activeSession, activeSessionWindow?.id ?? "", activePaneId)),
      canCloseSession: Boolean(activeSession),
      split,
      close: () => closePane(activePaneId),
      closeSession: closeActiveSession,
    }),
    {
      id: "terminal.bottom",
      title: "Jump to Bottom",
      shortcut: "Cmd/Ctrl Alt Down",
      enabled: () => Boolean(activePtyId),
      run: jumpToBottom,
    },
    // Session-switch commands mirror the native Sessions menu (Cmd 1..0). They are gated on the
    // session count so only reachable slots appear in the palette.
    ...Array.from({ length: 10 }, (_, i) => ({
      id: `session.select.${i + 1}`,
      title: `Switch to Session ${i + 1}`,
      shortcut: `Cmd ${(i + 1) % 10}`,
      enabled: () => sessions.length > i,
      run: () => selectSessionByIndex(i),
    })),
  ] satisfies Command[];
  $: if (activeSession && (!activePaneId || !activeSession.panes[activePaneId])) {
    activePaneId = firstPaneId(activeSession);
  }

  function backendError(err: unknown): string {
    const message = err instanceof Error ? err.message : String(err);
    if (!message || message.includes("Not Found") || message.includes("Failed to fetch")) {
      return "Wails runtime is unavailable. Run `task dev:app` and use the macOS app window, not the Vite browser URL.";
    }
    return message;
  }

  function applyStartupView(view: StartupView) {
    const target = startupTarget(view);
    selectMain(target.main);
    activeSidebar = target.sidebar;
  }

  function loadLocalSettings() {
    try {
      const raw = localStorage.getItem(SETTINGS_KEY);
      if (!raw) return;
      const parsed = JSON.parse(raw) as Partial<{
        railSide: RailSide;
        terminalFontSize: number;
        terminalCursorBlink: boolean;
        closePanePromptDisabled: boolean;
      }>;
      if (parsed.railSide === "left" || parsed.railSide === "right") railSide = parsed.railSide;
      if (typeof parsed.terminalFontSize === "number") terminalFontSize = parsed.terminalFontSize;
      if (typeof parsed.terminalCursorBlink === "boolean") {
        terminalCursorBlink = parsed.terminalCursorBlink;
      }
      if (typeof parsed.closePanePromptDisabled === "boolean") {
        closePanePromptDisabled = parsed.closePanePromptDisabled;
      }
    } catch {
      return;
    }
  }

  async function loadSettings() {
    loadLocalSettings();
    try {
      const loaded = await LoadAppSettings();
      startupView = normalizeStartupView(loaded.startupView);
      keepDaemonAlive = loaded.keepDaemonAlive;
      if (typeof loaded.autoRestartManagedDaemon === "boolean") {
        autoRestartManagedDaemon = loaded.autoRestartManagedDaemon;
      }
      if (typeof loaded.hookLogEnabled === "boolean") hookLogEnabled = loaded.hookLogEnabled;
      if (typeof loaded.clearHookLogAfterSession === "boolean") {
        clearHookLogAfterSession = loaded.clearHookLogAfterSession;
      }
      if (typeof loaded.worktrunkPath === "string") worktrunkPath = loaded.worktrunkPath;
      applyStartupView(startupView);
      await SetAgentHookLogSettings({
        enabled: hookLogEnabled,
        clearAfterSession: clearHookLogAfterSession,
      });
    } catch (err) {
      error = `Load settings failed: ${backendError(err)}`;
    }
  }

  function saveSettings() {
    try {
      localStorage.setItem(
        SETTINGS_KEY,
        JSON.stringify({ railSide, terminalFontSize, terminalCursorBlink, closePanePromptDisabled }),
      );
    } catch {
      return;
    }
  }

  async function persistAppSettings() {
    try {
      const saved = await SaveAppSettings({
        startupView,
        keepDaemonAlive,
        autoRestartManagedDaemon,
        hookLogEnabled,
        clearHookLogAfterSession,
        worktrunkPath,
      });
      startupView = normalizeStartupView(saved.startupView);
      keepDaemonAlive = saved.keepDaemonAlive;
      autoRestartManagedDaemon =
        typeof saved.autoRestartManagedDaemon === "boolean" ? saved.autoRestartManagedDaemon : false;
      if (typeof saved.hookLogEnabled === "boolean") hookLogEnabled = saved.hookLogEnabled;
      if (typeof saved.clearHookLogAfterSession === "boolean") {
        clearHookLogAfterSession = saved.clearHookLogAfterSession;
      }
      if (typeof saved.worktrunkPath === "string") worktrunkPath = saved.worktrunkPath;
    } catch (err) {
      error = `Save settings failed: ${backendError(err)}`;
    }
  }

  async function setStartupView(view: StartupView) {
    startupView = view;
    await persistAppSettings();
  }

  async function setKeepDaemonAlive(keep: boolean) {
    keepDaemonAlive = keep;
    await persistAppSettings();
  }

  async function setAutoRestartManagedDaemon(enabled: boolean) {
    autoRestartManagedDaemon = enabled;
    await persistAppSettings();
  }

  async function setWorktrunkPath(path: string) {
    worktrunkPath = path;
    await persistAppSettings();
  }

  function applyDaemonStatus(status: DaemonStatus) {
    daemonStatus = status;
    if (status.address) daemonAddress = status.address;
    daemonControlToken = status.controlToken ?? "";
    const transition = nextDaemonLinkSnapshot(daemonLink, status);
    daemonLink = transition.snapshot;
    if (!transition.changed) return;
    if (transition.shouldResetEventCursor) lastRuntimeEventSeq = 0;
    closePTYStreamsForDaemonGeneration();
    if (runtimeReadModelsLoaded && transition.shouldReconcile) {
      void reconcileDaemonGeneration(transition.snapshot.generation).catch((err) => {
        error = backendError(err);
      });
    }
  }

  async function refreshDaemonStatus() {
    applyDaemonStatus(await LoadDaemonStatus());
  }

  async function refreshSessions() {
    sessions = await ListSessions();
    if (!activeSessionId && sessions.length > 0) {
      activeSessionId = sessions[0].id;
      activePaneId = firstPaneId(sessions[0]);
    }
    if (activeSessionId && !sessions.some((session) => session.id === activeSessionId)) {
      activeSessionId = sessions[0]?.id ?? "";
      activePaneId = firstPaneId(sessions[0]);
    }
  }

  async function refreshPTYs() {
    loadingPtys = true;
    loadingPTYHistory = true;
    try {
      const [nextPtys, nextHistory] = await Promise.all([ListPTYs(), ListPTYHistory()]);
      ptys = nextPtys;
      ptyHistory = nextHistory;
      outputChunkStartOffsets = Object.fromEntries(
        nextPtys.map((pty) => [pty.id, outputChunkStartOffsets[pty.id] ?? []]),
      );
      if (selectedPTYHistory && !nextHistory.some((item) => item.ptyId === selectedPTYHistory?.ptyId)) {
        selectedPTYHistory = null;
      }
    } finally {
      loadingPtys = false;
      loadingPTYHistory = false;
    }
  }

  async function selectPTYHistory(ptyId: string) {
    try {
      selectedPTYHistory = await ReadPTYHistory(ptyId);
    } catch (err) {
      error = `Load PTY history failed: ${backendError(err)}`;
    }
  }

  function jumpToBottom() {
    if (!activePtyId) return;
    bottomJumpRevisions = {
      ...bottomJumpRevisions,
      [activePtyId]: (bottomJumpRevisions[activePtyId] ?? 0) + 1,
    };
  }

  async function refreshProjects() {
    loadingWork = true;
    try {
      workflowDefinitions = await ListWorkflowDefinitions();
      projects = await ListProjects();
      if (!activeProjectId && projects.length > 0) {
        activeProjectId = projects[0].id;
      }
      if (activeProjectId && !projects.some((project) => project.id === activeProjectId)) {
        activeProjectId = projects[0]?.id ?? "";
      }
      await refreshProjectDetail();
      await refreshWorkState();
    } finally {
      loadingWork = false;
    }
  }

  async function refreshWorkflowDefinitions() {
    workflowDefinitions = await ListWorkflowDefinitions();
  }

  async function refreshProjectDetail() {
    if (!activeProjectId) {
      projectDetail = null;
      return;
    }
    projectDetail = projectDetailWithStoreSessions(
      await LoadProjectDetail(activeProjectId),
      activeProjectId,
      sessions,
    );
  }

  async function refreshAgentProfiles() {
    agentProfiles = await ListAgentProfiles();
  }

  function syncActiveProjectDetailSessions() {
    if (!projectDetail || projectDetail.project.id !== activeProjectId) return;
    projectDetail = projectDetailWithStoreSessions(projectDetail, activeProjectId, sessions);
  }

  async function refreshWorkItems() {
    if (!activeProjectId) {
      workItems = [];
      workflowActionsByItem = {};
      return;
    }
    workItems = await ListWorkItems(activeProjectId);
  }

  async function refreshWorkItemWorkflowActions() {
    if (!workItems.length) {
      workflowActionsByItem = {};
      return;
    }
    const entries = await Promise.all(
      workItems.map(async (item) => [item.id, await ListWorkItemWorkflowActions(item.id)] as const),
    );
    workflowActionsByItem = Object.fromEntries(entries);
  }

  async function refreshWorkItemLinks() {
    if (!activeProjectId) {
      workItemLinks = [];
      readyWork = emptyReadyWorkExplanation();
      return;
    }
    const [nextLinks, nextReadyWork] = await Promise.all([
      ListWorkItemLinks(""),
      ReadyWork({ projectId: activeProjectId }),
    ]);
    workItemLinks = nextLinks;
    readyWork = nextReadyWork;
  }

  async function refreshWorkItemRuns() {
    workItemRuns = await ListWorkItemRuns("");
  }

  async function refreshWorkflowRecords() {
    if (!activeProjectId) {
      artifacts = [];
      questions = [];
      gateReports = [];
      workflowEvents = [];
      return;
    }
    const [nextArtifacts, nextQuestions, nextGates, nextEvents] = await Promise.all([
      ListArtifacts(""),
      ListQuestions(""),
      ListGateReports(""),
      ListWorkflowEvents(""),
    ]);
    artifacts = nextArtifacts;
    questions = nextQuestions;
    gateReports = nextGates;
    workflowEvents = nextEvents;
  }

  async function refreshWorkState() {
    await refreshWorkItems();
    await refreshWorkItemWorkflowActions();
    await refreshWorkItemLinks();
    await refreshWorkItemRuns();
    await refreshWorkflowRecords();
  }

  async function refreshRuntimeReadModels() {
    await Promise.all([
      refreshSessions(),
      refreshPTYs(),
      refreshStatusEvents(),
      refreshWorkState(),
    ]);
    if (activeMain === "projects") await refreshProjectDetail();
  }

  async function refreshVisibleWorkState() {
    if (activeMain !== "work" && activeMain !== "projects" && activeSidebar !== "work") return;
    await Promise.all([
      refreshSessions(),
      refreshPTYs(),
      activeProjectId ? refreshWorkState() : Promise.resolve(),
      activeProjectId ? refreshProjectDetail() : Promise.resolve(),
    ]);
  }

  async function refreshStatusEvents() {
    loadingStatusEvents = true;
    try {
      const [nextStatusEvents, nextPrompts, nextApprovals, nextBridgeEvents] = await Promise.all([
        ListStatusEvents({ unreadOnly: true }),
        ListAgentPrompts({ status: "pending" }),
        ListAgentBridgeApprovals({ status: "pending" }),
        ListAgentBridgeEvents({ status: "pending" }),
      ]);
      statusEvents = nextStatusEvents;
      agentPrompts = nextPrompts;
      agentBridgeApprovals = nextApprovals;
      agentBridgeEvents = nextBridgeEvents;
    } finally {
      loadingStatusEvents = false;
    }
  }

  async function clearNotifications() {
    const hookNotifications = agentBridgeEvents.filter(isAgentHookNotification);
    if (statusEvents.length === 0 && hookNotifications.length === 0) return;
    loadingStatusEvents = true;
    try {
      await Promise.all([
        ...statusEvents.map((event) => MarkStatusEventRead({ id: event.id })),
        ...hookNotifications.map((event) => MarkAgentBridgeEventRead({ id: event.id })),
      ]);
      await refreshStatusEvents();
    } catch (err) {
      error = `Clear notifications failed: ${backendError(err)}`;
    } finally {
      loadingStatusEvents = false;
    }
  }

  async function clearAgentHookEvents() {
    if (agentBridgeEvents.length === 0) return;
    loadingStatusEvents = true;
    try {
      await Promise.all(agentBridgeEvents.map((event) => MarkAgentBridgeEventRead({ id: event.id })));
      await refreshStatusEvents();
    } catch (err) {
      error = `Clear hook events failed: ${backendError(err)}`;
    } finally {
      loadingStatusEvents = false;
    }
  }

  async function refreshVisibleOutput() {
    await Promise.all(visiblePTYIds.map((ptyId) => refreshOutput(ptyId).catch(handleOutputError)));
  }

  function handleOutputError(err: unknown) {
    if (isStalePTYError(err)) return;
    throw err;
  }

  async function refreshOutput(ptyId: string) {
    const generation = daemonLink.generation;
    const inFlightGeneration = outputFetchInFlight.get(ptyId);
    if (inFlightGeneration === generation) {
      outputFetchAgain.add(ptyId);
      return;
    }
    outputFetchInFlight.set(ptyId, generation);
    try {
      do {
        outputFetchAgain.delete(ptyId);
        const fromOffset = offsets[ptyId] ?? 0;
        const snapshot = await Output({
          ptyId,
          fromOffset,
        });
        if (!isCurrentDaemonGeneration(daemonLink, generation)) return;
        const currentOffset = offsets[ptyId] ?? 0;
        const snapshotChunk = outputSnapshotChunkAfterOffset(snapshot, currentOffset);
        if (snapshotChunk) {
          outputChunks = {
            ...outputChunks,
            [ptyId]: [...(outputChunks[ptyId] ?? []), snapshotChunk.outputBase64],
          };
          outputChunkStartOffsets = {
            ...outputChunkStartOffsets,
            [ptyId]: [...(outputChunkStartOffsets[ptyId] ?? []), snapshotChunk.startOffset],
          };
        }
        offsets = { ...offsets, [ptyId]: Math.max(currentOffset, snapshot.offset) };
      } while (outputFetchAgain.has(ptyId));
    } finally {
      if (outputFetchInFlight.get(ptyId) === generation) outputFetchInFlight.delete(ptyId);
    }
  }

  async function loadDaemonAddress() {
    if (!daemonLink.canUseDaemon) return "";
    if (daemonAddress) return daemonAddress;
    if (!daemonStatus) await refreshDaemonStatus();
    daemonAddress = daemonStatus?.address ?? "";
    return daemonAddress;
  }

  function hasActivePTYStream(ptyId: string) {
    const socket = ptyStreams[ptyId];
    return Boolean(socket && socket.readyState !== WebSocket.CLOSED && socket.readyState !== WebSocket.CLOSING);
  }

  function clearPTYReconnectTimer(ptyId: string) {
    const timer = ptyReconnectTimers[ptyId];
    if (timer === undefined) return;
    window.clearTimeout(timer);
    const { [ptyId]: _, ...remaining } = ptyReconnectTimers;
    ptyReconnectTimers = remaining;
  }

  function clearPTYReconnectTimers() {
    for (const timer of Object.values(ptyReconnectTimers)) window.clearTimeout(timer);
    ptyReconnectTimers = {};
  }

  function closePTYStreamsForDaemonGeneration() {
    clearPTYReconnectTimers();
    ptyDialInFlight.clear();
    ptyReconnectAttempts = {};
    const sockets = Object.values(ptyStreams);
    ptyStreams = {};
    for (const socket of sockets) socket.close();
  }

  function schedulePTYReconnect(ptyId: string, generation: number) {
    if (ptyReconnectTimers[ptyId] !== undefined) return;
    const attempt = ptyReconnectAttempts[ptyId] ?? 0;
    ptyReconnectAttempts = { ...ptyReconnectAttempts, [ptyId]: attempt + 1 };
    const timer = window.setTimeout(() => {
      const { [ptyId]: _, ...remaining } = ptyReconnectTimers;
      ptyReconnectTimers = remaining;
      if (stopped || !isCurrentDaemonGeneration(daemonLink, generation) || !visiblePTYIds.includes(ptyId)) return;
      void openPTYStream(ptyId).catch((err) => {
        if (!isStalePTYError(err)) error = backendError(err);
      });
    }, reconnectBackoffDelayMs(attempt));
    ptyReconnectTimers = { ...ptyReconnectTimers, [ptyId]: timer };
  }

  async function reconcileDaemonGeneration(generation: number) {
    if (!isCurrentDaemonGeneration(daemonLink, generation) || !daemonLink.canUseDaemon) return;
    await refreshRuntimeReadModels();
    if (!isCurrentDaemonGeneration(daemonLink, generation)) return;
    await refreshVisibleOutput();
    if (!isCurrentDaemonGeneration(daemonLink, generation)) return;
    syncPTYStreams(visiblePTYIds);
  }

  function appendPTYStreamOutput(frame: PTYStreamFrame) {
    if (frame.type !== "output") return;
    const currentOffset = offsets[frame.ptyId] ?? 0;
    const nextOffset = nextPTYStreamOffset(currentOffset, frame);
    if (nextOffset === currentOffset) return;
    outputChunks = {
      ...outputChunks,
      [frame.ptyId]: [...(outputChunks[frame.ptyId] ?? []), frame.outputBase64],
    };
    outputChunkStartOffsets = {
      ...outputChunkStartOffsets,
      [frame.ptyId]: [...(outputChunkStartOffsets[frame.ptyId] ?? []), frame.offset],
    };
    offsets = { ...offsets, [frame.ptyId]: nextOffset };
  }

  async function openPTYStream(ptyId: string) {
    if (!visiblePTYIds.includes(ptyId)) return;
    if (!daemonLink.canUseDaemon) return;
    const generation = daemonLink.generation;
    const dialClaimKey = `${generation}:${ptyId}`;
    if (!claimSingleFlight(ptyDialInFlight, dialClaimKey)) return;
    try {
      const existing = ptyStreams[ptyId];
      if (existing && existing.readyState !== WebSocket.CLOSED && existing.readyState !== WebSocket.CLOSING) return;
      const address = await loadDaemonAddress();
      if (!address || !isCurrentDaemonGeneration(daemonLink, generation) || !daemonLink.canUseDaemon) return;
      const socket = new WebSocket(ptyAttachWebSocketURL(address, ptyId, offsets[ptyId] ?? 0, daemonControlToken));
      ptyStreams = { ...ptyStreams, [ptyId]: socket };
      socket.onopen = () => {
        if (ptyStreams[ptyId] !== socket || !isCurrentDaemonGeneration(daemonLink, generation)) return;
        ptyReconnectAttempts = { ...ptyReconnectAttempts, [ptyId]: 0 };
      };
      socket.onmessage = (event) => {
        if (ptyStreams[ptyId] !== socket || !isCurrentDaemonGeneration(daemonLink, generation)) return;
        const frame = JSON.parse(String(event.data)) as PTYStreamFrame;
        if (frame.type === "output") {
          appendPTYStreamOutput(frame);
          ptyReconnectAttempts = { ...ptyReconnectAttempts, [ptyId]: 0 };
        } else if (frame.type === "error") {
          error = frame.message;
        }
      };
      socket.onclose = () => {
        if (ptyStreams[ptyId] === socket) {
          const { [ptyId]: _, ...remaining } = ptyStreams;
          ptyStreams = remaining;
        }
        if (!isCurrentDaemonGeneration(daemonLink, generation)) return;
        if (!stopped && daemonLink.canUseDaemon && visiblePTYIds.includes(ptyId)) {
          refreshOutput(ptyId).catch((err) => {
            if (!isStalePTYError(err)) error = backendError(err);
          });
          schedulePTYReconnect(ptyId, generation);
        }
      };
      socket.onerror = () => {
        socket.close();
      };
    } finally {
      releaseSingleFlight(ptyDialInFlight, dialClaimKey);
    }
  }

  function syncPTYStreams(ptyIds: string[]) {
    const visible = new Set(ptyIds);
    for (const [ptyId, socket] of Object.entries(ptyStreams)) {
      if (!visible.has(ptyId)) {
        clearPTYReconnectTimer(ptyId);
        socket.close();
      }
    }
    for (const ptyId of ptyIds) {
      void openPTYStream(ptyId).catch((err) => {
        error = backendError(err);
      });
    }
  }

  async function writePTYInput(ptyId: string, data: string) {
    const socket = ptyStreams[ptyId];
    if (writePTYInputOverSocket(socket, ptyId, data)) {
      if (ptyTraceEnabled) void LogPTYTrace(ptyInputTraceLine("frontend.websocket", ptyId, data, performance.now()));
      return;
    }
    if (!daemonLink.canUseDaemon) return;
    await WritePTY({ ptyId, data });
    if (ptyTraceEnabled) void LogPTYTrace(ptyInputTraceLine("frontend.missing-websocket", ptyId, data, performance.now()));
    if (!hasActivePTYStream(ptyId)) {
      void refreshOutput(ptyId).catch((err) => {
        if (!isStalePTYError(err)) error = backendError(err);
      });
    }
  }

  function openNewSession() {
    clearNavigationStack();
    error = "";
    settingsOpen = false;
    pendingSessionProjectId = "";
    pendingSessionRootDir = "";
    pendingSessionWorkingDir = "";
    selectMain("session");
    newSessionOpen = true;
  }

  function openNewProjectSession(projectId: string) {
    const project = projects.find((candidate) => candidate.id === projectId);
    if (!project) return;
    clearNavigationStack();
    error = "";
    settingsOpen = false;
    pendingSessionProjectId = project.id;
    pendingSessionRootDir = project.rootDir;
    pendingSessionWorkingDir = project.rootDir;
    newSessionOpen = true;
  }

  async function createSession(request: {
    name: string;
    rootDir: string;
    workingDir: string;
    initialPty: {
      cols: number;
      rows: number;
      command: string;
      agentBridge?: { enabled: boolean; provider: string };
    } | null;
  }) {
    error = "";
    loadingSession = true;
    try {
      const created = await CreateSession({
        name: request.name,
        rootDir: request.rootDir,
        workingDir: request.workingDir,
        projectId: pendingSessionProjectId,
        initialPty: request.initialPty,
      });
      sessions = [created.session, ...sessions.filter((session) => session.id !== created.session.id)];
      if (pendingSessionProjectId) {
        activeProjectId = pendingSessionProjectId;
        selectMain("projects");
        activeSidebar = null;
        await refreshProjects();
      } else {
        activeSessionId = created.session.id;
        activePaneId = created.paneId;
        selectMain("session");
        activeSidebar = "sessions";
      }
      pendingSessionProjectId = "";
      pendingSessionRootDir = "";
      pendingSessionWorkingDir = "";
      newSessionOpen = false;
      await refreshPTYs();
      if (created.ptyId) await refreshOutput(created.ptyId);
    } catch (err) {
      error = backendError(err);
    } finally {
      loadingSession = false;
    }
  }

  async function refreshAgentHookIntegrations() {
    const [integrations, logStatus] = await Promise.all([
      ListAgentHookIntegrations(),
      LoadAgentHookLogStatus(),
    ]);
    agentHookIntegrations = integrations;
    agentHookLogStatus = logStatus;
    hookLogEnabled = logStatus.enabled;
    clearHookLogAfterSession = logStatus.clearAfterSession;
  }

  async function refreshPlugins() {
    plugins = await ListPlugins();
  }

  async function refreshOnboarding(openIfNeeded = false) {
    onboardingStatus = await LoadOnboardingStatus();
    if (openIfNeeded && onboardingStatus.shouldShow) {
      onboardingOpen = true;
    }
  }

  async function applyOnboarding(itemIds: string[]) {
    onboardingBusy = true;
    error = "";
    try {
      onboardingStatus = await ApplyOnboarding({ itemIds });
      await Promise.all([refreshAgentHookIntegrations(), refreshPlugins()]);
      onboardingOpen = onboardingStatus.shouldShow;
    } catch (err) {
      error = `Onboarding failed: ${backendError(err)}`;
    } finally {
      onboardingBusy = false;
    }
  }

  async function rescanPlugins() {
    error = "";
    try {
      plugins = await RescanPlugins();
    } catch (err) {
      error = `Rescan plugins failed: ${backendError(err)}`;
    }
  }

  async function setPluginTrusted(pluginId: string, trusted: boolean) {
    error = "";
    try {
      const status = trusted ? await TrustPlugin(pluginId) : await UntrustPlugin(pluginId);
      plugins = plugins.map((plugin) => (plugin.id === pluginId ? status : plugin));
    } catch (err) {
      error = `Update plugin trust failed: ${backendError(err)}`;
    }
  }

  async function refreshRegistryPlugins() {
    error = "";
    try {
      registryPlugins = (await ListRegistryPlugins()) ?? [];
    } catch (err) {
      error = `List registry plugins failed: ${backendError(err)}`;
    }
  }

  async function installPlugin(registry: string, pluginId: string) {
    error = "";
    installingPluginId = `${registry}/${pluginId}`;
    try {
      await InstallPlugin(registry, pluginId);
      // Installed plugins land untrusted; refresh both the discovered set and
      // the registry so install state and trust toggles reflect the new plugin.
      await Promise.all([refreshPlugins(), refreshRegistryPlugins()]);
    } catch (err) {
      error = `Install plugin failed: ${backendError(err)}`;
    } finally {
      installingPluginId = "";
    }
  }

  async function setHookLogEnabled(enabled: boolean) {
    hookLogEnabled = enabled;
    await runAgentHookAction("hook-log:settings", async () => {
      agentHookLogStatus = await SetAgentHookLogSettings({ enabled });
      await SaveAppSettings({
        startupView,
        keepDaemonAlive,
        autoRestartManagedDaemon,
        hookLogEnabled: agentHookLogStatus.enabled,
        clearHookLogAfterSession,
        worktrunkPath,
      });
      agentHookNotice = `Hook logging ${agentHookLogStatus.enabled ? "enabled" : "disabled"}.`;
    });
  }

  async function setClearHookLogAfterSession(enabled: boolean) {
    clearHookLogAfterSession = enabled;
    await runAgentHookAction("hook-log:settings", async () => {
      agentHookLogStatus = await SetAgentHookLogSettings({ clearAfterSession: enabled });
      await SaveAppSettings({
        startupView,
        keepDaemonAlive,
        autoRestartManagedDaemon,
        hookLogEnabled,
        clearHookLogAfterSession: agentHookLogStatus.clearAfterSession,
        worktrunkPath,
      });
      agentHookNotice = `Clear hook log after session ${agentHookLogStatus.clearAfterSession ? "enabled" : "disabled"}.`;
    });
  }

  async function clearAgentHookLog() {
    await runAgentHookAction("hook-log:clear", async () => {
      agentHookLogStatus = await ClearAgentHookLog();
      agentHookNotice = "Hook log cleared.";
    });
  }

  async function openAgentHookLog() {
    await runAgentHookAction("hook-log:open", async () => {
      agentHookLogStatus = await OpenAgentHookLog();
      agentHookNotice = "Hook log opened.";
    });
  }

  async function copyAgentHookLogPath(path: string) {
    await runAgentHookAction("hook-log:copy", async () => {
      await navigator.clipboard.writeText(path);
      agentHookNotice = "Hook log path copied.";
    });
  }

  async function checkAgentHookIntegration(provider: string) {
    agentHookNotice = `Checking ${agentHookProviderLabel(provider)} hooks...`;
    await runAgentHookAction(`check:${provider}`, async () => {
      const integration = await CheckAgentHookIntegration({ provider });
      upsertAgentHookIntegration(integration, provider);
      await refreshAgentHookIntegrations();
      agentHookNotice = `${agentHookProviderLabel(provider)} hooks are ${integration.status}.`;
    });
  }

  async function installAgentHookIntegration(provider: string) {
    agentHookNotice = `Installing ${agentHookProviderLabel(provider)} hooks...`;
    await runAgentHookAction(`install:${provider}`, async () => {
      const integration = await InstallAgentHookIntegration({ provider });
      upsertAgentHookIntegration(integration, provider);
      await refreshAgentHookIntegrations();
      agentHookNotice = `${agentHookProviderLabel(provider)} hooks are ${integration.status}.`;
    });
  }

  async function removeAgentHookIntegration(provider: string) {
    agentHookNotice = `Removing ${agentHookProviderLabel(provider)} hooks...`;
    await runAgentHookAction(`remove:${provider}`, async () => {
      const integration = await RemoveAgentHookIntegration({ provider });
      upsertAgentHookIntegration(integration, provider);
      await refreshAgentHookIntegrations();
      agentHookNotice = `${agentHookProviderLabel(provider)} hooks are ${integration.status}.`;
    });
  }

  async function runAgentHookAction(action: string, run: () => Promise<void>) {
    error = "";
    agentHookAction = action;
    try {
      await run();
    } catch (err) {
      error = backendError(err);
      agentHookNotice = `Agent hook action failed: ${backendError(err)}`;
    } finally {
      agentHookAction = "";
    }
  }

  function upsertAgentHookIntegration(integration: AgentHookIntegration, provider: string) {
    agentHookIntegrations = upsertAgentHookIntegrationView(
      agentHookIntegrations,
      integration,
      provider,
    );
  }

  function agentHookProviderLabel(provider: string) {
    if (provider === "claude") return "Claude Code";
    if (provider === "codex") return "Codex";
    return provider;
  }

  async function split(direction: "horizontal" | "vertical") {
    if (!activeSession || !activePaneId) return;
    error = "";
    try {
      const result = await SplitPane({
        sessionId: activeSession.id,
        windowId: activeSessionWindow?.id ?? "",
        targetPaneId: activePaneId,
        direction,
        initialPty: { cols: 0, rows: 0 },
      });
      sessions = sessions.map((session) =>
        session.id === result.session.id ? result.session : session,
      );
      activePaneId = result.paneId;
      await refreshPTYs();
      if (result.ptyId) await refreshOutput(result.ptyId);
    } catch (err) {
      error = backendError(err);
    }
  }

  async function handleRuntimeEvent(event: RuntimeEvent) {
    const targets = runtimeRefreshTargets(event);
    if (targets.sessions) await refreshSessions();
    if (targets.ptys) await refreshPTYs();
    if (
      targets.outputPtyId &&
      !hasActivePTYStream(targets.outputPtyId)
    ) await refreshOutput(targets.outputPtyId);
    if (targets.statusEvents) await refreshStatusEvents();
    if (targets.agentBridgeApprovals) await refreshStatusEvents();
    if (targets.agentHookEvents) await refreshStatusEvents();
    if (targets.work) {
      await refreshWorkState();
      if (activeMain === "projects") await refreshProjectDetail();
    }
  }

  async function runEventLoop() {
    if (eventLoopRunning) return;
    eventLoopRunning = true;
    let backoffAttempt = 0;
    while (!stopped) {
      if (!daemonLink.canUseDaemon) {
        await new Promise((resolve) => window.setTimeout(resolve, reconnectBackoffDelayMs(backoffAttempt)));
        backoffAttempt += 1;
        continue;
      }
      const generation = daemonLink.generation;
      try {
        const response = await NextEvent({ timeoutMs: 30000, afterSeq: lastRuntimeEventSeq });
        if (!isCurrentDaemonGeneration(daemonLink, generation)) continue;
        backoffAttempt = 0;
        if (response.event.seq) lastRuntimeEventSeq = response.event.seq;
        if (response.missed) {
          await refreshRuntimeReadModels();
          await refreshVisibleOutput();
        }
        await handleRuntimeEvent(response.event);
      } catch {
        if (!stopped && isCurrentDaemonGeneration(daemonLink, generation)) {
          await new Promise((resolve) => window.setTimeout(resolve, reconnectBackoffDelayMs(backoffAttempt)));
          backoffAttempt += 1;
        }
      }
    }
  }

  function selectSession(session: Session) {
    selectMain("session");
    activeSessionId = session.id;
    activePaneId = firstPaneId(session);
    void refreshVisibleOutput().catch((err) => {
      error = backendError(err);
    });
  }

  // selectSessionByIndex activates the Nth session in the session bar (0-based), driven by the
  // Cmd+1..Cmd+0 native shortcuts. Out-of-range indices (fewer sessions than the pressed number)
  // are ignored.
  function selectSessionByIndex(index: number) {
    if (index >= 0 && index < sessions.length) {
      selectSession(sessions[index]);
    }
  }

  // syncSessionMenu pushes the given session list (in bar order) to the native menu so the Sessions
  // menu shows live names and the Cmd+1..Cmd+0 shortcuts map to the right sessions.
  function syncSessionMenu(list: Session[]) {
    void SyncSessionMenu(list.map((session) => ({ id: session.id, name: session.name }))).catch(
      (err) => {
        error = backendError(err);
      },
    );
  }

  function selectProject(projectId: string) {
    selectMain("work");
    activeProjectId = projectId;
    workFilterQuery = "";
    workFilterStageId = "";
    workFilterRunState = "";
    void refreshWorkState().catch((err) => {
      error = backendError(err);
    });
  }

  function selectProjectDetail(projectId: string) {
    selectMain("projects");
    activeSidebar = "projects";
    activeProjectId = projectId;
    void Promise.all([refreshProjectDetail(), refreshWorkState()]).catch((err) => {
      error = backendError(err);
    });
  }

  async function selectStatusEvent(event: StatusEvent) {
    const target = targetForStatusEvent(event, sessions);
    if (target.main === "work") {
      selectMain("work");
      activeSidebar = "work";
    } else {
      selectMain("session");
      if (target.sessionId) activeSessionId = target.sessionId;
      if (target.paneId) activePaneId = target.paneId;
    }
    try {
      await MarkStatusEventRead({ id: event.id });
      await refreshStatusEvents();
      if (target.main === "work") await refreshWorkState();
      if (target.main === "session") await refreshVisibleOutput();
    } catch (err) {
      error = backendError(err);
    }
  }

  async function selectStatusNotificationActivation(activation: StatusNotificationActivation) {
    if (!activation.event?.id) return;
    await selectStatusEvent(activation.event);
  }

  async function selectAgentBridgeEvent(event: AgentBridgeEvent) {
    const target = agentHookNotificationClickTarget(event, sessions);
    selectMain("session");
    if (target.sessionId) activeSessionId = target.sessionId;
    if (target.paneId) activePaneId = target.paneId;
    try {
      await MarkAgentBridgeEventRead({ id: target.readEventId });
      await refreshStatusEvents();
      await refreshVisibleOutput();
    } catch (err) {
      error = backendError(err);
    }
  }

  function paneIdForPty(session: Session | undefined, ptyId: string) {
    if (!session || !ptyId) return "";
    return Object.entries(session.panes).find(([, pane]) => pane?.currentPtyId === ptyId)?.[0] ?? "";
  }

  async function selectAgentPrompt(prompt: AgentPrompt) {
    selectMain("session");
    if (prompt.sessionId) activeSessionId = prompt.sessionId;
    const session = sessions.find((candidate) => candidate.id === prompt.sessionId);
    const paneId = paneIdForPty(session, prompt.ptyId || "");
    if (paneId) activePaneId = paneId;
    await refreshVisibleOutput();
  }

  async function resolveAgentBridgeApproval(approvalId: string, action: "allow" | "deny") {
    error = "";
    loadingStatusEvents = true;
    try {
      await ResolveAgentBridgeApproval(approvalId, { action });
      await refreshStatusEvents();
    } catch (err) {
      error = `Resolve approval failed: ${backendError(err)}`;
    } finally {
      loadingStatusEvents = false;
    }
  }

  async function resolveAgentPrompt(prompt: AgentPrompt, answer: string, tuiInput = "") {
    error = "";
    loadingStatusEvents = true;
    try {
      if (tuiInput && prompt.ptyId) await writePTYInput(prompt.ptyId, tuiInput);
      await ResolveAgentPrompt(prompt.id, { answer });
      await refreshStatusEvents();
      await refreshVisibleOutput();
    } catch (err) {
      error = `Resolve prompt failed: ${backendError(err)}`;
    } finally {
      loadingStatusEvents = false;
    }
  }

  async function openWorkItemRun(run: WorkItemRun) {
    try {
      const nextSessions = await ListSessions();
      sessions = nextSessions;
      const target = targetForStatusEvent(
        {
          id: run.id,
          kind: "run",
          sessionId: run.sessionId,
          ptyId: run.ptyId,
        },
        nextSessions,
      );
      selectMain("session");
      activeSidebar = "sessions";
      if (target.sessionId) activeSessionId = target.sessionId;
      if (target.paneId) activePaneId = target.paneId;
      if (run.ptyId) await refreshOutput(run.ptyId);
    } catch (err) {
      error = `Open run failed: ${backendError(err)}`;
    }
  }

  function openNewProject() {
    if (activeMain !== "projects") {
      selectMain("work");
      activeSidebar = "work";
    }
    newProjectOpen = true;
  }

  async function createProject(request: { name: string; description: string; rootDir: string }) {
    error = "";
    loadingWork = true;
    const nextMain = activeMain === "projects" ? "projects" : "work";
    try {
      const project = await CreateProject({
        name: request.name,
        description: request.description,
        rootDir: request.rootDir,
      });
      activeProjectId = project.id;
      await refreshProjects();
      selectMain(nextMain);
      activeSidebar = nextMain === "work" ? "work" : null;
      newProjectOpen = false;
    } catch (err) {
      error = `Create project failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function updateProject(projectId: string, request: { name: string; description: string }) {
    error = "";
    loadingWork = true;
    try {
      await UpdateProject(projectId, {
        name: request.name,
        description: request.description,
      });
      await refreshProjects();
    } catch (err) {
      error = `Update project failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function setProjectWorkflowDefinition(projectId: string, id: string, version: number) {
    if (!projectId || !id || version <= 0) return;
    error = "";
    loadingWork = true;
    try {
      const updatedProject = await SetProjectWorkflowDefinition(projectId, { id, version });
      workflowMigrationPlan = null;
      await refreshProjects();
      projects = projects.map((project) => (project.id === updatedProject.id ? updatedProject : project));
      if (projectDetail?.project.id === updatedProject.id) {
        projectDetail = { ...projectDetail, project: updatedProject };
      }
    } catch (err) {
      error = `Set project workflow failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function planProjectWorkflowMigration(projectId: string, id: string, version: number) {
    if (!projectId || !id || version <= 0) return;
    error = "";
    loadingWork = true;
    try {
      workflowMigrationPlan = await PlanProjectWorkflowMigration(projectId, { id, version });
    } catch (err) {
      error = `Plan workflow migration failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function validateWorkflowFile(path: string) {
    const trimmed = path.trim();
    if (!trimmed) return;
    error = "";
    loadingWork = true;
    try {
      workflowValidationReport = await ValidateWorkflowDefinitionFile({ path: trimmed });
    } catch (err) {
      error = `Validate workflow file failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function importWorkflowFile(path: string) {
    const trimmed = path.trim();
    if (!trimmed) return;
    error = "";
    loadingWork = true;
    try {
      await ImportWorkflowDefinitionFile({ path: trimmed });
      workflowValidationReport = null;
      await refreshWorkflowDefinitions();
    } catch (err) {
      error = `Import workflow file failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function exportWorkflowFile(id: string, version: number, path: string) {
    const trimmed = path.trim();
    if (!id || version <= 0 || !trimmed) return;
    error = "";
    loadingWork = true;
    try {
      await ExportWorkflowDefinitionFile({ id, version, path: trimmed });
    } catch (err) {
      error = `Export workflow file failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function deleteWorkflowDefinition(id: string, version: number) {
    if (!id || version <= 0) return;
    error = "";
    loadingWork = true;
    try {
      await DeleteWorkflowDefinition(id, version);
      workflowMigrationPlan = null;
      await refreshWorkflowDefinitions();
    } catch (err) {
      error = `Delete workflow definition failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function setPhaseAgent(projectId: string, preset: string, agentProfileId: string) {
    if (!projectId || !preset) return;
    error = "";
    loadingWork = true;
    try {
      await UpdateProject(projectId, { defaultPhaseAgents: { [preset]: agentProfileId } });
      await refreshProjects();
    } catch (err) {
      error = `Set phase agent failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function setInteractiveAgentShell(projectId: string, enabled: boolean) {
    if (!projectId) return;
    error = "";
    loadingWork = true;
    try {
      await UpdateProject(projectId, { useInteractiveAgentShell: enabled });
      await refreshProjects();
    } catch (err) {
      error = `Set agent shell failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function deleteProject(projectId: string) {
    error = "";
    loadingWork = true;
    try {
      await DeleteProject(projectId, {});
      if (activeProjectId === projectId) {
        activeProjectId = "";
        projectDetail = null;
        workItems = [];
        workItemLinks = [];
        readyWork = emptyReadyWorkExplanation();
        workItemRuns = [];
        artifacts = [];
        questions = [];
        gateReports = [];
        workflowEvents = [];
      }
      await refreshProjects();
    } catch (err) {
      error = `Delete project failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function createWorkItem(request: {
    projectId: string;
    title: string;
    bodyMarkdown: string;
  }) {
    error = "";
    loadingWork = true;
    try {
      await CreateWorkItem({
        projectId: request.projectId,
        title: request.title,
        bodyMarkdown: request.bodyMarkdown,
      });
      await refreshWorkState();
      if (activeMain === "projects") await refreshProjectDetail();
    } catch (err) {
      error = `Create work item failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function updateWorkItem(request: {
    id: string;
    title: string;
    bodyMarkdown: string;
  }) {
    error = "";
    loadingWork = true;
    try {
      await UpdateWorkItem({
        id: request.id,
        title: request.title,
        bodyMarkdown: request.bodyMarkdown,
      });
      await refreshWorkState();
      if (activeMain === "projects") await refreshProjectDetail();
    } catch (err) {
      error = `Update work item failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function moveWorkItem(workItemId: string, stageId: string) {
    error = "";
    loadingWork = true;
    try {
      await MoveWorkItem({ id: workItemId, stageId });
      await refreshWorkState();
    } catch (err) {
      error = `Move work item failed: ${backendError(err)}`;
      await refreshWorkItems().catch(() => undefined);
    } finally {
      loadingWork = false;
    }
  }

  async function addWorkItemLink(request: {
    sourceWorkItemId: string;
    targetWorkItemId: string;
    type: string;
  }) {
    error = "";
    loadingWork = true;
    try {
      await AddWorkItemLink(request);
      await refreshWorkState();
      if (activeMain === "projects") await refreshProjectDetail();
    } catch (err) {
      error = `Add dependency failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function generateWorktree(request: {
    workItemId: string;
    branch: string;
  }) {
    error = "";
    loadingWork = true;
    try {
      const item = workItems.find((candidate) => candidate.id === request.workItemId);
      const project = projects.find((candidate) => candidate.id === item?.projectId);
      if (!item || !project) {
        throw new Error("work item project not found");
      }
      const created = await CreateWorktree({
        repoPath: project.rootDir,
        branch: request.branch,
        base: "",
        overridePath: worktrunkPath,
      });
      await BindWorkItemWorktree({
        id: request.workItemId,
        branch: request.branch,
        worktreePath: created.path,
      });
      await refreshWorkState();
    } catch (err) {
      error = `Generate worktree failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function attachFile(workItemId: string, path: string) {
    error = "";
    loadingWork = true;
    try {
      await AddWorkItemAttachment({
        workItemId,
        kind: "file",
        scope: "external",
        path,
      });
      await refreshWorkState();
    } catch (err) {
      error = `Attach file failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function addProjectAttachment(request: {
    projectId: string;
    kind: string;
    title: string;
    path: string;
    url: string;
    note: string;
    provider: string;
    target: string;
    includeInContext: boolean;
  }) {
    error = "";
    loadingWork = true;
    try {
      await AddProjectAttachment({
        ...request,
        scope: request.kind === "file" ? "external" : "",
      });
      await Promise.all([refreshProjects(), refreshProjectDetail()]);
    } catch (err) {
      error = `Add attachment failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function deleteProjectAttachment(projectId: string, attachmentId: string) {
    error = "";
    loadingWork = true;
    try {
      await DeleteProjectAttachment(attachmentId, { projectId });
      await Promise.all([refreshProjects(), refreshProjectDetail()]);
    } catch (err) {
      error = `Delete attachment failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function updateProjectAttachment(
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
  ) {
    error = "";
    loadingWork = true;
    try {
      await UpdateProjectAttachment(attachmentId, { ...request, projectId });
      await Promise.all([refreshProjects(), refreshProjectDetail()]);
    } catch (err) {
      error = `Update attachment failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function runPluginProjectAttachmentTemplate(request: {
    pluginId: string;
    templateId: string;
    projectId: string;
    values: Record<string, string>;
  }) {
    error = "";
    loadingWork = true;
    try {
      await RunPluginProjectAttachmentTemplate(request.pluginId, request.templateId, {
        projectId: request.projectId,
        values: request.values,
      });
      await Promise.all([refreshProjects(), refreshProjectDetail()]);
    } catch (err) {
      error = `Add plugin attachment failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function deleteWorkItem(workItemId: string) {
    error = "";
    loadingWork = true;
    try {
      await DeleteWorkItem({ id: workItemId });
      await refreshWorkState();
      if (activeMain === "projects") await refreshProjectDetail();
    } catch (err) {
      error = `Delete work item failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function cancelWorkItemRun(runId: string) {
    error = "";
    loadingWork = true;
    try {
      await CancelWorkItemRun({ id: runId });
      await refreshWorkState();
    } catch (err) {
      error = `Cancel run failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function launchWorkItemRun(runId: string, agentProfileId = "") {
    error = "";
    loadingWork = true;
    try {
      await LaunchWorkItemRun({ id: runId, agentProfileId, worktreeOverridePath: worktrunkPath });
      await refreshWorkState();
      await refreshSessions();
      await refreshPTYs();
    } catch (err) {
      error = `Launch run failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function startPlanning(workItemId: string) {
    error = "";
    loadingWork = true;
    try {
      await StartPlanning({ workItemId, launch: true });
      await refreshWorkState();
      await refreshSessions();
      await refreshPTYs();
    } catch (err) {
      error = `Start planning failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function submitPlan(request: { workItemId: string; runId: string; title: string; body: string }) {
    error = "";
    loadingWork = true;
    try {
      await SubmitDraftPlan(request);
      await refreshWorkState();
    } catch (err) {
      error = `Submit plan failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function approvePlan(workItemId: string, artifactId: string) {
    error = "";
    loadingWork = true;
    try {
      await ApprovePlan({ workItemId, artifactId });
      await refreshWorkState();
    } catch (err) {
      error = `Approve plan failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function queueExecution(workItemId: string) {
    error = "";
    loadingWork = true;
    try {
      await QueueExecution({ workItemId });
      await refreshWorkState();
    } catch (err) {
      error = `Queue execution failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function launchExecution(workItemId: string, agentProfileId = "") {
    error = "";
    loadingWork = true;
    try {
      await LaunchExecution({ workItemId, agentProfileId, worktreeOverridePath: worktrunkPath });
      await refreshWorkState();
      await refreshSessions();
      await refreshPTYs();
    } catch (err) {
      error = `Launch execution failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function completeExecution(request: { workItemId: string; runId: string; message: string }) {
    error = "";
    loadingWork = true;
    try {
      await CompleteExecution(request);
      await refreshWorkState();
    } catch (err) {
      error = `Complete execution failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function submitReviewFeedback(request: { workItemId: string; runId: string; body: string }) {
    error = "";
    loadingWork = true;
    try {
      await SubmitReviewFeedback(request);
      await refreshWorkState();
    } catch (err) {
      error = `Submit feedback failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function askQuestion(request: { workItemId: string; runId: string; prompt: string }) {
    error = "";
    loadingWork = true;
    try {
      await AskQuestion(request);
      await refreshWorkState();
    } catch (err) {
      error = `Ask question failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function answerQuestion(questionId: string, answer: string) {
    error = "";
    loadingWork = true;
    try {
      await AnswerQuestion({ id: questionId, answer });
      await refreshWorkState();
    } catch (err) {
      error = `Answer question failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function completeGate(request: { id: string; status: string; overrideReason: string }) {
    error = "";
    loadingWork = true;
    try {
      await CompleteGate(request);
      await refreshWorkState();
    } catch (err) {
      error = `Complete gate failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function approveDone(workItemId: string, reason: string) {
    error = "";
    loadingWork = true;
    try {
      await ApproveDone({ workItemId, reason });
      await refreshWorkState();
    } catch (err) {
      error = `Approve done failed: ${backendError(err)}`;
    } finally {
      loadingWork = false;
    }
  }

  async function closeSession(session: Session) {
    await closeSessionById(session.id);
  }

  async function closeSessionById(sessionId: string) {
    error = "";
    loadingSession = true;
    try {
      sessions = await CloseSession({ sessionId });
      syncActiveProjectDetailSessions();
      if (activeSessionId === sessionId) {
        activeSessionId = sessions[0]?.id ?? "";
        activePaneId = firstPaneId(sessions[0]);
      }
      await refreshPTYs();
      await refreshVisibleOutput();
      if (activeMain === "projects") await refreshProjectDetail();
    } catch (err) {
      error = `Close session failed: ${backendError(err)}`;
    } finally {
      loadingSession = false;
    }
  }

  async function closePane(paneId: string) {
    const target = closePaneTarget(activeSession, activeSessionWindow?.id ?? "", paneId);
    if (!target) return;
    const ptyId = target.ptyId;
    if (ptyId && !closePanePromptDisabled) {
      openCloseConfirmation(
        target,
        "Close Terminal?",
        `The terminal still has a running process. If you close the terminal the process will be killed.\n\n${ptyId}`,
      );
      return;
    }
    await performClosePaneTarget(target);
  }

  async function closeActiveSession() {
    if (!activeSession) return;
    const ptyIds = Object.values(activeSession.panes)
      .map((pane) => pane?.currentPtyId ?? "")
      .filter((ptyId) => ptyId !== "");
    const target = { kind: "session" as const, sessionId: activeSession.id, ptyId: ptyIds[0] ?? "" };
    if (ptyIds.length > 0 && !closePanePromptDisabled) {
      openCloseConfirmation(
        target,
        "Close Session?",
        `The session has ${ptyIds.length} running terminal${ptyIds.length === 1 ? "" : "s"}. If you close the session ${ptyIds.length === 1 ? "the process" : "their processes"} will be killed.\n\n${ptyIds.slice(0, 3).join("\n")}${ptyIds.length > 3 ? `\n+${ptyIds.length - 3} more` : ""}`,
      );
      return;
    }
    await performClosePaneTarget(target);
  }

  function openCloseConfirmation(
    target: NonNullable<ReturnType<typeof closePaneTarget>>,
    title: string,
    message: string,
  ) {
    pendingClosePaneTarget = target;
    closeDialogTitle = title;
    closeDialogMessage = message;
    closePaneDialogOpen = true;
  }

  async function confirmClosePane(dontAskAgain: boolean) {
    const target = pendingClosePaneTarget;
    closePaneDialogOpen = false;
    pendingClosePaneTarget = null;
    closeDialogTitle = "";
    closeDialogMessage = "";
    if (!target) return;
    if (dontAskAgain) {
      closePanePromptDisabled = true;
      saveSettings();
    }
    await performClosePaneTarget(target);
  }

  function cancelClosePane() {
    closePaneDialogOpen = false;
    pendingClosePaneTarget = null;
    closeDialogTitle = "";
    closeDialogMessage = "";
  }

  async function performClosePaneTarget(target: NonNullable<ReturnType<typeof closePaneTarget>>) {
    if (target.kind === "session") {
      await closeSessionById(target.sessionId);
      return;
    }
    await performClosePane(target.request);
  }

  async function performClosePane(req: NonNullable<ReturnType<typeof closePaneRequest>>) {
    error = "";
    loadingSession = true;
    try {
      const updated = await ClosePane(req);
      sessions = sessions.map((session) => (session.id === updated.id ? updated : session));
      activePaneId = updated.panes[activePaneId] ? activePaneId : firstPaneId(updated);
      await refreshVisibleOutput();
    } catch (err) {
      error = `Close pane failed: ${backendError(err)}`;
    } finally {
      loadingSession = false;
    }
  }

  async function killPanePTY(paneId: string) {
    const req = killPTYRequest(activeSession?.panes[paneId]);
    if (!req) return;
    await killPTY(req.ptyId);
  }

  async function killPTY(ptyId: string) {
    if (!window.confirm(`Kill PTY ${ptyId}?`)) return;
    error = "";
    try {
      await KillPTY({ ptyId });
      await refreshPTYs();
      await refreshOutput(ptyId);
    } catch (err) {
      error = `Kill PTY failed: ${backendError(err)}`;
    }
  }

  async function deletePTY(ptyId: string) {
    if (!window.confirm(`Delete PTY ${ptyId}?`)) return;
    error = "";
    try {
      await DeletePTY({ ptyId });
      await refreshPTYs();
    } catch (err) {
      error = `Delete PTY failed: ${backendError(err)}`;
    }
  }

  async function setSessionProject(sessionId: string, projectId: string) {
    error = "";
    loadingSession = true;
    try {
      await SetSessionProject({ sessionId, projectId });
      await refreshSessions();
      syncActiveProjectDetailSessions();
      if (activeProjectId) await refreshProjectDetail();
    } catch (err) {
      error = `${projectId ? "Move" : "Remove"} session ${projectId ? "to" : "from"} project failed: ${backendError(err)}`;
    } finally {
      loadingSession = false;
    }
  }

  async function unassignProjectSession(sessionId: string) {
    await setSessionProject(sessionId, "");
  }

  async function openSessionById(sessionId: string) {
    await refreshSessions();
    const session = sessions.find((candidate) => candidate.id === sessionId);
    activeSessionId = sessionId;
    activePaneId = firstPaneId(session);
    selectMain("session");
    activeSidebar = "sessions";
    await refreshVisibleOutput().catch((err) => (error = backendError(err)));
  }

  function toggleSidebar(id: SidebarId) {
    if (id === "projects") {
      selectMain("projects");
      activeSidebar = activeSidebar === "projects" ? null : "projects";
      settingsOpen = false;
      void Promise.all([refreshProjects(), refreshProjectDetail()]).catch((err) => {
        error = backendError(err);
      });
      return;
    }
    if (id === "work") {
      selectMain("work");
      void refreshVisibleWorkState().catch((err) => {
        error = backendError(err);
      });
    }
    if (id === "sessions" || id === "ptys") selectMain("session");
    activeSidebar = activeSidebar === id ? null : id;
    settingsOpen = false;
  }

  function toggleCurrentSidebar() {
    clearNavigationStack();
    activeSidebar = nextSidebarAfterToggle(activeSidebar, activeMain);
    settingsOpen = false;
  }

  function toggleSettings() {
    settingsOpen = !settingsOpen;
    if (!settingsOpen) return;
    void Promise.all([refreshAgentHookIntegrations(), refreshPlugins(), refreshRegistryPlugins()]).catch((err) => {
      error = backendError(err);
    });
  }

  async function executeCommand(id: string) {
    try {
      const ran = await runCommand(commands, id);
      if (ran) commandPaletteOpen = false;
    } catch (err) {
      error = `Command failed: ${backendError(err)}`;
    }
  }

  // openPreferences opens the settings panel for the Cmd+, shortcut, without toggling it shut when
  // it is already open.
  function openPreferences() {
    if (!settingsOpen) {
      toggleSettings();
    }
  }

  function openOnboarding() {
    onboardingOpen = true;
    void refreshOnboarding(false).catch((err) => {
      error = backendError(err);
    });
  }

  onMount(() => {
    stopCommandEvents = Events.On("command:run", (event) => {
      void executeCommand(String(event.data));
    });
    stopDaemonStatusEvents = Events.On(DAEMON_STATUS_EVENT, (event) => {
      applyDaemonStatus(event.data as DaemonStatus);
    });
    stopStatusNotificationEvents = Events.On(STATUS_NOTIFICATION_ACTIVATED_EVENT, (event) => {
      void selectStatusNotificationActivation(event.data as StatusNotificationActivation);
    });
    updateWindowFocusState();
    window.addEventListener("focus", updateWindowFocusState);
    window.addEventListener("blur", updateWindowFocusState);
    document.addEventListener("visibilitychange", updateWindowFocusState);
    PTYTraceEnabled()
      .then((enabled) => {
        ptyTraceEnabled = enabled;
      })
      .catch(() => {
        ptyTraceEnabled = false;
      });
    loadSettings()
      .then(() => {
        settingsLoaded = true;
      })
      .then(refreshDaemonStatus)
      .then(refreshSessions)
      .then(refreshPTYs)
      .then(refreshPlugins)
      .then(() => refreshOnboarding(true))
      .then(refreshAgentProfiles)
      .then(refreshWorkflowDefinitions)
      .then(refreshProjects)
      .then(refreshStatusEvents)
      .then(() => {
        runtimeReadModelsLoaded = true;
      })
      .then(refreshVisibleOutput)
      .catch((err) => {
        error = backendError(err);
      });
    void runEventLoop();
    workReconcileTimer = window.setInterval(() => {
      refreshVisibleWorkState().catch((err) => {
        error = backendError(err);
      });
    }, 2000);
  });

  $: if (settingsLoaded) syncPTYStreams(visiblePTYIds);

  $: if (settingsLoaded) saveSettings();

  // Keep the native Sessions menu in step with the session bar whenever the list changes.
  $: if (settingsLoaded) syncSessionMenu(sessions);

  $: if (settingsLoaded) {
    activeMain;
    activeSessionId;
    activePaneId;
    windowFocused;
    syncNotificationFocusContext();
  }

  onDestroy(() => {
    stopped = true;
    stopCommandEvents?.();
    stopDaemonStatusEvents?.();
    stopStatusNotificationEvents?.();
    window.removeEventListener("focus", updateWindowFocusState);
    window.removeEventListener("blur", updateWindowFocusState);
    document.removeEventListener("visibilitychange", updateWindowFocusState);
    clearPTYReconnectTimers();
    if (workReconcileTimer) window.clearInterval(workReconcileTimer);
    for (const socket of Object.values(ptyStreams)) socket.close();
  });
</script>

<main class="flex h-screen flex-col overflow-hidden bg-bg-deep text-text-primary">
  <div class="flex min-h-0 flex-1 flex-row">
    <AppSidebar
      side={railSide}
      {activeMain}
      {activeSidebar}
      {settingsOpen}
      {notificationCount}
      {sessions}
      {ptys}
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
      onSidebar={toggleSidebar}
      onSettings={toggleSettings}
      onClose={() => (activeSidebar = null)}
      onNewSession={openNewSession}
      onSelectSession={selectSession}
      onCloseSession={closeSession}
      onSetSessionProject={(sessionId, projectId) => void setSessionProject(sessionId, projectId)}
      onRefreshPtys={() => void refreshPTYs()}
      onKillPTY={(ptyId) => void killPTY(ptyId)}
      onDeletePTY={(ptyId) => void deletePTY(ptyId)}
      onSelectPTYHistory={(ptyId) => void selectPTYHistory(ptyId)}
      onRefreshStatusEvents={() => void refreshStatusEvents()}
      onClearNotifications={() => void executeCommand("notifications.clear")}
      onSelectStatusEvent={(event) => void selectStatusEvent(event)}
      onSelectAgentPrompt={(prompt) => void selectAgentPrompt(prompt)}
      onSelectAgentBridgeEvent={(event) => void selectAgentBridgeEvent(event)}
      onResolveAgentBridgeApproval={(id, action) => void resolveAgentBridgeApproval(id, action)}
      onResolveAgentPrompt={(prompt, answer, tuiInput) => void resolveAgentPrompt(prompt, answer, tuiInput)}
      onRefreshWork={() => void refreshProjects()}
      onNewProject={openNewProject}
      onSelectProject={selectProject}
      onSelectProjectDetail={selectProjectDetail}
      onCreateWorkItem={createWorkItem}
    />

    <section class="relative flex min-w-0 flex-1 flex-col overflow-hidden bg-bg-deep">
      <header class="flex h-10 shrink-0 items-center justify-between border-b border-hairline bg-bg-base/80 px-3">
        <div class="min-w-0">
          <div class="truncate text-[13px] font-semibold text-text-primary">
            {activeMain === "projects" ? "Projects" : activeMain === "work" ? "Work" : (activeSession?.name ?? "No active session")}
          </div>
          <div class="truncate font-mono text-[10px] text-text-muted">
            {activeMain === "work" || activeMain === "projects" ? (activeProjectId || "No project selected") : (activePaneId || "No pane selected")}
          </div>
        </div>
      </header>

      {#if error}
        <div
          class="absolute bottom-3 right-3 z-30 max-w-[520px] rounded border border-red/30 bg-red/10 px-3 py-2 text-[12px] text-red shadow-lg shadow-black/30"
        >
          {error}
        </div>
      {/if}

      <div class="relative flex min-h-0 flex-1 flex-col">
        <MainRouter
          {activeMain}
          {activeSession}
          {activeSessionWindow}
          {outputChunks}
          {outputChunkStartOffsets}
          {bottomJumpRevisions}
          {activePaneId}
          {terminalFontSize}
          {terminalCursorBlink}
          {loadingSession}
          {projects}
          projectDetail={projectDetail}
          {activeProjectId}
          {loadingWork}
          {workBoardOpenItemId}
          canNavigateBack={navigationStack.length > 0}
          {workItems}
          {workItemLinks}
          {readyWork}
          {workItemRuns}
          {artifacts}
          {questions}
          {gateReports}
          {workflowDefinitions}
          {workflowActionsByItem}
          {workflowMigrationPlan}
          {workflowValidationReport}
          {workflowEvents}
          {agentProfiles}
          {workFilterQuery}
          {workFilterStageId}
          {workFilterRunState}
          pluginAttachmentTemplates={projectAttachmentTemplates}
          onUpdateProject={(projectId, request) => void updateProject(projectId, request)}
          onSetProjectWorkflowDefinition={(projectId, id, version) => void setProjectWorkflowDefinition(projectId, id, version)}
          onPlanProjectWorkflowMigration={(projectId, id, version) => void planProjectWorkflowMigration(projectId, id, version)}
          onValidateWorkflowFile={(path) => void validateWorkflowFile(path)}
          onImportWorkflowFile={(path) => void importWorkflowFile(path)}
          onExportWorkflowFile={(id, version, path) => void exportWorkflowFile(id, version, path)}
          onDeleteWorkflowDefinition={(id, version) => void deleteWorkflowDefinition(id, version)}
          onDeleteProject={(projectId) => void deleteProject(projectId)}
          onNewProjectSession={(projectId) => openNewProjectSession(projectId)}
          onOpenSession={(sessionId) => void openSessionById(sessionId)}
          onRemoveSession={(sessionId) => void unassignProjectSession(sessionId)}
          onCreateWorkItem={createWorkItem}
          onDeleteWorkItem={deleteWorkItem}
          onOpenWorkItem={(workItemId) => navigateTo("work", { openItemId: workItemId })}
          onOpenRunTerminal={(run) => void openWorkItemRun(run)}
          onAddProjectAttachment={addProjectAttachment}
          onRunPluginProjectAttachmentTemplate={runPluginProjectAttachmentTemplate}
          onUpdateProjectAttachment={updateProjectAttachment}
          onDeleteProjectAttachment={deleteProjectAttachment}
          onDetailClose={navigateBack}
          onRefreshWork={() => void refreshProjects()}
          onUpdateWorkItem={updateWorkItem}
          onMoveWorkItem={moveWorkItem}
          onAddWorkItemLink={addWorkItemLink}
          onGenerateWorktree={generateWorktree}
          onAttachFile={attachFile}
          onCancelRun={cancelWorkItemRun}
          onLaunchRun={launchWorkItemRun}
          onStartPlanning={startPlanning}
          onSubmitPlan={submitPlan}
          onApprovePlan={approvePlan}
          onQueueExecution={queueExecution}
          onLaunchExecution={launchExecution}
          onSetPhaseAgent={setPhaseAgent}
          onSetInteractiveAgentShell={setInteractiveAgentShell}
          onCompleteExecution={completeExecution}
          onSubmitReviewFeedback={submitReviewFeedback}
          onAskQuestion={askQuestion}
          onAnswerQuestion={answerQuestion}
          onCompleteGate={completeGate}
          onApproveDone={approveDone}
          onFocusPane={(paneId) => (activePaneId = paneId)}
          onPtyInput={(ptyId) => refreshOutput(ptyId).catch((err) => {
            if (!isStalePTYError(err)) error = backendError(err);
          })}
          onWriteInput={writePTYInput}
          onClosePane={(paneId) => void closePane(paneId)}
          onKillPanePTY={(paneId) => void killPanePTY(paneId)}
          canClosePane={(paneId) =>
            Boolean(closePaneTarget(activeSession, activeSessionWindow?.id ?? "", paneId))}
          onNewSession={openNewSession}
        />

        <NewSessionDialog
          visible={newSessionOpen}
          loading={loadingSession}
          initialRootDir={pendingSessionRootDir}
          initialWorkingDir={pendingSessionWorkingDir}
          onclose={() => (newSessionOpen = false)}
          oncreate={createSession}
        />

        <NewProjectDialog
          visible={newProjectOpen}
          loading={loadingWork}
          onclose={() => (newProjectOpen = false)}
          oncreate={createProject}
        />

        <ConfirmDialog
          visible={closePaneDialogOpen}
          title={closeDialogTitle}
          message={closeDialogMessage}
          cancelLabel="Cancel"
          confirmLabel="Close"
          checkboxLabel="Do not ask again"
          oncancel={cancelClosePane}
          onconfirm={(checked) => void confirmClosePane(checked)}
        />

        <SettingsView
          visible={settingsOpen}
          {railSide}
          {startupView}
          {terminalFontSize}
          {terminalCursorBlink}
          {keepDaemonAlive}
          {autoRestartManagedDaemon}
          {daemonStatus}
          {worktrunkPath}
          {agentHookIntegrations}
          {plugins}
          {registryPlugins}
          {installingPluginId}
          {agentHookLogStatus}
          {agentBridgeEvents}
          {agentHookAction}
          {agentHookNotice}
          onclose={() => (settingsOpen = false)}
          onRailSide={(side) => (railSide = side)}
          onStartupView={(view) => void setStartupView(view)}
          onTerminalFontSize={(size) => (terminalFontSize = size)}
          onTerminalCursorBlink={(blink) => (terminalCursorBlink = blink)}
          onKeepDaemonAlive={(keep) => void setKeepDaemonAlive(keep)}
          onAutoRestartManagedDaemon={(enabled) => void setAutoRestartManagedDaemon(enabled)}
          onDaemonStatus={applyDaemonStatus}
          onWorktrunkPath={(path) => void setWorktrunkPath(path)}
          onRefreshAgentHookIntegrations={() => void refreshAgentHookIntegrations()}
          onRefreshPlugins={() => void rescanPlugins()}
          onSetPluginTrusted={(pluginId, trusted) => void setPluginTrusted(pluginId, trusted)}
          onRefreshRegistry={() => void refreshRegistryPlugins()}
          onInstallPlugin={(registry, pluginId) => void installPlugin(registry, pluginId)}
          onCheckAgentHookIntegration={(provider) => void checkAgentHookIntegration(provider)}
          onInstallAgentHookIntegration={(provider) => void installAgentHookIntegration(provider)}
          onRemoveAgentHookIntegration={(provider) => void removeAgentHookIntegration(provider)}
          onHookLogEnabled={(enabled) => void setHookLogEnabled(enabled)}
          onClearHookLogAfterSession={(enabled) => void setClearHookLogAfterSession(enabled)}
          onClearAgentHookLog={() => void clearAgentHookLog()}
          onClearAgentHookEvents={() => void clearAgentHookEvents()}
          onOpenAgentHookLog={() => void openAgentHookLog()}
          onCopyAgentHookLogPath={(path) => void copyAgentHookLogPath(path)}
          onRefreshAgentHookEvents={() => void refreshStatusEvents()}
          onRunOnboarding={openOnboarding}
        />

        <OnboardingPanel
          visible={onboardingOpen}
          status={onboardingStatus}
          busy={onboardingBusy}
          onclose={() => (onboardingOpen = false)}
          onapply={(ids) => void applyOnboarding(ids)}
        />
      </div>
    </section>
  </div>
  <CommandPalette
    visible={commandPaletteOpen}
    {commands}
    onclose={() => (commandPaletteOpen = false)}
    onrun={(id) => void executeCommand(id)}
  />
</main>
