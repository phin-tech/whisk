<script lang="ts">
  import RotateCcw from "@lucide/svelte/icons/rotate-ccw";
  import { onMount } from "svelte";
  import type { CommandView } from "../bindings/github.com/phin-tech/whisk/internal/appmenu/models";
  import {
    LoadKeybindings,
    SaveKeybindings,
  } from "../bindings/github.com/phin-tech/whisk/internal/wailsapp/service";
  import { displayAccelerator, findConflicts, formatAccelerator } from "./keybindingsView";
  import Button from "./ui/Button.svelte";
  import IconButton from "./ui/IconButton.svelte";
  import List from "./ui/List.svelte";
  import ListRow from "./ui/ListRow.svelte";

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
      <List class="border-y border-hairline">
        {#each group.commands as command}
          {@const accelerator = command.editable ? (draft[command.id] ?? "") : command.default}
          {@const conflicting = conflictingIds.has(command.id)}
          <ListRow class="flex items-center justify-between gap-3">
            <div class="min-w-0">
              <div class="truncate text-[13px] text-text-primary">{command.label}</div>
              {#if conflicting}
                <div class="mt-0.5 text-[11px] text-red">Conflicts with another shortcut</div>
              {/if}
            </div>
            {#if command.editable}
              <div class="flex shrink-0 items-center gap-1">
                <Button
                  size="sm"
                  class="min-w-[84px] font-mono {recordingId ===
                  command.id
                    ? '!border-accent !bg-accent-dim/20 !text-accent'
                    : conflicting
                      ? '!border-red/40 !bg-red/10 !text-red hover:!border-red'
                      : '!border-border !bg-bg-deep !text-text-primary hover:!border-accent-dim'}"
                  onclick={(event) => startRecording(command.id, event)}
                  onkeydown={(event: KeyboardEvent) => onRecordKey(event, command.id)}
                  onblur={() => recordingId === command.id && (recordingId = null)}
                >
                  {#if recordingId === command.id}
                    Press keys…
                  {:else}
                    {displayAccelerator(accelerator) || "Unset"}
                  {/if}
                </Button>
                <IconButton
                  label={`Reset ${command.label} to default`}
                  disabled={accelerator === command.default}
                  onclick={() => resetToDefault(command)}
                >
                  <RotateCcw size={13} />
                </IconButton>
              </div>
            {:else}
              <div
                class="shrink-0 rounded border border-border bg-bg-deep px-2 py-1 font-mono text-[12px] text-text-muted"
              >
                {displayAccelerator(accelerator)}
              </div>
            {/if}
          </ListRow>
        {/each}
      </List>
    </div>
  {/each}

  <div class="mt-5 flex items-center gap-3">
    <Button
      variant="primary"
      disabled={saving || hasConflicts || !dirty}
      onclick={save}
    >
      {saving ? "Saving…" : "Save Shortcuts"}
    </Button>
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
