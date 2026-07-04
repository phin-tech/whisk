import { describe, expect, it } from "vitest";
import {
  CLIENT_VIEW_STATE_KEY,
  LEGACY_UI_SETTINGS_KEY,
  MAX_TERMINAL_FONT_SIZE,
  MIN_TERMINAL_FONT_SIZE,
  defaultClientViewState,
  hydrateClientViewState,
  legacyCollapsedStageStorageKey,
  migrateLegacyLocalState,
  parseClientViewState,
  persistClientViewState,
  reconcileClientViewState,
  serializeClientViewState,
  type ClientViewStateV1,
  type ViewStateStorage,
} from "./viewState";

describe("client view state parsing", () => {
  it("returns fresh safe defaults for missing, malformed, and unknown-version JSON", () => {
    expect(parseClientViewState(null)).toEqual(defaultClientViewState());
    expect(parseClientViewState("{not json")).toEqual(defaultClientViewState());
    expect(parseClientViewState(JSON.stringify({ version: 999, preferences: { railSide: "left" } }))).toEqual(
      defaultClientViewState(),
    );

    const first = defaultClientViewState();
    first.work.collapsedStageIdsByProject.project_1 = ["todo"];
    expect(defaultClientViewState().work.collapsedStageIdsByProject).toEqual({});
  });

  it("sanitizes v1 data, clamps preferences, validates persisted IDs, and dedupes lists", () => {
    const state = parseClientViewState(
      JSON.stringify({
        version: 1,
        preferences: {
          railSide: "left",
          sidebarWidthPx: 900.4,
          terminalFontSize: 4,
          terminalCursorBlink: false,
          closePanePromptDisabled: true,
        },
        selection: {
          activeMain: "work",
          activeSidebar: null,
          activeSessionId: " sess_01 ",
          activePaneId: "pane_01",
          activeProjectId: " project_01 ",
          workBoardOpenItemId: " item_01 ",
          selectedPtyHistoryId: " pty_01 ",
        },
        work: {
          filterQuery: " blocking ",
          filterStageId: " doing ",
          filterRunState: " running ",
          collapsedStageIdsByProject: {
            " project_01 ": ["z", "a", "z", "", " a "],
            " ": ["ignored"],
          },
        },
        commandPalette: {
          recentCommandIds: ["work.refresh", "session.new", "work.refresh", " ", "preferences.open"],
        },
      }),
    );

    expect(state.preferences).toEqual({
      railSide: "left",
      sidebarWidthPx: 800,
      terminalFontSize: MIN_TERMINAL_FONT_SIZE,
      terminalCursorBlink: false,
      closePanePromptDisabled: true,
    });
    expect(state.selection).toEqual({
      activeMain: "work",
      activeSidebar: null,
      activeSessionId: "sess_01",
      activePaneId: "pane_01",
      activeProjectId: "project_01",
      workBoardOpenItemId: "item_01",
      selectedPtyHistoryId: "pty_01",
    });
    expect(state.work).toEqual({
      filterQuery: "blocking",
      filterStageId: "doing",
      filterRunState: "running",
      collapsedStageIdsByProject: { project_01: ["a", "z"] },
    });
    expect(state.commandPalette.recentCommandIds).toEqual([
      "work.refresh",
      "session.new",
      "preferences.open",
    ]);
  });

  it("serializes through the same sanitizer used at hydration boundaries", () => {
    const invalidState = {
      ...defaultClientViewState(),
      preferences: {
        ...defaultClientViewState().preferences,
        sidebarWidthPx: 12,
        terminalFontSize: 42,
      },
    } satisfies ClientViewStateV1;

    const serialized = JSON.parse(serializeClientViewState(invalidState)) as ClientViewStateV1;
    expect(serialized.preferences.sidebarWidthPx).toBe(240);
    expect(serialized.preferences.terminalFontSize).toBe(MAX_TERMINAL_FONT_SIZE);
  });
});

describe("legacy migration", () => {
  it("migrates the old unversioned UI settings key into hydrated v1 state", () => {
    const storage = memoryStorage({
      [LEGACY_UI_SETTINGS_KEY]: JSON.stringify({
        railSide: "left",
        terminalFontSize: 99,
        terminalCursorBlink: false,
        closePanePromptDisabled: true,
      }),
    });

    expect(migrateLegacyLocalState(storage).preferences).toEqual({
      railSide: "left",
      terminalFontSize: MAX_TERMINAL_FONT_SIZE,
      terminalCursorBlink: false,
      closePanePromptDisabled: true,
    });

    const hydrated = hydrateClientViewState({ storage });
    expect(hydrated.preferences).toEqual({
      ...defaultClientViewState().preferences,
      railSide: "left",
      terminalFontSize: MAX_TERMINAL_FONT_SIZE,
      terminalCursorBlink: false,
      closePanePromptDisabled: true,
    });
    expect(JSON.parse(storage.get(CLIENT_VIEW_STATE_KEY) ?? "{}")).toMatchObject({ version: 1 });
    expect(storage.get(LEGACY_UI_SETTINGS_KEY)).not.toBeNull();
  });

  it("migrates legacy WorkBoard collapsed-stage keys for known projects and prunes stale stages", () => {
    const storage = memoryStorage({
      [legacyCollapsedStageStorageKey("project_01")]: JSON.stringify([
        "done",
        "doing",
        "doing",
        "todo",
      ]),
    });

    const hydrated = hydrateClientViewState({
      storage,
      projects: [
        {
          id: "project_01",
          workflow: { stages: [{ id: "todo" }, { id: "doing" }] },
        },
      ],
    });

    expect(hydrated.work.collapsedStageIdsByProject).toEqual({
      project_01: ["doing", "todo"],
    });
  });

  it("prefers an existing v1 value over legacy settings while still filling missing legacy collapsed stages", () => {
    const storage = memoryStorage({
      [CLIENT_VIEW_STATE_KEY]: serializeClientViewState({
        ...defaultClientViewState(),
        preferences: { ...defaultClientViewState().preferences, railSide: "right" },
      }),
      [LEGACY_UI_SETTINGS_KEY]: JSON.stringify({ railSide: "left" }),
      [legacyCollapsedStageStorageKey("project_01")]: JSON.stringify(["todo"]),
    });

    const hydrated = hydrateClientViewState({
      storage,
      projects: [{ id: "project_01", workflow: { stages: [{ id: "todo" }] } }],
    });

    expect(hydrated.preferences.railSide).toBe("right");
    expect(hydrated.work.collapsedStageIdsByProject).toEqual({ project_01: ["todo"] });
  });

  it("keeps explicit empty collapsed-stage overrides ahead of stale legacy keys", () => {
    const storage = memoryStorage({
      [CLIENT_VIEW_STATE_KEY]: serializeClientViewState({
        ...defaultClientViewState(),
        work: {
          ...defaultClientViewState().work,
          collapsedStageIdsByProject: { project_01: [] },
        },
      }),
      [legacyCollapsedStageStorageKey("project_01")]: JSON.stringify(["todo"]),
    });

    const hydrated = hydrateClientViewState({
      storage,
      projects: [{ id: "project_01", workflow: { stages: [{ id: "todo" }] } }],
    });

    expect(hydrated.work.collapsedStageIdsByProject).toEqual({ project_01: [] });
    expect(JSON.parse(storage.get(CLIENT_VIEW_STATE_KEY) ?? "{}")).toMatchObject({
      work: { collapsedStageIdsByProject: { project_01: [] } },
    });
  });
});

describe("reconciliation", () => {
  it("preserves stored hints when read models are not available yet", () => {
    const state = parseClientViewState(
      JSON.stringify({
        version: 1,
        selection: {
          activeSessionId: "session_01",
          activePaneId: "pane_01",
          activeProjectId: "project_01",
          workBoardOpenItemId: "item_01",
          selectedPtyHistoryId: "pty_01",
        },
        work: {
          filterStageId: "doing",
          collapsedStageIdsByProject: { project_01: ["doing", "todo"] },
        },
      }),
    );

    expect(reconcileClientViewState(state, {})).toMatchObject({
      selection: {
        activeSessionId: "session_01",
        activePaneId: "pane_01",
        activeProjectId: "project_01",
        workBoardOpenItemId: "item_01",
        selectedPtyHistoryId: "pty_01",
      },
      work: {
        filterStageId: "doing",
        collapsedStageIdsByProject: { project_01: ["doing", "todo"] },
      },
    });
  });

  it("repairs only the read-model families that are explicitly available", () => {
    const state = parseClientViewState(
      JSON.stringify({
        version: 1,
        selection: {
          activeSessionId: "missing_session",
          activePaneId: "missing_pane",
          activeProjectId: "project_01",
          workBoardOpenItemId: "item_01",
          selectedPtyHistoryId: "pty_01",
        },
        work: {
          filterStageId: "doing",
          collapsedStageIdsByProject: { project_01: ["doing"] },
        },
      }),
    );

    const reconciled = reconcileClientViewState(state, {
      sessions: [{ id: "session_02", panes: { pane_02: { id: "pane_02" } } }],
    });

    expect(reconciled.selection).toMatchObject({
      activeSessionId: "session_02",
      activePaneId: "pane_02",
      activeProjectId: "project_01",
      workBoardOpenItemId: "item_01",
      selectedPtyHistoryId: "pty_01",
    });
    expect(reconciled.work.filterStageId).toBe("doing");
    expect(reconciled.work.collapsedStageIdsByProject).toEqual({ project_01: ["doing"] });
  });

  it("treats persisted daemon IDs as hints and repairs stale selection against read models", () => {
    const state = parseClientViewState(
      JSON.stringify({
        version: 1,
        selection: {
          activeSessionId: "missing_session",
          activePaneId: "missing_pane",
          activeProjectId: "missing_project",
          workBoardOpenItemId: "missing_item",
          selectedPtyHistoryId: "missing_pty",
        },
        work: {
          filterStageId: "old_stage",
          collapsedStageIdsByProject: {
            project_01: ["old_stage", "todo"],
            missing_project: ["todo"],
          },
        },
      }),
    );

    const reconciled = reconcileClientViewState(state, {
      sessions: [{ id: "session_01", panes: { pane_02: { id: "pane_02" } } }],
      projects: [{ id: "project_01", workflow: { stages: [{ id: "todo" }, { id: "doing" }] } }],
      ptyHistory: [{ ptyId: "pty_01" }],
      workItems: [{ id: "item_01", projectId: "project_01" }],
    });

    expect(reconciled.selection).toMatchObject({
      activeSessionId: "session_01",
      activePaneId: "pane_02",
      activeProjectId: "",
      workBoardOpenItemId: "",
      selectedPtyHistoryId: "",
    });
    expect(reconciled.work.filterStageId).toBe("");
    expect(reconciled.work.collapsedStageIdsByProject).toEqual({ project_01: ["todo"] });
  });

  it("keeps valid selection, repairs stale panes, and clears work items from the wrong project", () => {
    const state = parseClientViewState(
      JSON.stringify({
        version: 1,
        selection: {
          activeSessionId: "session_01",
          activePaneId: "stale_pane",
          activeProjectId: "project_01",
          workBoardOpenItemId: "item_other",
          selectedPtyHistoryId: "pty_01",
        },
        work: {
          filterStageId: "doing",
          collapsedStageIdsByProject: { project_01: ["doing"] },
        },
      }),
    );

    const reconciled = reconcileClientViewState(state, {
      sessions: [
        {
          id: "session_01",
          panes: { pane_01: { id: "pane_01" }, pane_02: { id: "pane_02" } },
        },
      ],
      projects: [{ id: "project_01", workflow: { stages: [{ id: "todo" }, { id: "doing" }] } }],
      ptyHistory: [{ ptyId: "pty_01" }],
      workItems: [{ id: "item_other", projectId: "project_02" }],
    });

    expect(reconciled.selection).toMatchObject({
      activeSessionId: "session_01",
      activePaneId: "pane_01",
      activeProjectId: "project_01",
      workBoardOpenItemId: "",
      selectedPtyHistoryId: "pty_01",
    });
    expect(reconciled.work.filterStageId).toBe("doing");
    expect(reconciled.work.collapsedStageIdsByProject).toEqual({ project_01: ["doing"] });
  });

  it("clears session and pane selection when the daemon has no session read models", () => {
    const reconciled = reconcileClientViewState(
      parseClientViewState(
        JSON.stringify({
          version: 1,
          selection: { activeSessionId: "session_01", activePaneId: "pane_01" },
        }),
      ),
      { sessions: [] },
    );

    expect(reconciled.selection.activeSessionId).toBe("");
    expect(reconciled.selection.activePaneId).toBe("");
  });
});

describe("storage helpers", () => {
  it("persists sanitized v1 JSON through injected storage", () => {
    const storage = memoryStorage();
    persistClientViewState(storage, {
      ...defaultClientViewState(),
      commandPalette: { recentCommandIds: ["b", "a", "b"] },
    });

    expect(parseClientViewState(storage.get(CLIENT_VIEW_STATE_KEY))).toMatchObject({
      version: 1,
      commandPalette: { recentCommandIds: ["b", "a"] },
    });
  });
});

function memoryStorage(seed: Record<string, string> = {}): ViewStateStorage & {
  get(key: string): string | null;
} {
  const values = new Map(Object.entries(seed));
  return {
    getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => {
      values.set(key, value);
    },
    get: (key: string) => values.get(key) ?? null,
  };
}
