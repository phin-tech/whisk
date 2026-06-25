<script lang="ts">
  import Check from "@lucide/svelte/icons/check";
  import ChevronRight from "@lucide/svelte/icons/chevron-right";
  import Copy from "@lucide/svelte/icons/copy";
  import Download from "@lucide/svelte/icons/download";
  import ExternalLink from "@lucide/svelte/icons/external-link";
  import Plug from "@lucide/svelte/icons/plug";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import type { AgentBridgeEvent, AgentHookIntegration, AgentHookLogStatus } from "../../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import { agentHookDebugDetailRows, agentHookDebugRows, agentHookIntegrationFor } from "../agentHooksView";
  import Button from "../ui/Button.svelte";
  import IconButton from "../ui/IconButton.svelte";
  import List from "../ui/List.svelte";
  import ListRow from "../ui/ListRow.svelte";
  import Switch from "../ui/Switch.svelte";

  type Props = {
    agentHookIntegrations: AgentHookIntegration[];
    agentHookLogStatus: AgentHookLogStatus | null;
    agentBridgeEvents: AgentBridgeEvent[];
    agentHookAction: string;
    agentHookNotice: string;
    onRefreshAgentHookIntegrations: () => void;
    onCheckAgentHookIntegration: (provider: string) => void;
    onInstallAgentHookIntegration: (provider: string) => void;
    onRemoveAgentHookIntegration: (provider: string) => void;
    onHookLogEnabled: (enabled: boolean) => void;
    onClearHookLogAfterSession: (enabled: boolean) => void;
    onClearAgentHookLog: () => void;
    onClearAgentHookEvents: () => void;
    onOpenAgentHookLog: () => void;
    onCopyAgentHookLogPath: (path: string) => void;
    onRefreshAgentHookEvents: () => void;
  };

  let {
    agentHookIntegrations,
    agentHookLogStatus,
    agentBridgeEvents,
    agentHookAction,
    agentHookNotice,
    onRefreshAgentHookIntegrations,
    onCheckAgentHookIntegration,
    onInstallAgentHookIntegration,
    onRemoveAgentHookIntegration,
    onHookLogEnabled,
    onClearHookLogAfterSession,
    onClearAgentHookLog,
    onClearAgentHookEvents,
    onOpenAgentHookLog,
    onCopyAgentHookLogPath,
    onRefreshAgentHookEvents,
  }: Props = $props();

  const providers = [
    { id: "claude", label: "Claude Code" },
    { id: "codex", label: "Codex" },
  ];

  const hookEventRows = $derived(agentHookDebugRows(agentBridgeEvents));
  const hookEventsById = $derived(new Map(agentBridgeEvents.map((event) => [event.id, event])));
  let expandedHookEventIds = $state(new Set<string>());

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
</script>

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
  <div class="mb-3 rounded border border-accent-dim/40 bg-accent-dim/10 px-2.5 py-2 text-[12px] text-text-secondary">
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
      <div class="mt-0.5 text-[11px] text-text-muted">Redacted JSONL hook payloads for debugging.</div>
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
    <div class="mt-1 text-[10px] text-text-muted">{bytesLabel(agentHookLogStatus?.sizeBytes)}</div>
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
    <Button size="sm" disabled={agentHookAction !== ""} onclick={onOpenAgentHookLog}>
      <ExternalLink size={13} />
      <span>Open in Editor</span>
    </Button>
    <Button variant="danger" size="sm" disabled={agentHookAction !== ""} onclick={onClearAgentHookLog}>
      <Trash2 size={13} />
      <span>Clear Log</span>
    </Button>
  </div>

  <div class="mt-3 flex items-center justify-between gap-3 py-1">
    <div>
      <div class="text-[12px] text-text-primary">Clear after session</div>
      <div class="mt-0.5 text-[11px] text-text-muted">Remove hook logs when the daemon shuts down.</div>
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
        <div class="mt-0.5 text-[11px] text-text-muted">Passive provider hook events.</div>
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
                <div class="truncate font-mono text-[10px] text-text-muted">{event.createdAt}</div>
              </div>
              <Button
                variant="ghost"
                align="start"
                class="!h-auto min-w-0 flex-col !items-start gap-0 !border-transparent !bg-transparent !px-0 !py-0 text-left hover:!bg-transparent hover:!text-inherit"
                ondblclick={() => toggleHookEventExpanded(event.id)}
              >
                <div class="truncate text-[12px] text-text-primary">{event.title}</div>
                <div class="mt-0.5 truncate text-[11px] text-text-secondary">{event.message}</div>
                <div class="mt-0.5 truncate font-mono text-[10px] text-text-muted">{event.meta}</div>
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
