<script lang="ts">
  export let visible = false;
  export let title = "";
  export let message = "";
  export let confirmLabel = "Confirm";
  export let cancelLabel = "Cancel";
  export let checkboxLabel = "";
  export let onconfirm: (checked: boolean) => void;
  export let oncancel: () => void;

  let dialog: HTMLDivElement;
  let checked = false;

  function handleKey(event: KeyboardEvent) {
    if (event.key === "Escape") {
      event.preventDefault();
      oncancel();
    }
    if (event.key === "Enter") {
      event.preventDefault();
      onconfirm(checked);
    }
  }

  $: if (!visible) checked = false;
  $: if (visible) requestAnimationFrame(() => dialog?.focus());
</script>

{#if visible}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/45 px-4">
    <div
      bind:this={dialog}
      class="w-full max-w-[300px] rounded-xl border border-white/20 bg-[#1c1c1f] px-4 py-4 text-text-primary shadow-[0_24px_80px_rgba(0,0,0,0.55)]"
      role="dialog"
      aria-modal="true"
      aria-labelledby="confirm-dialog-title"
      tabindex="-1"
      on:keydown={handleKey}
    >
      <h2 id="confirm-dialog-title" class="text-[13px] font-bold leading-4">{title}</h2>
      <p class="mt-2.5 whitespace-pre-line text-[12px] font-medium leading-4 text-text-secondary">{message}</p>
      {#if checkboxLabel}
        <label class="mt-3 flex items-center gap-2 text-[10px] text-text-secondary">
          <input
            type="checkbox"
            class="h-3.5 w-3.5 rounded border-border-subtle bg-bg-surface accent-accent"
            bind:checked
          />
          <span>{checkboxLabel}</span>
        </label>
      {/if}
      <div class="mt-4 grid grid-cols-2 gap-2.5">
        <button
          type="button"
          class="h-9 rounded-full bg-white/10 text-[12px] font-semibold text-text-primary transition-colors hover:bg-white/15 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent"
          on:click={oncancel}
        >
          {cancelLabel}
        </button>
        <button
          type="button"
          class="h-9 rounded-full bg-blue-500 text-[12px] font-semibold text-white transition-colors hover:bg-blue-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent"
          on:click={() => onconfirm(checked)}
        >
          {confirmLabel}
        </button>
      </div>
    </div>
  </div>
{/if}
