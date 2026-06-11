import { describe, expect, it } from "vitest";
import { paneIds, visiblePtyIds } from "./sessionView";

describe("paneIds", () => {
  it("walks nested split layouts in render order", () => {
    expect(
      paneIds({
        kind: "split",
        direction: "horizontal",
        children: [
          {
            kind: "split",
            direction: "vertical",
            children: [
              { kind: "leaf", paneId: "pane_01" },
              { kind: "leaf", paneId: "pane_02" },
            ],
          },
          { kind: "leaf", paneId: "pane_03" },
        ],
      }),
    ).toEqual(["pane_01", "pane_02", "pane_03"]);
  });
});

describe("visiblePtyIds", () => {
  it("puts the active pane first and removes duplicates", () => {
    const sessions = [
      {
        id: "sess_01",
        name: "one",
        workingDir: ".",
        focusedPaneId: "pane_02",
        layout: {
          kind: "split",
          direction: "horizontal",
          children: [
            { kind: "leaf", paneId: "pane_01" },
            { kind: "leaf", paneId: "pane_02" },
          ],
        },
        panes: {
          pane_01: { id: "pane_01", ptyId: "pty_01" },
          pane_02: { id: "pane_02", ptyId: "pty_02" },
        },
      },
      {
        id: "sess_02",
        name: "two",
        workingDir: ".",
        focusedPaneId: "pane_03",
        layout: { kind: "leaf", paneId: "pane_03" },
        panes: {
          pane_03: { id: "pane_03", ptyId: "pty_02" },
        },
      },
    ];

    expect(visiblePtyIds(sessions, "sess_01", "pane_02")).toEqual(["pty_02", "pty_01"]);
  });
});
