<script lang="ts">
  type Indicator = {
    id: string;
    label: string;
    tone: "info" | "success" | "warning" | "danger";
  };

  type Props = {
    indicators?: Indicator[];
    class?: string;
  };

  let { indicators = [], class: className = "" }: Props = $props();

  function dotClass(tone: Indicator["tone"]) {
    if (tone === "danger") return "text-red";
    if (tone === "warning") return "text-amber";
    if (tone === "success") return "text-green";
    return "text-blue";
  }

  const classes = $derived(`flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 ${className}`.trim());
</script>

{#if indicators.length > 0}
  <div class={classes}>
    {#each indicators as indicator (indicator.id)}
      <span class="inline-flex min-w-0 items-center gap-1 text-[11px]" title={indicator.label}>
        <span aria-hidden="true" class={`shrink-0 ${dotClass(indicator.tone)}`}>●</span>
        <span class="truncate text-text-muted">{indicator.label}</span>
      </span>
    {/each}
  </div>
{/if}
