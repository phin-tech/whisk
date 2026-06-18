import type { Command } from "./commands";

export const CommandSplitPaneVertical = "split-pane-vertical";
export const CommandSplitPaneHorizontal = "split-pane-horizontal";

export function commandIdForShortcut(event: KeyboardEvent): string | null {
  if (!(event.metaKey || event.ctrlKey) || event.altKey) return null;
  if (event.key.toLowerCase() !== "d") return null;
  return event.shiftKey ? CommandSplitPaneHorizontal : CommandSplitPaneVertical;
}

export function sessionSplitCommands(options: {
  canSplit: boolean;
  split: (direction: "horizontal" | "vertical") => void | Promise<void>;
}): Command[] {
  return [
    {
      id: CommandSplitPaneVertical,
      title: "Split Pane Vertically",
      shortcut: "Cmd/Ctrl D",
      enabled: () => options.canSplit,
      run: () => options.split("vertical"),
    },
    {
      id: CommandSplitPaneHorizontal,
      title: "Split Pane Horizontally",
      shortcut: "Cmd/Ctrl Shift D",
      enabled: () => options.canSplit,
      run: () => options.split("horizontal"),
    },
  ];
}
