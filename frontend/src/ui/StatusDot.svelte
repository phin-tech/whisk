<script lang="ts">
  type Props = {
    status: string;
    label?: string;
    showLabel?: boolean;
    class?: string;
    [key: string]: unknown;
  };

  let {
    status,
    label,
    showLabel = false,
    class: className = "",
    ...restProps
  }: Props = $props();

  const statusClass = $derived(
    status === "running" || status === "awaiting_input"
      ? "text-green"
      : status === "queued"
        ? "text-blue"
        : status === "failed" || status === "cancelled"
          ? "text-red"
          : "text-text-muted",
  );
  const classes = $derived(`inline-flex min-w-0 items-center gap-1.5 text-[12px] ${className}`.trim());
  const dotClasses = $derived(`shrink-0 ${statusClass}`.trim());
  const displayLabel = $derived(label ?? status);
</script>

<span class={classes} {...restProps}>
  <span aria-hidden="true" class={dotClasses}>●</span>
  {#if showLabel}
    <span class="min-w-0 truncate text-text-muted">{displayLabel}</span>
  {/if}
</span>
