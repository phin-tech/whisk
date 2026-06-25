<script lang="ts">
  import Check from "@lucide/svelte/icons/check";
  import ChevronDown from "@lucide/svelte/icons/chevron-down";
  import { Select } from "bits-ui";

  type Option = {
    value: string;
    label: string;
    disabled?: boolean;
  };
  type Props = {
    value?: string;
    label: string;
    options: Option[];
    placeholder?: string;
    disabled?: boolean;
    onValueChange?: (value: string) => void;
    class?: string;
  };

  let {
    value = $bindable(""),
    label,
    options,
    placeholder = "Select",
    disabled = false,
    onValueChange = () => {},
    class: className = "",
  }: Props = $props();

  const triggerClass = $derived(
    `inline-flex h-8 min-w-0 items-center justify-between gap-2 rounded border border-border bg-bg-deep px-2 text-[12px] text-text-primary outline-none transition-colors hover:border-border-subtle focus-visible:border-accent-dim focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50 disabled:cursor-not-allowed disabled:opacity-60 ${className}`.trim(),
  );
  const contentClass =
    "z-[60] min-w-[var(--bits-select-anchor-width)] rounded border border-border-subtle bg-bg-surface py-1 shadow-lg";
  const itemClass =
    "flex cursor-default select-none items-center justify-between gap-2 px-2 py-1.5 text-[12px] text-text-secondary outline-none transition-colors data-[highlighted]:bg-bg-hover data-[highlighted]:text-text-primary data-[selected]:text-accent data-[disabled]:pointer-events-none data-[disabled]:opacity-50";

  function handleValueChange(next: string) {
    value = next;
    onValueChange(next);
  }
</script>

<Select.Root type="single" bind:value items={options} {disabled} onValueChange={handleValueChange}>
  <Select.Trigger aria-label={label} class={triggerClass}>
    <Select.Value {placeholder} class="min-w-0 truncate" />
    <ChevronDown size={13} class="shrink-0 text-text-muted" />
  </Select.Trigger>
  <Select.Portal>
    <Select.Content sideOffset={4} class={contentClass}>
      <Select.Viewport>
        {#each options as option}
          <Select.Item
            value={option.value}
            label={option.label}
            disabled={option.disabled}
            class={itemClass}
          >
            {#snippet children({ selected })}
              <span class="truncate">{option.label}</span>
              {#if selected}
                <Check size={12} class="shrink-0" />
              {/if}
            {/snippet}
          </Select.Item>
        {/each}
      </Select.Viewport>
    </Select.Content>
  </Select.Portal>
</Select.Root>
