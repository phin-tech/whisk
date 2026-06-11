<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import type { Session } from "../bindings/github.com/phin-tech/whisk/internal/domain/session/models";
  import type { PTYInfo, RuntimeEvent } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import {
    CloseSession,
    CreateSession,
    ListPTYs,
    ListSessions,
    NextEvent,
    Output,
    SplitPane,
  } from "../bindings/github.com/phin-tech/whisk/internal/wailsapp/service";
  import ActivityRail from "./ActivityRail.svelte";
  import LayoutView from "./LayoutView.svelte";
  import NewSessionDialog from "./NewSessionDialog.svelte";
  import SettingsView from "./SettingsView.svelte";
  import SidebarDock from "./SidebarDock.svelte";
  import { activeWindow, firstPaneId, runtimeRefreshTargets, visiblePtyIds } from "./sessionView";

  type SidebarId = "sessions" | "ptys";
  type RailSide = "left" | "right";

  const SETTINGS_KEY = "whisk.ui.settings";

  let sessions: Session[] = [];
  let ptys: PTYInfo[] = [];
  let activeSessionId = "";
  let activePaneId = "";
  let activeSidebar: SidebarId | null = "sessions";
  let newSessionOpen = false;
  let settingsOpen = false;
  let railSide: RailSide = "right";
  let terminalFontSize = 13;
  let terminalCursorBlink = true;
  let error = "";
  let loadingSession = false;
  let loadingPtys = false;
  let outputChunks: Record<string, string[]> = {};
  let offsets: Record<string, number> = {};
  let reconcileTimer: number | undefined;
  let eventLoopRunning = false;
  let settingsLoaded = false;
  let stopped = false;
  const textEncoder = new TextEncoder();
  const outputFetchInFlight = new Set<string>();
  const outputFetchAgain = new Set<string>();

  $: activeSession = sessions.find((session) => session.id === activeSessionId) ?? null;
  $: activeSessionWindow = activeWindow(activeSession, activePaneId);
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

  function loadSettings() {
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
    activeSessionId = session.id;
    activePaneId = firstPaneId(session);
    void refreshVisibleOutput().catch((err) => {
      error = backendError(err);
    });
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
    activeSidebar = activeSidebar === id ? null : id;
    settingsOpen = false;
  }

  function toggleSettings() {
    settingsOpen = !settingsOpen;
  }

  onMount(() => {
    loadSettings();
    settingsLoaded = true;
    refreshSessions()
      .then(refreshPTYs)
      .then(refreshVisibleOutput)
      .catch((err) => {
        error = backendError(err);
      });
    void runEventLoop();
    reconcileTimer = window.setInterval(() => {
      refreshVisibleOutput().catch((err) => {
        error = backendError(err);
      });
    }, 2000);
  });

  $: if (settingsLoaded) saveSettings();

  onDestroy(() => {
    stopped = true;
    if (reconcileTimer) window.clearInterval(reconcileTimer);
  });
</script>

<main class="flex h-screen flex-col overflow-hidden bg-bg-deep text-text-primary">
  <div class="flex min-h-0 flex-1 flex-row">
    {#if railSide === "left"}
      <div class="flex h-full w-[36px] shrink-0 flex-col border-r border-hairline bg-bg-base/96">
        <ActivityRail
          {activeSidebar}
          {settingsOpen}
          onSidebar={toggleSidebar}
          onSettings={toggleSettings}
        />
      </div>
      <SidebarDock
        activePanel={activeSidebar}
        {sessions}
        {ptys}
        {activeSessionId}
        {loadingSession}
        {loadingPtys}
        {railSide}
        onClose={() => (activeSidebar = null)}
        onNewSession={openNewSession}
        onSelectSession={selectSession}
        onCloseSession={closeSession}
        onRefreshPtys={() => void refreshPTYs()}
      />
    {/if}

    <section class="relative flex min-w-0 flex-1 flex-col overflow-hidden bg-bg-deep">
      <header class="flex h-10 shrink-0 items-center justify-between border-b border-hairline bg-bg-base/80 px-3">
        <div class="min-w-0">
          <div class="truncate text-[13px] font-semibold text-text-primary">
            {activeSession?.name ?? "No active session"}
          </div>
          <div class="truncate font-mono text-[10px] text-text-muted">
            {activePaneId || "No pane selected"}
          </div>
        </div>
        <div class="flex items-center gap-1">
          <button
            type="button"
            class="rounded border border-border-subtle bg-bg-surface/60 px-2.5 py-1 text-[11px] text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary"
            disabled={!activeSession}
            on:click={() => split("horizontal")}
          >
            Split right
          </button>
          <button
            type="button"
            class="rounded border border-border-subtle bg-bg-surface/60 px-2.5 py-1 text-[11px] text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary"
            disabled={!activeSession}
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
        {#if activeSession}
          {#if activeSessionWindow}
            <LayoutView
              node={activeSessionWindow.layout}
              panes={activeSession.panes}
              {outputChunks}
              {activePaneId}
              {terminalFontSize}
              {terminalCursorBlink}
              onFocus={(paneId) => (activePaneId = paneId)}
              onInput={() => refreshVisibleOutput().catch((err) => (error = backendError(err)))}
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

        <SettingsView
          visible={settingsOpen}
          {railSide}
          {terminalFontSize}
          {terminalCursorBlink}
          onclose={() => (settingsOpen = false)}
          onRailSide={(side) => (railSide = side)}
          onTerminalFontSize={(size) => (terminalFontSize = size)}
          onTerminalCursorBlink={(blink) => (terminalCursorBlink = blink)}
        />
      </div>
    </section>

    {#if railSide === "right"}
      <SidebarDock
        activePanel={activeSidebar}
        {sessions}
        {ptys}
        {activeSessionId}
        {loadingSession}
        {loadingPtys}
        {railSide}
        onClose={() => (activeSidebar = null)}
        onNewSession={openNewSession}
        onSelectSession={selectSession}
        onCloseSession={closeSession}
        onRefreshPtys={() => void refreshPTYs()}
      />
      <div class="flex h-full w-[36px] shrink-0 flex-col border-l border-hairline bg-bg-base/96">
        <ActivityRail
          {activeSidebar}
          {settingsOpen}
          onSidebar={toggleSidebar}
          onSettings={toggleSettings}
        />
      </div>
    {/if}
  </div>
</main>
