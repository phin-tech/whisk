import { describe, expect, it } from "vitest";
import source from "./TerminalPane.svelte?raw";

describe("TerminalPane", () => {
  it("focuses xterm when the pane becomes focused", () => {
    expect(source).toMatch(/if \(focused && terminal\)[\s\S]*terminal\.focus\(\)/);
  });
});
