<script lang="ts">
  import Check from "@lucide/svelte/icons/check";
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
  import type { AgentHookIntegration, AgentHookLogStatus, PluginStatus, RegistryPlugin } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import { agentHookIntegrationFor } from "./agentHooksView";
  import DaemonSettings from "./DaemonSettings.svelte";
  import KeybindingsPanel from "./KeybindingsPanel.svelte";

  export let visible = false;
  export let railSide: "left" | "right" = "right";
  export let startupView: "sessions" | "kanban" = "sessions";
  export let terminalFontSize = 13;
  export let terminalCursorBlink = true;
  export let keepDaemonAlive = true;
  export let agentHookIntegrations: AgentHookIntegration[] = [];
  export let plugins: PluginStatus[] = [];
  export let agentHookLogStatus: AgentHookLogStatus | null = null;
  export let agentHookAction = "";
  export let agentHookNotice = "";
  export let onclose: () => void;
  export let onRailSide: (side: "left" | "right") => void;
  export let onStartupView: (view: "sessions" | "kanban") => void;
  export let onTerminalFontSize: (size: number) => void;
  export let onTerminalCursorBlink: (blink: boolean) => void;
  export let onKeepDaemonAlive: (keep: boolean) => void;
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
  export let onOpenAgentHookLog: () => void;
  export let onCopyAgentHookLogPath: (path: string) => void;
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

          <div class="mt-4 flex items-center justify-between gap-3 py-2">
            <div>
              <div class="text-[13px]">Onboarding</div>
              <div class="mt-0.5 text-[11px] text-text-muted">
                Agent hooks, skills, and plugin trust.
              </div>
            </div>
            <button
              type="button"
              class="inline-flex h-7 items-center justify-center gap-1 rounded border border-border-subtle bg-bg-surface/60 px-2 text-[11px] text-text-primary transition-colors hover:border-accent hover:text-accent"
              on:click={onRunOnboarding}
            >
              <RefreshCw size={13} />
              <span>Re-run</span>
            </button>
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
        {:else if selected === "shortcuts"}
          <KeybindingsPanel visible={visible && selected === "shortcuts"} />
        {:else if selected === "daemon"}
          <DaemonSettings {keepDaemonAlive} onKeepDaemonAlive={onKeepDaemonAlive} />
        {:else if selected === "plugins"}
          <div class="flex items-center justify-between gap-3 pb-3">
            <div>
              <div class="text-[13px]">Plugins</div>
              <div class="mt-0.5 text-[11px] text-text-muted">
                Daemon-loaded plugins from configured plugin directories.
              </div>
            </div>
            <button
              type="button"
              aria-label="Rescan plugins"
              class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded border border-border-subtle bg-bg-surface/60 text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary"
              on:click={onRefreshPlugins}
            >
              <RefreshCw size={14} />
            </button>
          </div>

          <div class="divide-y divide-hairline border-y border-hairline">
            {#if plugins.length === 0}
              <div class="py-3 text-[12px] text-text-muted">No plugins found.</div>
            {:else}
              {#each plugins as plugin (plugin.id)}
                <div class="grid gap-3 py-3 md:grid-cols-[minmax(180px,240px)_1fr_auto] md:items-start">
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

                  <button
                    type="button"
                    aria-label={`${plugin.trusted ? "Untrust" : "Trust"} ${plugin.name || plugin.id}`}
                    class="relative h-5 w-9 rounded-full border transition-all {plugin.trusted
                      ? 'border-accent bg-accent-dim'
                      : 'border-border bg-bg-deep'}"
                    disabled={!plugin.valid}
                    on:click={() => onSetPluginTrusted(plugin.id, !plugin.trusted)}
                  >
                    <div
                      class="absolute top-0.5 h-3.5 w-3.5 rounded-full transition-all {plugin.trusted
                        ? 'left-[18px] bg-accent'
                        : 'left-0.5 bg-text-secondary'}"
                    ></div>
                  </button>
                </div>
              {/each}
            {/if}
          </div>

          <div class="mt-6 flex items-center justify-between gap-3 pb-3">
            <div>
              <div class="text-[13px]">Available plugins</div>
              <div class="mt-0.5 text-[11px] text-text-muted">
                Installable from the configured plugin registries. Installed plugins start untrusted.
              </div>
            </div>
            <button
              type="button"
              aria-label="Refresh registries"
              class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded border border-border-subtle bg-bg-surface/60 text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary"
              on:click={onRefreshRegistry}
            >
              <RefreshCw size={14} />
            </button>
          </div>

          {#if registryPlugins.length === 0}
            <div class="border-y border-hairline py-3 text-[12px] text-text-muted">No registry plugins available.</div>
          {:else}
            {#each registryGroups as [registry, entries] (registry)}
              <div class="mb-2 mt-3 text-[11px] font-semibold uppercase tracking-wider text-text-muted">{registry}</div>
              <div class="divide-y divide-hairline border-y border-hairline">
                {#each entries as entry (entry.id)}
                  <div class="grid gap-3 py-3 md:grid-cols-[minmax(180px,240px)_1fr_auto] md:items-start">
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

                    <button
                      type="button"
                      class="inline-flex h-7 items-center gap-1.5 rounded border border-border-subtle bg-bg-surface/60 px-2.5 text-[11px] text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary disabled:opacity-50"
                      disabled={entry.installed || installingPluginId === `${entry.registry}/${entry.id}`}
                      on:click={() => onInstallPlugin(entry.registry, entry.id)}
                    >
                      <Download size={12} />
                      {installingPluginId === `${entry.registry}/${entry.id}` ? "Installing…" : entry.installed ? "Installed" : "Install"}
                    </button>
                  </div>
                {/each}
              </div>
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
            <button
              type="button"
              aria-label="Refresh agent hook integrations"
              class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded border border-border-subtle bg-bg-surface/60 text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary disabled:cursor-wait disabled:opacity-60"
              disabled={agentHookAction !== ""}
              on:click={onRefreshAgentHookIntegrations}
            >
              <RefreshCw size={14} />
            </button>
          </div>

          {#if agentHookNotice}
            <div
              class="mb-3 rounded border border-accent-dim/40 bg-accent-dim/10 px-2.5 py-2 text-[12px] text-text-secondary"
            >
              {agentHookNotice}
            </div>
          {/if}

          <div class="divide-y divide-hairline border-y border-hairline">
            {#each providers as provider}
              {@const integration = integrationFor(provider.id)}
              {@const installed = isInstalled(integration)}
              <div class="grid gap-3 py-3 md:grid-cols-[minmax(160px,220px)_1fr_auto] md:items-start">
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
                  <button
                    type="button"
                    aria-label={`Check ${provider.label} hooks`}
                    class="inline-flex h-7 w-7 items-center justify-center rounded border border-border-subtle bg-bg-surface/60 text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary disabled:cursor-wait disabled:opacity-60"
                    disabled={actionBusy(provider.id)}
                    on:click={() => onCheckAgentHookIntegration(provider.id)}
                  >
                    <Check size={14} />
                  </button>
                  <button
                    type="button"
                    aria-label={`${installLabel(integration.status)} ${provider.label} hooks`}
                    class="inline-flex h-7 items-center justify-center gap-1 rounded border border-border-subtle bg-bg-surface/60 px-2 text-[11px] text-text-primary transition-colors hover:border-accent hover:text-accent disabled:cursor-wait disabled:opacity-60"
                    disabled={actionBusy(provider.id)}
                    on:click={() => onInstallAgentHookIntegration(provider.id)}
                  >
                    <Download size={13} />
                    <span>{installLabel(integration.status)}</span>
                  </button>
                  <button
                    type="button"
                    aria-label={`Remove ${provider.label} hooks`}
                    class="inline-flex h-7 w-7 items-center justify-center rounded border border-border-subtle bg-bg-surface/60 text-text-secondary transition-colors hover:border-red/40 hover:text-red disabled:cursor-wait disabled:opacity-50"
                    disabled={!installed || actionBusy(provider.id)}
                    on:click={() => onRemoveAgentHookIntegration(provider.id)}
                  >
                    <Trash2 size={13} />
                  </button>
                </div>
              </div>
            {/each}
          </div>

          <div class="mt-5 border-y border-hairline py-3">
            <div class="flex items-start justify-between gap-3">
              <div>
                <div class="text-[13px] font-medium text-text-primary">Hook log</div>
                <div class="mt-0.5 text-[11px] text-text-muted">
                  Redacted JSONL hook payloads for debugging.
                </div>
              </div>
              <button
                type="button"
                aria-label="Toggle hook logging"
                class="relative h-5 w-9 rounded-full border transition-all {agentHookLogStatus?.enabled
                  ? 'border-accent bg-accent-dim'
                  : 'border-border bg-bg-deep'}"
                disabled={agentHookAction !== ""}
                on:click={() => onHookLogEnabled(!(agentHookLogStatus?.enabled ?? true))}
              >
                <div
                  class="absolute top-0.5 h-3.5 w-3.5 rounded-full transition-all {agentHookLogStatus?.enabled
                    ? 'left-[18px] bg-accent'
                    : 'left-0.5 bg-text-secondary'}"
                ></div>
              </button>
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
              <button
                type="button"
                class="inline-flex h-7 items-center justify-center gap-1 rounded border border-border-subtle bg-bg-surface/60 px-2 text-[11px] text-text-primary transition-colors hover:border-accent hover:text-accent disabled:cursor-wait disabled:opacity-60"
                disabled={!agentHookLogStatus?.path || agentHookAction !== ""}
                on:click={() => agentHookLogStatus?.path && onCopyAgentHookLogPath(agentHookLogStatus.path)}
              >
                <Copy size={13} />
                <span>Copy Location</span>
              </button>
              <button
                type="button"
                class="inline-flex h-7 items-center justify-center gap-1 rounded border border-border-subtle bg-bg-surface/60 px-2 text-[11px] text-text-primary transition-colors hover:border-accent hover:text-accent disabled:cursor-wait disabled:opacity-60"
                disabled={agentHookAction !== ""}
                on:click={onOpenAgentHookLog}
              >
                <ExternalLink size={13} />
                <span>Open in Editor</span>
              </button>
              <button
                type="button"
                class="inline-flex h-7 items-center justify-center gap-1 rounded border border-red/30 bg-red/10 px-2 text-[11px] text-red transition-colors hover:border-red disabled:cursor-wait disabled:opacity-60"
                disabled={agentHookAction !== ""}
                on:click={onClearAgentHookLog}
              >
                <Trash2 size={13} />
                <span>Clear Log</span>
              </button>
            </div>

            <div class="mt-3 flex items-center justify-between gap-3 py-1">
              <div>
                <div class="text-[12px] text-text-primary">Clear after session</div>
                <div class="mt-0.5 text-[11px] text-text-muted">
                  Remove hook logs when the daemon shuts down.
                </div>
              </div>
              <button
                type="button"
                aria-label="Toggle clear hook log after session"
                class="relative h-5 w-9 rounded-full border transition-all {agentHookLogStatus?.clearAfterSession
                  ? 'border-accent bg-accent-dim'
                  : 'border-border bg-bg-deep'}"
                disabled={agentHookAction !== ""}
                on:click={() => onClearHookLogAfterSession(!(agentHookLogStatus?.clearAfterSession ?? false))}
              >
                <div
                  class="absolute top-0.5 h-3.5 w-3.5 rounded-full transition-all {agentHookLogStatus?.clearAfterSession
                    ? 'left-[18px] bg-accent'
                    : 'left-0.5 bg-text-secondary'}"
                ></div>
              </button>
            </div>
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}
