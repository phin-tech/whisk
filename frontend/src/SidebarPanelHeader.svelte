<script lang="ts">
  import PanelRightClose from "@lucide/svelte/icons/panel-right-close";
  import IconButton from "./ui/IconButton.svelte";

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
    <IconButton
      label={`Collapse ${title} sidebar`}
      size="sm"
      onclick={onclose}
    >
      <PanelRightClose size={13} />
    </IconButton>
  </div>
</header>
