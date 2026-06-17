<script lang="ts">
  import { Events } from "@wailsio/runtime";
  import { onDestroy, onMount } from "svelte";
  import type { Session } from "../bindings/github.com/phin-tech/whisk/internal/domain/session/models";
  import type {
    AgentBridgeApproval,
    AgentBridgeEvent,
    AgentHookIntegration,
    AgentHookLogStatus,
    Project,
    PTYInfo,
    RuntimeEvent,
    StatusEvent,
    Artifact,
    GateReport,
    Question,
    WorkItem,
    WorkItemRun,
    WorkflowEvent,
  } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import {
    AddWorkItemAttachment,
    AnswerQuestion,
    ApproveDone,
    ApprovePlan,
    AskQuestion,
    BindWorkItemWorktree,
    CancelWorkItemRun,
    CloseSession,
    CompleteExecution,
    CompleteGate,
    CreateProject,
    CreateSession,
    CreateWorkItem,
    CreateWorktree,
    DeleteWorkItem,
    CheckAgentHookIntegration,
    AgentHookLogStatus as LoadAgentHookLogStatus,
    ClearAgentHookLog,
    InstallAgentHookIntegration,
    ListAgentBridgeApprovals,
    ListAgentBridgeEvents,
    ListAgentHookIntegrations,
    ListArtifacts,
    ListGateReports,
    ListPTYs,
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
    MarkAgentBridgeEventRead,
    MarkStatusEventRead,
    MoveWorkItem,
    NextEvent,
    Output,
    OpenAgentHookLog,
    QueueExecution,
    RemoveAgentHookIntegration,
    ResolveAgentBridgeApproval,
    SaveAppSettings,
    SetAgentHookLogSettings,
    SplitPane,
    StartPlanning,
    SubmitDraftPlan,
    SubmitReviewFeedback,
    SyncSessionMenu,
  } from "../bindings/github.com/phin-tech/whisk/internal/wailsapp/service";
  import ActivityRail from "./ActivityRail.svelte";
  import CommandPalette from "./CommandPalette.svelte";
  import LayoutView from "./LayoutView.svelte";
  import NewProjectDialog from "./NewProjectDialog.svelte";
  import NewSessionDialog from "./NewSessionDialog.svelte";
  import SettingsView from "./SettingsView.svelte";
  import SidebarDock from "./SidebarDock.svelte";
  import WorkBoard from "./WorkBoard.svelte";
  import { upsertAgentHookIntegration as upsertAgentHookIntegrationView } from "./agentHooksView";
  import type { Command } from "./commands";
  import { runCommand } from "./commands";
  import { notificationBadgeCount, targetForStatusEvent } from "./notificationsView";
  import { activeWindow, firstPaneId, runtimeRefreshTargets, visiblePtyIds } from "./sessionView";
  import {
    normalizeStartupView,
    startupTarget,
    type StartupView,
  } from "./startupView";

  type SidebarId = "sessions" | "ptys" | "work" | "notifications";
  type MainView = "session" | "work";
  type RailSide = "left" | "right";

  const SETTINGS_KEY = "whisk.ui.settings";

  let sessions: Session[] = [];
  let ptys: PTYInfo[] = [];
  let projects: Project[] = [];
  let workItems: WorkItem[] = [];
  let workItemRuns: WorkItemRun[] = [];
  let artifacts: Artifact[] = [];
  let questions: Question[] = [];
  let gateReports: GateReport[] = [];
  let workflowEvents: WorkflowEvent[] = [];
  let statusEvents: StatusEvent[] = [];
  let agentBridgeApprovals: AgentBridgeApproval[] = [];
  let agentBridgeEvents: AgentBridgeEvent[] = [];
  let agentHookIntegrations: AgentHookIntegration[] = [];
  let agentHookLogStatus: AgentHookLogStatus | null = null;
  let agentHookAction = "";
  let agentHookNotice = "";
  let commands: Command[] = [];
  let activeSessionId = "";
  let activePaneId = "";
  let activeProjectId = "";
  let activeMain: MainView = "session";
  let activeSidebar: SidebarId | null = "sessions";
  let commandPaletteOpen = false;
  let newProjectOpen = false;
  let newSessionOpen = false;
  let settingsOpen = false;
  let railSide: RailSide = "right";
  let startupView: StartupView = "sessions";
  let terminalFontSize = 13;
  let terminalCursorBlink = true;
  let keepDaemonAlive = true;
  let hookLogEnabled = true;
  let clearHookLogAfterSession = false;
  let error = "";
  let loadingSession = false;
  let loadingPtys = false;
  let loadingWork = false;
  let loadingStatusEvents = false;
  let outputChunks: Record<string, string[]> = {};
  let offsets: Record<string, number> = {};
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
  $: notificationCount = notificationBadgeCount(statusEvents) + agentBridgeApprovals.length + agentBridgeEvents.length;
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
      enabled: () => statusEvents.length > 0 || agentBridgeEvents.length > 0,
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
      }>;
      if (parsed.railSide === "left" || parsed.railSide === "right") railSide = parsed.railSide;
      if (typeof parsed.terminalFontSize === "number") terminalFontSize = parsed.terminalFontSize;
      if (typeof parsed.terminalCursorBlink === "boolean") {
        terminalCursorBlink = parsed.terminalCursorBlink;
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
        JSON.stringify({ railSide, terminalFontSize, terminalCursorBlink }),
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
    try {
      ptys = await ListPTYs();
    } finally {
      loadingPtys = false;
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
      await refreshWorkState();
    } finally {
      loadingWork = false;
    }
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
    if (activeMain !== "work" && activeSidebar !== "work") return;
    await Promise.all([
      refreshSessions(),
      refreshPTYs(),
      activeProjectId ? refreshWorkState() : Promise.resolve(),
    ]);
  }

  async function refreshStatusEvents() {
    loadingStatusEvents = true;
    try {
      const [nextStatusEvents, nextApprovals, nextBridgeEvents] = await Promise.all([
        ListStatusEvents({ unreadOnly: true }),
        ListAgentBridgeApprovals({ status: "pending" }),
        ListAgentBridgeEvents({ status: "pending" }),
      ]);
      statusEvents = nextStatusEvents;
      agentBridgeApprovals = nextApprovals;
      agentBridgeEvents = nextBridgeEvents;
    } finally {
      loadingStatusEvents = false;
    }
  }

  async function clearNotifications() {
    if (statusEvents.length === 0 && agentBridgeEvents.length === 0) return;
    loadingStatusEvents = true;
    try {
      await Promise.all([
        ...statusEvents.map((event) => MarkStatusEventRead({ id: event.id })),
        ...agentBridgeEvents.map((event) => MarkAgentBridgeEventRead({ id: event.id })),
      ]);
      await refreshStatusEvents();
    } catch (err) {
      error = `Clear notifications failed: ${backendError(err)}`;
    } finally {
      loadingStatusEvents = false;
    }
  }

  async function refreshVisibleOutput() {
    const ids = visiblePtyIds(sessions, activeSessionId, activePaneId);
    await Promise.all(ids.map((ptyId) => refreshOutput(ptyId)));
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

  function openNewSession() {
    error = "";
    settingsOpen = false;
    activeMain = "session";
    newSessionOpen = true;
  }

  async function createSession(request: {
    name: string;
    rootDir: string;
    initialPty: { cols: number; rows: number; command: string } | null;
  }) {
    error = "";
    loadingSession = true;
    try {
      const created = await CreateSession({
        name: request.name,
        rootDir: request.rootDir,
        initialPty: request.initialPty,
      });
      sessions = [created.session, ...sessions.filter((session) => session.id !== created.session.id)];
      activeSessionId = created.session.id;
      activePaneId = created.paneId;
      activeMain = "session";
      activeSidebar = "sessions";
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
    if (targets.outputPtyId) await refreshOutput(targets.outputPtyId);
    if (targets.statusEvents) await refreshStatusEvents();
    if (targets.agentBridgeApprovals) await refreshStatusEvents();
    if (targets.agentHookEvents) await refreshStatusEvents();
    if (targets.work) await refreshWorkState();
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
    void refreshWorkState().catch((err) => {
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
    activeMain = "work";
    activeSidebar = "work";
    newProjectOpen = true;
  }

  async function createProject(request: { name: string; rootDir: string }) {
    error = "";
    loadingWork = true;
    try {
      const project = await CreateProject({
        name: request.name,
        rootDir: request.rootDir,
      });
      activeProjectId = project.id;
      await refreshProjects();
      activeMain = "work";
      activeSidebar = "work";
      newProjectOpen = false;
    } catch (err) {
      error = `Create project failed: ${backendError(err)}`;
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

  async function deleteWorkItem(workItemId: string) {
    error = "";
    loadingWork = true;
    try {
      await DeleteWorkItem({ id: workItemId });
      await refreshWorkState();
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
    error = "";
    loadingSession = true;
    try {
      sessions = await CloseSession({ sessionId: session.id });
      if (activeSessionId === session.id) {
        activeSessionId = sessions[0]?.id ?? "";
        activePaneId = firstPaneId(sessions[0]);
      }
      await refreshPTYs();
      await refreshVisibleOutput();
    } catch (err) {
      error = `Close session failed: ${backendError(err)}`;
    } finally {
      loadingSession = false;
    }
  }

  function toggleSidebar(id: SidebarId) {
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
    void refreshAgentHookIntegrations().catch((err) => {
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

  onMount(() => {
    stopCommandEvents = Events.On("command:run", (event) => {
      void executeCommand(String(event.data));
    });
    loadSettings()
      .then(() => {
        settingsLoaded = true;
      })
      .then(refreshSessions)
      .then(refreshPTYs)
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

  $: if (settingsLoaded) saveSettings();

  // Keep the native Sessions menu in step with the session bar whenever the list changes.
  $: if (settingsLoaded) syncSessionMenu(sessions);

  onDestroy(() => {
    stopped = true;
    stopCommandEvents?.();
    if (outputReconcileTimer) window.clearInterval(outputReconcileTimer);
    if (workReconcileTimer) window.clearInterval(workReconcileTimer);
  });
</script>

<svelte:window on:keydown={handleCommandKey} />

<main class="flex h-screen flex-col overflow-hidden bg-bg-deep text-text-primary">
  <div class="flex min-h-0 flex-1 flex-row">
    {#if railSide === "left"}
      <div class="flex h-full w-[36px] shrink-0 flex-col border-r border-hairline bg-bg-base/96">
        <ActivityRail
          activeSidebar={activeSidebar ?? (activeMain === "work" ? "work" : null)}
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
        {projects}
        {workItems}
          {statusEvents}
          {agentBridgeApprovals}
          {agentBridgeEvents}
        {activeSessionId}
        {activeProjectId}
        {loadingSession}
        {loadingPtys}
        {loadingWork}
        {loadingStatusEvents}
        {railSide}
        onClose={() => (activeSidebar = null)}
        onNewSession={openNewSession}
        onSelectSession={selectSession}
        onCloseSession={closeSession}
        onRefreshPtys={() => void refreshPTYs()}
        onRefreshStatusEvents={() => void refreshStatusEvents()}
        onClearNotifications={() => void executeCommand("notifications.clear")}
        onSelectStatusEvent={(event) => void selectStatusEvent(event)}
        onResolveAgentBridgeApproval={(id, action) => void resolveAgentBridgeApproval(id, action)}
        onRefreshWork={() => void refreshProjects()}
        onNewProject={openNewProject}
        onSelectProject={selectProject}
        onCreateWorkItem={createWorkItem}
        onMoveWorkItem={moveWorkItem}
        onGenerateWorktree={generateWorktree}
        onAttachFile={attachFile}
        onDeleteWorkItem={deleteWorkItem}
      />
    {/if}

    <section class="relative flex min-w-0 flex-1 flex-col overflow-hidden bg-bg-deep">
      <header class="flex h-10 shrink-0 items-center justify-between border-b border-hairline bg-bg-base/80 px-3">
        <div class="min-w-0">
          <div class="truncate text-[13px] font-semibold text-text-primary">
            {activeMain === "work" ? "Work" : (activeSession?.name ?? "No active session")}
          </div>
          <div class="truncate font-mono text-[10px] text-text-muted">
            {activeMain === "work" ? (activeProjectId || "No project selected") : (activePaneId || "No pane selected")}
          </div>
        </div>
        <div class="flex items-center gap-1">
          <button
            type="button"
            class="rounded border border-border-subtle bg-bg-surface/60 px-2.5 py-1 text-[11px] text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary"
            disabled={!activeSession || activeMain === "work"}
            on:click={() => split("horizontal")}
          >
            Split right
          </button>
          <button
            type="button"
            class="rounded border border-border-subtle bg-bg-surface/60 px-2.5 py-1 text-[11px] text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary"
            disabled={!activeSession || activeMain === "work"}
            on:click={() => split("vertical")}
          >
            Split down
          </button>
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
        {#if activeMain === "work"}
          <WorkBoard
            {projects}
            {workItems}
            {workItemRuns}
            {artifacts}
            {questions}
            {gateReports}
            {workflowEvents}
            {activeProjectId}
            loading={loadingWork}
            onRefresh={() => void refreshProjects()}
            onNewProject={openNewProject}
            onSelectProject={selectProject}
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
              onInput={(ptyId) => refreshOutput(ptyId).catch((err) => (error = backendError(err)))}
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
          onclose={() => (newSessionOpen = false)}
          oncreate={createSession}
        />

        <NewProjectDialog
          visible={newProjectOpen}
          loading={loadingWork}
          onclose={() => (newProjectOpen = false)}
          oncreate={createProject}
        />

        <SettingsView
          visible={settingsOpen}
          {railSide}
          {startupView}
          {terminalFontSize}
          {terminalCursorBlink}
          {keepDaemonAlive}
          {agentHookIntegrations}
          {agentHookLogStatus}
          {agentHookAction}
          {agentHookNotice}
          onclose={() => (settingsOpen = false)}
          onRailSide={(side) => (railSide = side)}
          onStartupView={(view) => void setStartupView(view)}
          onTerminalFontSize={(size) => (terminalFontSize = size)}
          onTerminalCursorBlink={(blink) => (terminalCursorBlink = blink)}
          onKeepDaemonAlive={(keep) => void setKeepDaemonAlive(keep)}
          onRefreshAgentHookIntegrations={() => void refreshAgentHookIntegrations()}
          onCheckAgentHookIntegration={(provider) => void checkAgentHookIntegration(provider)}
          onInstallAgentHookIntegration={(provider) => void installAgentHookIntegration(provider)}
          onRemoveAgentHookIntegration={(provider) => void removeAgentHookIntegration(provider)}
          onHookLogEnabled={(enabled) => void setHookLogEnabled(enabled)}
          onClearHookLogAfterSession={(enabled) => void setClearHookLogAfterSession(enabled)}
          onClearAgentHookLog={() => void clearAgentHookLog()}
          onOpenAgentHookLog={() => void openAgentHookLog()}
          onCopyAgentHookLogPath={(path) => void copyAgentHookLogPath(path)}
        />
      </div>
    </section>

    {#if railSide === "right"}
      <SidebarDock
        activePanel={activeSidebar}
        {sessions}
        {ptys}
        {projects}
        {workItems}
        {statusEvents}
        {agentBridgeApprovals}
        {agentBridgeEvents}
        {activeSessionId}
        {activeProjectId}
        {loadingSession}
        {loadingPtys}
        {loadingWork}
        {loadingStatusEvents}
        {railSide}
        onClose={() => (activeSidebar = null)}
        onNewSession={openNewSession}
        onSelectSession={selectSession}
        onCloseSession={closeSession}
        onRefreshPtys={() => void refreshPTYs()}
        onRefreshStatusEvents={() => void refreshStatusEvents()}
        onClearNotifications={() => void executeCommand("notifications.clear")}
        onSelectStatusEvent={(event) => void selectStatusEvent(event)}
        onResolveAgentBridgeApproval={(id, action) => void resolveAgentBridgeApproval(id, action)}
        onRefreshWork={() => void refreshProjects()}
        onNewProject={openNewProject}
        onSelectProject={selectProject}
        onCreateWorkItem={createWorkItem}
        onMoveWorkItem={moveWorkItem}
        onGenerateWorktree={generateWorktree}
        onAttachFile={attachFile}
        onDeleteWorkItem={deleteWorkItem}
      />
      <div class="flex h-full w-[36px] shrink-0 flex-col border-l border-hairline bg-bg-base/96">
        <ActivityRail
          activeSidebar={activeSidebar ?? (activeMain === "work" ? "work" : null)}
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
