<script lang="ts">
  import Plus from "@lucide/svelte/icons/plus";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import type { Session } from "../../bindings/github.com/phin-tech/whisk/internal/domain/session/models";
  import type { WorkItem, WorkItemRun } from "../../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import Button from "../ui/Button.svelte";
  import EmptyState from "../ui/EmptyState.svelte";
  import IconButton from "../ui/IconButton.svelte";
  import List from "../ui/List.svelte";
  import ListRow from "../ui/ListRow.svelte";
  import SectionHeader from "../ui/SectionHeader.svelte";
  import StatusDot from "../ui/StatusDot.svelte";

  export let sessions: Session[] = [];
  export let sortedRuns: WorkItemRun[] = [];
  export let workItems: WorkItem[] = [];
  export let loading = false;
  export let onNewSession: () => void;
  export let onOpenSession: (sessionId: string) => void;
  export let onRemoveSession: (sessionId: string) => void;
  export let sessionNameSuffix: (session: Session, sessions: Session[]) => string;
</script>

<section>
  <SectionHeader title="Sessions" class="mb-2">
    <Button size="sm" disabled={loading} onclick={onNewSession}>
      <Plus size={13} />
      <span>Add</span>
    </Button>
  </SectionHeader>
  <List>
    {#if sessions.length === 0}
      <EmptyState message="No sessions." />
    {:else}
      {#each sessions as session (session.id)}
        {@const sessionRun = sortedRuns.find((run) => run.sessionId === session.id) ?? null}
        {@const sessionItem = sessionRun ? workItems.find((item) => item.id === sessionRun.workItemId) ?? null : null}
        <ListRow cols="grid-cols-[minmax(0,1fr)_auto_48px_32px]">
          <div class="min-w-0">
            <div class="truncate text-[13px] font-medium text-text-primary">
              {session.name}
              {#if sessionNameSuffix(session, sessions)}
                <span class="font-mono text-[10px] text-text-muted">#{sessionNameSuffix(session, sessions)}</span>
              {/if}
            </div>
            {#if sessionItem}
              <div class="mt-0.5 flex min-w-0 items-center gap-1.5 text-[11px] text-text-muted">
                <span class="truncate">{sessionItem.title}</span>
                {#if sessionItem.stageId}
                  <span class="opacity-40">·</span>
                  <span class="shrink-0">{sessionItem.stageId}</span>
                {/if}
              </div>
            {:else}
              <div class="truncate font-mono text-[10px] text-text-muted">{session.rootDir}</div>
            {/if}
          </div>
          {#if sessionRun}
            <StatusDot status={sessionRun.status} showLabel class="shrink-0 text-[11px]" />
          {:else}
            <span></span>
          {/if}
          <Button size="sm" disabled={loading} onclick={() => onOpenSession(session.id)}>Open</Button>
          <IconButton label="Remove session" tone="danger" disabled={loading} onclick={() => onRemoveSession(session.id)}>
            <Trash2 size={13} />
          </IconButton>
        </ListRow>
      {/each}
    {/if}
  </List>
</section>
