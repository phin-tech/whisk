import { describe, expect, it } from "vitest";

const modules = import.meta.glob("./*.svelte", {
  eager: true,
  import: "default",
  query: "?raw",
});
const svelteSources = Object.entries(modules).map(([path, source]) => ({
  file: path.replace("./", ""),
  source: String(source),
}));

function sourceViolations(pattern: RegExp, allowed: (match: string) => boolean = () => false) {
  return svelteSources.flatMap(({ file, source }) =>
    Array.from(source.matchAll(pattern), ([match]) => match)
      .filter((match) => !allowed(match))
      .map((match) => `${file}: ${match}`),
  );
}

describe("design system adoption", () => {
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
});
