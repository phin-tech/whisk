<script lang="ts">
  import Button from "./ui/Button.svelte";
  import Checkbox from "./ui/Checkbox.svelte";
  import ModalShell from "./ui/ModalShell.svelte";

  type Props = {
    visible?: boolean;
    title?: string;
    message?: string;
    confirmLabel?: string;
    cancelLabel?: string;
    checkboxLabel?: string;
    onconfirm: (checked: boolean) => void;
    oncancel: () => void;
  };

  let {
    visible = false,
    title = "",
    message = "",
    confirmLabel = "Confirm",
    cancelLabel = "Cancel",
    checkboxLabel = "",
    onconfirm,
    oncancel,
  }: Props = $props();

  let checked = $state(false);

  function handleEscape(event: KeyboardEvent) {
    event.preventDefault();
    oncancel();
  }

  function handleKey(event: KeyboardEvent) {
    if (event.key !== "Enter") return;
    event.preventDefault();
    onconfirm(checked);
  }

  function handleOpenChange(open: boolean) {
    if (!open && visible) oncancel();
  }

  $effect(() => {
    if (!visible) checked = false;
  });
</script>

<ModalShell
  open={visible}
  titleId="confirm-dialog-title"
  class="max-w-[300px] bg-bg-surface px-4 py-4 shadow-[0_24px_80px_rgba(0,0,0,0.55)]"
  interactOutsideBehavior="ignore"
  onOpenChange={handleOpenChange}
  onEscapeKeydown={handleEscape}
  onkeydown={handleKey}
>
  {#snippet heading()}
    {title}
  {/snippet}
  <p class="mt-2.5 whitespace-pre-line text-[12px] font-medium leading-4 text-text-secondary">{message}</p>
  {#if checkboxLabel}
    <Checkbox class="mt-3" bind:checked>{checkboxLabel}</Checkbox>
  {/if}
  <div class="mt-4 grid grid-cols-2 gap-2.5">
    <Button type="button" variant="outline" size="lg" class="w-full" onclick={oncancel}>
      {cancelLabel}
    </Button>
    <Button type="button" variant="primary" size="lg" class="w-full" onclick={() => onconfirm(checked)}>
      {confirmLabel}
    </Button>
  </div>
</ModalShell>
