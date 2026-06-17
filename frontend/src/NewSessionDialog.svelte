<script lang="ts">
  import { Dialogs } from "@wailsio/runtime";
  import FolderOpen from "@lucide/svelte/icons/folder-open";
  import TerminalIcon from "@lucide/svelte/icons/terminal";
  import X from "@lucide/svelte/icons/x";

  export let visible = false;
  export let loading = false;
  export let initialRootDir = "";
  export let onclose: () => void;
  export let oncreate: (request: {
    name: string;
    rootDir: string;
    initialPty: { cols: number; rows: number; command: string } | null;
  }) => void;

  let name = "";
  let rootDir = "";
  let command = "";
  let initialPty = true;
  let localError = "";
  let previousVisible = false;

  $: canCreate = rootDir.trim().length > 0 && !loading;
  $: if (visible && !previousVisible) reset();
  $: previousVisible = visible;

  function reset() {
    name = "";
    rootDir = initialRootDir;
    command = "";
    initialPty = true;
    localError = "";
  }

  function submit() {
    if (!canCreate) return;
    localError = "";
    oncreate({
      name: name.trim(),
      rootDir: rootDir.trim(),
      initialPty: initialPty ? { cols: 0, rows: 0, command: command.trim() } : null,
    });
  }

  async function chooseRootDir() {
    localError = "";
    try {
      const selected = await Dialogs.OpenFile({
        Title: "Root directory",
        ButtonText: "Choose",
        Directory: rootDir || undefined,
        CanChooseDirectories: true,
        CanChooseFiles: false,
        CanCreateDirectories: true,
        AllowsMultipleSelection: false,
      });
      if (typeof selected === "string" && selected.length > 0) {
        rootDir = selected;
      }
    } catch (err) {
      localError = err instanceof Error ? err.message : String(err);
    }
  }

  function handleKey(event: KeyboardEvent) {
    if (event.key === "Escape" && !loading) {
      event.preventDefault();
      onclose();
    }
  }
</script>

<svelte:window on:keydown={visible ? handleKey : undefined} />

{#if visible}
  <div
    class="absolute inset-0 z-40 flex items-center justify-center bg-black/45 px-4"
    role="dialog"
    aria-modal="true"
    aria-label="New session"
  >
    <form
      class="w-full max-w-[520px] rounded-lg border border-border bg-bg-base shadow-[0_24px_80px_rgba(0,0,0,0.45)]"
      on:submit|preventDefault={submit}
    >
      <div class="flex h-11 items-center justify-between border-b border-hairline px-4">
        <div class="flex items-center gap-2 text-sm font-semibold text-text-primary">
          <TerminalIcon size={15} />
          <span>New session</span>
        </div>
        <button
          type="button"
          aria-label="Close"
          class="inline-flex h-7 w-7 items-center justify-center rounded border border-transparent text-text-muted transition-colors hover:border-border-subtle hover:bg-bg-hover hover:text-text-primary focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50"
          disabled={loading}
          on:click={onclose}
        >
          <X size={14} />
        </button>
      </div>

      <div class="space-y-4 px-4 py-4">
        <label class="block">
          <span class="mb-1 block text-[11px] font-semibold uppercase tracking-widest text-text-muted">
            Name
          </span>
          <input
            class="h-9 w-full rounded border border-border bg-bg-deep px-2.5 text-[13px] text-text-primary outline-none transition-colors placeholder:text-text-muted focus:border-accent-dim"
            type="text"
            bind:value={name}
            placeholder="Derived from root"
            disabled={loading}
          />
        </label>

        <label class="block">
          <span class="mb-1 block text-[11px] font-semibold uppercase tracking-widest text-text-muted">
            Root directory
          </span>
          <div class="flex gap-2">
            <input
              class="h-9 min-w-0 flex-1 rounded border border-border bg-bg-deep px-2.5 font-mono text-[12px] text-text-primary outline-none transition-colors placeholder:text-text-muted focus:border-accent-dim"
              type="text"
              bind:value={rootDir}
              placeholder="/path/to/project"
              disabled={loading}
            />
            <button
              type="button"
              aria-label="Choose root directory"
              title="Choose root directory"
              class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded border border-border-subtle bg-bg-surface/60 text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50"
              disabled={loading}
              on:click={chooseRootDir}
            >
              <FolderOpen size={15} />
            </button>
          </div>
        </label>

        <div class="flex items-center justify-between gap-3 rounded border border-border-subtle bg-bg-surface/25 px-3 py-2">
          <div class="flex items-center gap-2 text-[13px] text-text-primary">
            <TerminalIcon size={14} />
            <span>Initial PTY</span>
          </div>
          <button
            type="button"
            aria-label="Toggle initial PTY"
            class="relative h-5 w-9 rounded-full border transition-all {initialPty
              ? 'border-accent bg-accent-dim'
              : 'border-border bg-bg-deep'}"
            disabled={loading}
            on:click={() => (initialPty = !initialPty)}
          >
            <div
              class="absolute top-0.5 h-3.5 w-3.5 rounded-full transition-all {initialPty
                ? 'left-[18px] bg-accent'
                : 'left-0.5 bg-text-secondary'}"
            ></div>
          </button>
        </div>

        {#if initialPty}
          <label class="block">
            <span class="mb-1 block text-[11px] font-semibold uppercase tracking-widest text-text-muted">
              Initial command
            </span>
            <input
              class="h-9 w-full rounded border border-border bg-bg-deep px-2.5 font-mono text-[12px] text-text-primary outline-none transition-colors placeholder:text-text-muted focus:border-accent-dim"
              type="text"
              bind:value={command}
              placeholder="Optional"
              disabled={loading}
            />
          </label>
        {/if}

        {#if localError}
          <div class="rounded border border-red/30 bg-red/10 px-3 py-2 text-[12px] text-red">
            {localError}
          </div>
        {/if}
      </div>

      <div class="flex justify-end gap-2 border-t border-hairline px-4 py-3">
        <button
          type="button"
          class="rounded border border-border-subtle bg-bg-surface/40 px-3 py-1.5 text-[12px] text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary"
          disabled={loading}
          on:click={onclose}
        >
          Cancel
        </button>
        <button
          type="submit"
          class="rounded border border-accent-dim bg-accent-dim px-3 py-1.5 text-[12px] font-semibold text-text-primary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:border-border disabled:bg-bg-surface/30 disabled:text-text-muted"
          disabled={!canCreate}
        >
          Create
        </button>
      </div>
    </form>
  </div>
{/if}
