<script lang="ts">
  import FolderTree from "@lucide/svelte/icons/folder-tree";
  import ListTodo from "@lucide/svelte/icons/list-todo";
  import SettingsIcon from "@lucide/svelte/icons/settings";
  import TerminalSquare from "@lucide/svelte/icons/square-terminal";

  export let activeSidebar: "sessions" | "ptys" | "work" | null;
  export let settingsOpen = false;
  export let onSidebar: (id: "sessions" | "ptys" | "work") => void;
  export let onSettings: () => void;

  const items = [
    { id: "sessions" as const, label: "Sessions", icon: FolderTree },
    { id: "work" as const, label: "Work", icon: ListTodo },
    { id: "ptys" as const, label: "PTYs", icon: TerminalSquare },
  ];
</script>

<div class="flex h-full flex-col items-center p-1">
  {#each items as item (item.id)}
    {@const Icon = item.icon}
    <button
      type="button"
      aria-label={item.label}
      aria-pressed={activeSidebar === item.id}
      class="group relative flex h-7 w-7 shrink-0 items-center justify-center rounded text-text-secondary transition-colors hover:bg-white/5 hover:text-text-primary focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50 {activeSidebar ===
      item.id
        ? 'bg-white/10 text-text-primary'
        : ''}"
      on:click={() => onSidebar(item.id)}
    >
      <Icon size={16} />
      <span
        class="pointer-events-none absolute right-full top-1/2 z-50 mr-1 -translate-y-1/2 whitespace-nowrap rounded border border-border-subtle bg-bg-elevated px-2 py-1 text-[11px] font-medium text-text-primary opacity-0 shadow-lg shadow-black/30 transition-opacity duration-75 group-hover:opacity-100 group-focus-visible:opacity-100"
        aria-hidden="true">{item.label}</span
      >
    </button>
  {/each}

  <div class="flex-1"></div>

  <button
    type="button"
    aria-label="Settings"
    aria-pressed={settingsOpen}
    class="group relative flex h-7 w-7 shrink-0 items-center justify-center rounded text-text-secondary transition-colors hover:bg-white/5 hover:text-text-primary focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50 {settingsOpen
      ? 'bg-white/10 text-text-primary'
      : ''}"
    on:click={onSettings}
  >
    <SettingsIcon size={16} />
    <span
      class="pointer-events-none absolute right-full top-1/2 z-50 mr-1 -translate-y-1/2 whitespace-nowrap rounded border border-border-subtle bg-bg-elevated px-2 py-1 text-[11px] font-medium text-text-primary opacity-0 shadow-lg shadow-black/30 transition-opacity duration-75 group-hover:opacity-100 group-focus-visible:opacity-100"
      aria-hidden="true">Settings</span
    >
  </button>
</div>
