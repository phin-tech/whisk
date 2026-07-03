<script lang="ts">
  import Bell from "@lucide/svelte/icons/bell";
  import FolderTree from "@lucide/svelte/icons/folder-tree";
  import ListTodo from "@lucide/svelte/icons/list-todo";
  import PanelsTopLeft from "@lucide/svelte/icons/panels-top-left";
  import SettingsIcon from "@lucide/svelte/icons/settings";
  import TerminalSquare from "@lucide/svelte/icons/square-terminal";
  import IconButton from "./ui/IconButton.svelte";

  export let activeSidebar: "sessions" | "ptys" | "work" | "projects" | "notifications" | null;
  export let settingsOpen = false;
  export let notificationCount = 0;
  export let onSidebar: (id: "sessions" | "ptys" | "work" | "projects" | "notifications") => void;
  export let onSettings: () => void;

  const items = [
    { id: "sessions" as const, label: "Sessions", icon: FolderTree },
    { id: "notifications" as const, label: "Notifications", icon: Bell },
    { id: "work" as const, label: "Work", icon: ListTodo },
    { id: "projects" as const, label: "Projects", icon: PanelsTopLeft },
    { id: "ptys" as const, label: "PTYs", icon: TerminalSquare },
  ];
</script>

<div class="flex h-full flex-col items-center p-1">
  {#each items as item (item.id)}
    {@const Icon = item.icon}
    <IconButton
      label={item.label}
      aria-pressed={activeSidebar === item.id}
      class="group relative flex h-7 w-7 shrink-0 items-center justify-center rounded text-sidebar-foreground transition-colors hover:bg-sidebar-hover/60 hover:text-sidebar-active-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-sidebar-ring/50 {activeSidebar ===
      item.id
        ? 'bg-sidebar-active text-sidebar-active-foreground'
        : ''}"
      onclick={() => onSidebar(item.id)}
    >
      <Icon size={16} />
      {#if item.id === "notifications" && notificationCount > 0}
        <span
          class="absolute -right-0.5 -top-0.5 flex h-3.5 min-w-3.5 items-center justify-center rounded-full border border-sidebar-rail bg-red px-0.5 text-[8px] font-bold leading-none text-text-primary"
          aria-label="{notificationCount} unread notifications"
        >
          {notificationCount > 9 ? "9+" : notificationCount}
        </span>
      {/if}
      <span
        class="pointer-events-none absolute right-full top-1/2 z-50 mr-1 -translate-y-1/2 whitespace-nowrap rounded border border-border-subtle bg-bg-elevated px-2 py-1 text-[11px] font-medium text-text-primary opacity-0 shadow-lg shadow-black/30 transition-opacity duration-75 group-hover:opacity-100 group-focus-visible:opacity-100"
        aria-hidden="true">{item.label}</span
      >
    </IconButton>
  {/each}

  <div class="flex-1"></div>

  <IconButton
    label="Settings"
    aria-pressed={settingsOpen}
    class="group relative flex h-7 w-7 shrink-0 items-center justify-center rounded text-sidebar-foreground transition-colors hover:bg-sidebar-hover/60 hover:text-sidebar-active-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-sidebar-ring/50 {settingsOpen
      ? 'bg-sidebar-active text-sidebar-active-foreground'
      : ''}"
    onclick={onSettings}
  >
    <SettingsIcon size={16} />
    <span
      class="pointer-events-none absolute right-full top-1/2 z-50 mr-1 -translate-y-1/2 whitespace-nowrap rounded border border-border-subtle bg-bg-elevated px-2 py-1 text-[11px] font-medium text-text-primary opacity-0 shadow-lg shadow-black/30 transition-opacity duration-75 group-hover:opacity-100 group-focus-visible:opacity-100"
      aria-hidden="true">Settings</span
    >
  </IconButton>
</div>
