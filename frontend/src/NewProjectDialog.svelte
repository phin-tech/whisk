<script lang="ts">
  import { Dialogs } from "@wailsio/runtime";
  import FolderOpen from "@lucide/svelte/icons/folder-open";
  import FolderPlus from "@lucide/svelte/icons/folder-plus";
  import X from "@lucide/svelte/icons/x";
  import Button from "./ui/Button.svelte";
  import IconButton from "./ui/IconButton.svelte";
  import ModalShell from "./ui/ModalShell.svelte";
  import TextArea from "./ui/TextArea.svelte";
  import TextField from "./ui/TextField.svelte";

  type Props = {
    visible?: boolean;
    loading?: boolean;
    onclose: () => void;
    oncreate: (request: { name: string; description: string; rootDir: string }) => void;
  };

  let { visible = false, loading = false, onclose, oncreate }: Props = $props();

  let name = $state("");
  let description = $state("");
  let rootDir = $state("");
  let localError = $state("");
  let previousVisible = $state(false);

  const canCreate = $derived(name.trim().length > 0 && rootDir.trim().length > 0 && !loading);

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

  function handleEscape(event: KeyboardEvent) {
    event.preventDefault();
    if (!loading) onclose();
  }

  function handleOpenChange(open: boolean) {
    if (!open && visible && !loading) onclose();
  }

  function handleSubmit(event: SubmitEvent) {
    event.preventDefault();
    submit();
  }

  $effect(() => {
    if (visible && !previousVisible) reset();
    previousVisible = visible;
  });
</script>

<ModalShell
  open={visible}
  titleId="new-project-dialog-title"
  titleClass="sr-only"
  class="max-w-[520px] overflow-hidden bg-bg-base shadow-[0_24px_80px_rgba(0,0,0,0.45)]"
  onOpenChange={handleOpenChange}
  onEscapeKeydown={handleEscape}
>
  {#snippet heading()}
    New project
  {/snippet}

  <form onsubmit={handleSubmit}>
    <div class="flex h-11 items-center justify-between border-b border-hairline px-4">
      <div class="flex items-center gap-2 text-[13px] font-semibold text-text-primary">
        <FolderPlus size={15} />
        <span>New project</span>
      </div>
      <IconButton label="Close" disabled={loading} onclick={onclose}>
        <X size={14} />
      </IconButton>
    </div>

    <div class="space-y-4 px-4 py-4">
      <label class="block">
        <span class="mb-1 block text-[11px] font-semibold uppercase tracking-widest text-text-muted">
          Name
        </span>
        <TextField bind:value={name} placeholder="Project name" disabled={loading} />
      </label>

      <label class="block">
        <span class="mb-1 block text-[11px] font-semibold uppercase tracking-widest text-text-muted">
          Description
        </span>
        <TextArea
          bind:value={description}
          placeholder="What this project owns"
          disabled={loading}
          class="min-h-20"
        />
      </label>

      <label class="block">
        <span class="mb-1 block text-[11px] font-semibold uppercase tracking-widest text-text-muted">
          Root directory
        </span>
        <div class="flex gap-2">
          <TextField
            bind:value={rootDir}
            placeholder="/path/to/repo"
            disabled={loading}
            class="min-w-0 flex-1 font-mono"
          />
          <IconButton
            label="Choose project root"
            disabled={loading}
            class="h-8 w-8 shrink-0 border-border-subtle bg-bg-surface/60"
            onclick={chooseRootDir}
          >
            <FolderOpen size={15} />
          </IconButton>
        </div>
      </label>

      {#if localError}
        <div class="rounded border border-red/30 bg-red/10 px-3 py-2 text-[12px] text-red">
          {localError}
        </div>
      {/if}
    </div>

    <div class="flex justify-end gap-2 border-t border-hairline px-4 py-3">
      <Button type="button" variant="outline" disabled={loading} onclick={onclose}>Cancel</Button>
      <Button type="submit" variant="primary" disabled={!canCreate}>Create</Button>
    </div>
  </form>
</ModalShell>
