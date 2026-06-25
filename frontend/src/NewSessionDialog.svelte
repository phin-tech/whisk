<script lang="ts">
  import { Dialogs } from "@wailsio/runtime";
  import FolderOpen from "@lucide/svelte/icons/folder-open";
  import TerminalIcon from "@lucide/svelte/icons/terminal";
  import X from "@lucide/svelte/icons/x";
  import Button from "./ui/Button.svelte";
  import IconButton from "./ui/IconButton.svelte";
  import ModalShell from "./ui/ModalShell.svelte";
  import SelectField from "./ui/SelectField.svelte";
  import Switch from "./ui/Switch.svelte";
  import TextField from "./ui/TextField.svelte";

  type CreateSessionRequest = {
    name: string;
    rootDir: string;
    workingDir: string;
    initialPty: {
      cols: number;
      rows: number;
      command: string;
      agentBridge?: { enabled: boolean; provider: string };
    } | null;
  };
  type Props = {
    visible?: boolean;
    loading?: boolean;
    initialRootDir?: string;
    initialWorkingDir?: string;
    onclose: () => void;
    oncreate: (request: CreateSessionRequest) => void;
  };

  const agentProviderOptions = [
    { value: "claude", label: "Claude" },
    { value: "codex", label: "Codex" },
  ];

  let {
    visible = false,
    loading = false,
    initialRootDir = "",
    initialWorkingDir = "",
    onclose,
    oncreate,
  }: Props = $props();

  let name = $state("");
  let directory = $state("");
  let command = $state("");
  let initialPty = $state(true);
  let agentBridge = $state(false);
  let agentProvider = $state("claude");
  let agentBridgeTouched = $state(false);
  let localError = $state("");
  let previousVisible = $state(false);

  const canCreate = $derived(directory.trim().length > 0 && !loading);
  const detectedProvider = $derived(commandProvider(command));

  function reset() {
    name = "";
    directory = initialWorkingDir || initialRootDir;
    command = "";
    initialPty = true;
    agentBridge = false;
    agentProvider = "claude";
    agentBridgeTouched = false;
    localError = "";
  }

  function commandProvider(value: string) {
    const base = value.trim().split(/\s+/)[0]?.split(/[\\/]/).pop()?.toLowerCase() || "";
    if (base === "claude") return "claude";
    if (base === "codex") return "codex";
    return "";
  }

  function setAgentBridge(enabled: boolean) {
    agentBridgeTouched = true;
    agentBridge = enabled;
  }

  function submit() {
    if (!canCreate) return;
    localError = "";
    const selectedDir = directory.trim();
    oncreate({
      name: name.trim(),
      rootDir: initialRootDir.trim() || selectedDir,
      workingDir: selectedDir,
      initialPty: initialPty
        ? {
            cols: 0,
            rows: 0,
            command: command.trim(),
            agentBridge: agentBridge ? { enabled: true, provider: agentProvider } : undefined,
          }
        : null,
    });
  }

  async function chooseDirectory() {
    localError = "";
    try {
      const selected = await Dialogs.OpenFile({
        Title: "Directory",
        ButtonText: "Choose",
        Directory: directory || undefined,
        CanChooseDirectories: true,
        CanChooseFiles: false,
        CanCreateDirectories: true,
        AllowsMultipleSelection: false,
      });
      if (typeof selected === "string" && selected.length > 0) {
        directory = selected;
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
    if (initialPty && detectedProvider && !agentBridgeTouched) {
      agentBridge = true;
      agentProvider = detectedProvider;
    }
  });

  $effect(() => {
    if (visible && !previousVisible) reset();
    previousVisible = visible;
  });
</script>

<ModalShell
  open={visible}
  titleId="new-session-dialog-title"
  titleClass="sr-only"
  class="max-w-[520px] overflow-hidden bg-bg-base shadow-[0_24px_80px_rgba(0,0,0,0.45)]"
  onOpenChange={handleOpenChange}
  onEscapeKeydown={handleEscape}
>
  {#snippet heading()}
    New session
  {/snippet}

  <form onsubmit={handleSubmit}>
    <div class="flex h-11 items-center justify-between border-b border-hairline px-4">
      <div class="flex items-center gap-2 text-[13px] font-semibold text-text-primary">
        <TerminalIcon size={15} />
        <span>New session</span>
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
        <TextField bind:value={name} placeholder="Derived from root" disabled={loading} />
      </label>

      <label class="block">
        <span class="mb-1 block text-[11px] font-semibold uppercase tracking-widest text-text-muted">
          Directory
        </span>
        <div class="flex gap-2">
          <TextField
            bind:value={directory}
            placeholder="/path/to/project"
            disabled={loading}
            class="min-w-0 flex-1 font-mono"
          />
          <IconButton
            label="Choose directory"
            disabled={loading}
            class="h-8 w-8 shrink-0 border-border-subtle bg-bg-surface/60"
            onclick={chooseDirectory}
          >
            <FolderOpen size={15} />
          </IconButton>
        </div>
      </label>

      <div class="flex items-center justify-between gap-3 rounded border border-border-subtle bg-bg-surface/25 px-3 py-2">
        <div class="flex items-center gap-2 text-[13px] text-text-primary">
          <TerminalIcon size={14} />
          <span>Initial PTY</span>
        </div>
        <Switch bind:checked={initialPty} label="Toggle initial PTY" disabled={loading} />
      </div>

      {#if initialPty}
        <label class="block">
          <span class="mb-1 block text-[11px] font-semibold uppercase tracking-widest text-text-muted">
            Initial command
          </span>
          <TextField
            bind:value={command}
            placeholder="Optional"
            disabled={loading}
            class="font-mono"
          />
        </label>

        <div class="flex items-center justify-between gap-3 rounded border border-border-subtle bg-bg-surface/25 px-3 py-2">
          <div class="flex min-w-0 items-center gap-2 text-[13px] text-text-primary">
            <Switch
              bind:checked={agentBridge}
              label="Agent bridge"
              disabled={loading}
              onCheckedChange={setAgentBridge}
            />
            <span>Agent bridge</span>
          </div>
          {#if agentBridge}
            <SelectField
              bind:value={agentProvider}
              label="Agent provider"
              options={agentProviderOptions}
              disabled={loading}
              class="w-28"
            />
          {/if}
        </div>
      {/if}

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
