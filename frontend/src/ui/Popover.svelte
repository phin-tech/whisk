<script lang="ts">
  import { Popover as BitsPopover } from "bits-ui";
  import type { Snippet } from "svelte";

  type Align = "start" | "center" | "end";
  type Side = "top" | "right" | "bottom" | "left";
  type TriggerProps = { props: Record<string, unknown> };
  type Props = {
    open?: boolean;
    align?: Align;
    side?: Side;
    sideOffset?: number;
    onOpenChange?: (open: boolean) => void;
    onEscapeKeydown?: (event: KeyboardEvent) => void;
    class?: string;
    trigger?: Snippet<[TriggerProps]>;
    children?: Snippet;
    [key: string]: unknown;
  };

  let {
    open = $bindable(false),
    align = "end",
    side = "bottom",
    sideOffset = 4,
    onOpenChange = () => {},
    onEscapeKeydown = () => {},
    class: className = "",
    trigger,
    children,
    ...restProps
  }: Props = $props();

  const base =
    "z-[60] rounded border border-border-subtle bg-bg-surface py-1 shadow-lg outline-none";
  const classes = $derived(`${base} ${className}`.trim());

  function handleOpenChange(next: boolean) {
    open = next;
    onOpenChange(next);
  }

  function handleEscapeKeydown(event: KeyboardEvent) {
    event.stopPropagation();
    onEscapeKeydown(event);
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key !== "Escape") return;
    event.stopPropagation();
    open = false;
    onOpenChange(false);
  }

  function withTriggerKeydown(props: Record<string, unknown>) {
    return {
      ...props,
      onkeydown: (event: KeyboardEvent) => {
        if (event.key === "Escape") {
          event.stopPropagation();
          open = false;
          onOpenChange(false);
          return;
        }
        const handler = props.onkeydown;
        if (typeof handler === "function") {
          (handler as (event: KeyboardEvent) => void)(event);
        }
      },
    };
  }
</script>

<BitsPopover.Root bind:open onOpenChange={handleOpenChange}>
  <BitsPopover.Trigger>
    {#snippet child({ props })}
      {@render trigger?.({ props: withTriggerKeydown(props) })}
    {/snippet}
  </BitsPopover.Trigger>
  <BitsPopover.Portal>
    <BitsPopover.Content
      {align}
      {side}
      {sideOffset}
      trapFocus={false}
      class={classes}
      onEscapeKeydown={handleEscapeKeydown}
      onkeydown={handleKeydown}
      {...restProps}
    >
      {@render children?.()}
    </BitsPopover.Content>
  </BitsPopover.Portal>
</BitsPopover.Root>
