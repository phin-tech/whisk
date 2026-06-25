<script lang="ts">
  import type { Snippet } from "svelte";

  type Props = {
    class?: string;
    children?: Snippet;
  };

  let { class: className = "", children }: Props = $props();

  const classes = $derived(`min-w-44 py-0.5 ${className}`.trim());
</script>

<div role="menu" class={classes}>
  {@render children?.()}
</div>
