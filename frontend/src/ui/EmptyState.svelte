<script lang="ts">
  import type { Snippet } from "svelte";

  type Props = {
    title?: string;
    message?: string;
    class?: string;
    icon?: Snippet;
    children?: Snippet;
    [key: string]: unknown;
  };

  let {
    title = "",
    message = "",
    class: className = "",
    icon,
    children,
    ...restProps
  }: Props = $props();

  const classes = $derived(`px-3 py-3 text-[12px] text-text-muted ${className}`.trim());
  const messageClasses = $derived(`${title ? "mt-1 " : ""}max-w-sm leading-5`.trim());
  const childrenClasses = $derived(
    `${title || message ? "mt-3 " : ""}flex items-center gap-2`.trim(),
  );
</script>

<div class={classes} {...restProps}>
  {#if icon}
    <div class="mb-2 text-text-muted">
      {@render icon?.()}
    </div>
  {/if}
  {#if title}
    <p class="text-[13px] font-medium text-text-primary">{title}</p>
  {/if}
  {#if message}
    <p class={messageClasses}>{message}</p>
  {/if}
  {#if children}
    <div class={childrenClasses}>
      {@render children?.()}
    </div>
  {/if}
</div>
