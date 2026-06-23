import type { Command } from "./commands";

export const CommandSplitPaneVertical = "split-pane-vertical";
export const CommandSplitPaneHorizontal = "split-pane-horizontal";
export const CommandClosePane = "close-pane";
export const CommandCloseSession = "close-session";

export function sessionSplitCommands(options: {
  canSplit: boolean;
  canClose: boolean;
  canCloseSession: boolean;
  split: (direction: "horizontal" | "vertical") => void | Promise<void>;
  close: () => void | Promise<void>;
  closeSession: () => void | Promise<void>;
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
    {
      id: CommandClosePane,
      title: "Close Pane",
      shortcut: "Cmd/Ctrl W",
      enabled: () => options.canClose,
      run: options.close,
    },
    {
      id: CommandCloseSession,
      title: "Close Session",
      shortcut: "Cmd/Ctrl Shift W",
      enabled: () => options.canCloseSession,
      run: options.closeSession,
    },
  ];
}
