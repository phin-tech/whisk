export const MAX_JUMP_QUERY_BYTES = 2048;
export const DEFAULT_JUMP_RESULT_LIMIT = 50;

export type JumpTargetKind =
  | "session"
  | "pane"
  | "pty"
  | "project"
  | "work-item"
  | "work-item-run"
  | "plugin-command";

export type JumpTargetPayload =
  | { kind: "session"; sessionId: string; projectId?: string }
  | { kind: "pane"; sessionId: string; windowId?: string; paneId: string; ptyId?: string }
  | { kind: "pty"; ptyId: string; sessionId?: string; windowId?: string; paneId?: string }
  | { kind: "project"; projectId: string }
  | { kind: "work-item"; projectId: string; workItemId: string }
  | {
      kind: "work-item-run";
      projectId: string;
      workItemId: string;
      runId: string;
      sessionId?: string;
      ptyId?: string;
    }
  | { kind: "plugin-command"; pluginId: string; commandId: string };

export type JumpTarget = {
  id: string;
  kind: JumpTargetKind;
  title: string;
  subtitle?: string;
  detail?: string;
  keywords?: readonly string[];
  disabled?: boolean;
  current?: boolean;
  payload?: JumpTargetPayload;
};

export type PreparedJumpTarget = JumpTarget & {
  inputIndex: number;
  searchableText: string;
  normalized: {
    id: string;
    title: string;
    subtitle: string;
    detail: string;
    keywords: string[];
    searchableText: string;
  };
};

export const JUMP_TARGET_KIND_PRIORITY: Record<JumpTargetKind, number> = {
  session: 0,
  pane: 1,
  pty: 2,
  project: 3,
  "work-item": 4,
  "work-item-run": 5,
  "plugin-command": 6,
};

export function prepareJumpTargets(targets: readonly JumpTarget[]): PreparedJumpTarget[] {
  return targets.map((target, inputIndex) => {
    const normalized = {
      id: normalizeSearchValue(target.id),
      title: normalizeSearchValue(target.title),
      subtitle: normalizeSearchValue(target.subtitle),
      detail: normalizeSearchValue(target.detail),
      keywords: (target.keywords ?? []).map(normalizeSearchValue).filter(Boolean),
      searchableText: "",
    };
    const rawSearchableText = [
      target.id,
      target.title,
      target.subtitle,
      target.detail,
      ...(target.keywords ?? []),
    ]
      .filter((value): value is string => Boolean(value))
      .join(" ");
    normalized.searchableText = normalizeSearchValue(rawSearchableText);
    return {
      ...target,
      inputIndex,
      searchableText: normalized.searchableText,
      normalized,
    };
  });
}

export function rankJumpTargets(
  query: string,
  targets: readonly PreparedJumpTarget[],
  limit = DEFAULT_JUMP_RESULT_LIMIT,
): PreparedJumpTarget[] {
  if (limit <= 0) return [];
  if (queryByteLength(query) > MAX_JUMP_QUERY_BYTES) return [];

  const needle = normalizeSearchValue(query);
  const cappedLimit = Math.min(limit, DEFAULT_JUMP_RESULT_LIMIT);
  const enabledTargets = targets.filter((target) => target.disabled !== true);
  if (!needle) return enabledTargets.slice(0, cappedLimit);

  return enabledTargets
    .map((target) => ({ target, score: scoreTarget(target, needle) }))
    .filter((result) => result.score > 0)
    .sort((a, b) => {
      if (b.score !== a.score) return b.score - a.score;
      const kindDelta = JUMP_TARGET_KIND_PRIORITY[a.target.kind] - JUMP_TARGET_KIND_PRIORITY[b.target.kind];
      if (kindDelta !== 0) return kindDelta;
      return a.target.inputIndex - b.target.inputIndex;
    })
    .slice(0, cappedLimit)
    .map((result) => result.target);
}

function scoreTarget(target: PreparedJumpTarget, needle: string): number {
  const fields = target.normalized;
  let score = 0;

  score = Math.max(score, scoreIdentifier(fields.id, needle));
  score = Math.max(score, scoreTitle(fields.title, needle));

  for (const keyword of fields.keywords) {
    score = Math.max(score, scoreKeyword(keyword, needle));
  }

  score = Math.max(score, scoreSecondary(fields.subtitle, needle));
  score = Math.max(score, scoreSecondary(fields.detail, needle));

  const searchableIndex = fields.searchableText.indexOf(needle);
  if (searchableIndex >= 0) score = Math.max(score, 2500 - searchableIndex);

  score = Math.max(score, scoreFuzzy(fields.title, needle, 1800));
  score = Math.max(score, scoreFuzzy(fields.searchableText, needle, 1200));

  return score;
}

function scoreIdentifier(value: string, needle: string): number {
  if (!value) return 0;
  if (value === needle) return 10000;
  if (value.startsWith(needle)) return 7600 - needle.length;
  const index = value.indexOf(needle);
  return index >= 0 ? 5000 - index : 0;
}

function scoreTitle(value: string, needle: string): number {
  if (!value) return 0;
  if (value === needle) return 9000;
  if (value.startsWith(needle)) return 8500 - needle.length;
  const index = value.indexOf(needle);
  return index >= 0 ? 7200 - index : 0;
}

function scoreKeyword(value: string, needle: string): number {
  if (!value) return 0;
  if (value === needle) return isIdentifierKeyword(value) ? 9800 : 6800;
  if (value.startsWith(needle)) return 6200 - needle.length;
  const index = value.indexOf(needle);
  return index >= 0 ? 5200 - index : 0;
}

function scoreSecondary(value: string, needle: string): number {
  if (!value) return 0;
  if (value === needle) return 4700;
  if (value.startsWith(needle)) return 4200 - needle.length;
  const index = value.indexOf(needle);
  return index >= 0 ? 3300 - index : 0;
}

function scoreFuzzy(value: string, needle: string, baseScore: number): number {
  if (!value || !needle) return 0;
  let valueIndex = 0;
  let needleIndex = 0;
  let previousMatch = -1;
  let gapCost = 0;

  while (valueIndex < value.length && needleIndex < needle.length) {
    if (value[valueIndex] === needle[needleIndex]) {
      if (previousMatch >= 0) gapCost += valueIndex - previousMatch - 1;
      previousMatch = valueIndex;
      needleIndex += 1;
    }
    valueIndex += 1;
  }

  if (needleIndex !== needle.length) return 0;
  return Math.max(1, baseScore - gapCost - value.length);
}

function normalizeSearchValue(value: unknown): string {
  return String(value ?? "").trim().toLowerCase();
}

function queryByteLength(query: string): number {
  return new TextEncoder().encode(query).length;
}

function isIdentifierKeyword(value: string): boolean {
  if (value.startsWith("#")) return isDigits(value.slice(1));
  if (isDigits(value)) return true;
  return [
    "item_",
    "work-item:",
    "run_",
    "work-item-run:",
    "sess_",
    "session:",
    "pty_",
    "pty:",
    "proj_",
    "project:",
    "pane_",
    "pane:",
  ].some((prefix) => value.startsWith(prefix));
}

function isDigits(value: string): boolean {
  if (!value) return false;
  for (const char of value) {
    if (char < "0" || char > "9") return false;
  }
  return true;
}
