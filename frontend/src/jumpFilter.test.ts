import { describe, expect, it } from "vitest";
import {
  MAX_JUMP_QUERY_BYTES,
  prepareJumpTargets,
  rankJumpTargets,
  type PreparedJumpTarget,
} from "./jumpFilter";

describe("jumpFilter", () => {
  it("prepares stable searchable targets and preserves empty-query order", () => {
    const prepared = prepareJumpTargets([
      { id: "session:sess_1", kind: "session", title: "API", subtitle: "/repo/api" },
      { id: "project:proj_1", kind: "project", title: "Whisk", disabled: true },
      { id: "work-item:item_1", kind: "work-item", title: "Build palette" },
      { id: "pty:pty_1", kind: "pty", title: "PTY", detail: "/repo/api/frontend" },
    ]);

    expect(prepared.map((target) => target.inputIndex)).toEqual([0, 1, 2, 3]);
    expect(prepared[0].searchableText).toContain("session:sess_1");
    expect(rankJumpTargets("", prepared, 2).map((target) => target.id)).toEqual([
      "session:sess_1",
      "work-item:item_1",
    ]);
  });

  it("rejects oversized queries before scanning target fields", () => {
    const dangerous = {} as PreparedJumpTarget;
    Object.defineProperty(dangerous, "disabled", {
      get() {
        throw new Error("target fields should not be read");
      },
    });

    expect(rankJumpTargets("x".repeat(MAX_JUMP_QUERY_BYTES + 1), [dangerous])).toEqual([]);
  });

  it("prioritizes exact ids and work item numbers over title matches", () => {
    const prepared = prepareJumpTargets([
      { id: "project:proj_40", kind: "project", title: "Issue tracker 40" },
      {
        id: "work-item:item_abc",
        kind: "work-item",
        title: "Jump palette foundation",
        keywords: ["#40", "40", "item_abc"],
      },
    ]);

    expect(rankJumpTargets("40", prepared).map((target) => target.id)).toEqual([
      "work-item:item_abc",
      "project:proj_40",
    ]);
  });

  it("keeps strong title matches ahead of keyword, subtitle, detail, and fuzzy matches", () => {
    const prepared = prepareJumpTargets([
      {
        id: "session:sess_fuzzy",
        kind: "session",
        title: "Command utility",
        keywords: ["palette"],
      },
      {
        id: "project:proj_detail",
        kind: "project",
        title: "Frontend",
        detail: "palette",
      },
      {
        id: "work-item:item_title",
        kind: "work-item",
        title: "Palette actions",
      },
      {
        id: "pty:pty_fuzzy",
        kind: "pty",
        title: "P l t shell",
        detail: "/repo/palette",
      },
    ]);

    expect(rankJumpTargets("palette", prepared).map((target) => target.id)).toEqual([
      "work-item:item_title",
      "session:sess_fuzzy",
      "project:proj_detail",
      "pty:pty_fuzzy",
    ]);
  });

  it("supports path-like slash matching", () => {
    const prepared = prepareJumpTargets([
      { id: "pty:pty_api", kind: "pty", title: "API", detail: "/Users/dev/whisk/frontend/src" },
      { id: "pty:pty_other", kind: "pty", title: "Other", detail: "/Users/dev/whisk/internal/app" },
    ]);

    expect(rankJumpTargets("frontend/src", prepared).map((target) => target.id)).toEqual([
      "pty:pty_api",
    ]);
  });

  it("uses kind priority and input order for stable ties", () => {
    const prepared = prepareJumpTargets([
      { id: "work-item:item_1", kind: "work-item", title: "Alpha" },
      { id: "session:sess_1", kind: "session", title: "Alpha" },
      { id: "session:sess_2", kind: "session", title: "Alpha" },
      { id: "project:proj_1", kind: "project", title: "Alpha" },
    ]);

    expect(rankJumpTargets("alpha", prepared).map((target) => target.id)).toEqual([
      "session:sess_1",
      "session:sess_2",
      "project:proj_1",
      "work-item:item_1",
    ]);
  });

  it("caps result counts", () => {
    const prepared = prepareJumpTargets(
      Array.from({ length: 55 }, (_, index) => ({
        id: `project:${index}`,
        kind: "project" as const,
        title: `Project ${index}`,
      })),
    );

    expect(rankJumpTargets("project", prepared)).toHaveLength(50);
  });
});
