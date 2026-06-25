<script lang="ts">
  import { Switch as BitsSwitch } from "bits-ui";

  type Props = {
    checked?: boolean;
    label: string;
    disabled?: boolean;
    onCheckedChange?: (checked: boolean) => void;
    class?: string;
    [key: string]: unknown;
  };

  let {
    checked = $bindable(false),
    label,
    disabled = false,
    onCheckedChange = () => {},
    class: className = "",
    ...restProps
  }: Props = $props();

  const rootClass = $derived(
    `relative inline-flex h-5 w-9 shrink-0 items-center rounded-full border transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50 disabled:cursor-not-allowed disabled:opacity-50 ${
      checked ? "border-accent bg-accent-dim" : "border-border bg-bg-deep"
    } ${className}`.trim(),
  );
  const thumbClass = $derived(
    `block h-3.5 w-3.5 rounded-full transition-transform ${
      checked ? "translate-x-[18px] bg-accent" : "translate-x-0.5 bg-text-secondary"
    }`,
  );

  function handleCheckedChange(next: boolean) {
    checked = next;
    onCheckedChange(next);
  }
</script>

<BitsSwitch.Root
  bind:checked
  {disabled}
  aria-label={label}
  class={rootClass}
  onCheckedChange={handleCheckedChange}
  {...restProps}
>
  <BitsSwitch.Thumb class={thumbClass} />
</BitsSwitch.Root>
