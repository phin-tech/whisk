import { describe, expect, it } from "vitest";
import {
  applyRecentJumpTargets,
  reconcileJumpRecents,
  updateJumpRecents,
} from "./jumpRecents";

describe("updateJumpRecents", () => {
  it("prepends activated id and deduplicates", () => {
    expect(updateJumpRecents(["a", "b", "c"], "d", 10)).toEqual(["d", "a", "b", "c"]);
  });

  it("moves existing entry to front", () => {
    expect(updateJumpRecents(["a", "b", "c"], "b", 10)).toEqual(["b", "a", "c"]);
  });

  it("caps at max size", () => {
    const many = ["a", "b", "c", "d", "e"];
    expect(updateJumpRecents(many, "f", 3)).toEqual(["f", "a", "b"]);
  });

  it("returns empty when max size is zero or negative", () => {
    expect(updateJumpRecents(["a", "b"], "c", 0)).toEqual([]);
    expect(updateJumpRecents(["a", "b"], "c", -1)).toEqual([]);
  });

  it("ignores empty activation ids without clearing existing recents", () => {
    expect(updateJumpRecents([" a ", "b", "a"], " ", 10)).toEqual(["a", "b"]);
  });

  it("handles empty current list", () => {
    expect(updateJumpRecents([], "a", 10)).toEqual(["a"]);
  });

  it("caps after dedup preserves last-in semantics", () => {
    const list = ["x", "y", "z"];
    expect(updateJumpRecents(list, "y", 2)).toEqual(["y", "x"]);
  });
});

describe("reconcileJumpRecents", () => {
  it("keeps recent ids that appear in available set", () => {
    expect(reconcileJumpRecents(["a", "b", "c"], ["a", "c", "d"])).toEqual(["a", "c"]);
  });

  it("deduplicates and trims recents while preserving recency order", () => {
    expect(reconcileJumpRecents([" c ", "a", "c", "", "b"], ["a", "b", "c"])).toEqual([
      "c",
      "a",
      "b",
    ]);
  });

  it("returns empty when no recent ids are available", () => {
    expect(reconcileJumpRecents(["a", "b"], ["c", "d"])).toEqual([]);
  });

  it("returns empty for empty input", () => {
    expect(reconcileJumpRecents([], ["a", "b"])).toEqual([]);
    expect(reconcileJumpRecents(["a"], [])).toEqual([]);
  });
});

describe("applyRecentJumpTargets", () => {
  it("puts current targets first, then recent, then remaining", () => {
    const targets = [
      { id: "p", current: false },
      { id: "a", current: true },
      { id: "b", current: false },
      { id: "c", current: false },
    ];
    const recent = ["b", "z"];

    expect(applyRecentJumpTargets(targets, recent).map((t) => t.id)).toEqual([
      "a",
      "b",
      "p",
      "c",
    ]);
  });

  it("preserves input order within each group", () => {
    const targets = [
      { id: "a", current: false },
      { id: "b", current: false },
      { id: "c", current: true },
      { id: "d", current: false },
    ];
    const recent: string[] = [];

    expect(applyRecentJumpTargets(targets, recent).map((t) => t.id)).toEqual([
      "c",
      "a",
      "b",
      "d",
    ]);
  });

  it("handles empty targets", () => {
    expect(applyRecentJumpTargets([], ["a"])).toEqual([]);
  });

  it("handles empty recent set", () => {
    const targets = [
      { id: "x", current: false },
      { id: "y", current: true },
    ];
    expect(applyRecentJumpTargets(targets, []).map((t) => t.id)).toEqual([
      "y",
      "x",
    ]);
  });

  it("orders recent targets by most-recent id order instead of input order", () => {
    const targets = [
      { id: "a", current: false },
      { id: "b", current: false },
      { id: "c", current: false },
    ];

    expect(applyRecentJumpTargets(targets, ["c", "a"]).map((t) => t.id)).toEqual([
      "c",
      "a",
      "b",
    ]);
  });

  it("keeps current targets ahead of recent targets", () => {
    const targets = [
      { id: "a", current: false },
      { id: "b", current: true },
      { id: "c", current: false },
    ];

    expect(applyRecentJumpTargets(targets, ["c", "b", "a"]).map((t) => t.id)).toEqual([
      "b",
      "c",
      "a",
    ]);
  });

  it("places first duplicate-id target in recents, remaining in other group", () => {
    const targets = [
      { id: "dup", current: false },
      { id: "a", current: false },
      { id: "dup", current: false },
    ];
    const recent = ["dup"];

    expect(applyRecentJumpTargets(targets, recent).map((t) => t.id)).toEqual([
      "dup",
      "a",
      "dup",
    ]);
  });
});
