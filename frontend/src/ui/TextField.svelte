<script lang="ts">
  type Variant = "boxed" | "seamless";
  type Props = {
    value?: string;
    ref?: HTMLInputElement | null;
    variant?: Variant;
    type?: string;
    disabled?: boolean;
    placeholder?: string;
    class?: string;
    [key: string]: unknown;
  };

  let {
    value = $bindable(""),
    ref = $bindable(null),
    variant = "boxed",
    type = "text",
    disabled = false,
    placeholder = "",
    class: className = "",
    ...restProps
  }: Props = $props();

  const base =
    "w-full rounded border px-2 text-[12px] text-text-primary outline-none transition-colors placeholder:text-text-muted focus:border-accent-dim disabled:cursor-not-allowed disabled:opacity-60";
  const variantClass = $derived(
    variant === "seamless"
      ? "border-transparent bg-transparent py-1.5 hover:border-border-subtle focus:bg-bg-deep"
      : "h-8 border-border bg-bg-deep",
  );
  const classes = $derived(`${base} ${variantClass} ${className}`.trim());
</script>

<input bind:this={ref} bind:value {type} {disabled} {placeholder} class={classes} {...restProps} />
