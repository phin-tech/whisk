<script lang="ts">
  import FolderTree from "@lucide/svelte/icons/folder-tree";
  import Keyboard from "@lucide/svelte/icons/keyboard";
  import Plug from "@lucide/svelte/icons/plug";
  import Server from "@lucide/svelte/icons/server";
  import Settings from "@lucide/svelte/icons/settings";
  import TerminalIcon from "@lucide/svelte/icons/terminal";
  import X from "@lucide/svelte/icons/x";
  import type { AgentBridgeEvent, AgentHookIntegration, AgentHookLogStatus, PluginStatus, RegistryPlugin } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import type { DaemonStatus } from "../bindings/github.com/phin-tech/whisk/internal/wailsapp/models";
  import DaemonSettings from "./DaemonSettings.svelte";
  import KeybindingsPanel from "./KeybindingsPanel.svelte";
  import GeneralSettings from "./settings/GeneralSettings.svelte";
  import IntegrationsSettings from "./settings/IntegrationsSettings.svelte";
  import PluginsSettings from "./settings/PluginsSettings.svelte";
  import TerminalSettings from "./settings/TerminalSettings.svelte";
  import Button from "./ui/Button.svelte";
  import IconButton from "./ui/IconButton.svelte";

  export let visible = false;
  export let railSide: "left" | "right" = "right";
  export let startupView: "sessions" | "kanban" = "sessions";
  export let terminalFontSize = 13;
  export let terminalCursorBlink = true;
  export let keepDaemonAlive = true;
  export let autoRestartManagedDaemon = false;
  export let daemonStatus: DaemonStatus | null = null;
  export let worktrunkPath = "/opt/homebrew/bin/wt";
  export let agentHookIntegrations: AgentHookIntegration[] = [];
  export let plugins: PluginStatus[] = [];
  export let agentHookLogStatus: AgentHookLogStatus | null = null;
  export let agentBridgeEvents: AgentBridgeEvent[] = [];
  export let agentHookAction = "";
  export let agentHookNotice = "";
  export let onclose: () => void;
  export let onRailSide: (side: "left" | "right") => void;
  export let onStartupView: (view: "sessions" | "kanban") => void;
  export let onTerminalFontSize: (size: number) => void;
  export let onTerminalCursorBlink: (blink: boolean) => void;
  export let onKeepDaemonAlive: (keep: boolean) => void;
  export let onAutoRestartManagedDaemon: (enabled: boolean) => void;
  export let onDaemonStatus: (status: DaemonStatus) => void;
  export let onWorktrunkPath: (path: string) => void;
  export let onRefreshAgentHookIntegrations: () => void;
  export let onRefreshPlugins: () => void;
  export let onSetPluginTrusted: (pluginId: string, trusted: boolean) => void;
  export let registryPlugins: RegistryPlugin[] = [];
  export let installingPluginId = "";
  export let onRefreshRegistry: () => void;
  export let onInstallPlugin: (registry: string, pluginId: string) => void;
  export let onCheckAgentHookIntegration: (provider: string) => void;
  export let onInstallAgentHookIntegration: (provider: string) => void;
  export let onRemoveAgentHookIntegration: (provider: string) => void;
  export let onHookLogEnabled: (enabled: boolean) => void;
  export let onClearHookLogAfterSession: (enabled: boolean) => void;
  export let onClearAgentHookLog: () => void;
  export let onClearAgentHookEvents: () => void;
  export let onOpenAgentHookLog: () => void;
  export let onCopyAgentHookLogPath: (path: string) => void;
  export let onRefreshAgentHookEvents: () => void;
  export let onRunOnboarding: () => void;

  type Category = "general" | "sessions" | "terminal" | "shortcuts" | "daemon" | "plugins" | "integrations";

  let selected: Category = "general";

  const categories = [
    { id: "general" as const, label: "General", icon: Settings },
    { id: "sessions" as const, label: "Sessions", icon: FolderTree },
    { id: "terminal" as const, label: "Terminal", icon: TerminalIcon },
    { id: "shortcuts" as const, label: "Shortcuts", icon: Keyboard },
    { id: "daemon" as const, label: "Daemon", icon: Server },
    { id: "plugins" as const, label: "Plugins", icon: Plug },
    { id: "integrations" as const, label: "Integrations", icon: Plug },
  ];

  function handleKey(event: KeyboardEvent) {
    if (event.key === "Escape") {
      event.preventDefault();
      onclose();
    }
  }
</script>

<svelte:window onkeydown={visible ? handleKey : undefined} />

{#if visible}
  <div
    class="absolute inset-0 z-20 flex min-h-0 overflow-hidden border-l border-hairline bg-bg-deep"
    role="region"
    aria-label="Preferences"
  >
    <aside class="flex w-[180px] shrink-0 flex-col border-r border-hairline bg-bg-surface/30 py-3">
      <div class="flex items-center gap-2 px-3 pb-2">
        <IconButton label="Close settings" size="sm" onclick={onclose}>
          <X size={14} />
        </IconButton>
        <div class="text-[11px] font-semibold uppercase tracking-widest text-text-muted">
          Preferences
        </div>
      </div>
      <nav class="flex flex-col gap-0.5 px-2">
        {#each categories as category}
          {@const Icon = category.icon}
          <Button
            variant={selected === category.id ? "primary" : "ghost"}
            size="sm"
            align="start"
            class="h-8 w-full gap-2 border-transparent text-[13px] {selected === category.id
              ? ''
              : 'bg-transparent text-text-secondary'}"
            onclick={() => (selected = category.id)}
          >
            <Icon size={14} />
            <span>{category.label}</span>
          </Button>
        {/each}
      </nav>
    </aside>

    <div class="flex min-w-0 flex-1 flex-col">
      <div class="flex h-10 shrink-0 items-center border-b border-hairline px-4">
        <h2 class="text-[13px] font-semibold tracking-tight">
          {categories.find((category) => category.id === selected)?.label}
        </h2>
      </div>

      <div class="app-scrollbar flex-1 overflow-y-auto px-5 py-4">
        {#if selected === "general"}
          <GeneralSettings
            {railSide}
            {startupView}
            {onRailSide}
            {onStartupView}
            {onRunOnboarding}
          />
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
          <TerminalSettings
            {terminalFontSize}
            {terminalCursorBlink}
            {onTerminalFontSize}
            {onTerminalCursorBlink}
          />
        {:else if selected === "shortcuts"}
          <KeybindingsPanel visible={visible && selected === "shortcuts"} />
        {:else if selected === "daemon"}
          <DaemonSettings
            {keepDaemonAlive}
            {autoRestartManagedDaemon}
            status={daemonStatus}
            {worktrunkPath}
            onKeepDaemonAlive={onKeepDaemonAlive}
            onAutoRestartManagedDaemon={onAutoRestartManagedDaemon}
            onDaemonStatus={onDaemonStatus}
            onWorktrunkPath={onWorktrunkPath}
          />
        {:else if selected === "plugins"}
          <PluginsSettings
            {plugins}
            {registryPlugins}
            {installingPluginId}
            {onRefreshPlugins}
            {onSetPluginTrusted}
            {onRefreshRegistry}
            {onInstallPlugin}
          />
        {:else if selected === "integrations"}
          <IntegrationsSettings
            {agentHookIntegrations}
            {agentHookLogStatus}
            {agentBridgeEvents}
            {agentHookAction}
            {agentHookNotice}
            {onRefreshAgentHookIntegrations}
            {onCheckAgentHookIntegration}
            {onInstallAgentHookIntegration}
            {onRemoveAgentHookIntegration}
            {onHookLogEnabled}
            {onClearHookLogAfterSession}
            {onClearAgentHookLog}
            {onClearAgentHookEvents}
            {onOpenAgentHookLog}
            {onCopyAgentHookLogPath}
            {onRefreshAgentHookEvents}
          />
        {/if}
      </div>
    </div>
  </div>
{/if}
