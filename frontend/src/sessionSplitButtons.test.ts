import { describe, expect, it } from "vitest";
import source from "./App.svelte?raw";

describe("session split buttons", () => {
  it("does not render split right or split down controls", () => {
    expect(source).not.toMatch(/Split right/);
    expect(source).not.toMatch(/Split down/);
  });
});
