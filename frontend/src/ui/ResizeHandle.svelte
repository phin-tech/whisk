<script lang="ts">
  type Props = {
    label?: string;
    dragging?: boolean;
    onmousedown?: (event: MouseEvent) => void;
    class?: string;
    [key: string]: unknown;
  };

  let {
    label = "Resize sidebar",
    dragging = false,
    onmousedown,
    class: className = "",
    ...restProps
  }: Props = $props();

  const classes = $derived(
    `group relative flex w-1 shrink-0 cursor-col-resize self-stretch flex-col items-center border-0 bg-transparent p-0 ${className}`.trim(),
  );
  const railClasses = $derived(
    `min-h-0 max-w-[0.5px] min-w-[0.5px] flex-1 transition-all duration-150 ${
      dragging ? "bg-border" : "bg-border-subtle group-hover:bg-border-hairline"
    }`,
  );
</script>

<button
  type="button"
  aria-label={label}
  class={classes}
  {onmousedown}
  {...restProps}
>
  <div class={railClasses}></div>
</button>
