<script lang="ts">
  import type { WorkItemRun } from "../../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import Button from "../ui/Button.svelte";
  import EmptyState from "../ui/EmptyState.svelte";
  import List from "../ui/List.svelte";
  import ListRow from "../ui/ListRow.svelte";
  import SectionHeader from "../ui/SectionHeader.svelte";
  import StatusDot from "../ui/StatusDot.svelte";

  export let sortedRuns: WorkItemRun[] = [];
  export let runLabel: (run: WorkItemRun) => string;
  export let workItemTitle: (workItemId: string) => string;
  export let formattedTime: (value: unknown) => string;
  export let onOpenRunTerminal: (run: WorkItemRun) => void;
</script>

<section>
  <SectionHeader title="Runs" class="mb-2" />
  <List>
    {#if sortedRuns.length === 0}
      <EmptyState message="No runs." />
    {:else}
      {#each sortedRuns as run (run.id)}
        <ListRow cols="grid-cols-[minmax(0,1fr)_auto_88px]">
          <div class="min-w-0">
            <div class="truncate text-[13px] font-medium text-text-primary">
              {runLabel(run)}
            </div>
            <div class="truncate font-mono text-[10px] text-text-muted">
              {run.id}
            </div>
            <div class="truncate text-[11px] text-text-muted">
              {workItemTitle(run.workItemId)}
            </div>
            {#if formattedTime(run.updatedAt || run.createdAt)}
              <div class="truncate font-mono text-[10px] text-text-muted">
                {formattedTime(run.updatedAt || run.createdAt)}
              </div>
            {/if}
          </div>
          <StatusDot status={run.status} showLabel class="shrink-0 text-[11px]" />
          <Button size="sm" disabled={!run.sessionId && !run.ptyId} onclick={() => onOpenRunTerminal(run)}>
            Open
          </Button>
        </ListRow>
      {/each}
    {/if}
  </List>
</section>
