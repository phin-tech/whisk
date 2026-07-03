import { describe, expect, it } from "vitest";

// Vitest runs in Node; the frontend tsconfig intentionally omits Node types.
// @ts-ignore
const { readFile } = await import("node:fs/promises");
const stylesSource = await readFile(new URL("./styles.css", import.meta.url), "utf8");

const modules = import.meta.glob("./**/*.svelte", {
  eager: true,
  import: "default",
  query: "?raw",
});
const svelteSources = Object.entries(modules).map(([path, source]) => ({
  file: path.replace("./", ""),
  source: String(source),
}));
const sourceByFile = new Map(svelteSources.map(({ file, source }) => [file, source]));

function sourceFor(file: string) {
  const source = sourceByFile.get(file);
  if (!source) throw new Error(`Missing component source for ${file}`);
  return source;
}

function sourceViolations(pattern: RegExp, allowed: (match: string) => boolean = () => false) {
  return svelteSources.flatMap(({ file, source }) =>
    Array.from(source.matchAll(pattern), ([match]) => match)
      .filter((match) => !allowed(match))
      .map((match) => `${file}: ${match}`),
  );
}

describe("design system adoption", () => {
  it("keeps raw hex values in the canonical stylesheet instead of components", () => {
    expect(sourceViolations(/#[0-9a-fA-F]{3,8}\b/g)).toEqual([]);
  });

  it("uses tokenized color utilities instead of raw hex surface classes", () => {
    expect(sourceViolations(/\b(?:bg|text|border|shadow|ring)-\[#[0-9a-fA-F]{3,8}\]/g)).toEqual(
      [],
    );
  });

  it("uses tokenized opacity utilities except for the documented modal scrim", () => {
    expect(
      sourceViolations(
        /\b(?:bg|border|text|divide)-(?:white|black)\/[0-9]+/g,
        (match) => match === "bg-black/70",
      ),
    ).toEqual([]);
  });

  it("uses explicit design-system type sizes instead of the default Tailwind scale", () => {
    expect(
      sourceViolations(/\btext-(?:xs|sm|base|lg|xl|2xl|3xl|4xl|5xl|6xl|7xl|8xl|9xl)\b/g),
    ).toEqual([]);
  });

  it("publishes role tokens for sidebar chrome and terminal panes", () => {
    expect(stylesSource).toContain("--color-sidebar-rail");
    expect(stylesSource).toContain("--color-sidebar-surface");
    expect(stylesSource).toContain("--color-sidebar-foreground");
    expect(stylesSource).toContain("--color-sidebar-active");
    expect(stylesSource).toContain("--color-terminal-surface");
    expect(stylesSource).toContain("--color-terminal-foreground");
  });

  it("adopts sidebar role utilities in rail and dock chrome", () => {
    expect(sourceFor("AppSidebar.svelte")).toContain("bg-sidebar-rail");
    expect(sourceFor("ActivityRail.svelte")).toContain("text-sidebar-foreground");
    expect(sourceFor("ActivityRail.svelte")).toContain("bg-sidebar-active");
    expect(sourceFor("SidebarDock.svelte")).toContain("bg-sidebar-surface");
  });
});
