<script lang="ts">
  import Plus from "@lucide/svelte/icons/plus";
  import type { Session } from "../../bindings/github.com/phin-tech/whisk/internal/domain/session/models";
  import type { WorkItem, WorkItemRun } from "../../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import Button from "../ui/Button.svelte";
  import EmptyState from "../ui/EmptyState.svelte";
  import List from "../ui/List.svelte";
  import ListRow from "../ui/ListRow.svelte";
  import StatusDot from "../ui/StatusDot.svelte";

  type Counts = {
    sessions: number;
    workItems: number;
    runs: number;
  };

  export let counts: Counts;
  export let recentSessions: Session[] = [];
  export let recentWorkItems: WorkItem[] = [];
  export let recentRuns: WorkItemRun[] = [];
  export let sessions: Session[] = [];
  export let loading = false;
  export let onSelectTab: (tab: "sessions" | "cards" | "runs") => void;
  export let onNewSession: () => void;
  export let onOpenSession: (sessionId: string) => void;
  export let onOpenWorkItem: (workItemId: string) => void;
  export let onOpenRunTerminal: (run: WorkItemRun) => void;
  export let sessionNameSuffix: (session: Session, sessions: Session[]) => string;
  export let runLabel: (run: WorkItemRun) => string;
  export let workItemTitle: (workItemId: string) => string;
</script>

<div class="grid gap-5 xl:grid-cols-3">
  <section>
    <div class="mb-2 flex items-center justify-between gap-2">
      <Button variant="ghost" size="sm" align="start" class="border-transparent bg-transparent px-0 uppercase text-text-muted" onclick={() => onSelectTab("sessions")}>
        Sessions <span class="font-mono">{counts.sessions}</span>
      </Button>
      <Button size="sm" disabled={loading} onclick={onNewSession}>
        <Plus size={13} />
        <span>Add</span>
      </Button>
    </div>
    <List>
      {#if recentSessions.length === 0}
        <EmptyState message="No sessions." />
      {:else}
        {#each recentSessions as session (session.id)}
          <ListRow as="button" onclick={() => onOpenSession(session.id)}>
            <div class="truncate text-[13px] font-medium text-text-primary">
              {session.name}
              {#if sessionNameSuffix(session, sessions)}
                <span class="font-mono text-[10px] text-text-muted">#{sessionNameSuffix(session, sessions)}</span>
              {/if}
            </div>
            <div class="truncate font-mono text-[10px] text-text-muted">{session.rootDir}</div>
          </ListRow>
        {/each}
        {#if counts.sessions > recentSessions.length}
          <ListRow as="button" class="text-[12px] text-accent hover:text-accent/80" onclick={() => onSelectTab("sessions")}>
            View all ({counts.sessions}) →
          </ListRow>
        {/if}
      {/if}
    </List>
  </section>

  <section>
    <div class="mb-2 flex items-center justify-between gap-2">
      <Button variant="ghost" size="sm" align="start" class="border-transparent bg-transparent px-0 uppercase text-text-muted" onclick={() => onSelectTab("cards")}>
        Recent cards <span class="font-mono">{counts.workItems}</span>
      </Button>
      <Button size="sm" disabled={loading} onclick={() => onSelectTab("cards")}>
        <Plus size={13} />
        <span>Add</span>
      </Button>
    </div>
    <List>
      {#if recentWorkItems.length === 0}
        <EmptyState message="No cards." />
      {:else}
        {#each recentWorkItems as item (item.id)}
          <ListRow as="button" cols="grid-cols-[56px_minmax(0,1fr)]" onclick={() => onOpenWorkItem(item.id)}>
            <span class="font-mono text-[11px] text-text-muted">#{item.number}</span>
            <span class="min-w-0">
              <span class="block truncate text-[13px] font-medium text-text-primary">{item.title}</span>
              <span class="block truncate text-[11px] text-text-muted">{item.stageId}</span>
            </span>
          </ListRow>
        {/each}
        {#if counts.workItems > recentWorkItems.length}
          <ListRow as="button" class="text-[12px] text-accent hover:text-accent/80" onclick={() => onSelectTab("cards")}>
            View all ({counts.workItems}) →
          </ListRow>
        {/if}
      {/if}
    </List>
  </section>

  <section>
    <div class="mb-2 flex items-center justify-between gap-2">
      <Button variant="ghost" size="sm" align="start" class="border-transparent bg-transparent px-0 uppercase text-text-muted" onclick={() => onSelectTab("runs")}>
        Latest runs <span class="font-mono">{counts.runs}</span>
      </Button>
    </div>
    <List>
      {#if recentRuns.length === 0}
        <EmptyState message="No runs." />
      {:else}
        {#each recentRuns as run (run.id)}
          <ListRow as="button" disabled={!run.sessionId && !run.ptyId} onclick={() => onOpenRunTerminal(run)}>
            <div class="flex min-w-0 items-center gap-2">
              <div class="min-w-0 flex-1 truncate text-[13px] font-medium text-text-primary">{runLabel(run)}</div>
              <StatusDot status={run.status} showLabel class="shrink-0 text-[11px]" />
            </div>
            <div class="truncate text-[11px] text-text-muted">{workItemTitle(run.workItemId)}</div>
          </ListRow>
        {/each}
        {#if counts.runs > recentRuns.length}
          <ListRow as="button" class="text-[12px] text-accent hover:text-accent/80" onclick={() => onSelectTab("runs")}>
            View all ({counts.runs}) →
          </ListRow>
        {/if}
      {/if}
    </List>
  </section>
</div>
