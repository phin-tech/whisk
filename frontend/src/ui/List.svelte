<script lang="ts">
  import type { Snippet } from "svelte";

  type Props = {
    class?: string;
    children?: Snippet;
    [key: string]: unknown;
  };

  let { class: className = "", children, ...restProps }: Props = $props();

  const classes = $derived(`divide-y divide-hairline ${className}`.trim());
</script>

<div class={classes} {...restProps}>
  {@render children?.()}
</div>
