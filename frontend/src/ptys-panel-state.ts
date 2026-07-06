import {
  deriveVirtualIndexWindow,
  deriveVirtualRows,
  type VirtualIndexWindow,
  type VirtualRow,
} from "./ui/virtual-list";

type PtyInfoLike = {
  id: string;
  workingDir: string;
  cols: number;
  rows: number;
  running: boolean;
  status?: string;
  sessionId: string;
  paneId: string;
  title?: string;
  terminalWorkingDirectory?: string;
  agentStatus?: PtyAgentStatusLike | null;
};

type PtyAgentStatusLike = {
  label?: string;
  state?: string;
  title?: string;
  prompt?: string;
};

type PtyHistorySummaryLike = {
  ptyId: string;
  sessionId?: string;
  paneId?: string;
  workingDir?: string;
  createdAt?: unknown;
  exitCode?: number | null;
};

type PtyHistoryLike = {
  ptyId: string;
  output?: string;
};

export type PtysPanelSectionRow = {
  kind: "section";
  key: string;
  section: "live" | "history";
  title: string;
  count: number;
};

export type PtysPanelLiveRow = {
  kind: "live";
  key: string;
  id: string;
  title: string;
  subtitle: string;
  detail: string;
  running: boolean;
  status: string;
  statusTone: "running" | "waiting" | "working" | "idle" | "exited";
  canDelete: boolean;
  agentLabel: string;
  agentTitle: string;
};

export type PtysPanelHistoryRow = {
  kind: "history";
  key: string;
  id: string;
  title: string;
  subtitle: string;
  detail: string;
  createdAt: unknown;
  exitCode: number | null;
  statusLabel: string;
  selected: boolean;
};

export type PtysPanelHistoryOutputRow = {
  kind: "history-output";
  key: string;
  ptyId: string;
  output: string;
};

export type PtysPanelRow =
  | PtysPanelSectionRow
  | PtysPanelLiveRow
  | PtysPanelHistoryRow
  | PtysPanelHistoryOutputRow;

export type PtysPanelViewInput = {
  ptys: readonly PtyInfoLike[];
  ptyHistory: readonly PtyHistorySummaryLike[];
  selectedPTYHistory: PtyHistoryLike | null;
  loading: boolean;
  loadingHistory: boolean;
};

export type PtysPanelView = {
  rows: PtysPanelRow[];
  liveCount: number;
  historyCount: number;
  showLoadingEmpty: boolean;
  showEmpty: boolean;
};

export type PtysPanelVirtualState = {
  window: VirtualIndexWindow;
  virtualRows: VirtualRow<PtysPanelRow>[];
};

export const PTYS_PANEL_SECTION_ROW_HEIGHT = 28;
export const PTYS_PANEL_ITEM_ROW_HEIGHT = 66;
export const PTYS_PANEL_HISTORY_OUTPUT_ROW_HEIGHT = 264;

export function derivePtysPanelView(input: PtysPanelViewInput): PtysPanelView {
  const liveRows = derivePtyLiveRows(input.ptys);
  const historyRows = derivePtyHistoryRows(input.ptyHistory, input.selectedPTYHistory);
  const rows: PtysPanelRow[] = [];

  if (liveRows.length > 0) {
    rows.push(sectionRow("live", "Live", liveRows.length));
    rows.push(...liveRows);
  }

  if (historyRows.length > 0) {
    rows.push(sectionRow("history", "History", historyRows.length));
    for (const row of historyRows) {
      rows.push(row);
      if (row.selected) {
        rows.push({
          kind: "history-output",
          key: `history-output:${row.id}`,
          ptyId: row.id,
          output: input.selectedPTYHistory?.output || "(no output)",
        });
      }
    }
  }

  const showLoadingEmpty = (input.loading || input.loadingHistory) && rows.length === 0;

  return {
    rows,
    liveCount: liveRows.length,
    historyCount: historyRows.length,
    showLoadingEmpty,
    showEmpty: !showLoadingEmpty && rows.length === 0,
  };
}

export function derivePtyLiveRows(ptys: readonly PtyInfoLike[]): PtysPanelLiveRow[] {
  return ptys.map((pty) => {
    const agentState = normalizedAgentState(pty.agentStatus?.state);
    const status = agentState || pty.status || (pty.running ? "running" : "exited");
    return {
      kind: "live",
      key: `live:${pty.id}`,
      id: pty.id,
      title: pty.title || pty.id,
      subtitle: `${pty.sessionId || "unowned"} / ${pty.paneId || "detached"}`,
      detail: `${pty.terminalWorkingDirectory || pty.workingDir || "."} / ${pty.cols}x${pty.rows}`,
      running: pty.running,
      status,
      statusTone: ptyStatusTone(pty.running, agentState),
      canDelete: !pty.running,
      agentLabel: pty.agentStatus?.label || "",
      agentTitle: pty.agentStatus?.title || pty.agentStatus?.prompt || "",
    };
  });
}

export function derivePtyHistoryRows(
  history: readonly PtyHistorySummaryLike[],
  selectedPTYHistory: PtyHistoryLike | null = null,
): PtysPanelHistoryRow[] {
  const selectedPtyId = selectedPTYHistory?.ptyId ?? "";
  return history.map((item) => {
    const exitCode = item.exitCode ?? null;
    return {
      kind: "history",
      key: `history:${item.ptyId}`,
      id: item.ptyId,
      title: item.ptyId,
      subtitle: `${item.sessionId || "unowned"} / ${item.paneId || "detached"}`,
      detail: item.workingDir || ".",
      createdAt: item.createdAt || "",
      exitCode,
      statusLabel: exitCode === null ? "saved" : `exit ${exitCode}`,
      selected: item.ptyId === selectedPtyId,
    };
  });
}

export function ptysPanelRowHeight(row: PtysPanelRow): number {
  if (row.kind === "section") return PTYS_PANEL_SECTION_ROW_HEIGHT;
  if (row.kind === "history-output") return PTYS_PANEL_HISTORY_OUTPUT_ROW_HEIGHT;
  return PTYS_PANEL_ITEM_ROW_HEIGHT;
}

export function derivePtysPanelVirtualState(input: {
  rows: readonly PtysPanelRow[];
  viewportHeight: number;
  scrollOffset: number;
  overscan?: number;
}): PtysPanelVirtualState {
  const heights = input.rows.map(ptysPanelRowHeight);
  const window = deriveVirtualIndexWindow({
    count: input.rows.length,
    heights,
    viewportHeight: input.viewportHeight,
    scrollOffset: input.scrollOffset,
    overscan: input.overscan,
  });
  return {
    window,
    virtualRows: deriveVirtualRows(input.rows, heights, window),
  };
}

function sectionRow(
  section: PtysPanelSectionRow["section"],
  title: string,
  count: number,
): PtysPanelSectionRow {
  return {
    kind: "section",
    key: `section:${section}`,
    section,
    title,
    count,
  };
}

function normalizedAgentState(state: string | undefined): string {
  if (!state || state === "unknown") return "";
  if (state === "waiting" || state === "working" || state === "idle" || state === "done") {
    return state;
  }
  return "";
}

function ptyStatusTone(
  running: boolean,
  agentState: string,
): PtysPanelLiveRow["statusTone"] {
  if (!running) return "exited";
  if (agentState === "waiting") return "waiting";
  if (agentState === "working") return "working";
  if (agentState === "idle" || agentState === "done") return "idle";
  return "running";
}
