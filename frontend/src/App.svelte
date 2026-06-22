<script lang="ts">
  import { Events } from "@wailsio/runtime";
  import { onDestroy, onMount } from "svelte";
  import type { Session } from "../bindings/github.com/phin-tech/whisk/internal/domain/session/models";
  import type {
    AgentBridgeApproval,
    AgentBridgeEvent,
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
    WorkItemRun,
    WorkflowEvent,
  } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import {
    AddProjectAttachment,
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
    ListGateReports,
    ListPTYHistory,
    ListPTYs,
    ProjectDetail as LoadProjectDetail,
    ListProjects,
    ListQuestions,
    ListSessions,
    ListStatusEvents,
    ListWorkItemRuns,
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
    QueueExecution,
    RemoveAgentHookIntegration,
    RescanPlugins,
    ResolveAgentBridgeApproval,
    ResolveAgentPrompt,
    ReadPTYHistory,
    RunPluginProjectAttachmentTemplate,
    SaveAppSettings,
    SetAgentHookLogSettings,
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
    WritePTY,
    ListPlugins,
    OnboardingStatus as LoadOnboardingStatus,
    ListRegistryPlugins,
    InstallPlugin,
  } from "../bindings/github.com/phin-tech/whisk/internal/wailsapp/service";
  import ActivityRail from "./ActivityRail.svelte";
  import CommandPalette from "./CommandPalette.svelte";
  import ConfirmDialog from "./ConfirmDialog.svelte";
  import LayoutView from "./LayoutView.svelte";
  import NewProjectDialog from "./NewProjectDialog.svelte";
  import NewSessionDialog from "./NewSessionDialog.svelte";
  import OnboardingPanel from "./OnboardingPanel.svelte";
  import ProjectsView from "./ProjectsView.svelte";
  import SettingsView from "./SettingsView.svelte";
  import SidebarDock from "./SidebarDock.svelte";
  import WorkBoard from "./WorkBoard.svelte";
  import { agentHookNotificationClickTarget, isAgentHookNotification, upsertAgentHookIntegration as upsertAgentHookIntegrationView } from "./agentHooksView";
  import type { Command } from "./commands";
  import { runCommand } from "./commands";
  import { notificationSurfaceCount, targetForStatusEvent } from "./notificationsView";
  import {
    nextPTYStreamOffset,
    ptyAttachWebSocketURL,
    ptyInputTraceLine,
    type PTYStreamFrame,
    writePTYInputOverSocket,
  } from "./ptyStream";
  import { commandIdForShortcut, sessionSplitCommands } from "./sessionCommands";
  import { activeWindow, closePaneRequest, closePaneTarget, firstPaneId, isStalePTYError, killPTYRequest, runtimeRefreshTargets, visiblePtyIds } from "./sessionView";
  import {
    normalizeStartupView,
    startupTarget,
    type StartupView,
  } from "./startupView";

  type SidebarId = "sessions" | "ptys" | "work" | "projects" | "notifications";
  type MainView = "session" | "work" | "projects";
  type RailSide = "left" | "right";

  const SETTINGS_KEY = "whisk.ui.settings";

  let sessions: Session[] = [];
  let ptys: PTYInfo[] = [];
  let ptyHistory: PTYHistorySummary[] = [];
  let selectedPTYHistory: PTYHistory | null = null;
  let projects: Project[] = [];
  let projectDetail: ProjectDetail | null = null;
  let workItems: WorkItem[] = [];
  let workItemRuns: WorkItemRun[] = [];
  let artifacts: Artifact[] = [];
  let questions: Question[] = [];
  let gateReports: GateReport[] = [];
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
  let hookLogEnabled = true;
  let clearHookLogAfterSession = false;
  let error = "";
  let loadingSession = false;
  let loadingPtys = false;
  let loadingPTYHistory = false;
  let loadingWork = false;
  let loadingStatusEvents = false;
  let outputChunks: Record<string, string[]> = {};
  let offsets: Record<string, number> = {};
  let daemonAddress = "";
  let ptyStreams: Record<string, WebSocket> = {};
  let ptyTraceEnabled = false;
  let outputReconcileTimer: number | undefined;
  let workReconcileTimer: number | undefined;
  let stopCommandEvents: (() => void) | undefined;
  let eventLoopRunning = false;
  let settingsLoaded = false;
  let stopped = false;
  const textEncoder = new TextEncoder();
  const outputFetchInFlight = new Set<string>();
  const outputFetchAgain = new Set<string>();

  $: activeSession = sessions.find((session) => session.id === activeSessionId) ?? null;
  $: activeSessionWindow = activeWindow(activeSession, activePaneId);
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
    activeMain = target.main;
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
      if (typeof loaded.hookLogEnabled === "boolean") hookLogEnabled = loaded.hookLogEnabled;
      if (typeof loaded.clearHookLogAfterSession === "boolean") {
        clearHookLogAfterSession = loaded.clearHookLogAfterSession;
      }
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
        hookLogEnabled,
        clearHookLogAfterSession,
      });
      startupView = normalizeStartupView(saved.startupView);
      keepDaemonAlive = saved.keepDaemonAlive;
      if (typeof saved.hookLogEnabled === "boolean") hookLogEnabled = saved.hookLogEnabled;
      if (typeof saved.clearHookLogAfterSession === "boolean") {
        clearHookLogAfterSession = saved.clearHookLogAfterSession;
      }
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

  async function refreshProjects() {
    loadingWork = true;
    try {
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

  async function refreshProjectDetail() {
    if (!activeProjectId) {
      projectDetail = null;
      return;
    }
    projectDetail = await LoadProjectDetail(activeProjectId);
  }

  function syncActiveProjectDetailSessions() {
    if (!projectDetail || projectDetail.project.id !== activeProjectId) return;
    projectDetail = {
      ...projectDetail,
      sessions: sessions.filter((session) => session.projectId === activeProjectId),
    };
  }

  async function refreshWorkItems() {
    if (!activeProjectId) {
      workItems = [];
      return;
    }
    workItems = await ListWorkItems(activeProjectId);
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
    await refreshWorkItemRuns();
    await refreshWorkflowRecords();
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
      const [nextStatusEvents, nextPrompts, nextBridgeEvents] = await Promise.all([
        ListStatusEvents({ unreadOnly: true }),
        ListAgentPrompts({ status: "pending" }),
        ListAgentBridgeEvents({ status: "pending" }),
      ]);
      statusEvents = nextStatusEvents;
      agentPrompts = nextPrompts;
      agentBridgeApprovals = [];
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
    if (outputFetchInFlight.has(ptyId)) {
      outputFetchAgain.add(ptyId);
      return;
    }
    outputFetchInFlight.add(ptyId);
    try {
      do {
        outputFetchAgain.delete(ptyId);
        const snapshot = await Output({
          ptyId,
          fromOffset: offsets[ptyId] ?? 0,
        });
        if (snapshot.outputBase64) {
          outputChunks = {
            ...outputChunks,
            [ptyId]: [...(outputChunks[ptyId] ?? []), snapshot.outputBase64],
          };
        } else if (snapshot.output) {
          const bytes = textEncoder.encode(snapshot.output);
          let binary = "";
          for (const byte of bytes) binary += String.fromCharCode(byte);
          outputChunks = {
            ...outputChunks,
            [ptyId]: [...(outputChunks[ptyId] ?? []), btoa(binary)],
          };
        }
        offsets = { ...offsets, [ptyId]: snapshot.offset };
      } while (outputFetchAgain.has(ptyId));
    } finally {
      outputFetchInFlight.delete(ptyId);
    }
  }

  async function loadDaemonAddress() {
    if (daemonAddress) return daemonAddress;
    const status = await LoadDaemonStatus();
    daemonAddress = status.address;
    return daemonAddress;
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
    offsets = { ...offsets, [frame.ptyId]: nextOffset };
  }

  async function openPTYStream(ptyId: string) {
    if (!visiblePTYIds.includes(ptyId)) return;
    const existing = ptyStreams[ptyId];
    if (existing && existing.readyState !== WebSocket.CLOSED && existing.readyState !== WebSocket.CLOSING) return;
    const address = await loadDaemonAddress();
    const socket = new WebSocket(ptyAttachWebSocketURL(address, ptyId, offsets[ptyId] ?? 0));
    ptyStreams = { ...ptyStreams, [ptyId]: socket };
    socket.onmessage = (event) => {
      const frame = JSON.parse(String(event.data)) as PTYStreamFrame;
      if (frame.type === "output") {
        appendPTYStreamOutput(frame);
      } else if (frame.type === "error") {
        error = frame.message;
      }
    };
    socket.onclose = () => {
      if (ptyStreams[ptyId] === socket) {
        const { [ptyId]: _, ...remaining } = ptyStreams;
        ptyStreams = remaining;
      }
      if (!stopped && visiblePTYIds.includes(ptyId)) {
        refreshOutput(ptyId).catch((err) => {
          if (!isStalePTYError(err)) error = backendError(err);
        });
        window.setTimeout(() => {
          openPTYStream(ptyId).catch((err) => {
            if (!isStalePTYError(err)) error = backendError(err);
          });
        }, 500);
      }
    };
    socket.onerror = () => {
      socket.close();
    };
  }

  function syncPTYStreams(ptyIds: string[]) {
    const visible = new Set(ptyIds);
    for (const [ptyId, socket] of Object.entries(ptyStreams)) {
      if (!visible.has(ptyId)) {
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
    await WritePTY({ ptyId, data });
    if (ptyTraceEnabled) void LogPTYTrace(ptyInputTraceLine("frontend.missing-websocket", ptyId, data, performance.now()));
  }

  function openNewSession() {
    error = "";
    settingsOpen = false;
    pendingSessionProjectId = "";
    pendingSessionRootDir = "";
    pendingSessionWorkingDir = "";
    activeMain = "session";
    newSessionOpen = true;
  }

  function openNewProjectSession(projectId: string) {
    const project = projects.find((candidate) => candidate.id === projectId);
    if (!project) return;
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
        activeMain = "projects";
        activeSidebar = null;
        await refreshProjects();
      } else {
        activeSessionId = created.session.id;
        activePaneId = created.paneId;
        activeMain = "session";
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
        hookLogEnabled: agentHookLogStatus.enabled,
        clearHookLogAfterSession,
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
        hookLogEnabled,
        clearHookLogAfterSession: agentHookLogStatus.clearAfterSession,
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
      !ptyStreams[targets.outputPtyId]
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
    while (!stopped) {
      try {
        const event = await NextEvent({ timeoutMs: 30000 });
        await handleRuntimeEvent(event);
      } catch {
        if (!stopped) {
          await new Promise((resolve) => window.setTimeout(resolve, 250));
        }
      }
    }
  }

  function selectSession(session: Session) {
    activeMain = "session";
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
    activeMain = "work";
    activeProjectId = projectId;
    workFilterQuery = "";
    workFilterStageId = "";
    workFilterRunState = "";
    void refreshWorkState().catch((err) => {
      error = backendError(err);
    });
  }

  function selectProjectDetail(projectId: string) {
    activeMain = "projects";
    activeSidebar = "projects";
    activeProjectId = projectId;
    void Promise.all([refreshProjectDetail(), refreshWorkState()]).catch((err) => {
      error = backendError(err);
    });
  }

  async function selectStatusEvent(event: StatusEvent) {
    const target = targetForStatusEvent(event, sessions);
    if (target.main === "work") {
      activeMain = "work";
      activeSidebar = "work";
    } else {
      activeMain = "session";
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

  async function selectAgentBridgeEvent(event: AgentBridgeEvent) {
    const target = agentHookNotificationClickTarget(event, sessions);
    activeMain = "session";
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
    activeMain = "session";
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
      activeMain = "session";
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
      activeMain = "work";
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
      activeMain = nextMain;
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

  async function deleteProject(projectId: string) {
    error = "";
    loadingWork = true;
    try {
      await DeleteProject(projectId, {});
      if (activeProjectId === projectId) {
        activeProjectId = "";
        projectDetail = null;
        workItems = [];
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

  async function launchWorkItemRun(runId: string) {
    error = "";
    loadingWork = true;
    try {
      await LaunchWorkItemRun({ id: runId });
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

  async function launchExecution(workItemId: string) {
    error = "";
    loadingWork = true;
    try {
      await LaunchExecution({ workItemId });
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
    activeMain = "session";
    activeSidebar = "sessions";
    await refreshVisibleOutput().catch((err) => (error = backendError(err)));
  }

  function toggleSidebar(id: SidebarId) {
    if (id === "projects") {
      activeMain = "projects";
      activeSidebar = activeSidebar === "projects" ? null : "projects";
      settingsOpen = false;
      void Promise.all([refreshProjects(), refreshProjectDetail()]).catch((err) => {
        error = backendError(err);
      });
      return;
    }
    if (id === "work") {
      activeMain = "work";
      void refreshVisibleWorkState().catch((err) => {
        error = backendError(err);
      });
    }
    if (id === "sessions" || id === "ptys") activeMain = "session";
    activeSidebar = activeSidebar === id ? null : id;
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

  function handleCommandKey(event: KeyboardEvent) {
    const commandId = commandIdForShortcut(event);
    if (commandId) {
      event.preventDefault();
      void executeCommand(commandId);
      return;
    }
    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
      event.preventDefault();
      void executeCommand("palette.open");
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
      .then(refreshSessions)
      .then(refreshPTYs)
      .then(refreshPlugins)
      .then(() => refreshOnboarding(true))
      .then(refreshProjects)
      .then(refreshStatusEvents)
      .then(refreshVisibleOutput)
      .catch((err) => {
        error = backendError(err);
      });
    void runEventLoop();
    outputReconcileTimer = window.setInterval(() => {
      refreshVisibleOutput().catch((err) => {
        error = backendError(err);
      });
    }, 2000);
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

  onDestroy(() => {
    stopped = true;
    stopCommandEvents?.();
    if (outputReconcileTimer) window.clearInterval(outputReconcileTimer);
    if (workReconcileTimer) window.clearInterval(workReconcileTimer);
    for (const socket of Object.values(ptyStreams)) socket.close();
  });
</script>

<svelte:window on:keydown={handleCommandKey} />

<main class="flex h-screen flex-col overflow-hidden bg-bg-deep text-text-primary">
  <div class="flex min-h-0 flex-1 flex-row">
    {#if railSide === "left"}
      <div class="flex h-full w-[36px] shrink-0 flex-col border-r border-hairline bg-bg-base/96">
        <ActivityRail
          activeSidebar={activeSidebar ??
            (activeMain === "work" ? "work" : activeMain === "projects" ? "projects" : null)}
          {settingsOpen}
          {notificationCount}
          onSidebar={toggleSidebar}
          onSettings={toggleSettings}
        />
      </div>
      <SidebarDock
        activePanel={activeSidebar}
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
        {railSide}
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
        onSelectProject={activeSidebar === "projects" ? selectProjectDetail : selectProject}
        onCreateWorkItem={createWorkItem}
      />
    {/if}

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
        {#if activeMain === "projects"}
          <ProjectsView
            {projects}
            detail={projectDetail}
            {activeProjectId}
            loading={loadingWork || loadingSession}
            onUpdateProject={(projectId, request) => void updateProject(projectId, request)}
            onDeleteProject={(projectId) => void deleteProject(projectId)}
            onNewSession={(projectId) => openNewProjectSession(projectId)}
            onOpenSession={(sessionId) => void openSessionById(sessionId)}
            onRemoveSession={(sessionId) => void unassignProjectSession(sessionId)}
            onCreateWorkItem={createWorkItem}
            onDeleteWorkItem={deleteWorkItem}
            onOpenRunTerminal={(run) => void openWorkItemRun(run)}
            pluginAttachmentTemplates={projectAttachmentTemplates}
            onAddProjectAttachment={addProjectAttachment}
            onRunPluginProjectAttachmentTemplate={runPluginProjectAttachmentTemplate}
            onUpdateProjectAttachment={updateProjectAttachment}
            onDeleteProjectAttachment={deleteProjectAttachment}
          />
        {:else if activeMain === "work"}
          <WorkBoard
            {projects}
            {workItems}
            {workItemRuns}
            {artifacts}
            {questions}
            {gateReports}
            {workflowEvents}
            {activeProjectId}
            filterQuery={workFilterQuery}
            filterStageId={workFilterStageId}
            filterRunState={workFilterRunState}
            loading={loadingWork}
            onRefresh={() => void refreshProjects()}
            onCreateWorkItem={createWorkItem}
            onMoveWorkItem={moveWorkItem}
            onGenerateWorktree={generateWorktree}
            onAttachFile={attachFile}
            onDeleteWorkItem={deleteWorkItem}
            onCancelRun={cancelWorkItemRun}
            onLaunchRun={launchWorkItemRun}
            onOpenRunTerminal={(run) => void openWorkItemRun(run)}
            onStartPlanning={startPlanning}
            onSubmitPlan={submitPlan}
            onApprovePlan={approvePlan}
            onQueueExecution={queueExecution}
            onLaunchExecution={launchExecution}
            onCompleteExecution={completeExecution}
            onSubmitReviewFeedback={submitReviewFeedback}
            onAskQuestion={askQuestion}
            onAnswerQuestion={answerQuestion}
            onCompleteGate={completeGate}
            onApproveDone={approveDone}
          />
        {:else if activeSession}
          {#if activeSessionWindow}
            <LayoutView
              node={activeSessionWindow.layout}
              panes={activeSession.panes}
              {outputChunks}
              {activePaneId}
              {terminalFontSize}
              {terminalCursorBlink}
              onFocus={(paneId) => (activePaneId = paneId)}
              onInput={(ptyId) => refreshOutput(ptyId).catch((err) => {
                if (!isStalePTYError(err)) error = backendError(err);
              })}
              onWriteInput={writePTYInput}
              onClose={(paneId) => void closePane(paneId)}
              onKillPTY={(paneId) => void killPanePTY(paneId)}
              canClose={(paneId) =>
                Boolean(closePaneTarget(activeSession, activeSessionWindow?.id ?? "", paneId))}
            />
          {/if}
        {:else}
          <div class="flex flex-1 flex-col items-center justify-center gap-4 text-center text-text-secondary">
            <div
              class="flex h-16 w-16 items-center justify-center rounded-2xl border border-border-subtle bg-bg-surface/80 text-accent shadow-[0_18px_40px_rgba(2,6,23,0.45)]"
            >
              <span class="text-3xl">W</span>
            </div>
            <div class="space-y-1">
              <p class="text-base font-semibold tracking-tight text-text-primary">
                No active sessions
              </p>
              <p class="text-sm text-text-secondary">
                Start a daemon-owned shell session.
              </p>
            </div>
            <button
              type="button"
              class="rounded-lg border border-border-subtle bg-bg-surface/80 px-4 py-2 text-sm font-semibold text-text-primary shadow-[0_18px_40px_rgba(2,6,23,0.45)] transition-colors hover:border-accent hover:text-accent"
              disabled={loadingSession}
              on:click={openNewSession}
            >
              New Session
            </button>
          </div>
        {/if}

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

    {#if railSide === "right"}
      <SidebarDock
        activePanel={activeSidebar}
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
        {railSide}
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
        onSelectProject={activeSidebar === "projects" ? selectProjectDetail : selectProject}
        onCreateWorkItem={createWorkItem}
      />
      <div class="flex h-full w-[36px] shrink-0 flex-col border-l border-hairline bg-bg-base/96">
        <ActivityRail
          activeSidebar={activeSidebar ??
            (activeMain === "work" ? "work" : activeMain === "projects" ? "projects" : null)}
          {settingsOpen}
          {notificationCount}
          onSidebar={toggleSidebar}
          onSettings={toggleSettings}
        />
      </div>
    {/if}
  </div>
  <CommandPalette
    visible={commandPaletteOpen}
    {commands}
    onclose={() => (commandPaletteOpen = false)}
    onrun={(id) => void executeCommand(id)}
  />
</main>
