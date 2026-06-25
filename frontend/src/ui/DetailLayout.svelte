<script lang="ts">
  import type { Snippet } from "svelte";

  type Props = {
    class?: string;
    mainClass?: string;
    asideClass?: string;
    main?: Snippet;
    aside?: Snippet;
    children?: Snippet;
    [key: string]: unknown;
  };

  let {
    class: className = "",
    mainClass = "",
    asideClass = "",
    main,
    aside,
    children,
    ...restProps
  }: Props = $props();

  const classes = $derived(
    `grid gap-6 xl:grid-cols-[minmax(0,1fr)_280px] xl:items-start ${className}`.trim(),
  );
  const mainClasses = $derived(`grid min-w-0 gap-6 ${mainClass}`.trim());
  const asideClasses = $derived(`min-w-0 ${asideClass}`.trim());
</script>

<div class={classes} {...restProps}>
  {#if main}
    <main class={mainClasses}>
      {@render main?.()}
    </main>
  {:else}
    {@render children?.()}
  {/if}
  {#if aside}
    <aside class={asideClasses}>
      {@render aside?.()}
    </aside>
  {/if}
</div>
