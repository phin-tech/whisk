<script lang="ts">
  import { Dialogs } from "@wailsio/runtime";
  import FolderOpen from "@lucide/svelte/icons/folder-open";
  import FolderPlus from "@lucide/svelte/icons/folder-plus";
  import X from "@lucide/svelte/icons/x";

  export let visible = false;
  export let loading = false;
  export let onclose: () => void;
  export let oncreate: (request: { name: string; description: string; rootDir: string }) => void;

  let name = "";
  let description = "";
  let rootDir = "";
  let localError = "";
  let previousVisible = false;

  $: canCreate = name.trim().length > 0 && rootDir.trim().length > 0 && !loading;
  $: if (visible && !previousVisible) reset();
  $: previousVisible = visible;

  function reset() {
    name = "";
    description = "";
    rootDir = "";
    localError = "";
  }

  function submit() {
    if (!canCreate) return;
    localError = "";
    oncreate({ name: name.trim(), description: description.trim(), rootDir: rootDir.trim() });
  }

  async function chooseRootDir() {
    localError = "";
    try {
      const selected = await Dialogs.OpenFile({
        Title: "Project root",
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
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/45 px-4"
    role="dialog"
    aria-modal="true"
    aria-label="New project"
  >
    <form
      class="w-full max-w-[520px] rounded-lg border border-border bg-bg-base shadow-[0_24px_80px_rgba(0,0,0,0.45)]"
      on:submit|preventDefault={submit}
    >
      <div class="flex h-11 items-center justify-between border-b border-hairline px-4">
        <div class="flex items-center gap-2 text-sm font-semibold text-text-primary">
          <FolderPlus size={15} />
          <span>New project</span>
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
            placeholder="Project name"
            disabled={loading}
          />
        </label>

        <label class="block">
          <span class="mb-1 block text-[11px] font-semibold uppercase tracking-widest text-text-muted">
            Description
          </span>
          <textarea
            class="min-h-20 w-full resize-y rounded border border-border bg-bg-deep px-2.5 py-2 text-[13px] text-text-primary outline-none transition-colors placeholder:text-text-muted focus:border-accent-dim"
            bind:value={description}
            placeholder="What this project owns"
            disabled={loading}
          ></textarea>
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
              placeholder="/path/to/repo"
              disabled={loading}
            />
            <button
              type="button"
              aria-label="Choose project root"
              title="Choose project root"
              class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded border border-border-subtle bg-bg-surface/60 text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50"
              disabled={loading}
              on:click={chooseRootDir}
            >
              <FolderOpen size={15} />
            </button>
          </div>
        </label>

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
