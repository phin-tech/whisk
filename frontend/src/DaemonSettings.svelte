<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import Play from "@lucide/svelte/icons/play";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import RotateCw from "@lucide/svelte/icons/rotate-cw";
  import Square from "@lucide/svelte/icons/square";
  import {
    DaemonStatus as FetchDaemonStatus,
    RestartDaemon,
    StartDaemon,
    StopDaemon,
  } from "../bindings/github.com/phin-tech/whisk/internal/wailsapp/service";
  import type { DaemonStatus } from "../bindings/github.com/phin-tech/whisk/internal/wailsapp/models";

  export let keepDaemonAlive = true;
  export let worktrunkPath = "/opt/homebrew/bin/wt";
  export let onKeepDaemonAlive: (value: boolean) => void;
  export let onWorktrunkPath: (value: string) => void;

  let status: DaemonStatus | null = null;
  let busy = false;
  let actionError = "";
  let pollTimer: number | undefined;
  let pendingWorktrunkPath = worktrunkPath;
  let lastWorktrunkPath = worktrunkPath;

  const POLL_MS = 3000;

  function describeError(err: unknown): string {
    return err instanceof Error ? err.message : String(err);
  }

  async function refresh() {
    try {
      status = await FetchDaemonStatus();
    } catch (err) {
      actionError = describeError(err);
    }
  }

  async function run(action: () => Promise<DaemonStatus>) {
    if (busy) return;
    busy = true;
    actionError = "";
    try {
      status = await action();
    } catch (err) {
      actionError = describeError(err);
      await refresh();
    } finally {
      busy = false;
    }
  }

  const start = () => run(StartDaemon);
  const stop = () => run(StopDaemon);
  const restart = () => run(RestartDaemon);

  function saveWorktrunkPath() {
    const next = pendingWorktrunkPath.trim();
    if (next !== worktrunkPath) onWorktrunkPath(next);
  }

  function pathKey(event: KeyboardEvent) {
    if (event.key !== "Enter") return;
    event.preventDefault();
    saveWorktrunkPath();
  }

  onMount(() => {
    void refresh();
    pollTimer = window.setInterval(() => {
      if (!busy) void refresh();
    }, POLL_MS);
  });

  onDestroy(() => {
    if (pollTimer !== undefined) window.clearInterval(pollTimer);
  });

  $: running = status?.running ?? false;
  $: buildLabel =
    status?.running && status.version
      ? `${status.version}${status.gitSha ? ` · ${status.gitSha.slice(0, 7)}` : ""}${status.dirty ? " dirty" : ""}`
      : "";
  $: versionLabel = status?.running
    ? `${buildLabel || "unknown build"} · API v${status.apiVersion}`
    : "";
  $: if (worktrunkPath !== lastWorktrunkPath) {
    pendingWorktrunkPath = worktrunkPath;
    lastWorktrunkPath = worktrunkPath;
  }
</script>

<div class="rounded-xl border border-border-subtle bg-bg-surface/35 p-3">
  <div class="flex items-start justify-between gap-3">
    <div class="min-w-0">
      <div class="flex items-center gap-2">
        <span
          class="h-2 w-2 shrink-0 rounded-full {running ? 'bg-green' : 'bg-red'}"
        ></span>
        <span class="text-[13px] font-medium">
          {running ? "Running" : "Stopped"}
        </span>
        {#if status?.managed}
          <span
            class="rounded border border-border-subtle bg-bg-deep px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-text-muted"
          >
            Started by Whisk
          </span>
        {/if}
      </div>
      <div class="mt-1 truncate text-[11px] text-text-muted">
        {status?.address ?? "—"}{versionLabel ? ` · ${versionLabel}` : ""}
      </div>
    </div>
    <button
      type="button"
      aria-label="Refresh daemon status"
      class="shrink-0 rounded border border-transparent p-1 text-text-muted transition-colors hover:border-border-subtle hover:bg-bg-hover hover:text-text-primary disabled:opacity-50"
      disabled={busy}
      on:click={() => void refresh()}
    >
      <RefreshCw size={14} class={busy ? "animate-spin" : ""} />
    </button>
  </div>

  <div class="mt-3 flex items-center gap-2">
    <button
      type="button"
      class="inline-flex items-center gap-1.5 rounded-lg border border-border-subtle bg-bg-surface/80 px-3 py-1.5 text-[12px] font-semibold text-text-secondary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:border-border-subtle disabled:hover:text-text-secondary"
      disabled={busy || running}
      on:click={start}
    >
      <Play size={13} /> Start
    </button>
    <button
      type="button"
      class="inline-flex items-center gap-1.5 rounded-lg border border-border-subtle bg-bg-surface/80 px-3 py-1.5 text-[12px] font-semibold text-text-secondary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:border-border-subtle disabled:hover:text-text-secondary"
      disabled={busy || !running}
      on:click={stop}
    >
      <Square size={13} /> Stop
    </button>
    <button
      type="button"
      class="inline-flex items-center gap-1.5 rounded-lg border border-border-subtle bg-bg-surface/80 px-3 py-1.5 text-[12px] font-semibold text-text-secondary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:border-border-subtle disabled:hover:text-text-secondary"
      disabled={busy}
      on:click={restart}
    >
      <RotateCw size={13} /> Restart
    </button>
  </div>

  {#if actionError}
    <div class="mt-2 text-[11px] text-red">{actionError}</div>
  {/if}
</div>

<div class="mt-4 py-2">
  <div class="flex items-start justify-between gap-3">
    <div>
      <div class="text-[13px]">Worktrunk binary</div>
      <div class="mt-0.5 text-[11px] text-text-muted">
        Path used when generating worktrees.
      </div>
    </div>
    <input
      class="w-[260px] rounded border border-border bg-bg-deep px-2 py-1 text-right font-mono text-[12px] text-text-primary outline-none focus:border-accent-dim"
      type="text"
      bind:value={pendingWorktrunkPath}
      placeholder="/opt/homebrew/bin/wt"
      on:blur={saveWorktrunkPath}
      on:keydown={pathKey}
      aria-label="Worktrunk binary path"
    />
  </div>
</div>

<div class="mt-4 flex items-center justify-between gap-3 py-2">
  <div>
    <div class="text-[13px]">Keep daemon running after quitting</div>
    <div class="mt-0.5 text-[11px] text-text-muted">
      Leaves the background daemon alive so your sessions persist between launches. When off, a
      daemon Whisk started is stopped on quit.
    </div>
  </div>
  <button
    type="button"
    aria-label="Toggle keep daemon running"
    class="relative h-5 w-9 shrink-0 rounded-full border transition-all {keepDaemonAlive
      ? 'border-accent bg-accent-dim'
      : 'border-border bg-bg-deep'}"
    on:click={() => onKeepDaemonAlive(!keepDaemonAlive)}
  >
    <div
      class="absolute top-0.5 h-3.5 w-3.5 rounded-full transition-all {keepDaemonAlive
        ? 'left-[18px] bg-accent'
        : 'left-0.5 bg-text-secondary'}"
    ></div>
  </button>
</div>
