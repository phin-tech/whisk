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
  it("maps iTerm split shortcuts to command ids", () => {
    expect(commandIdForShortcut(keyEvent({ key: "d", metaKey: true }))).toBe("split-pane-vertical");
    expect(commandIdForShortcut(keyEvent({ key: "D", metaKey: true, shiftKey: true }))).toBe(
      "split-pane-horizontal",
    );
  });

  it("runs split actions through the app command ids", async () => {
    const directions: string[] = [];
    const commands = sessionSplitCommands({
      canSplit: true,
      split: async (direction) => {
        directions.push(direction);
      },
    });

    await commands.find((command) => command.id === "split-pane-vertical")?.run();
    await commands.find((command) => command.id === "split-pane-horizontal")?.run();

    expect(directions).toEqual(["vertical", "horizontal"]);
  });
});
