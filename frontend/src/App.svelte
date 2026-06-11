<script lang="ts">
  import type { LayoutNode, Session } from "../bindings/github.com/phin-tech/whisk/internal/domain/session/models";
  import { CreateSession, ListSessions, Output, SplitPane } from "../bindings/github.com/phin-tech/whisk/internal/wailsapp/service";
  import LayoutView from "./LayoutView.svelte";

  let sessions: Session[] = [];
  let activeSessionId = "";
  let activePaneId = "";
  let error = "";
  let loading = false;
  let outputs: Record<string, string> = {};
  let offsets: Record<string, number> = {};
  let poller: number | undefined;
  let polling = false;
  let pollAgain = false;

  const pollIntervalMs = 40;

  $: activeSession = sessions.find((session) => session.id === activeSessionId) ?? null;
  $: if (activeSession && !activePaneId) activePaneId = activeSession.focusedPaneId;

  function backendError(err: unknown): string {
    const message = err instanceof Error ? err.message : String(err);
    if (!message || message.includes("Not Found") || message.includes("Failed to fetch")) {
      return "Wails runtime is unavailable. Run `task dev:app` and use the macOS app window, not the Vite browser URL.";
    }
    return message;
  }

  async function refreshSessions() {
    sessions = await ListSessions();
    if (!activeSessionId && sessions.length > 0) {
      activeSessionId = sessions[0].id;
      activePaneId = sessions[0].focusedPaneId;
    }
  }

  async function createSession() {
    error = "";
    loading = true;
    try {
      const created = await CreateSession({
        name: "Local shell",
        workingDir: ".",
        cols: 100,
        rows: 28,
      });
      sessions = [created.Session, ...sessions.filter((session) => session.id !== created.Session.id)];
      activeSessionId = created.Session.id;
      activePaneId = created.Session.focusedPaneId;
      await pollOnce();
    } catch (err) {
      error = backendError(err);
    } finally {
      loading = false;
    }
  }

  async function split(direction: "horizontal" | "vertical") {
    if (!activeSession || !activePaneId) return;
    error = "";
    try {
      const result = await SplitPane({
        sessionId: activeSession.id,
        targetPaneId: activePaneId,
        direction,
        cols: 100,
        rows: 28,
      });
      sessions = sessions.map((session) =>
        session.id === result.Session.id ? result.Session : session,
      );
      activePaneId = result.PaneID;
      await pollOnce();
    } catch (err) {
      error = backendError(err);
    }
  }

  function paneIds(node: LayoutNode | undefined): string[] {
    if (!node) return [];
    if (node.kind === "leaf") return node.paneId ? [node.paneId] : [];
    return (node.children ?? []).flatMap(paneIds);
  }

  function visiblePtyIds(): string[] {
    const ptys: string[] = [];
    const seen = new Set<string>();
    const activePtyID = activeSession?.panes[activePaneId]?.ptyId;
    if (activePtyID) {
      ptys.push(activePtyID);
      seen.add(activePtyID);
    }
    for (const session of sessions) {
      for (const paneID of paneIds(session.layout)) {
        const ptyID = session.panes[paneID]?.ptyId;
        if (ptyID && !seen.has(ptyID)) {
          ptys.push(ptyID);
          seen.add(ptyID);
        }
      }
    }
    return ptys;
  }

  async function pollOutputs() {
    const ptys = visiblePtyIds();
    await Promise.all(
      ptys.map(async (ptyID) => {
        const snapshot = await Output({
          ptyId: ptyID,
          fromOffset: offsets[ptyID] ?? 0,
        });
        if (snapshot.output) {
          outputs = {
            ...outputs,
            [ptyID]: (outputs[ptyID] ?? "") + snapshot.output,
          };
        }
        offsets = { ...offsets, [ptyID]: snapshot.offset };
      }),
    );
  }

  async function pollOnce() {
    if (polling) {
      pollAgain = true;
      return;
    }
    polling = true;
    try {
      do {
        pollAgain = false;
        await pollOutputs();
      } while (pollAgain);
    } finally {
      polling = false;
    }
  }

  refreshSessions().catch((err) => {
    error = backendError(err);
  });

  poller = window.setInterval(() => {
    pollOnce().catch((err) => {
      error = backendError(err);
    });
  }, pollIntervalMs);
</script>

<svelte:window
  on:beforeunload={() => {
    if (poller) window.clearInterval(poller);
  }}
/>

<main class="app-shell">
  <aside class="sidebar">
    <div class="brand">
      <div class="mark">W</div>
      <div>
        <h1>Whisk</h1>
        <p>Local runtime</p>
      </div>
    </div>

    <button class="primary-action" on:click={createSession} disabled={loading}>
      {loading ? "Creating..." : "New session"}
    </button>

    <div class="session-list">
      {#if sessions.length === 0}
        <p class="empty">No sessions yet.</p>
      {:else}
        {#each sessions as session}
          <button
            class:active={session.id === activeSessionId}
            class="session-row"
            on:click={() => {
              activeSessionId = session.id;
              activePaneId = session.focusedPaneId;
            }}
          >
            <span>{session.name}</span>
            <small>{session.workingDir}</small>
          </button>
        {/each}
      {/if}
    </div>
  </aside>

  <section class="workspace">
    <header class="toolbar">
      <div>
        <strong>{activeSession?.name ?? "No active session"}</strong>
        <span>{activePaneId || "No pane selected"}</span>
      </div>
      <nav>
        <button on:click={() => split("horizontal")} disabled={!activeSession}>Split right</button>
        <button on:click={() => split("vertical")} disabled={!activeSession}>Split down</button>
      </nav>
    </header>

    {#if error}
      <div class="error">{error}</div>
    {/if}

    <div class="pane-stage">
      {#if activeSession}
        <LayoutView
          node={activeSession.layout}
          panes={activeSession.panes}
          outputs={outputs}
          activePaneId={activePaneId}
          onFocus={(paneId) => (activePaneId = paneId)}
          onInput={pollOnce}
        />
      {:else}
        <div class="start-panel">
          <h2>Start a local shell</h2>
          <p>Create a session to spawn a backend-owned PTY and render it here.</p>
          <button on:click={createSession}>New session</button>
        </div>
      {/if}
    </div>
  </section>
</main>
