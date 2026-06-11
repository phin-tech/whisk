<script lang="ts">
  import PanelRightClose from "@lucide/svelte/icons/panel-right-close";

  export let title: string;
  export let onclose: () => void;
</script>

<header
  class="flex h-10 shrink-0 items-center justify-between border-b border-hairline px-3"
>
  <div class="min-w-0 text-[11px] font-semibold uppercase tracking-widest text-text-muted">
    {title}
  </div>
  <div class="flex items-center gap-1">
    <slot name="actions" />
    <button
      type="button"
      aria-label={`Collapse ${title} sidebar`}
      class="inline-flex h-6 w-6 items-center justify-center rounded border border-transparent text-text-muted transition-colors hover:border-border-subtle hover:bg-bg-hover hover:text-text-primary focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50"
      on:click={onclose}
    >
      <PanelRightClose size={13} />
    </button>
  </div>
</header>
