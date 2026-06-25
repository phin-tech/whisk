import { describe, expect, it } from "vitest";
import source from "./NewSessionDialog.svelte?raw";

describe("NewSessionDialog", () => {
  it("does not expose separate root and working directory fields by default", () => {
    expect(source).not.toMatch(/Root directory/);
    expect(source).not.toMatch(/Working directory/);
  });

  it("lets new sessions opt into agent bridge env injection", () => {
    expect(source).toMatch(/Agent bridge/);
    expect(source).toMatch(/agentBridge: agentBridge/);
    expect(source).toContain('from "./ui/Switch.svelte"');
    expect(source).toContain('from "./ui/SelectField.svelte"');
  });

  it("auto-detects known agent commands for bridge provider defaults", () => {
    expect(source).toMatch(/function commandProvider/);
    expect(source).toMatch(/base === "claude"/);
    expect(source).toMatch(/base === "codex"/);
  });
});
