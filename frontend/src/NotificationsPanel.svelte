<script lang="ts">
  import AlertTriangle from "@lucide/svelte/icons/alert-triangle";
  import Ban from "@lucide/svelte/icons/ban";
  import CheckCircle2 from "@lucide/svelte/icons/check-circle-2";
  import ChevronRight from "@lucide/svelte/icons/chevron-right";
  import CircleHelp from "@lucide/svelte/icons/circle-help";
  import ShieldQuestion from "@lucide/svelte/icons/shield-question";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import X from "@lucide/svelte/icons/x";
  import type { Session } from "../bindings/github.com/phin-tech/whisk/internal/domain/session/models";
  import type { AgentBridgeApproval, AgentBridgeEvent, AgentPrompt, StatusEvent } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import { agentHookNotificationRows } from "./agentHooksView";
  import { notificationClearEnabled, notificationDetailRows, notificationRows } from "./notificationsView";
  import SidebarPanelHeader from "./SidebarPanelHeader.svelte";

  export let sessions: Session[] = [];
  export let statusEvents: StatusEvent[] = [];
  export let agentBridgeApprovals: AgentBridgeApproval[] = [];
  export let agentPrompts: AgentPrompt[] = [];
  export let agentBridgeEvents: AgentBridgeEvent[] = [];
  export let loading = false;
  export let onclose: () => void;
  export let onRefresh: () => void;
  export let onClearNotifications: () => void;
  export let onSelectStatusEvent: (event: StatusEvent) => void;
  export let onSelectAgentPrompt: (prompt: AgentPrompt) => void;
  export let onSelectAgentBridgeEvent: (event: AgentBridgeEvent) => void;
  export let onResolveAgentBridgeApproval: (id: string, action: "allow" | "deny") => void;
  export let onResolveAgentPrompt: (prompt: AgentPrompt, answer: string, tuiInput?: string) => void;

  let expandedIds = new Set<string>();
  let promptAnswers: Record<string, string> = {};

  $: rows = notificationRows(statusEvents);
  $: promptMessages = new Set(agentPrompts.map((prompt) => prompt.message).filter(Boolean));
  $: hookRows = agentHookNotificationRows(agentBridgeEvents, sessions).filter(
    (hook) => !promptMessages.has(hook.title) && !promptMessages.has(hook.message),
  );
  $: hasRows = rows.length > 0 || agentPrompts.length > 0 || agentBridgeApprovals.length > 0 || hookRows.length > 0;
  $: canClear = notificationClearEnabled(statusEvents, agentBridgeEvents);

  function iconForTone(tone: string) {
    if (tone === "done") return CheckCircle2;
    if (tone === "warning") return AlertTriangle;
    return CircleHelp;
  }

  function eventById(id: string) {
    return statusEvents.find((event) => event.id === id);
  }

  function isExpanded(id: string) {
    return expandedIds.has(id);
  }

  function toggleExpanded(id: string) {
    const next = new Set(expandedIds);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    expandedIds = next;
  }

  function promptAnswer(id: string) {
    return (promptAnswers[id] || "").trim();
  }

  function setPromptAnswer(id: string, event: Event) {
    promptAnswers = { ...promptAnswers, [id]: (event.currentTarget as HTMLInputElement).value };
  }

  function resolveTextPrompt(prompt: AgentPrompt) {
    const answer = promptAnswer(prompt.id);
    if (!answer) return;
    onResolveAgentPrompt(prompt, answer);
  }

  function resolveOptionPrompt(prompt: AgentPrompt, answer: string, index: number) {
    const tuiInput = prompt.provider === "claude" && prompt.toolName === "AskUserQuestion" ? `${index + 1}\r` : "";
    onResolveAgentPrompt(prompt, answer, tuiInput);
  }
</script>

<div class="flex h-full min-h-0 w-full flex-col bg-bg-deep">
  <SidebarPanelHeader title="Notifications" {onclose}>
    <div slot="actions" class="flex items-center gap-1">
      <button
        type="button"
        class="inline-flex h-6 w-6 items-center justify-center rounded border border-transparent text-text-muted transition-colors hover:border-border-subtle hover:bg-bg-hover hover:text-text-primary focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50 disabled:cursor-default disabled:opacity-60"
        disabled={loading || !canClear}
        aria-label="Clear notifications"
        title="Clear notifications"
        on:click={onClearNotifications}
      >
        <Trash2 size={13} />
      </button>
      <button
        type="button"
        class="inline-flex h-6 w-6 items-center justify-center rounded border border-transparent text-text-muted transition-colors hover:border-border-subtle hover:bg-bg-hover hover:text-text-primary focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50 disabled:cursor-default disabled:opacity-60"
        disabled={loading}
        aria-label="Refresh notifications"
        title="Refresh notifications"
        on:click={onRefresh}
      >
        <RefreshCw size={13} class={loading ? "animate-spin" : ""} />
      </button>
    </div>
  </SidebarPanelHeader>

  <div class="app-scrollbar min-h-0 flex-1 overflow-y-auto p-2">
    {#if !hasRows}
      <div class="flex h-full items-center justify-center px-4 text-center text-[13px] text-text-muted">
        No unread notifications.
      </div>
    {:else}
      <div class="space-y-1">
        {#each agentPrompts as prompt (prompt.id)}
          {@const expanded = isExpanded(prompt.id)}
          {@const hasOptions = (prompt.options || []).length > 0}
          <div
            class="rounded border border-accent-dim/50 bg-accent-dim/10 px-2.5 py-2 text-text-primary"
          >
            <div class="flex min-w-0 items-start gap-2">
              <button
                type="button"
                class="mt-0.5 inline-flex h-4 w-4 shrink-0 items-center justify-center rounded text-accent transition-colors hover:bg-bg-surface/60 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50"
                aria-label={expanded ? "Collapse response controls" : "Expand response controls"}
                aria-expanded={expanded}
                on:click={() => toggleExpanded(prompt.id)}
              >
                <ChevronRight size={13} class="transition-transform {expanded ? 'rotate-90' : ''}" />
              </button>
              <div class="min-w-0 flex-1">
                <button
                  type="button"
                  class="w-full min-w-0 text-left focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50"
                  on:click={() => toggleExpanded(prompt.id)}
                  on:dblclick|stopPropagation={() => onSelectAgentPrompt(prompt)}
                >
                  <div class="flex min-w-0 items-center justify-between gap-2">
                    <div class="truncate text-[12px] font-semibold">
                      {prompt.message}
                    </div>
                    <div class="max-w-[72px] shrink-0 truncate rounded border border-border-subtle px-1.5 py-0.5 text-[10px] uppercase text-text-muted">
                      {prompt.provider || "unknown"}
                    </div>
                  </div>
                  <div class="mt-1 line-clamp-3 text-[12px] leading-4 text-text-secondary">
                    {prompt.provider ? `${prompt.provider.charAt(0).toUpperCase()}${prompt.provider.slice(1)} ` : "Agent "}{prompt.kind}
                  </div>
                  <div class="mt-1 min-w-0 break-all font-mono text-[10px] leading-4 text-text-muted">
                    {prompt.cwd || `${prompt.sessionId || "unowned"} / ${prompt.ptyId || "no pty"}`}
                  </div>
                  {#if !expanded}
                    <div class="mt-1 text-[10px] text-text-muted">Click to respond</div>
                  {/if}
                </button>
                {#if expanded}
                  {#if hasOptions}
                    <div class="mt-2 grid grid-cols-2 gap-1">
                      {#each prompt.options || [] as option, index}
                        <button
                          type="button"
                          class="inline-flex h-7 min-w-0 items-center justify-center gap-1 rounded border text-[12px] font-semibold text-text-primary transition-colors disabled:cursor-default disabled:opacity-60 {option.value ===
                          'allow'
                            ? 'border-green/30 bg-green/10 hover:border-green/60 hover:bg-green/15'
                            : option.value === 'deny'
                              ? 'border-red/30 bg-red/10 hover:border-red/60 hover:bg-red/15'
                              : 'border-accent-dim/50 bg-accent-dim/15 hover:border-accent hover:bg-accent-dim/20'}"
                          disabled={loading}
                          on:click|stopPropagation={() => resolveOptionPrompt(prompt, option.value, index)}
                        >
                          {#if option.value === "allow"}
                            <CheckCircle2 size={13} />
                          {:else if option.value === "deny"}
                            <Ban size={13} />
                          {/if}
                          <span class="truncate">{option.label}</span>
                        </button>
                      {/each}
                    </div>
                  {:else}
                    <form class="mt-2 flex min-w-0 gap-1" on:submit|preventDefault={() => resolveTextPrompt(prompt)}>
                      <input
                        class="min-w-0 flex-1 rounded border border-border-subtle bg-bg-base px-2 py-1 text-[12px] text-text-primary outline-none placeholder:text-text-muted focus:border-accent"
                        placeholder="Type response"
                        value={promptAnswers[prompt.id] || ""}
                        on:input={(event) => setPromptAnswer(prompt.id, event)}
                      />
                      <button
                        type="submit"
                        class="shrink-0 rounded border border-accent-dim/50 bg-accent-dim/15 px-2 py-1 text-[12px] font-semibold text-text-primary transition-colors hover:border-accent disabled:cursor-default disabled:opacity-60"
                        disabled={loading || !promptAnswer(prompt.id)}
                      >
                        Send
                      </button>
                    </form>
                  {/if}
                {/if}
              </div>
            </div>
          </div>
        {/each}
        {#each agentBridgeApprovals as approval (approval.id)}
          <div
            class="rounded border border-accent-dim/50 bg-accent-dim/10 px-2.5 py-2 text-text-primary"
          >
            <div class="flex min-w-0 items-start gap-2">
              <ShieldQuestion size={14} class="mt-0.5 shrink-0 text-accent" />
              <div class="min-w-0 flex-1">
                <div class="flex min-w-0 items-center justify-between gap-2">
                  <div class="truncate text-[12px] font-semibold">Agent approval</div>
                  <div class="shrink-0 rounded border border-border-subtle px-1.5 py-0.5 text-[10px] uppercase text-text-muted">
                    {approval.provider}
                  </div>
                </div>
                <div class="mt-1 line-clamp-3 font-mono text-[12px] leading-4 text-text-secondary">
                  {approval.toolName}{approval.toolInput?.command ? `: ${approval.toolInput.command}` : ""}
                </div>
                <div class="mt-1 truncate font-mono text-[10px] text-text-muted">
                  {approval.sessionId || "unowned"} / {approval.ptyId || "no pty"}
                </div>
                <div class="mt-2 grid grid-cols-2 gap-1">
                  <button
                    type="button"
                    class="inline-flex h-7 items-center justify-center gap-1 rounded border border-green/30 bg-green/10 text-[12px] font-semibold text-text-primary transition-colors hover:border-green/60 hover:bg-green/15 disabled:cursor-default disabled:opacity-60"
                    disabled={loading}
                    on:click={() => onResolveAgentBridgeApproval(approval.id, "allow")}
                  >
                    <CheckCircle2 size={13} />
                    Allow
                  </button>
                  <button
                    type="button"
                    class="inline-flex h-7 items-center justify-center gap-1 rounded border border-red/30 bg-red/10 text-[12px] font-semibold text-text-primary transition-colors hover:border-red/60 hover:bg-red/15 disabled:cursor-default disabled:opacity-60"
                    disabled={loading}
                    on:click={() => onResolveAgentBridgeApproval(approval.id, "deny")}
                  >
                    <Ban size={13} />
                    Deny
                  </button>
                </div>
              </div>
            </div>
          </div>
        {/each}
        {#each hookRows as hook (hook.id)}
          <div class="rounded border border-accent-dim/40 bg-accent-dim/10 px-2.5 py-2 text-text-primary">
            <button
              type="button"
              class="flex w-full min-w-0 items-start gap-2 text-left focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50"
              on:click={() => {
                const event = agentBridgeEvents.find((candidate) => candidate.id === hook.id);
                if (event) onSelectAgentBridgeEvent(event);
              }}
            >
              <CircleHelp size={14} class="mt-0.5 shrink-0 text-accent" />
              <div class="min-w-0 flex-1">
                <div class="flex min-w-0 items-center justify-between gap-2">
                  <div class="truncate text-[12px] font-semibold">{hook.title}</div>
                  <div class="max-w-[72px] shrink-0 truncate rounded border border-border-subtle px-1.5 py-0.5 text-[10px] uppercase text-text-muted">
                    {hook.provider || "unknown"}
                  </div>
                </div>
                <div class="mt-1 line-clamp-3 text-[12px] leading-4 text-text-secondary">
                  {hook.message}
                </div>
                <div class="mt-1 min-w-0 break-all font-mono text-[10px] leading-4 text-text-muted">
                  {hook.meta}
                </div>
              </div>
            </button>
          </div>
        {/each}
        {#each rows as row (row.id)}
          {@const Icon = iconForTone(row.tone)}
          {@const event = eventById(row.id)}
          {@const expanded = isExpanded(row.id)}
          <div
            class="rounded border transition-colors {row.tone ===
            'done'
              ? 'border-green/25 bg-green/5 text-text-secondary hover:border-green/40 hover:bg-green/10'
              : row.tone === 'warning'
                ? 'border-amber/30 bg-amber/5 text-text-primary hover:border-amber/50 hover:bg-amber/10'
                : 'border-accent-dim/40 bg-accent-dim/10 text-text-primary hover:border-accent hover:bg-accent-dim/15'}"
          >
            <div class="flex min-w-0 items-start gap-2 px-2.5 py-2">
              <button
                type="button"
                class="mt-0.5 inline-flex h-4 w-4 shrink-0 items-center justify-center rounded text-text-muted transition-colors hover:bg-bg-surface/60 hover:text-text-primary focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50"
                aria-label={expanded ? "Collapse notification details" : "Expand notification details"}
                aria-expanded={expanded}
                on:click={() => toggleExpanded(row.id)}
              >
                <ChevronRight size={13} class="transition-transform {expanded ? 'rotate-90' : ''}" />
              </button>
              <button
                type="button"
                class="min-w-0 flex-1 text-left focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50"
                on:click={() => event && onSelectStatusEvent(event)}
                on:dblclick|stopPropagation={() => toggleExpanded(row.id)}
              >
                <div class="flex min-w-0 items-start gap-2">
                  <Icon size={14} class="mt-0.5 shrink-0" />
                  <div class="min-w-0 flex-1">
                    <div class="flex min-w-0 items-center justify-between gap-2">
                      <div class="truncate text-[12px] font-semibold">{row.title}</div>
                      <X size={11} class="shrink-0 opacity-45" />
                    </div>
                    <div class="mt-1 line-clamp-3 text-[12px] leading-4 text-text-secondary">
                      {row.message}
                    </div>
                    <div class="mt-1 truncate font-mono text-[10px] text-text-muted">
                      {row.meta}
                    </div>
                  </div>
                </div>
              </button>
            </div>
            {#if expanded && event}
              <div class="border-t border-hairline px-2.5 py-2">
                <div class="grid grid-cols-[72px_1fr] gap-x-2 gap-y-1">
                  {#each notificationDetailRows(event, sessions) as detail}
                    <div class="truncate text-[10px] uppercase text-text-muted">{detail.label}</div>
                    <div class="min-w-0 break-all font-mono text-[11px] text-text-secondary">
                      {detail.value}
                    </div>
                  {/each}
                </div>
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>
