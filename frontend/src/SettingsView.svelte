<script lang="ts">
  import Check from "@lucide/svelte/icons/check";
  import ChevronRight from "@lucide/svelte/icons/chevron-right";
  import Copy from "@lucide/svelte/icons/copy";
  import Download from "@lucide/svelte/icons/download";
  import ExternalLink from "@lucide/svelte/icons/external-link";
  import FolderTree from "@lucide/svelte/icons/folder-tree";
  import Keyboard from "@lucide/svelte/icons/keyboard";
  import Plug from "@lucide/svelte/icons/plug";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import Server from "@lucide/svelte/icons/server";
  import Settings from "@lucide/svelte/icons/settings";
  import TerminalIcon from "@lucide/svelte/icons/terminal";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import X from "@lucide/svelte/icons/x";
  import type { AgentBridgeEvent, AgentHookIntegration, AgentHookLogStatus, PluginStatus, RegistryPlugin } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import { agentHookDebugDetailRows, agentHookDebugRows, agentHookIntegrationFor } from "./agentHooksView";
  import DaemonSettings from "./DaemonSettings.svelte";
  import KeybindingsPanel from "./KeybindingsPanel.svelte";
  import Button from "./ui/Button.svelte";
  import IconButton from "./ui/IconButton.svelte";
  import List from "./ui/List.svelte";
  import ListRow from "./ui/ListRow.svelte";
  import Switch from "./ui/Switch.svelte";
  import TextField from "./ui/TextField.svelte";

  export let visible = false;
  export let railSide: "left" | "right" = "right";
  export let startupView: "sessions" | "kanban" = "sessions";
  export let terminalFontSize = 13;
  export let terminalCursorBlink = true;
  export let keepDaemonAlive = true;
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
  export let onWorktrunkPath: (path: string) => void;
  export let onRefreshAgentHookIntegrations: () => void;
  export let onRefreshPlugins: () => void;
  export let onSetPluginTrusted: (pluginId: string, trusted: boolean) => void;
  export let registryPlugins: RegistryPlugin[] = [];
  export let installingPluginId = "";
  export let onRefreshRegistry: () => void;
  export let onInstallPlugin: (registry: string, pluginId: string) => void;

  // Group available plugins by their registry namespace for display.
  $: registryGroups = Object.entries(
    registryPlugins.reduce<Record<string, RegistryPlugin[]>>((groups, plugin) => {
      (groups[plugin.registry] ??= []).push(plugin);
      return groups;
    }, {}),
  );
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
  let expandedHookEventIds = new Set<string>();

  const categories = [
    { id: "general" as const, label: "General", icon: Settings },
    { id: "sessions" as const, label: "Sessions", icon: FolderTree },
    { id: "terminal" as const, label: "Terminal", icon: TerminalIcon },
    { id: "shortcuts" as const, label: "Shortcuts", icon: Keyboard },
    { id: "daemon" as const, label: "Daemon", icon: Server },
    { id: "plugins" as const, label: "Plugins", icon: Plug },
    { id: "integrations" as const, label: "Integrations", icon: Plug },
  ];

  $: hookEventRows = agentHookDebugRows(agentBridgeEvents);
  $: hookEventsById = new Map(agentBridgeEvents.map((event) => [event.id, event]));

  const providers = [
    { id: "claude", label: "Claude Code" },
    { id: "codex", label: "Codex" },
  ];

  function handleKey(event: KeyboardEvent) {
    if (event.key === "Escape") {
      event.preventDefault();
      onclose();
    }
  }

  function hookEventById(id: string) {
    return hookEventsById.get(id);
  }

  function isHookEventExpanded(id: string) {
    return expandedHookEventIds.has(id);
  }

  function toggleHookEventExpanded(id: string) {
    const next = new Set(expandedHookEventIds);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    expandedHookEventIds = next;
  }

  function integrationFor(provider: string): AgentHookIntegration {
    return agentHookIntegrationFor(agentHookIntegrations, provider);
  }

  function isInstalled(integration: AgentHookIntegration) {
    return ["current", "modified", "outdated", "untrusted"].includes(integration.status);
  }

  function statusClass(status: string) {
    if (status === "current") return "border-green/35 bg-green/10 text-green";
    if (status === "untrusted" || status === "outdated") {
      return "border-amber/35 bg-amber/10 text-amber";
    }
    if (status === "modified" || status === "unavailable") {
      return "border-red/30 bg-red/10 text-red";
    }
    return "border-border bg-bg-deep text-text-muted";
  }

  function installLabel(status: string) {
    if (status === "modified") return "Repair";
    if (status === "outdated" || status === "untrusted") return "Update";
    return "Install";
  }

  function actionBusy(provider: string) {
    return agentHookAction.endsWith(`:${provider}`);
  }

  function bytesLabel(bytes: number | undefined) {
    if (!bytes) return "0 B";
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  }

  function pluginStatusClass(plugin: PluginStatus) {
    if (!plugin.valid) return "border-red/30 bg-red/10 text-red";
    if (plugin.trusted) return "border-green/35 bg-green/10 text-green";
    return "border-border bg-bg-deep text-text-muted";
  }

  function pluginResolverLabels(plugin: PluginStatus) {
    return (plugin.resolvers ?? []).map((resolver) => resolver.provider).join(", ");
  }

  function updateTerminalFontSize(event: Event) {
    onTerminalFontSize(Number((event.currentTarget as HTMLInputElement).value));
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
        <IconButton
          label="Close settings"
          size="sm"
          onclick={onclose}
        >
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
            class="h-8 w-full gap-2 border-transparent text-[13px] {selected ===
            category.id
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
          <div class="rounded-xl border border-border-subtle bg-bg-surface/35 p-3">
            <div class="flex items-start justify-between gap-3">
              <div>
                <div class="text-[13px]">Theme</div>
                <div class="mt-0.5 text-[11px] text-text-muted">
                  Refined Zinc
                </div>
              </div>
              <div class="rounded border border-border bg-bg-deep px-2 py-1 text-[12px] text-text-secondary">
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
                <Button
                  variant={startupView === option.id ? "primary" : "ghost"}
                  size="sm"
                  class="h-7 rounded-none border-transparent text-[11px] {startupView === option.id ? '' : 'bg-transparent'}"
                  aria-pressed={startupView === option.id}
                  onclick={() => onStartupView(option.id as "sessions" | "kanban")}
                >
                  {option.label}
                </Button>
              {/each}
            </div>
          </div>

          <div class="mt-4 flex items-center justify-between gap-3 py-2">
            <div>
              <div class="text-[13px]">Onboarding</div>
              <div class="mt-0.5 text-[11px] text-text-muted">
                Agent hooks, skills, and plugin trust.
              </div>
            </div>
            <Button
              size="sm"
              onclick={onRunOnboarding}
            >
              <RefreshCw size={13} />
              <span>Re-run</span>
            </Button>
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
                <Button
                  variant={railSide === side ? "primary" : "ghost"}
                  size="sm"
                  class="h-7 rounded-none border-transparent text-[11px] {railSide === side ? '' : 'bg-transparent'}"
                  aria-pressed={railSide === side}
                  onclick={() => onRailSide(side as "left" | "right")}
                >
                  {side}
                </Button>
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
            <TextField
              class="w-20 text-right"
              type="number"
              min="10"
              max="20"
              value={String(terminalFontSize)}
              oninput={updateTerminalFontSize}
            />
          </div>

          <div class="flex items-center justify-between py-2">
            <div>
              <div class="text-[13px]">Cursor blink</div>
              <div class="mt-0.5 text-[11px] text-text-muted">
                Applies to newly mounted terminal panes.
              </div>
            </div>
            <Switch
              label="Toggle cursor blink"
              checked={terminalCursorBlink}
              onCheckedChange={onTerminalCursorBlink}
            />
          </div>
        {:else if selected === "shortcuts"}
          <KeybindingsPanel visible={visible && selected === "shortcuts"} />
        {:else if selected === "daemon"}
          <DaemonSettings {keepDaemonAlive} {worktrunkPath} onKeepDaemonAlive={onKeepDaemonAlive} onWorktrunkPath={onWorktrunkPath} />
        {:else if selected === "plugins"}
          <div class="flex items-center justify-between gap-3 pb-3">
            <div>
              <div class="text-[13px]">Plugins</div>
              <div class="mt-0.5 text-[11px] text-text-muted">
                Daemon-loaded plugins from configured plugin directories.
              </div>
            </div>
            <IconButton
              label="Rescan plugins"
              class="shrink-0"
              onclick={onRefreshPlugins}
            >
              <RefreshCw size={14} />
            </IconButton>
          </div>

          <List class="border-y border-hairline">
            {#if plugins.length === 0}
              <div class="py-3 text-[12px] text-text-muted">No plugins found.</div>
            {:else}
              {#each plugins as plugin (plugin.id)}
                <ListRow class="grid gap-3 py-3 md:grid-cols-[minmax(180px,240px)_1fr_auto] md:items-start">
                  <div class="min-w-0">
                    <div class="flex items-center gap-2">
                      <Plug size={14} class="shrink-0 text-text-muted" />
                      <span class="truncate text-[13px] font-medium text-text-primary">
                        {plugin.name || plugin.id}
                      </span>
                    </div>
                    <span class="mt-1 inline-flex rounded border px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider {pluginStatusClass(plugin)}">
                      {plugin.valid ? (plugin.trusted ? "trusted" : "untrusted") : "invalid"}
                    </span>
                  </div>

                  <div class="min-w-0 space-y-1 text-[11px] text-text-muted">
                    <div class="truncate">
                      ID <span class="font-mono text-text-secondary">{plugin.id}</span>
                      {plugin.version ? ` · v${plugin.version}` : ""}
                    </div>
                    <div class="truncate">
                      Path <span class="font-mono text-text-secondary">{plugin.dir}</span>
                    </div>
                    {#if pluginResolverLabels(plugin)}
                      <div class="truncate">
                        Resolvers <span class="font-mono text-text-secondary">{pluginResolverLabels(plugin)}</span>
                      </div>
                    {/if}
                    {#if plugin.projectAttachmentTemplates?.length}
                      <div class="truncate">
                        Attachments
                        <span class="text-text-secondary">
                          {plugin.projectAttachmentTemplates.map((template) => template.label || template.id).join(", ")}
                        </span>
                      </div>
                    {/if}
                    {#if plugin.error}
                      <div class="text-red">{plugin.error}</div>
                    {/if}
                  </div>

                  <Switch
                    label={`${plugin.trusted ? "Untrust" : "Trust"} ${plugin.name || plugin.id}`}
                    checked={plugin.trusted}
                    disabled={!plugin.valid}
                    onCheckedChange={(trusted) => onSetPluginTrusted(plugin.id, trusted)}
                  />
                </ListRow>
              {/each}
            {/if}
          </List>

          <div class="mt-6 flex items-center justify-between gap-3 pb-3">
            <div>
              <div class="text-[13px]">Available plugins</div>
              <div class="mt-0.5 text-[11px] text-text-muted">
                Installable from the configured plugin registries. Installed plugins start untrusted.
              </div>
            </div>
            <IconButton
              label="Refresh registries"
              class="shrink-0"
              onclick={onRefreshRegistry}
            >
              <RefreshCw size={14} />
            </IconButton>
          </div>

          {#if registryPlugins.length === 0}
            <div class="border-y border-hairline py-3 text-[12px] text-text-muted">No registry plugins available.</div>
          {:else}
            {#each registryGroups as [registry, entries] (registry)}
              <div class="mb-2 mt-3 text-[11px] font-semibold uppercase tracking-wider text-text-muted">{registry}</div>
              <List class="border-y border-hairline">
                {#each entries as entry (entry.id)}
                  <ListRow class="grid gap-3 py-3 md:grid-cols-[minmax(180px,240px)_1fr_auto] md:items-start">
                    <div class="min-w-0">
                      <div class="flex items-center gap-2">
                        <Plug size={14} class="shrink-0 text-text-muted" />
                        <span class="truncate text-[13px] font-medium text-text-primary">
                          {entry.name || entry.id}
                        </span>
                      </div>
                      <span class="mt-1 inline-flex rounded border px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider {entry.installed ? 'border-green/35 bg-green/10 text-green' : 'border-border bg-bg-deep text-text-muted'}">
                        {entry.installed ? (entry.trusted ? "installed · trusted" : "installed") : "available"}
                      </span>
                    </div>

                    <div class="min-w-0 space-y-1 text-[11px] text-text-muted">
                      <div class="truncate">
                        ID <span class="font-mono text-text-secondary">{entry.id}</span>
                        {entry.sourceType ? ` · ${entry.sourceType}` : ""}
                      </div>
                      {#if entry.description}
                        <div class="truncate text-text-secondary">{entry.description}</div>
                      {/if}
                    </div>

                    <Button
                      size="sm"
                      disabled={entry.installed || installingPluginId === `${entry.registry}/${entry.id}`}
                      onclick={() => onInstallPlugin(entry.registry, entry.id)}
                    >
                      <Download size={12} />
                      {installingPluginId === `${entry.registry}/${entry.id}` ? "Installing…" : entry.installed ? "Installed" : "Install"}
                    </Button>
                  </ListRow>
                {/each}
              </List>
            {/each}
          {/if}
        {:else if selected === "integrations"}
          <div class="flex items-center justify-between gap-3 pb-3">
            <div>
              <div class="text-[13px]">Agent hooks</div>
              <div class="mt-0.5 text-[11px] text-text-muted">
                Global provider hooks managed under <span class="font-mono">~/.config/whisk</span>.
              </div>
            </div>
            <IconButton
              label="Refresh agent hook integrations"
              class="shrink-0"
              disabled={agentHookAction !== ""}
              onclick={onRefreshAgentHookIntegrations}
            >
              <RefreshCw size={14} />
            </IconButton>
          </div>

          {#if agentHookNotice}
            <div
              class="mb-3 rounded border border-accent-dim/40 bg-accent-dim/10 px-2.5 py-2 text-[12px] text-text-secondary"
            >
              {agentHookNotice}
            </div>
          {/if}

          <List class="border-y border-hairline">
            {#each providers as provider}
              {@const integration = integrationFor(provider.id)}
              {@const installed = isInstalled(integration)}
              <ListRow class="grid gap-3 py-3 md:grid-cols-[minmax(160px,220px)_1fr_auto] md:items-start">
                <div class="flex min-w-0 items-start gap-2">
                  <Plug size={14} class="mt-0.5 shrink-0 text-text-muted" />
                  <div class="min-w-0">
                    <span class="block text-[13px] font-medium text-text-primary">{provider.label}</span>
                    <span
                      class="mt-1 inline-flex rounded border px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider {statusClass(
                        integration.status,
                      )}"
                    >
                      {integration.status || "missing"}
                    </span>
                  </div>
                </div>

                <div class="min-w-0 space-y-1 text-[11px] text-text-muted">
                  {#if integration.detail}
                    <div class="text-text-secondary">{integration.detail}</div>
                  {/if}
                  <div class="truncate">
                    Config <span class="font-mono text-text-secondary">{integration.configPath || "not checked"}</span>
                  </div>
                  <div class="truncate">
                    Helper <span class="font-mono text-text-secondary">{integration.helperPath || "not installed"}</span>
                  </div>
                  <div>
                    Version
                    <span class="font-mono text-text-secondary">
                      {integration.installedVersion || "none"} / {integration.latestVersion || "unknown"}
                    </span>
                  </div>
                </div>

                <div class="flex items-center gap-1 md:justify-end">
                  <IconButton
                    label={`Check ${provider.label} hooks`}
                    disabled={actionBusy(provider.id)}
                    onclick={() => onCheckAgentHookIntegration(provider.id)}
                  >
                    <Check size={14} />
                  </IconButton>
                  <Button
                    size="sm"
                    disabled={actionBusy(provider.id)}
                    aria-label={`${installLabel(integration.status)} ${provider.label} hooks`}
                    onclick={() => onInstallAgentHookIntegration(provider.id)}
                  >
                    <Download size={13} />
                    <span>{installLabel(integration.status)}</span>
                  </Button>
                  <IconButton
                    label={`Remove ${provider.label} hooks`}
                    tone="danger"
                    disabled={!installed || actionBusy(provider.id)}
                    onclick={() => onRemoveAgentHookIntegration(provider.id)}
                  >
                    <Trash2 size={13} />
                  </IconButton>
                </div>
              </ListRow>
            {/each}
          </List>

          <div class="mt-5 border-y border-hairline py-3">
            <div class="flex items-start justify-between gap-3">
              <div>
                <div class="text-[13px] font-medium text-text-primary">Hook log</div>
                <div class="mt-0.5 text-[11px] text-text-muted">
                  Redacted JSONL hook payloads for debugging.
                </div>
              </div>
              <Switch
                label="Toggle hook logging"
                checked={agentHookLogStatus?.enabled ?? true}
                disabled={agentHookAction !== ""}
                onCheckedChange={onHookLogEnabled}
              />
            </div>

            <div class="mt-3 rounded border border-border-subtle bg-bg-surface/25 p-2">
              <div class="select-text break-all font-mono text-[11px] leading-4 text-text-secondary">
                {agentHookLogStatus?.path || "~/.config/whisk/agent-hooks/hooks.jsonl"}
              </div>
              <div class="mt-1 text-[10px] text-text-muted">
                {bytesLabel(agentHookLogStatus?.sizeBytes)}
              </div>
            </div>

            <div class="mt-2 flex flex-wrap gap-1">
              <Button
                size="sm"
                disabled={!agentHookLogStatus?.path || agentHookAction !== ""}
                onclick={() => agentHookLogStatus?.path && onCopyAgentHookLogPath(agentHookLogStatus.path)}
              >
                <Copy size={13} />
                <span>Copy Location</span>
              </Button>
              <Button
                size="sm"
                disabled={agentHookAction !== ""}
                onclick={onOpenAgentHookLog}
              >
                <ExternalLink size={13} />
                <span>Open in Editor</span>
              </Button>
              <Button
                variant="danger"
                size="sm"
                disabled={agentHookAction !== ""}
                onclick={onClearAgentHookLog}
              >
                <Trash2 size={13} />
                <span>Clear Log</span>
              </Button>
            </div>

            <div class="mt-3 flex items-center justify-between gap-3 py-1">
              <div>
                <div class="text-[12px] text-text-primary">Clear after session</div>
                <div class="mt-0.5 text-[11px] text-text-muted">
                  Remove hook logs when the daemon shuts down.
                </div>
              </div>
              <Switch
                label="Toggle clear hook log after session"
                checked={agentHookLogStatus?.clearAfterSession ?? false}
                disabled={agentHookAction !== ""}
                onCheckedChange={onClearHookLogAfterSession}
              />
            </div>

            <div class="mt-4 border-t border-hairline pt-3">
              <div class="flex items-center justify-between gap-3">
                <div>
                  <div class="text-[12px] font-medium text-text-primary">Recent hook events</div>
                  <div class="mt-0.5 text-[11px] text-text-muted">
                    Passive provider hook events.
                  </div>
                </div>
                <div class="flex items-center gap-1">
                  <IconButton
                    label="Refresh hook events"
                    disabled={agentHookAction !== ""}
                    onclick={onRefreshAgentHookEvents}
                  >
                    <RefreshCw size={13} />
                  </IconButton>
                  <IconButton
                    label="Clear hook events"
                    tone="danger"
                    disabled={hookEventRows.length === 0 || agentHookAction !== ""}
                    onclick={onClearAgentHookEvents}
                  >
                    <Trash2 size={13} />
                  </IconButton>
                </div>
              </div>

              {#if hookEventRows.length === 0}
                <div class="mt-3 rounded border border-border-subtle bg-bg-surface/25 px-2.5 py-2 text-[11px] text-text-muted">
                  No pending hook events.
                </div>
              {:else}
                <List class="mt-3 border-y border-hairline">
                  {#each hookEventRows as event (event.id)}
                    {@const rawEvent = hookEventById(event.id)}
                    {@const expanded = isHookEventExpanded(event.id)}
                    <ListRow class="py-2">
                      <div class="grid gap-2 md:grid-cols-[20px_92px_1fr]">
                        <IconButton
                          label={expanded ? "Collapse hook event details" : "Expand hook event details"}
                          size="sm"
                          aria-expanded={expanded}
                          onclick={() => toggleHookEventExpanded(event.id)}
                        >
                          <ChevronRight size={13} class="transition-transform {expanded ? 'rotate-90' : ''}" />
                        </IconButton>
                        <div class="min-w-0">
                          <div class="truncate text-[11px] font-semibold uppercase text-text-muted">
                            {event.provider || "unknown"}
                          </div>
                          <div class="truncate font-mono text-[10px] text-text-muted">
                            {event.createdAt}
                          </div>
                        </div>
                        <Button
                          variant="ghost"
                          align="start"
                          class="!h-auto min-w-0 flex-col !items-start gap-0 !border-transparent !bg-transparent !px-0 !py-0 text-left hover:!bg-transparent hover:!text-inherit"
                          ondblclick={() => toggleHookEventExpanded(event.id)}
                        >
                          <div class="truncate text-[12px] text-text-primary">{event.title}</div>
                          <div class="mt-0.5 truncate text-[11px] text-text-secondary">
                            {event.message}
                          </div>
                          <div class="mt-0.5 truncate font-mono text-[10px] text-text-muted">
                            {event.meta}
                          </div>
                        </Button>
                      </div>
                      {#if expanded && rawEvent}
                        <div class="mt-2 border-t border-hairline pt-2 md:ml-[112px]">
                          <div class="grid grid-cols-[84px_1fr] gap-x-2 gap-y-1">
                            {#each agentHookDebugDetailRows(rawEvent) as detail}
                              <div class="truncate text-[10px] uppercase text-text-muted">{detail.label}</div>
                              <div class="min-w-0 break-all font-mono text-[11px] text-text-secondary">
                                {detail.value}
                              </div>
                            {/each}
                          </div>
                        </div>
                      {/if}
                    </ListRow>
                  {/each}
                </List>
              {/if}
            </div>
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}
