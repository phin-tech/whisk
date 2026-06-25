<script lang="ts">
  import Download from "@lucide/svelte/icons/download";
  import Plug from "@lucide/svelte/icons/plug";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import type { PluginStatus, RegistryPlugin } from "../../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import Button from "../ui/Button.svelte";
  import IconButton from "../ui/IconButton.svelte";
  import List from "../ui/List.svelte";
  import ListRow from "../ui/ListRow.svelte";
  import Switch from "../ui/Switch.svelte";

  type Props = {
    plugins: PluginStatus[];
    registryPlugins: RegistryPlugin[];
    installingPluginId: string;
    onRefreshPlugins: () => void;
    onSetPluginTrusted: (pluginId: string, trusted: boolean) => void;
    onRefreshRegistry: () => void;
    onInstallPlugin: (registry: string, pluginId: string) => void;
  };

  let {
    plugins,
    registryPlugins,
    installingPluginId,
    onRefreshPlugins,
    onSetPluginTrusted,
    onRefreshRegistry,
    onInstallPlugin,
  }: Props = $props();

  const registryGroups = $derived(
    Object.entries(
      registryPlugins.reduce<Record<string, RegistryPlugin[]>>((groups, plugin) => {
        (groups[plugin.registry] ??= []).push(plugin);
        return groups;
      }, {}),
    ),
  );

  function pluginStatusClass(plugin: PluginStatus) {
    if (!plugin.valid) return "border-red/30 bg-red/10 text-red";
    if (plugin.trusted) return "border-green/35 bg-green/10 text-green";
    return "border-border bg-bg-deep text-text-muted";
  }

  function pluginResolverLabels(plugin: PluginStatus) {
    return (plugin.resolvers ?? []).map((resolver) => resolver.provider).join(", ");
  }
</script>

<div class="flex items-center justify-between gap-3 pb-3">
  <div>
    <div class="text-[13px]">Plugins</div>
    <div class="mt-0.5 text-[11px] text-text-muted">
      Daemon-loaded plugins from configured plugin directories.
    </div>
  </div>
  <IconButton label="Rescan plugins" class="shrink-0" onclick={onRefreshPlugins}>
    <RefreshCw size={14} />
  </IconButton>
</div>

<List class="border-y border-hairline">
  {#if plugins.length === 0}
    <div class="py-3 text-[12px] text-text-muted">No plugins found.</div>
  {:else}
    {#each plugins as plugin (plugin.id)}
      <ListRow class="grid gap-3 py-3 md:grid-cols-[minmax(180px,240px)_1fr_auto] md:items-start">
        <div class="min-w-0">
          <div class="flex items-center gap-2">
            <Plug size={14} class="shrink-0 text-text-muted" />
            <span class="truncate text-[13px] font-medium text-text-primary">
              {plugin.name || plugin.id}
            </span>
          </div>
          <span class="mt-1 inline-flex rounded border px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider {pluginStatusClass(plugin)}">
            {plugin.valid ? (plugin.trusted ? "trusted" : "untrusted") : "invalid"}
          </span>
        </div>

        <div class="min-w-0 space-y-1 text-[11px] text-text-muted">
          <div class="truncate">
            ID <span class="font-mono text-text-secondary">{plugin.id}</span>
            {plugin.version ? ` · v${plugin.version}` : ""}
          </div>
          <div class="truncate">
            Path <span class="font-mono text-text-secondary">{plugin.dir}</span>
          </div>
          {#if pluginResolverLabels(plugin)}
            <div class="truncate">
              Resolvers <span class="font-mono text-text-secondary">{pluginResolverLabels(plugin)}</span>
            </div>
          {/if}
          {#if plugin.projectAttachmentTemplates?.length}
            <div class="truncate">
              Attachments
              <span class="text-text-secondary">
                {plugin.projectAttachmentTemplates.map((template) => template.label || template.id).join(", ")}
              </span>
            </div>
          {/if}
          {#if plugin.error}
            <div class="text-red">{plugin.error}</div>
          {/if}
        </div>

        <Switch
          label={`${plugin.trusted ? "Untrust" : "Trust"} ${plugin.name || plugin.id}`}
          checked={plugin.trusted}
          disabled={!plugin.valid}
          onCheckedChange={(trusted) => onSetPluginTrusted(plugin.id, trusted)}
        />
      </ListRow>
    {/each}
  {/if}
</List>

<div class="mt-6 flex items-center justify-between gap-3 pb-3">
  <div>
    <div class="text-[13px]">Available plugins</div>
    <div class="mt-0.5 text-[11px] text-text-muted">
      Installable from the configured plugin registries. Installed plugins start untrusted.
    </div>
  </div>
  <IconButton label="Refresh registries" class="shrink-0" onclick={onRefreshRegistry}>
    <RefreshCw size={14} />
  </IconButton>
</div>

{#if registryPlugins.length === 0}
  <div class="border-y border-hairline py-3 text-[12px] text-text-muted">No registry plugins available.</div>
{:else}
  {#each registryGroups as [registry, entries] (registry)}
    <div class="mb-2 mt-3 text-[11px] font-semibold uppercase tracking-wider text-text-muted">{registry}</div>
    <List class="border-y border-hairline">
      {#each entries as entry (entry.id)}
        <ListRow class="grid gap-3 py-3 md:grid-cols-[minmax(180px,240px)_1fr_auto] md:items-start">
          <div class="min-w-0">
            <div class="flex items-center gap-2">
              <Plug size={14} class="shrink-0 text-text-muted" />
              <span class="truncate text-[13px] font-medium text-text-primary">
                {entry.name || entry.id}
              </span>
            </div>
            <span class="mt-1 inline-flex rounded border px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider {entry.installed ? 'border-green/35 bg-green/10 text-green' : 'border-border bg-bg-deep text-text-muted'}">
              {entry.installed ? (entry.trusted ? "installed · trusted" : "installed") : "available"}
            </span>
          </div>

          <div class="min-w-0 space-y-1 text-[11px] text-text-muted">
            <div class="truncate">
              ID <span class="font-mono text-text-secondary">{entry.id}</span>
              {entry.sourceType ? ` · ${entry.sourceType}` : ""}
            </div>
            {#if entry.description}
              <div class="truncate text-text-secondary">{entry.description}</div>
            {/if}
          </div>

          <Button
            size="sm"
            disabled={entry.installed || installingPluginId === `${entry.registry}/${entry.id}`}
            onclick={() => onInstallPlugin(entry.registry, entry.id)}
          >
            <Download size={12} />
            {installingPluginId === `${entry.registry}/${entry.id}` ? "Installing…" : entry.installed ? "Installed" : "Install"}
          </Button>
        </ListRow>
      {/each}
    </List>
  {/each}
{/if}
