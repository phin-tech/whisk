<script lang="ts">
  type Variant = "boxed" | "seamless";
  type Props = {
    value?: string;
    variant?: Variant;
    disabled?: boolean;
    placeholder?: string;
    rows?: number;
    class?: string;
    [key: string]: unknown;
  };

  let {
    value = $bindable(""),
    variant = "boxed",
    disabled = false,
    placeholder = "",
    rows,
    class: className = "",
    ...restProps
  }: Props = $props();

  const base =
    "min-h-16 w-full resize-y rounded border px-2 py-2 text-[12px] text-text-primary outline-none transition-colors placeholder:text-text-muted focus:border-accent-dim disabled:cursor-not-allowed disabled:opacity-60";
  const variantClass = $derived(
    variant === "seamless"
      ? "border-transparent bg-transparent hover:border-border-subtle focus:bg-bg-deep"
      : "border-border bg-bg-deep",
  );
  const classes = $derived(`${base} ${variantClass} ${className}`.trim());
</script>

<textarea bind:value {disabled} {placeholder} {rows} class={classes} {...restProps}></textarea>
