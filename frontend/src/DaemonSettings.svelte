<script lang="ts">
  import { onMount } from "svelte";
  import Play from "@lucide/svelte/icons/play";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import RotateCw from "@lucide/svelte/icons/rotate-cw";
  import Square from "@lucide/svelte/icons/square";
  import Button from "./ui/Button.svelte";
  import IconButton from "./ui/IconButton.svelte";
  import Switch from "./ui/Switch.svelte";
  import TextField from "./ui/TextField.svelte";
  import {
    DaemonStatus as FetchDaemonStatus,
    RestartDaemon,
    StartDaemon,
    StopDaemon,
  } from "../bindings/github.com/phin-tech/whisk/internal/wailsapp/service";
  import type { DaemonStatus } from "../bindings/github.com/phin-tech/whisk/internal/wailsapp/models";

  export let keepDaemonAlive = true;
  export let autoRestartManagedDaemon = false;
  export let status: DaemonStatus | null = null;
  export let worktrunkPath = "/opt/homebrew/bin/wt";
  export let onKeepDaemonAlive: (value: boolean) => void;
  export let onAutoRestartManagedDaemon: (value: boolean) => void;
  export let onDaemonStatus: (status: DaemonStatus) => void;
  export let onWorktrunkPath: (value: string) => void;

  let busy = false;
  let actionError = "";
  let pendingWorktrunkPath = worktrunkPath;
  let lastWorktrunkPath = worktrunkPath;

  function describeError(err: unknown): string {
    return err instanceof Error ? err.message : String(err);
  }

  async function refresh() {
    try {
      const next = await FetchDaemonStatus();
      status = next;
      onDaemonStatus(next);
    } catch (err) {
      actionError = describeError(err);
    }
  }

  async function run(action: () => Promise<DaemonStatus>) {
    if (busy) return;
    busy = true;
    actionError = "";
    try {
      const next = await action();
      status = next;
      onDaemonStatus(next);
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
    if (!status) void refresh();
  });

  $: running = status?.running ?? false;
  $: buildLabel =
    status?.running && status.version
      ? `${status.version}${status.gitSha ? ` · ${status.gitSha.slice(0, 7)}` : ""}${status.dirty ? " dirty" : ""}`
      : "";
  $: versionLabel = status?.running
    ? `${buildLabel || "unknown build"} · API v${status.apiVersion}`
    : "";
  $: restartLabel = status?.restarting
    ? `Auto-restart attempt ${status.restartAttempt} of ${status.restartMaxAttempts}`
    : status?.autoRestartExhausted
      ? `Auto-restart stopped after ${status.restartAttempt} attempts`
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
    <IconButton
      label="Refresh daemon status"
      size="sm"
      class="shrink-0"
      disabled={busy}
      onclick={() => void refresh()}
    >
      <RefreshCw size={14} class={busy ? "animate-spin" : ""} />
    </IconButton>
  </div>

  <div class="mt-3 flex items-center gap-2">
    <Button
      size="sm"
      disabled={busy || running}
      onclick={start}
    >
      <Play size={13} /> Start
    </Button>
    <Button
      size="sm"
      disabled={busy || !running}
      onclick={stop}
    >
      <Square size={13} /> Stop
    </Button>
    <Button
      size="sm"
      disabled={busy}
      onclick={restart}
    >
      <RotateCw size={13} /> Restart
    </Button>
  </div>

  {#if actionError}
    <div class="mt-2 text-[11px] text-red">{actionError}</div>
  {/if}
  {#if restartLabel}
    <div class="mt-2 text-[11px] text-text-secondary">{restartLabel}</div>
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
    <TextField
      class="w-[260px] text-right font-mono"
      type="text"
      bind:value={pendingWorktrunkPath}
      placeholder="/opt/homebrew/bin/wt"
      onblur={saveWorktrunkPath}
      onkeydown={pathKey}
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
  <Switch
    label="Toggle keep daemon running"
    checked={keepDaemonAlive}
    onCheckedChange={onKeepDaemonAlive}
  />
</div>

<div class="mt-4 flex items-center justify-between gap-3 py-2">
  <div>
    <div class="text-[13px]">Auto-restart managed daemon</div>
    <div class="mt-0.5 text-[11px] text-text-muted">
      Re-ensures a daemon Whisk started when it exits unexpectedly. Unmanaged daemons are left alone.
    </div>
  </div>
  <Switch
    label="Toggle managed daemon auto-restart"
    checked={autoRestartManagedDaemon}
    onCheckedChange={onAutoRestartManagedDaemon}
  />
</div>
