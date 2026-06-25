<script lang="ts">
  import Plus from "@lucide/svelte/icons/plus";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import type { WorkItem } from "../../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import Button from "../ui/Button.svelte";
  import EmptyState from "../ui/EmptyState.svelte";
  import IconButton from "../ui/IconButton.svelte";
  import List from "../ui/List.svelte";
  import ListRow from "../ui/ListRow.svelte";
  import SectionHeader from "../ui/SectionHeader.svelte";
  import TextArea from "../ui/TextArea.svelte";
  import TextField from "../ui/TextField.svelte";

  export let workItems: WorkItem[] = [];
  export let newCardTitle = "";
  export let newCardBody = "";
  export let loading = false;
  export let createCard: () => void;
  export let deleteCard: (item: WorkItem) => void;
  export let onOpenWorkItem: (workItemId: string) => void;

  function submitOnEnter(event: KeyboardEvent) {
    if (event.key !== "Enter" || event.shiftKey) return;
    event.preventDefault();
    createCard();
  }

  function stopDelete(event: MouseEvent, item: WorkItem) {
    event.stopPropagation();
    deleteCard(item);
  }
</script>

<section>
  <SectionHeader title="Cards" class="mb-2" />
  <div class="mb-2 grid gap-2 border border-border-subtle bg-bg-surface/25 p-2">
    <TextField
      bind:value={newCardTitle}
      placeholder="Card title"
      disabled={loading}
      onkeydown={submitOnEnter}
    />
    <TextArea bind:value={newCardBody} placeholder="Notes" disabled={loading} class="py-1.5" />
    <div class="flex justify-end">
      <Button variant="primary" disabled={loading || !newCardTitle.trim()} onclick={createCard}>
        <Plus size={14} />
        <span>Add card</span>
      </Button>
    </div>
  </div>
  <List>
    {#if workItems.length === 0}
      <EmptyState message="No cards." />
    {:else}
      {#each workItems as item (item.id)}
        <ListRow
          as="button"
          cols="grid-cols-[72px_minmax(0,1fr)_120px_32px]"
          class="cursor-pointer"
          onclick={() => onOpenWorkItem(item.id)}
        >
          <div class="font-mono text-[11px] text-text-muted">#{item.number}</div>
          <div class="min-w-0">
            <div class="truncate text-[13px] font-medium text-text-primary">
              {item.title}
            </div>
            {#if item.bodyMarkdown}
              <div class="truncate text-[11px] text-text-muted">{item.bodyMarkdown}</div>
            {/if}
          </div>
          <div class="truncate text-right text-[11px] text-text-muted">{item.stageId}</div>
          <IconButton label="Delete card" tone="danger" disabled={loading} onclick={(event) => stopDelete(event, item)}>
            <Trash2 size={13} />
          </IconButton>
        </ListRow>
      {/each}
    {/if}
  </List>
</section>
