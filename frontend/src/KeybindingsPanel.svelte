<script lang="ts">
  import RotateCcw from "@lucide/svelte/icons/rotate-ccw";
  import { onMount } from "svelte";
  import type { CommandView } from "../bindings/github.com/phin-tech/whisk/internal/appmenu/models";
  import {
    LoadKeybindings,
    SaveKeybindings,
  } from "../bindings/github.com/phin-tech/whisk/internal/wailsapp/service";
  import { displayAccelerator, findConflicts, formatAccelerator } from "./keybindingsView";

  export let visible = false;

  let commands: CommandView[] = [];
  // draft holds the in-progress accelerator for each editable command, keyed by command id.
  let draft: Record<string, string> = {};
  let recordingId: string | null = null;
  let loaded = false;
  let saving = false;
  let loadError = "";
  let saveError = "";
  let savedNotice = "";

  // Reload whenever the panel becomes visible so it reflects the persisted bindings.
  $: if (visible && !loaded) {
    void load();
  }

  async function load() {
    try {
      const view = await LoadKeybindings();
      commands = view.commands;
      draft = {};
      for (const command of commands) {
        if (command.editable) {
          draft[command.id] = command.accelerator;
        }
      }
      loaded = true;
      loadError = "";
    } catch (err) {
      loadError = errorText(err);
    }
  }

  function errorText(err: unknown): string {
    if (err instanceof Error) return err.message;
    return String(err);
  }

  // effectiveBindings maps every command id to its current accelerator (draft for editable rows,
  // the fixed default for read-only standard rows) so conflicts across the whole map are surfaced.
  $: effectiveBindings = Object.fromEntries(
    commands.map((command) => [
      command.id,
      command.editable ? (draft[command.id] ?? "") : command.default,
    ]),
  );
  $: conflicts = findConflicts(effectiveBindings);
  $: conflictingIds = new Set(Object.values(conflicts).flat());
  $: hasConflicts = Object.keys(conflicts).length > 0;
  $: dirty = commands.some(
    (command) => command.editable && (draft[command.id] ?? "") !== command.accelerator,
  );

  function startRecording(id: string, event: MouseEvent) {
    recordingId = id;
    savedNotice = "";
    (event.currentTarget as HTMLElement).focus();
  }

  function onRecordKey(event: KeyboardEvent, id: string) {
    if (recordingId !== id) return;
    event.preventDefault();
    event.stopPropagation();
    // Escape with no modifiers cancels recording without changing the binding.
    if (event.key === "Escape" && !event.metaKey && !event.ctrlKey && !event.altKey) {
      recordingId = null;
      return;
    }
    const accelerator = formatAccelerator(event);
    if (accelerator === "") return; // still only modifiers held
    draft[id] = accelerator;
    draft = draft;
    recordingId = null;
  }

  function resetToDefault(command: CommandView) {
    draft[command.id] = command.default;
    draft = draft;
    savedNotice = "";
  }

  async function save() {
    saving = true;
    saveError = "";
    try {
      const overrides: Record<string, string> = {};
      for (const command of commands) {
        if (command.editable) {
          overrides[command.id] = draft[command.id] ?? "";
        }
      }
      const view = await SaveKeybindings(overrides);
      commands = view.commands;
      draft = {};
      for (const command of commands) {
        if (command.editable) {
          draft[command.id] = command.accelerator;
        }
      }
      savedNotice = "Shortcuts saved.";
    } catch (err) {
      saveError = errorText(err);
    } finally {
      saving = false;
    }
  }

  // Group commands by category while preserving registry order.
  $: groups = (() => {
    const order: string[] = [];
    const byCategory: Record<string, CommandView[]> = {};
    for (const command of commands) {
      if (!(command.category in byCategory)) {
        byCategory[command.category] = [];
        order.push(command.category);
      }
      byCategory[command.category].push(command);
    }
    return order.map((category) => ({ category, commands: byCategory[category] }));
  })();
</script>

{#if loadError}
  <div class="rounded border border-red/30 bg-red/10 px-2.5 py-2 text-[12px] text-red">
    {loadError}
  </div>
{:else if !loaded}
  <div class="text-[12px] text-text-muted">Loading shortcuts…</div>
{:else}
  <div class="mb-3 text-[11px] text-text-muted">
    Rebind application shortcuts. Standard macOS shortcuts are shown for reference. Changes apply to
    the menu bar immediately when saved.
  </div>

  {#each groups as group}
    <div class="mt-4 first:mt-0">
      <div class="mb-1 text-[10px] font-semibold uppercase tracking-widest text-text-muted">
        {group.category}
      </div>
      <div class="divide-y divide-hairline border-y border-hairline">
        {#each group.commands as command}
          {@const accelerator = command.editable ? (draft[command.id] ?? "") : command.default}
          {@const conflicting = conflictingIds.has(command.id)}
          <div class="flex items-center justify-between gap-3 py-2">
            <div class="min-w-0">
              <div class="truncate text-[13px] text-text-primary">{command.label}</div>
              {#if conflicting}
                <div class="mt-0.5 text-[11px] text-red">Conflicts with another shortcut</div>
              {/if}
            </div>
            {#if command.editable}
              <div class="flex shrink-0 items-center gap-1">
                <button
                  type="button"
                  class="min-w-[84px] rounded border px-2 py-1 text-center font-mono text-[12px] transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/60 {recordingId ===
                  command.id
                    ? 'border-accent bg-accent-dim/20 text-accent'
                    : conflicting
                      ? 'border-red/40 bg-red/10 text-red hover:border-red'
                      : 'border-border bg-bg-deep text-text-primary hover:border-accent-dim'}"
                  on:click={(event) => startRecording(command.id, event)}
                  on:keydown={(event) => onRecordKey(event, command.id)}
                  on:blur={() => recordingId === command.id && (recordingId = null)}
                >
                  {#if recordingId === command.id}
                    Press keys…
                  {:else}
                    {displayAccelerator(accelerator) || "Unset"}
                  {/if}
                </button>
                <button
                  type="button"
                  aria-label={`Reset ${command.label} to default`}
                  class="inline-flex h-7 w-7 items-center justify-center rounded border border-border-subtle bg-bg-surface/60 text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary disabled:opacity-40"
                  disabled={accelerator === command.default}
                  on:click={() => resetToDefault(command)}
                >
                  <RotateCcw size={13} />
                </button>
              </div>
            {:else}
              <div
                class="shrink-0 rounded border border-border bg-bg-deep px-2 py-1 font-mono text-[12px] text-text-muted"
              >
                {displayAccelerator(accelerator)}
              </div>
            {/if}
          </div>
        {/each}
      </div>
    </div>
  {/each}

  <div class="mt-5 flex items-center gap-3">
    <button
      type="button"
      class="inline-flex h-8 items-center justify-center rounded border border-accent bg-accent-dim/30 px-3 text-[12px] font-medium text-text-primary transition-colors hover:bg-accent-dim/50 disabled:cursor-not-allowed disabled:border-border disabled:bg-bg-deep disabled:text-text-muted"
      disabled={saving || hasConflicts || !dirty}
      on:click={save}
    >
      {saving ? "Saving…" : "Save Shortcuts"}
    </button>
    {#if hasConflicts}
      <span class="text-[11px] text-red">Resolve conflicts before saving.</span>
    {:else if saveError}
      <span class="text-[11px] text-red">{saveError}</span>
    {:else if savedNotice}
      <span class="text-[11px] text-green">{savedNotice}</span>
    {:else if dirty}
      <span class="text-[11px] text-text-muted">Unsaved changes</span>
    {/if}
  </div>
{/if}
