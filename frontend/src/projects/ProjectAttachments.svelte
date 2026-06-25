<script lang="ts">
  import ExternalLink from "@lucide/svelte/icons/external-link";
  import Pencil from "@lucide/svelte/icons/pencil";
  import Plus from "@lucide/svelte/icons/plus";
  import Save from "@lucide/svelte/icons/save";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import type { ProjectAttachmentTemplate } from "../../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import Button from "../ui/Button.svelte";
  import Checkbox from "../ui/Checkbox.svelte";
  import EmptyState from "../ui/EmptyState.svelte";
  import IconButton from "../ui/IconButton.svelte";
  import List from "../ui/List.svelte";
  import ListRow from "../ui/ListRow.svelte";
  import SectionHeader from "../ui/SectionHeader.svelte";
  import TextField from "../ui/TextField.svelte";

  type Attachment = {
    id: string;
    kind: string;
    title?: string;
    path?: string;
    url?: string;
    note?: string;
    provider?: string;
    target?: string;
    includeInContext?: boolean;
  };

  export let attachments: Attachment[] = [];
  export let pluginAttachmentTemplates: (ProjectAttachmentTemplate & { pluginId: string })[] = [];
  export let selectedPluginTemplate: (ProjectAttachmentTemplate & { pluginId: string }) | undefined;
  export let loading = false;
  export let attachmentFormOpen = false;
  export let attachmentKind = "file";
  export let attachmentTitle = "";
  export let attachmentTarget = "";
  export let attachmentNote = "";
  export let attachmentInContext = true;
  export let attachmentEditId = "";
  export let openAttachmentForm: (mode: string) => void;
  export let closeAttachmentForm: () => void;
  export let createAttachment: () => void;
  export let openAttachmentEditor: (attachment: Attachment) => void;
  export let deleteAttachment: (attachmentId: string) => void;
  export let openAttachmentURL: (attachment: Attachment) => void;
  export let externalAttachmentURL: (attachment: Attachment) => string;
  export let attachmentSummary: (attachment: Attachment) => string;
  export let attachmentTargetLabel: (kind: string) => string;
  export let pluginFieldValue: (id: string) => string;
  export let setPluginFieldValue: (id: string, value: string) => void;

  function fieldInput(fieldId: string, event: Event) {
    setPluginFieldValue(fieldId, (event.currentTarget as HTMLInputElement).value);
  }
</script>

<section>
  <SectionHeader title="Attachments" class="mb-2" />
  <div class="mb-2 border border-border-subtle bg-bg-surface/25 p-2">
    <div class="flex flex-wrap gap-1.5">
      {#each ["file", "url", "note"] as mode}
        <Button disabled={loading} onclick={() => openAttachmentForm(mode)}>
          <Plus size={14} />
          <span>{mode}</span>
        </Button>
      {/each}
      {#each pluginAttachmentTemplates as template (`${template.pluginId}:${template.id}`)}
        <Button disabled={loading} onclick={() => openAttachmentForm(`${template.pluginId}:${template.id}`)}>
          <Plus size={14} />
          <span>{template.label || template.id}</span>
        </Button>
      {/each}
    </div>

    {#if attachmentFormOpen}
      <div class="mt-2 grid gap-2 border-t border-hairline pt-2">
        {#if selectedPluginTemplate && !attachmentEditId}
          <div class="text-[12px] font-medium text-text-primary">
            {selectedPluginTemplate.label || selectedPluginTemplate.id}
          </div>
          <div class="grid gap-2 md:grid-cols-2">
            {#each selectedPluginTemplate.fields ?? [] as field (field.id)}
              <TextField
                value={pluginFieldValue(field.id)}
                placeholder={field.placeholder || field.label || field.id}
                disabled={loading}
                oninput={(event: Event) => fieldInput(field.id, event)}
              />
            {/each}
          </div>
        {:else}
          <div class="grid gap-2 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
            <TextField bind:value={attachmentTitle} placeholder="Title" disabled={loading} />
            {#if attachmentKind === "note"}
              <TextField bind:value={attachmentNote} placeholder="Note" disabled={loading} />
            {:else}
              <TextField bind:value={attachmentTarget} placeholder={attachmentTargetLabel(attachmentKind)} disabled={loading} />
            {/if}
          </div>
        {/if}

        <div class="flex flex-wrap items-center justify-between gap-2">
          <Checkbox bind:checked={attachmentInContext} disabled={loading || Boolean(selectedPluginTemplate)} class="text-[12px]">
            Use as agent context
          </Checkbox>
          <div class="flex gap-1.5">
            <Button onclick={closeAttachmentForm}>Cancel</Button>
            <Button variant="primary" disabled={loading} onclick={createAttachment}>
              {#if attachmentEditId}
                <Save size={14} />
                <span>Save</span>
              {:else}
                <Plus size={14} />
                <span>Add</span>
              {/if}
            </Button>
          </div>
        </div>
      </div>
    {/if}
  </div>

  <List>
    {#if attachments.length === 0}
      <EmptyState message="No attachments." />
    {:else}
      {#each attachments as attachment (attachment.id)}
        <ListRow cols="grid-cols-[96px_minmax(0,1fr)_72px_32px_32px_32px]">
          <div class="font-mono text-[11px] text-text-muted">{attachment.kind}</div>
          <div class="min-w-0">
            <div class="truncate text-[13px] font-medium text-text-primary">
              {attachment.title || attachmentSummary(attachment)}
            </div>
            <div class="truncate font-mono text-[10px] text-text-muted">
              {attachmentSummary(attachment)}
            </div>
          </div>
          <div class="text-[11px] text-text-muted">
            {attachment.includeInContext ? "context" : ""}
          </div>
          {#if externalAttachmentURL(attachment)}
            <IconButton label="Open attachment URL" onclick={() => openAttachmentURL(attachment)}>
              <ExternalLink size={13} />
            </IconButton>
          {:else}
            <span></span>
          {/if}
          <IconButton label="Edit attachment" disabled={loading} onclick={() => openAttachmentEditor(attachment)}>
            <Pencil size={13} />
          </IconButton>
          <IconButton label="Delete attachment" tone="danger" disabled={loading} onclick={() => deleteAttachment(attachment.id)}>
            <Trash2 size={13} />
          </IconButton>
        </ListRow>
      {/each}
    {/if}
  </List>
</section>
