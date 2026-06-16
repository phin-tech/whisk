import { describe, expect, it } from "vitest";
import { commandItems, runCommand } from "./commands";

describe("commands", () => {
  it("filters disabled commands from palette items", () => {
    expect(
      commandItems([
        { id: "enabled", title: "Enabled", run: () => {} },
        { id: "disabled", title: "Disabled", enabled: () => false, run: () => {} },
      ]),
    ).toEqual([{ id: "enabled", title: "Enabled" }]);
  });

  it("matches commands by id and title", () => {
    const commands = [
      { id: "notifications.clear", title: "Clear Notifications", run: () => {} },
      { id: "session.new", title: "New Session", run: () => {} },
    ];

    expect(commandItems(commands, "clear")).toEqual([
      { id: "notifications.clear", title: "Clear Notifications" },
    ]);
    expect(commandItems(commands, "session.new")).toEqual([
      { id: "session.new", title: "New Session" },
    ]);
  });

  it("runs enabled commands only", async () => {
    let ran = "";
    const commands = [
      { id: "yes", title: "Yes", run: () => { ran = "yes"; } },
      { id: "no", title: "No", enabled: () => false, run: () => { ran = "no"; } },
    ];

    expect(await runCommand(commands, "missing")).toBe(false);
    expect(await runCommand(commands, "no")).toBe(false);
    expect(ran).toBe("");

    expect(await runCommand(commands, "yes")).toBe(true);
    expect(ran).toBe("yes");
  });
});
