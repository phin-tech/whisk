<script lang="ts">
  import { Dialog } from "bits-ui";
  import type { Snippet } from "svelte";

  type LayerBehavior = "close" | "defer-otherwise-close" | "defer-otherwise-ignore" | "ignore";
  type Placement = "center" | "top";
  type Props = {
    open?: boolean;
    titleId?: string;
    titleClass?: string;
    placement?: Placement;
    onOpenChange?: (open: boolean) => void;
    onkeydown?: (event: KeyboardEvent) => void;
    onEscapeKeydown?: (event: KeyboardEvent) => void;
    interactOutsideBehavior?: LayerBehavior;
    escapeKeydownBehavior?: LayerBehavior;
    class?: string;
    heading?: Snippet;
    children?: Snippet;
  };

  let {
    open = false,
    titleId = "dialog-title",
    titleClass = "text-[13px] font-bold leading-4",
    placement = "center",
    onOpenChange = () => {},
    onkeydown,
    onEscapeKeydown,
    interactOutsideBehavior = "ignore",
    escapeKeydownBehavior = "close",
    class: className = "",
    heading,
    children,
  }: Props = $props();

  let content = $state<HTMLDivElement | null>(null);
  const placementClass = $derived(
    placement === "top"
      ? "left-1/2 top-[14vh] -translate-x-1/2"
      : "left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2",
  );
  const base = $derived(
    `fixed ${placementClass} z-50 w-full rounded-md border border-hairline bg-bg-deep text-text-primary outline-none`,
  );

  const classes = $derived(`${base} ${className}`.trim());

  function focusContent(event: Event) {
    event.preventDefault();
    requestAnimationFrame(() => content?.focus());
  }

  $effect(() => {
    if (!open) return;
    requestAnimationFrame(() => content?.focus());
  });
</script>

<Dialog.Root {open} {onOpenChange}>
  {#if open}
    <Dialog.Portal>
      <Dialog.Overlay class="fixed inset-0 z-50 bg-black/70" />
      <Dialog.Content
        bind:ref={content}
        class={classes}
        tabindex={-1}
        aria-labelledby={titleId}
        trapFocus={true}
        {interactOutsideBehavior}
        {escapeKeydownBehavior}
        onOpenAutoFocus={focusContent}
        {onEscapeKeydown}
        {onkeydown}
      >
        <Dialog.Title id={titleId} level={2} class={titleClass}>
          {@render heading?.()}
        </Dialog.Title>
        {@render children?.()}
      </Dialog.Content>
    </Dialog.Portal>
  {/if}
</Dialog.Root>
