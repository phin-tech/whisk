<script lang="ts">
  import FolderTree from "@lucide/svelte/icons/folder-tree";
  import Server from "@lucide/svelte/icons/server";
  import Settings from "@lucide/svelte/icons/settings";
  import TerminalIcon from "@lucide/svelte/icons/terminal";
  import X from "@lucide/svelte/icons/x";
  import DaemonSettings from "./DaemonSettings.svelte";

  export let visible = false;
  export let railSide: "left" | "right" = "right";
  export let startupView: "sessions" | "kanban" = "sessions";
  export let terminalFontSize = 13;
  export let terminalCursorBlink = true;
  export let keepDaemonAlive = true;
  export let onclose: () => void;
  export let onRailSide: (side: "left" | "right") => void;
  export let onStartupView: (view: "sessions" | "kanban") => void;
  export let onTerminalFontSize: (size: number) => void;
  export let onTerminalCursorBlink: (blink: boolean) => void;
  export let onKeepDaemonAlive: (keep: boolean) => void;

  type Category = "general" | "sessions" | "terminal" | "daemon";

  let selected: Category = "general";

  const categories = [
    { id: "general" as const, label: "General", icon: Settings },
    { id: "sessions" as const, label: "Sessions", icon: FolderTree },
    { id: "terminal" as const, label: "Terminal", icon: TerminalIcon },
    { id: "daemon" as const, label: "Daemon", icon: Server },
  ];

  function handleKey(event: KeyboardEvent) {
    if (event.key === "Escape") {
      event.preventDefault();
      onclose();
    }
  }
</script>

<svelte:window on:keydown={visible ? handleKey : undefined} />

{#if visible}
  <div
    class="absolute inset-0 z-20 flex min-h-0 overflow-hidden border-l border-hairline bg-bg-deep"
    role="region"
    aria-label="Preferences"
  >
    <aside class="flex w-[180px] shrink-0 flex-col border-r border-hairline bg-bg-surface/30 py-3">
      <div class="flex items-center gap-2 px-3 pb-2">
        <button
          type="button"
          aria-label="Close settings"
          class="rounded border border-transparent bg-transparent p-1 text-text-muted transition-colors hover:border-border-subtle hover:bg-bg-hover hover:text-text-primary focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50"
          on:click={onclose}
        >
          <X size={14} />
        </button>
        <div class="text-[11px] font-semibold uppercase tracking-widest text-text-muted">
          Preferences
        </div>
      </div>
      <nav class="flex flex-col gap-0.5 px-2">
        {#each categories as category}
          {@const Icon = category.icon}
          <button
            type="button"
            class="flex items-center gap-2 rounded-md px-2 py-1.5 text-left text-[13px] transition-colors {selected ===
            category.id
              ? 'bg-accent-dim text-text-primary'
              : 'text-text-secondary hover:bg-bg-hover'}"
            on:click={() => (selected = category.id)}
          >
            <Icon size={14} />
            <span>{category.label}</span>
          </button>
        {/each}
      </nav>
    </aside>

    <div class="flex min-w-0 flex-1 flex-col">
      <div class="flex h-10 shrink-0 items-center border-b border-hairline px-4">
        <h2 class="text-sm font-semibold tracking-tight">
          {categories.find((category) => category.id === selected)?.label}
        </h2>
      </div>

      <div class="app-scrollbar flex-1 overflow-y-auto px-5 py-4">
        {#if selected === "general"}
          <div class="rounded-xl border border-border-subtle bg-bg-surface/35 p-3">
            <div class="flex items-start justify-between gap-3">
              <div>
                <div class="text-[13px]">Theme</div>
                <div class="mt-0.5 text-[11px] text-text-muted">
                  Refined Zinc
                </div>
              </div>
              <div class="rounded border border-border bg-bg-deep px-2 py-1 text-xs text-text-secondary">
                Default
              </div>
            </div>
          </div>

          <div class="mt-4 flex items-center justify-between gap-3 py-2">
            <div>
              <div class="text-[13px]">Open to</div>
              <div class="mt-0.5 text-[11px] text-text-muted">
                Initial workspace after launch.
              </div>
            </div>
            <div class="flex overflow-hidden rounded border border-border bg-bg-deep">
              {#each [{ id: "sessions", label: "Sessions" }, { id: "kanban", label: "Kanban" }] as option}
                <button
                  type="button"
                  class="px-2.5 py-1 text-[11px] transition-colors {startupView ===
                  option.id
                    ? 'bg-accent-dim text-text-primary'
                    : 'text-text-secondary hover:bg-bg-hover'}"
                  aria-pressed={startupView === option.id}
                  on:click={() => onStartupView(option.id as "sessions" | "kanban")}
                >
                  {option.label}
                </button>
              {/each}
            </div>
          </div>

          <div class="mt-4 flex items-center justify-between py-2">
            <div>
              <div class="text-[13px]">Activity rail</div>
              <div class="mt-0.5 text-[11px] text-text-muted">
                Position of the icon rail and sidebar dock.
              </div>
            </div>
            <div class="flex overflow-hidden rounded border border-border bg-bg-deep">
              {#each ["left", "right"] as side}
                <button
                  type="button"
                  class="px-2.5 py-1 text-[11px] transition-colors {railSide ===
                  side
                    ? 'bg-accent-dim text-text-primary'
                    : 'text-text-secondary hover:bg-bg-hover'}"
                  aria-pressed={railSide === side}
                  on:click={() => onRailSide(side as "left" | "right")}
                >
                  {side}
                </button>
              {/each}
            </div>
          </div>
        {:else if selected === "sessions"}
          <div class="flex items-center justify-between py-2">
            <div>
              <div class="text-[13px]">Restore on launch</div>
              <div class="mt-0.5 text-[11px] text-text-muted">
                Placeholder for daemon-owned session restore settings.
              </div>
            </div>
            <div class="relative h-5 w-9 rounded-full border border-border bg-bg-deep">
              <div class="absolute left-0.5 top-0.5 h-3.5 w-3.5 rounded-full bg-text-secondary"></div>
            </div>
          </div>

          <div class="flex items-center justify-between py-2">
            <div>
              <div class="text-[13px]">On pane close</div>
              <div class="mt-0.5 text-[11px] text-text-muted">
                Kill/detach controls land when the daemon contract exists.
              </div>
            </div>
            <div class="rounded border border-border bg-bg-deep px-2.5 py-1 text-[11px] text-text-muted">
              Kill
            </div>
          </div>
        {:else if selected === "terminal"}
          <div class="flex items-center justify-between py-2">
            <span class="text-[13px]">Font size</span>
            <input
              class="w-20 rounded border border-border bg-bg-deep px-2 py-1 text-right text-xs text-text-primary outline-none focus:border-accent-dim"
              type="number"
              min="10"
              max="20"
              value={terminalFontSize}
              on:input={(event) =>
                onTerminalFontSize(Number(event.currentTarget.value))}
            />
          </div>

          <div class="flex items-center justify-between py-2">
            <div>
              <div class="text-[13px]">Cursor blink</div>
              <div class="mt-0.5 text-[11px] text-text-muted">
                Applies to newly mounted terminal panes.
              </div>
            </div>
            <button
              type="button"
              aria-label="Toggle cursor blink"
              class="relative h-5 w-9 rounded-full border transition-all {terminalCursorBlink
                ? 'border-accent bg-accent-dim'
                : 'border-border bg-bg-deep'}"
              on:click={() => onTerminalCursorBlink(!terminalCursorBlink)}
            >
              <div
                class="absolute top-0.5 h-3.5 w-3.5 rounded-full transition-all {terminalCursorBlink
                  ? 'left-[18px] bg-accent'
                  : 'left-0.5 bg-text-secondary'}"
              ></div>
            </button>
          </div>
        {:else if selected === "daemon"}
          <DaemonSettings {keepDaemonAlive} onKeepDaemonAlive={onKeepDaemonAlive} />
        {/if}
      </div>
    </div>
  </div>
{/if}
