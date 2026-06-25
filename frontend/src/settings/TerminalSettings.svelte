<script lang="ts">
  import Switch from "../ui/Switch.svelte";
  import TextField from "../ui/TextField.svelte";

  type Props = {
    terminalFontSize: number;
    terminalCursorBlink: boolean;
    onTerminalFontSize: (size: number) => void;
    onTerminalCursorBlink: (blink: boolean) => void;
  };

  let {
    terminalFontSize,
    terminalCursorBlink,
    onTerminalFontSize,
    onTerminalCursorBlink,
  }: Props = $props();

  function updateTerminalFontSize(event: Event) {
    onTerminalFontSize(Number((event.currentTarget as HTMLInputElement).value));
  }
</script>

<div class="flex items-center justify-between py-2">
  <span class="text-[13px]">Font size</span>
  <TextField
    class="w-20 text-right"
    type="number"
    min="10"
    max="20"
    value={String(terminalFontSize)}
    oninput={updateTerminalFontSize}
  />
</div>

<div class="flex items-center justify-between py-2">
  <div>
    <div class="text-[13px]">Cursor blink</div>
    <div class="mt-0.5 text-[11px] text-text-muted">Applies to newly mounted terminal panes.</div>
  </div>
  <Switch
    label="Toggle cursor blink"
    checked={terminalCursorBlink}
    onCheckedChange={onTerminalCursorBlink}
  />
</div>
