import type {
  PluginUIScope,
  UIContributionScope,
  UIContributionsResponse,
} from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
import type { JumpTarget } from "./jumpFilter";

export type PluginUIContributionScopeInput = {
  activeProjectId?: string;
  openWorkItemId?: string;
  activeSessionId?: string;
  activePaneId?: string;
  activePtyId?: string;
};

export type PluginCommandDescriptor = {
  id: string;
  title: string;
  pluginId: string;
  pluginName: string;
  commandId: string;
  commandLabel: string;
  commandScope: PluginUIScope;
  contributionScope: UIContributionScope;
};

const scopeKeyOrder = [
  "projectId",
  "workItemId",
  "runId",
  "sessionId",
  "paneId",
  "ptyId",
  "gateReportId",
  "phase",
] as const satisfies readonly (keyof UIContributionScope)[];

export function derivePluginUIContributionScope(input: PluginUIContributionScopeInput): UIContributionScope {
  return compactScope({
    projectId: input.activeProjectId,
    workItemId: input.openWorkItemId,
    sessionId: input.activeSessionId,
    paneId: input.activePaneId,
    ptyId: input.activePtyId,
  });
}

export function pluginUIContributionScopeKey(scope: UIContributionScope): string {
  return scopeKeyOrder.map((key) => `${key}=${scopeValue(scope[key])}`).join("|");
}

export function derivePluginCommandDescriptors(
  contributions: UIContributionsResponse | null | undefined,
): PluginCommandDescriptor[] {
  const scope = compactScope(contributions?.scope ?? {});
  const descriptors: PluginCommandDescriptor[] = [];
  for (const plugin of contributions?.plugins ?? []) {
    if (!plugin.trusted || !plugin.enabled) continue;
    const pluginId = scopeValue(plugin.pluginId);
    if (!pluginId) continue;
    const pluginName = scopeValue(plugin.name) || pluginId;
    for (const command of plugin.commands ?? []) {
      const commandId = scopeValue(command.id);
      const commandLabel = scopeValue(command.label) || commandId;
      const commandScope = scopeValue(command.scope);
      if (!commandId || !commandLabel || !commandScope) continue;
      descriptors.push({
        id: `plugin-command:${pluginId}:${commandId}`,
        title: `${pluginName}: ${commandLabel}`,
        pluginId,
        pluginName,
        commandId,
        commandLabel,
        commandScope,
        contributionScope: scope,
      });
    }
  }
  return descriptors;
}

export function derivePluginCommandJumpTargets(
  contributions: UIContributionsResponse | null | undefined,
): JumpTarget[] {
  return derivePluginCommandDescriptors(contributions).map((descriptor) => ({
    id: descriptor.id,
    kind: "plugin-command",
    title: descriptor.title,
    subtitle: "Plugin command",
    detail: descriptor.commandScope,
    keywords: [
      descriptor.pluginId,
      descriptor.pluginName,
      descriptor.commandId,
      descriptor.commandLabel,
      descriptor.commandScope,
      ...scopeKeywords(descriptor.contributionScope),
    ],
    payload: {
      kind: "plugin-command",
      pluginId: descriptor.pluginId,
      commandId: descriptor.commandId,
    },
  }));
}

function scopeKeywords(scope: UIContributionScope): string[] {
  return scopeKeyOrder.map((key) => scopeValue(scope[key])).filter(Boolean);
}

function compactScope(scope: Partial<UIContributionScope>): UIContributionScope {
  const out: UIContributionScope = {};
  for (const key of scopeKeyOrder) {
    const value = scopeValue(scope[key]);
    if (value) out[key] = value;
  }
  return out;
}

function scopeValue(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}
