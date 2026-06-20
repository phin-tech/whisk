import { describe, expect, it } from "vitest";
import {
  commandIdForShortcut,
  sessionSplitCommands,
} from "./sessionCommands";

function keyEvent(init: Partial<KeyboardEvent> & { key: string }): KeyboardEvent {
  return {
    key: init.key,
    metaKey: init.metaKey ?? false,
    ctrlKey: init.ctrlKey ?? false,
    shiftKey: init.shiftKey ?? false,
    altKey: init.altKey ?? false,
  } as KeyboardEvent;
}

describe("session split commands", () => {
  it("maps terminal pane shortcuts to command ids", () => {
    expect(commandIdForShortcut(keyEvent({ key: "d", metaKey: true }))).toBe("split-pane-vertical");
    expect(commandIdForShortcut(keyEvent({ key: "D", metaKey: true, shiftKey: true }))).toBe(
      "split-pane-horizontal",
    );
    expect(commandIdForShortcut(keyEvent({ key: "w", metaKey: true }))).toBe("close-pane");
    expect(commandIdForShortcut(keyEvent({ key: "w", metaKey: true, shiftKey: true }))).toBe("close-session");
  });

  it("runs pane actions through the app command ids", async () => {
    const directions: string[] = [];
    let closed = false;
    let sessionClosed = false;
    const commands = sessionSplitCommands({
      canSplit: true,
      canClose: true,
      canCloseSession: true,
      split: async (direction) => {
        directions.push(direction);
      },
      close: () => {
        closed = true;
      },
      closeSession: () => {
        sessionClosed = true;
      },
    });

    await commands.find((command) => command.id === "split-pane-vertical")?.run();
    await commands.find((command) => command.id === "split-pane-horizontal")?.run();
    await commands.find((command) => command.id === "close-pane")?.run();
    await commands.find((command) => command.id === "close-session")?.run();

    expect(directions).toEqual(["vertical", "horizontal"]);
    expect(closed).toBe(true);
    expect(sessionClosed).toBe(true);
    expect(commands.find((command) => command.id === "close-pane")).toMatchObject({
      title: "Close Pane",
      shortcut: "Cmd/Ctrl W",
    });
    expect(commands.find((command) => command.id === "close-session")).toMatchObject({
      title: "Close Session",
      shortcut: "Cmd/Ctrl Shift W",
    });
  });
});
