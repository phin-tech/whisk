import { describe, expect, it } from "vitest";
import source from "./NewSessionDialog.svelte?raw";

describe("NewSessionDialog", () => {
  it("does not expose separate root and working directory fields by default", () => {
    expect(source).not.toMatch(/Root directory/);
    expect(source).not.toMatch(/Working directory/);
  });
});
