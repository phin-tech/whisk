<script lang="ts">
  import ArrowLeft from "@lucide/svelte/icons/arrow-left";
  import type { Snippet } from "svelte";
  import type { WorkflowStage } from "../../bindings/github.com/phin-tech/whisk/internal/domain/workitem/models";
  import Button from "../ui/Button.svelte";
  import EmptyState from "../ui/EmptyState.svelte";
  import IconButton from "../ui/IconButton.svelte";
  import List from "../ui/List.svelte";
  import ListRow from "../ui/ListRow.svelte";

  export let stage: WorkflowStage;
  export let count = 0;
  export let collapsed = false;
  export let hasAttention = false;
  export let attentionClass = "";
  export let onToggle: (stageId: string) => void;
  export let children: Snippet | undefined;
</script>

{#if collapsed}
  <Button
    variant="outline"
    class="flex min-h-[420px] w-12 shrink-0 flex-col items-center justify-between bg-bg-base px-2 py-3 text-text-secondary hover:border-border hover:bg-bg-surface hover:text-text-primary"
    aria-label={`Expand ${stage.name}`}
    title={`Expand ${stage.name}`}
    onclick={() => onToggle(stage.id)}
  >
    <span class="writing-vertical truncate text-[13px] font-semibold">{stage.name}</span>
    <span class="flex flex-col items-center gap-2">
      {#if hasAttention}
        <span class="h-2 w-2 rounded-full {attentionClass}"></span>
      {/if}
      <span class="rounded border border-border-subtle bg-bg-deep/80 px-1.5 py-1 font-mono text-[12px]">
        {count}
      </span>
    </span>
  </Button>
{:else}
  <section class="flex h-full min-h-[420px] w-[320px] shrink-0 flex-col overflow-hidden rounded-md border border-border-subtle bg-bg-base">
    <div class="flex h-12 min-w-0 shrink-0 items-center justify-between gap-2 border-b border-hairline px-3">
      <div class="flex min-w-0 items-center gap-2">
        {#if hasAttention}
          <span class="h-2 w-2 shrink-0 rounded-full {attentionClass}"></span>
        {/if}
        <Button
          variant="ghost"
          size="sm"
          align="start"
          class="min-w-0 truncate border-transparent bg-transparent px-0 text-[13px] font-semibold text-text-primary hover:bg-transparent focus-visible:text-accent"
          aria-label={`Collapse ${stage.name}`}
          title="Collapse"
          onclick={() => onToggle(stage.id)}
        >
          {stage.name}
        </Button>
        <div class="rounded border border-hairline bg-bg-deep/70 px-1.5 py-0.5 font-mono text-[12px] text-text-secondary">
          {count}
        </div>
      </div>
      <IconButton label={`Collapse ${stage.name}`} onclick={() => onToggle(stage.id)}>
        <ArrowLeft size={13} />
      </IconButton>
    </div>

    {#if count === 0}
      <List class="min-h-0 min-w-0 flex-1">
        <ListRow class="flex min-h-24 items-center justify-center py-7 text-center">
          <EmptyState message="Empty" class="py-0 text-[13px]" />
        </ListRow>
      </List>
    {:else}
      {@render children?.()}
    {/if}
  </section>
{/if}
