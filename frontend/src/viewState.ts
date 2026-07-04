import {
  idString,
  parsePaneId,
  parseProjectId,
  parsePtyId,
  parseSessionId,
  parseWorkItemId,
} from "./ids";
import type { MainView } from "./navigation";
import type { SidebarId } from "./sidebarCommands";

export const CLIENT_VIEW_STATE_KEY = "whisk.clientViewState";
export const CLIENT_VIEW_STATE_VERSION = 1;
export const LEGACY_UI_SETTINGS_KEY = "whisk.ui.settings";

export const MIN_SIDEBAR_WIDTH_PX = 240;
export const MAX_SIDEBAR_WIDTH_PX = 800;
export const MIN_TERMINAL_FONT_SIZE = 10;
export const MAX_TERMINAL_FONT_SIZE = 20;
export const MAX_RECENT_COMMANDS = 20;
export const MAX_RECENT_JUMP_TARGETS = 20;

export type RailSide = "left" | "right";

export type ClientViewStateV1 = {
  version: 1;
  preferences: {
    railSide: RailSide;
    sidebarWidthPx: number;
    terminalFontSize: number;
    terminalCursorBlink: boolean;
    closePanePromptDisabled: boolean;
  };
  selection: {
    activeMain: MainView;
    activeSidebar: SidebarId | null;
    activeSessionId: string;
    activePaneId: string;
    activeProjectId: string;
    workBoardOpenItemId: string;
    selectedPtyHistoryId: string;
  };
  work: {
    filterQuery: string;
    filterStageId: string;
    filterRunState: string;
    collapsedStageIdsByProject: Record<string, string[]>;
  };
  commandPalette: {
    recentCommandIds: string[];
  };
  jumpPalette: {
    recentTargetIds: string[];
  };
};

export type PartialClientViewStateV1 = {
  preferences?: Partial<ClientViewStateV1["preferences"]>;
  selection?: Partial<ClientViewStateV1["selection"]>;
  work?: Partial<ClientViewStateV1["work"]>;
  commandPalette?: Partial<ClientViewStateV1["commandPalette"]>;
  jumpPalette?: Partial<ClientViewStateV1["jumpPalette"]>;
};

export type ViewStateStorageReader = Pick<Storage, "getItem">;
export type ViewStateStorage = Pick<Storage, "getItem" | "setItem">;

export type SessionReadModelLike = {
  id?: string | null;
  panes?: Record<string, { id?: string | null } | null | undefined> | null;
};

export type ProjectReadModelLike = {
  id?: string | null;
  workflow?: {
    stages?: Array<{ id?: string | null } | null | undefined> | null;
  } | null;
};

export type PtyHistoryReadModelLike = {
  id?: string | null;
  ptyId?: string | null;
};

export type WorkItemReadModelLike = {
  id?: string | null;
  projectId?: string | null;
};

export type ReconcileClientViewStateArgs = {
  sessions?: SessionReadModelLike[];
  projects?: ProjectReadModelLike[];
  ptyHistory?: PtyHistoryReadModelLike[];
  workItems?: WorkItemReadModelLike[];
};

export type HydrateClientViewStateArgs = ReconcileClientViewStateArgs & {
  storage: ViewStateStorage;
};

export function defaultClientViewState(): ClientViewStateV1 {
  return {
    version: CLIENT_VIEW_STATE_VERSION,
    preferences: {
      railSide: "right",
      sidebarWidthPx: 320,
      terminalFontSize: 13,
      terminalCursorBlink: true,
      closePanePromptDisabled: false,
    },
    selection: {
      activeMain: "session",
      activeSidebar: "sessions",
      activeSessionId: "",
      activePaneId: "",
      activeProjectId: "",
      workBoardOpenItemId: "",
      selectedPtyHistoryId: "",
    },
    work: {
      filterQuery: "",
      filterStageId: "",
      filterRunState: "",
      collapsedStageIdsByProject: {},
    },
    commandPalette: {
      recentCommandIds: [],
    },
    jumpPalette: {
      recentTargetIds: [],
    },
  };
}

export function parseClientViewState(raw: string | null): ClientViewStateV1 {
  return parseStoredClientViewState(raw) ?? defaultClientViewState();
}

export function serializeClientViewState(state: ClientViewStateV1): string {
  return JSON.stringify(sanitizeClientViewState(state));
}

export function persistClientViewState(storage: Pick<Storage, "setItem">, state: ClientViewStateV1) {
  storage.setItem(CLIENT_VIEW_STATE_KEY, serializeClientViewState(state));
}

export function hydrateClientViewState(args: HydrateClientViewStateArgs): ClientViewStateV1 {
  const raw = args.storage.getItem(CLIENT_VIEW_STATE_KEY);
  const persisted = parseStoredClientViewState(raw);
  const legacy = migrateLegacyLocalState(args.storage, { projects: args.projects });
  const merged = applyClientViewStatePatch(defaultClientViewState(), legacy);
  const hydrated = reconcileClientViewState(
    persisted ? mergeClientViewState(merged, persisted) : merged,
    args,
  );

  if (!persisted || serializeClientViewState(hydrated) !== raw) {
    try {
      persistClientViewState(args.storage, hydrated);
    } catch {
      return hydrated;
    }
  }

  return hydrated;
}

export function migrateLegacyLocalState(
  storage: ViewStateStorageReader,
  options: { projects?: ProjectReadModelLike[]; projectIds?: string[] } = {},
): PartialClientViewStateV1 {
  const migrated: PartialClientViewStateV1 = {};
  const legacyPreferences = parseLegacyPreferences(storage.getItem(LEGACY_UI_SETTINGS_KEY));
  if (legacyPreferences) migrated.preferences = legacyPreferences;

  const collapsedStageIdsByProject: Record<string, string[]> = {};
  for (const projectId of legacyCollapsedStageProjectIds(options)) {
    const collapsed = parseLegacyCollapsedStages(storage.getItem(legacyCollapsedStageStorageKey(projectId)));
    if (collapsed.length > 0) collapsedStageIdsByProject[projectId] = collapsed;
  }

  if (Object.keys(collapsedStageIdsByProject).length > 0) {
    migrated.work = { collapsedStageIdsByProject };
  }

  return migrated;
}

export function reconcileClientViewState(
  state: ClientViewStateV1,
  readModels: ReconcileClientViewStateArgs,
): ClientViewStateV1 {
  const reconciled = sanitizeClientViewState(state);

  if (Array.isArray(readModels.sessions)) {
    const sessions = readModels.sessions.map(sessionRecord).filter((session) => session.id);
    const selectedSession =
      sessions.find((session) => session.id === reconciled.selection.activeSessionId) ??
      sessions[0] ??
      null;

    if (selectedSession) {
      reconciled.selection.activeSessionId = selectedSession.id;
      reconciled.selection.activePaneId = selectedSession.paneIds.includes(reconciled.selection.activePaneId)
        ? reconciled.selection.activePaneId
        : selectedSession.paneIds[0] ?? "";
    } else {
      reconciled.selection.activeSessionId = "";
      reconciled.selection.activePaneId = "";
    }
  }

  const projects = Array.isArray(readModels.projects)
    ? readModels.projects.map(projectRecord).filter((project) => project.id)
    : null;
  if (projects) {
    const projectIds = new Set(projects.map((project) => project.id));
    if (!projectIds.has(reconciled.selection.activeProjectId)) {
      reconciled.selection.activeProjectId = "";
    }

    const stageIdsByProject = new Map(
      projects.map((project) => [project.id, new Set(project.stageIds)] as const),
    );
    const allStageIds = new Set(projects.flatMap((project) => project.stageIds));
    if (reconciled.work.filterStageId && !allStageIds.has(reconciled.work.filterStageId)) {
      reconciled.work.filterStageId = "";
    }

    const collapsedStageIdsByProject: Record<string, string[]> = {};
    for (const [projectId, collapsedStageIds] of Object.entries(
      reconciled.work.collapsedStageIdsByProject,
    )) {
      const allowedStageIds = stageIdsByProject.get(projectId);
      if (!allowedStageIds) continue;
      collapsedStageIdsByProject[projectId] = collapsedStageIds.filter((stageId) =>
        allowedStageIds.has(stageId),
      );
    }
    reconciled.work.collapsedStageIdsByProject = collapsedStageIdsByProject;
  }

  if (Array.isArray(readModels.ptyHistory)) {
    const ptyHistoryIds = new Set(
      readModels.ptyHistory
        .map((entry) => parsePtyId(entry.ptyId ?? entry.id))
        .filter((id) => id !== null)
        .map((id) => idString(id)),
    );
    if (!ptyHistoryIds.has(reconciled.selection.selectedPtyHistoryId)) {
      reconciled.selection.selectedPtyHistoryId = "";
    }
  }

  if (Array.isArray(readModels.workItems)) {
    const workItems = readModels.workItems
      .map((item) => ({
        id: parseWorkItemId(item.id),
        projectId: parseProjectId(item.projectId),
      }))
      .filter((item) => item.id);
    const selectedWorkItem = workItems.find(
      (item) => item.id && idString(item.id) === reconciled.selection.workBoardOpenItemId,
    );
    if (
      !selectedWorkItem ||
      (projects &&
        reconciled.selection.activeProjectId &&
        selectedWorkItem.projectId &&
        idString(selectedWorkItem.projectId) !== reconciled.selection.activeProjectId)
    ) {
      reconciled.selection.workBoardOpenItemId = "";
    }
  }

  return reconciled;
}

export function legacyCollapsedStageStorageKey(projectId: string) {
  return `whisk.workBoard.collapsedStages.${projectId}`;
}

function parseStoredClientViewState(raw: string | null): ClientViewStateV1 | null {
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw);
    if (!isRecord(parsed) || parsed.version !== CLIENT_VIEW_STATE_VERSION) return null;
    return sanitizeClientViewState(parsed);
  } catch {
    return null;
  }
}

function sanitizeClientViewState(value: unknown): ClientViewStateV1 {
  const defaults = defaultClientViewState();
  const record = isRecord(value) ? value : {};
  return {
    version: CLIENT_VIEW_STATE_VERSION,
    preferences: sanitizePreferences(record.preferences, defaults.preferences),
    selection: sanitizeSelection(record.selection, defaults.selection),
    work: sanitizeWork(record.work, defaults.work),
    commandPalette: sanitizeCommandPalette(record.commandPalette, defaults.commandPalette),
    jumpPalette: sanitizeJumpPalette(record.jumpPalette, defaults.jumpPalette),
  };
}

function sanitizePreferences(
  value: unknown,
  defaults: ClientViewStateV1["preferences"],
): ClientViewStateV1["preferences"] {
  const record = isRecord(value) ? value : {};
  return {
    railSide: record.railSide === "left" || record.railSide === "right" ? record.railSide : defaults.railSide,
    sidebarWidthPx: clampNumber(record.sidebarWidthPx, {
      min: MIN_SIDEBAR_WIDTH_PX,
      max: MAX_SIDEBAR_WIDTH_PX,
      fallback: defaults.sidebarWidthPx,
      round: true,
    }),
    terminalFontSize: clampNumber(record.terminalFontSize, {
      min: MIN_TERMINAL_FONT_SIZE,
      max: MAX_TERMINAL_FONT_SIZE,
      fallback: defaults.terminalFontSize,
    }),
    terminalCursorBlink:
      typeof record.terminalCursorBlink === "boolean"
        ? record.terminalCursorBlink
        : defaults.terminalCursorBlink,
    closePanePromptDisabled:
      typeof record.closePanePromptDisabled === "boolean"
        ? record.closePanePromptDisabled
        : defaults.closePanePromptDisabled,
  };
}

function sanitizeSelection(
  value: unknown,
  defaults: ClientViewStateV1["selection"],
): ClientViewStateV1["selection"] {
  const record = isRecord(value) ? value : {};
  return {
    activeMain: isMainView(record.activeMain) ? record.activeMain : defaults.activeMain,
    activeSidebar: isSidebarId(record.activeSidebar) || record.activeSidebar === null
      ? record.activeSidebar
      : defaults.activeSidebar,
    activeSessionId: parseOptionalIdString(record.activeSessionId, parseSessionId),
    activePaneId: parseOptionalIdString(record.activePaneId, parsePaneId),
    activeProjectId: parseOptionalIdString(record.activeProjectId, parseProjectId),
    workBoardOpenItemId: parseOptionalIdString(record.workBoardOpenItemId, parseWorkItemId),
    selectedPtyHistoryId: parseOptionalIdString(record.selectedPtyHistoryId, parsePtyId),
  };
}

function sanitizeWork(value: unknown, defaults: ClientViewStateV1["work"]): ClientViewStateV1["work"] {
  const record = isRecord(value) ? value : {};
  return {
    filterQuery: stringValue(record.filterQuery, defaults.filterQuery),
    filterStageId: stringValue(record.filterStageId, defaults.filterStageId),
    filterRunState: stringValue(record.filterRunState, defaults.filterRunState),
    collapsedStageIdsByProject: sanitizeCollapsedStageIdsByProject(
      record.collapsedStageIdsByProject,
    ),
  };
}

function sanitizeCommandPalette(
  value: unknown,
  defaults: ClientViewStateV1["commandPalette"],
): ClientViewStateV1["commandPalette"] {
  const record = isRecord(value) ? value : {};
  return {
    recentCommandIds: Array.isArray(record.recentCommandIds)
      ? sanitizeOrderedStringList(record.recentCommandIds).slice(0, MAX_RECENT_COMMANDS)
      : [...defaults.recentCommandIds],
  };
}

function sanitizeJumpPalette(
  value: unknown,
  defaults: ClientViewStateV1["jumpPalette"],
): ClientViewStateV1["jumpPalette"] {
  const record = isRecord(value) ? value : {};
  return {
    recentTargetIds: Array.isArray(record.recentTargetIds)
      ? sanitizeOrderedStringList(record.recentTargetIds).slice(0, MAX_RECENT_JUMP_TARGETS)
      : [...defaults.recentTargetIds],
  };
}

function parseLegacyPreferences(raw: string | null): Partial<ClientViewStateV1["preferences"]> | null {
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw);
    if (!isRecord(parsed)) return null;
    const preferences: Partial<ClientViewStateV1["preferences"]> = {};
    if (parsed.railSide === "left" || parsed.railSide === "right") preferences.railSide = parsed.railSide;
    if (typeof parsed.terminalFontSize === "number") {
      preferences.terminalFontSize = clampNumber(parsed.terminalFontSize, {
        min: MIN_TERMINAL_FONT_SIZE,
        max: MAX_TERMINAL_FONT_SIZE,
        fallback: defaultClientViewState().preferences.terminalFontSize,
      });
    }
    if (typeof parsed.terminalCursorBlink === "boolean") {
      preferences.terminalCursorBlink = parsed.terminalCursorBlink;
    }
    if (typeof parsed.closePanePromptDisabled === "boolean") {
      preferences.closePanePromptDisabled = parsed.closePanePromptDisabled;
    }
    return Object.keys(preferences).length > 0 ? preferences : null;
  } catch {
    return null;
  }
}

function parseLegacyCollapsedStages(raw: string | null): string[] {
  if (!raw) return [];
  try {
    return sanitizeSortedStringList(JSON.parse(raw));
  } catch {
    return [];
  }
}

function applyClientViewStatePatch(
  state: ClientViewStateV1,
  patch: PartialClientViewStateV1,
): ClientViewStateV1 {
  return sanitizeClientViewState({
    ...state,
    preferences: { ...state.preferences, ...patch.preferences },
    selection: { ...state.selection, ...patch.selection },
    work: {
      ...state.work,
      ...patch.work,
      collapsedStageIdsByProject: {
        ...state.work.collapsedStageIdsByProject,
        ...patch.work?.collapsedStageIdsByProject,
      },
    },
    commandPalette: { ...state.commandPalette, ...patch.commandPalette },
    jumpPalette: { ...state.jumpPalette, ...patch.jumpPalette },
  });
}

function mergeClientViewState(base: ClientViewStateV1, override: ClientViewStateV1): ClientViewStateV1 {
  return sanitizeClientViewState({
    ...base,
    preferences: { ...base.preferences, ...override.preferences },
    selection: { ...base.selection, ...override.selection },
    work: {
      ...base.work,
      ...override.work,
      collapsedStageIdsByProject: {
        ...base.work.collapsedStageIdsByProject,
        ...override.work.collapsedStageIdsByProject,
      },
    },
    commandPalette: { ...base.commandPalette, ...override.commandPalette },
    jumpPalette: { ...base.jumpPalette, ...override.jumpPalette },
  });
}

function legacyCollapsedStageProjectIds(options: {
  projects?: ProjectReadModelLike[];
  projectIds?: string[];
}) {
  const ids = [
    ...(options.projectIds ?? []),
    ...(options.projects ?? [])
      .map((project) => parseProjectId(project.id))
      .filter((id) => id !== null)
      .map((id) => idString(id)),
  ];
  return sanitizeOrderedStringList(ids);
}

function sessionRecord(session: SessionReadModelLike) {
  const id = parseOptionalIdString(session.id, parseSessionId);
  const paneIds = sanitizeOrderedStringList([
    ...Object.keys(session.panes ?? {}),
    ...Object.values(session.panes ?? {})
      .map((pane) => parsePaneId(pane?.id))
      .filter((paneId) => paneId !== null)
      .map((paneId) => idString(paneId)),
  ]);
  return { id, paneIds };
}

function projectRecord(project: ProjectReadModelLike) {
  const id = parseOptionalIdString(project.id, parseProjectId);
  const stageIds = sanitizeSortedStringList(project.workflow?.stages?.map((stage) => stage?.id ?? "") ?? []);
  return { id, stageIds };
}

function sanitizeCollapsedStageIdsByProject(value: unknown): Record<string, string[]> {
  if (!isRecord(value)) return {};
  const collapsedStageIdsByProject: Record<string, string[]> = {};
  for (const [rawProjectId, rawStageIds] of Object.entries(value)) {
    const projectId = parseProjectId(rawProjectId);
    if (!projectId || !Array.isArray(rawStageIds)) continue;
    const stageIds = sanitizeSortedStringList(rawStageIds);
    collapsedStageIdsByProject[idString(projectId)] = stageIds;
  }
  return collapsedStageIdsByProject;
}

function sanitizeOrderedStringList(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  const seen = new Set<string>();
  const result: string[] = [];
  for (const item of value) {
    const text = stringValue(item, "");
    if (!text || seen.has(text)) continue;
    seen.add(text);
    result.push(text);
  }
  return result;
}

function sanitizeSortedStringList(value: unknown): string[] {
  return sanitizeOrderedStringList(value).sort((a, b) => a.localeCompare(b));
}

function parseOptionalIdString<K extends string>(
  value: unknown,
  parse: (value: unknown) => (string & { readonly __brand?: K }) | null,
) {
  const parsed = parse(value);
  return parsed ? String(parsed) : "";
}

function stringValue(value: unknown, fallback: string) {
  return typeof value === "string" ? value.trim() : fallback;
}

function clampNumber(
  value: unknown,
  options: { min: number; max: number; fallback: number; round?: boolean },
) {
  if (typeof value !== "number" || !Number.isFinite(value)) return options.fallback;
  const normalized = options.round ? Math.round(value) : value;
  return Math.min(options.max, Math.max(options.min, normalized));
}

function isMainView(value: unknown): value is MainView {
  return value === "session" || value === "work" || value === "projects";
}

function isSidebarId(value: unknown): value is SidebarId {
  return (
    value === "sessions" ||
    value === "ptys" ||
    value === "work" ||
    value === "projects" ||
    value === "notifications"
  );
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}
