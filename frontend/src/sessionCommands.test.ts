import { describe, expect, it } from "vitest";
import {
  sessionSplitCommands,
} from "./sessionCommands";

describe("session split commands", () => {
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
