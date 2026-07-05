import { describe, expect, it } from "vitest";
import type {
  PTYHistory,
  PTYHistorySummary,
  PTYInfo,
} from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
import ptysPanelSource from "./PtysPanel.svelte?raw";
import {
  derivePtyHistoryRows,
  derivePtyLiveRows,
  derivePtysPanelView,
  derivePtysPanelVirtualState,
  PTYS_PANEL_HISTORY_OUTPUT_ROW_HEIGHT,
  PTYS_PANEL_ITEM_ROW_HEIGHT,
  PTYS_PANEL_SECTION_ROW_HEIGHT,
  ptysPanelRowHeight,
  type PtysPanelRow,
} from "./ptys-panel-state";

function pty(overrides: Partial<PTYInfo> = {}): PTYInfo {
  return {
    id: "pty_01",
    workingDir: "/repo",
    cols: 80,
    rows: 24,
    running: true,
    status: "",
    exitCode: null,
    sessionId: "sess_01",
    windowId: "win_01",
    paneId: "pane_01",
    originWindowId: "win_01",
    originPaneId: "pane_01",
    ...overrides,
  } as PTYInfo;
}

function historySummary(overrides: Partial<PTYHistorySummary> = {}): PTYHistorySummary {
  return {
    ptyId: "pty_01",
    sessionId: "sess_01",
    windowId: "win_01",
    paneId: "pane_01",
    workingDir: "/repo",
    createdAt: "2026-06-19T12:00:00Z",
    exitCode: 0,
    ...overrides,
  } as PTYHistorySummary;
}

function history(overrides: Partial<PTYHistory> = {}): PTYHistory {
  return {
    ptyId: "pty_01",
    sessionId: "sess_01",
    windowId: "win_01",
    paneId: "pane_01",
    workingDir: "/repo",
    createdAt: "2026-06-19T12:00:00Z",
    exitCode: 0,
    output: "hello from history",
    ...overrides,
  } as PTYHistory;
}

describe("derivePtyLiveRows", () => {
  it("formats daemon PTY inventory with stable panel keys", () => {
    expect(
      derivePtyLiveRows([
        pty({ id: "pty_01", running: true }),
        pty({ id: "pty_02", running: false, status: "killed" }),
      ]),
    ).toEqual([
      {
        kind: "live",
        key: "live:pty_01",
        id: "pty_01",
        title: "pty_01",
        subtitle: "sess_01 / pane_01",
        detail: "/repo / 80x24",
        running: true,
        status: "running",
        canDelete: false,
      },
      {
        kind: "live",
        key: "live:pty_02",
        id: "pty_02",
        title: "pty_02",
        subtitle: "sess_01 / pane_01",
        detail: "/repo / 80x24",
        running: false,
        status: "killed",
        canDelete: true,
      },
    ]);
  });
});

describe("derivePtyHistoryRows", () => {
  it("summarizes persisted PTYs and marks the selected history row", () => {
    expect(
      derivePtyHistoryRows(
        [
          historySummary({ ptyId: "pty_01", exitCode: 0 }),
          historySummary({ ptyId: "pty_02", exitCode: null }),
        ],
        history({ ptyId: "pty_02", output: "" }),
      ),
    ).toMatchObject([
      {
        kind: "history",
        key: "history:pty_01",
        id: "pty_01",
        title: "pty_01",
        subtitle: "sess_01 / pane_01",
        detail: "/repo",
        createdAt: "2026-06-19T12:00:00Z",
        exitCode: 0,
        statusLabel: "exit 0",
        selected: false,
      },
      {
        kind: "history",
        key: "history:pty_02",
        id: "pty_02",
        exitCode: null,
        statusLabel: "saved",
        selected: true,
      },
    ]);
  });
});

describe("derivePtysPanelView", () => {
  it("flattens live and history sections with selected output rows", () => {
    const view = derivePtysPanelView({
      ptys: [pty()],
      ptyHistory: [historySummary()],
      selectedPTYHistory: history({ output: "line 1\nline 2" }),
      loading: false,
      loadingHistory: false,
    });

    expect(view.showLoadingEmpty).toBe(false);
    expect(view.showEmpty).toBe(false);
    expect(view.liveCount).toBe(1);
    expect(view.historyCount).toBe(1);
    expect(view.rows.map((row) => row.kind)).toEqual([
      "section",
      "live",
      "section",
      "history",
      "history-output",
    ]);
    expect(view.rows[0]).toMatchObject({ key: "section:live", title: "Live", count: 1 });
    expect(view.rows[2]).toMatchObject({ key: "section:history", title: "History", count: 1 });
    expect(view.rows[4]).toMatchObject({
      kind: "history-output",
      key: "history-output:pty_01",
      ptyId: "pty_01",
      output: "line 1\nline 2",
    });
  });

  it("shows loading or empty state only when no rows are available", () => {
    expect(
      derivePtysPanelView({
        ptys: [],
        ptyHistory: [],
        selectedPTYHistory: null,
        loading: true,
        loadingHistory: false,
      }),
    ).toMatchObject({ showLoadingEmpty: true, showEmpty: false });

    expect(
      derivePtysPanelView({
        ptys: [],
        ptyHistory: [],
        selectedPTYHistory: null,
        loading: false,
        loadingHistory: false,
      }),
    ).toMatchObject({ showLoadingEmpty: false, showEmpty: true });
  });
});

describe("ptysPanelRowHeight", () => {
  it("keeps section, item, and selected-output rows at fixed heights", () => {
    const section: PtysPanelRow = {
      kind: "section",
      key: "section:live",
      section: "live",
      title: "Live",
      count: 0,
    };
    const live = derivePtyLiveRows([pty()])[0];
    const output: PtysPanelRow = {
      kind: "history-output",
      key: "history-output:pty_01",
      ptyId: "pty_01",
      output: "output",
    };

    expect(ptysPanelRowHeight(section)).toBe(PTYS_PANEL_SECTION_ROW_HEIGHT);
    expect(ptysPanelRowHeight(live)).toBe(PTYS_PANEL_ITEM_ROW_HEIGHT);
    expect(ptysPanelRowHeight(output)).toBe(PTYS_PANEL_HISTORY_OUTPUT_ROW_HEIGHT);
  });
});

describe("derivePtysPanelVirtualState", () => {
  it("returns bounded virtual rows with stable keys for large live lists", () => {
    const view = derivePtysPanelView({
      ptys: Array.from({ length: 80 }, (_, index) =>
        pty({ id: `pty_${String(index).padStart(2, "0")}` }),
      ),
      ptyHistory: [],
      selectedPTYHistory: null,
      loading: false,
      loadingHistory: false,
    });
    const virtualState = derivePtysPanelVirtualState({
      rows: view.rows,
      viewportHeight: PTYS_PANEL_ITEM_ROW_HEIGHT * 4,
      scrollOffset: PTYS_PANEL_SECTION_ROW_HEIGHT + PTYS_PANEL_ITEM_ROW_HEIGHT * 20,
      overscan: 2,
    });

    expect(virtualState.window.totalHeight).toBe(
      PTYS_PANEL_SECTION_ROW_HEIGHT + PTYS_PANEL_ITEM_ROW_HEIGHT * 80,
    );
    expect(virtualState.virtualRows.length).toBeLessThan(view.rows.length);
    expect(virtualState.virtualRows.map((row) => row.key)).toContain("live:pty_20");
    expect(virtualState.virtualRows[0].offsetTop).toBeGreaterThan(0);
  });
});

describe("PtysPanel state extraction and virtualization wiring", () => {
  it("derives rows through the beside-component state module", () => {
    expect(ptysPanelSource).toContain('from "./ptys-panel-state"');
    expect(ptysPanelSource).toContain("derivePtysPanelView");
    expect(ptysPanelSource).toContain("derivePtysPanelVirtualState");
    expect(ptysPanelSource).toContain("data-ptys-virtual-list");
    expect(ptysPanelSource).toContain("data-pty-virtual-row");
    expect(ptysPanelSource).not.toContain("ptyRowsFromInventory");
    expect(ptysPanelSource).not.toContain("ptyHistoryRows");
    expect(ptysPanelSource).not.toContain("@tanstack");
    expect(ptysPanelSource).not.toContain("localStorage");
  });
});
