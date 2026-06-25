<script lang="ts">
  import { Checkbox as BitsCheckbox } from "bits-ui";
  import type { Snippet } from "svelte";

  type Props = {
    checked?: boolean;
    disabled?: boolean;
    onCheckedChange?: (checked: boolean) => void;
    class?: string;
    children?: Snippet;
    [key: string]: unknown;
  };

  let {
    checked = $bindable(false),
    disabled = false,
    onCheckedChange = () => {},
    class: className = "",
    children,
    ...restProps
  }: Props = $props();

  const boxClass =
    "inline-flex h-3.5 w-3.5 shrink-0 items-center justify-center rounded border border-border-subtle bg-bg-surface text-accent transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50 disabled:opacity-50";
  const labelBase = "inline-flex items-center gap-2 text-[10px] text-text-secondary";

  const labelClass = $derived(`${labelBase} ${className}`.trim());

  function handleCheckedChange(next: boolean | "indeterminate") {
    checked = next === true;
    onCheckedChange(checked);
  }
</script>

<label class={labelClass}>
  <BitsCheckbox.Root
    bind:checked
    {disabled}
    class={boxClass}
    onCheckedChange={handleCheckedChange}
    {...restProps}
  >
    {#if checked}
      <span class="h-1.5 w-1.5 rounded-sm bg-accent"></span>
    {/if}
  </BitsCheckbox.Root>
  <span>{@render children?.()}</span>
</label>
