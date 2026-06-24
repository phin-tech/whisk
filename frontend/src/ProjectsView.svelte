<script lang="ts">
  import { Browser } from "@wailsio/runtime";
  import Ellipsis from "@lucide/svelte/icons/ellipsis";
  import ExternalLink from "@lucide/svelte/icons/external-link";
  import Folder from "@lucide/svelte/icons/folder";
  import Info from "@lucide/svelte/icons/info";
  import LayoutDashboard from "@lucide/svelte/icons/layout-dashboard";
  import ListChecks from "@lucide/svelte/icons/list-checks";
  import Paperclip from "@lucide/svelte/icons/paperclip";
  import Pencil from "@lucide/svelte/icons/pencil";
  import PlayCircle from "@lucide/svelte/icons/play-circle";
  import Plus from "@lucide/svelte/icons/plus";
  import Save from "@lucide/svelte/icons/save";
  import Terminal from "@lucide/svelte/icons/terminal";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import type { Session } from "../bindings/github.com/phin-tech/whisk/internal/domain/session/models";
  import type {
    Project,
    ProjectDetail,
    ProjectAttachmentTemplate,
    MetadataValue,
    WorkItem,
    WorkItemRun,
  } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import { externalAttachmentURL, openExternalURL } from "./externalLinks";
  import {
    buildProjectAttachmentUpdate,
    projectAttachmentEditValues,
    type ProjectAttachmentLike,
  } from "./projectAttachments";
  import {
    projectDetailCounts,
    selectedProjectDetail,
    sessionNameSuffix,
    sortRunsRecent,
  } from "./projectView";

  export let projects: Project[] = [];
  export let activeProjectId = "";
  export let detail: ProjectDetail | null = null;
  export let loading = false;
  export let onUpdateProject: (
    projectId: string,
    request: { name: string; description: string },
  ) => void;
  export let onDeleteProject: (projectId: string) => void;
  export let onNewSession: (projectId: string) => void;
  export let onOpenSession: (sessionId: string) => void;
  export let onRemoveSession: (sessionId: string) => void;
  export let onCreateWorkItem: (request: {
    projectId: string;
    title: string;
    bodyMarkdown: string;
  }) => void;
  export let onDeleteWorkItem: (workItemId: string) => void;
  export let onOpenWorkItem: (workItemId: string) => void;
  export let onOpenRunTerminal: (run: WorkItemRun) => void;
  export let onAddProjectAttachment: (request: {
    projectId: string;
    kind: string;
    title: string;
    path: string;
    url: string;
    note: string;
    provider: string;
    target: string;
    includeInContext: boolean;
    meta?: Record<string, MetadataValue>;
  }) => void;
  export let pluginAttachmentTemplates: (ProjectAttachmentTemplate & { pluginId: string })[] = [];
  export let onRunPluginProjectAttachmentTemplate: (request: {
    pluginId: string;
    templateId: string;
    projectId: string;
    values: Record<string, string>;
  }) => void;
  export let onUpdateProjectAttachment: (
    projectId: string,
    attachmentId: string,
    request: {
      title: string;
      path: string;
      url: string;
      note: string;
      provider: string;
      target: string;
      includeInContext: boolean;
      meta?: Record<string, MetadataValue>;
    },
  ) => void;
  export let onDeleteProjectAttachment: (projectId: string, attachmentId: string) => void;

  type ProjectTab = "overview" | "attachments" | "sessions" | "cards" | "runs";

  let editProjectId = "";
  let editName = "";
  let editDescription = "";
  let descOpen = false;
  let menuOpen = false;
  let newCardTitle = "";
  let newCardBody = "";
  let attachmentKind = "file";
  let attachmentTitle = "";
  let attachmentTarget = "";
  let attachmentNote = "";
  let attachmentProvider = "github";
  let attachmentInContext = true;
  let attachmentFormOpen = false;
  let attachmentEditId = "";
  let attachmentMode = "";
  let pluginFieldValues: Record<string, string> = {};
  let activeTab: ProjectTab = "overview";

  $: visibleDetail = selectedProjectDetail(projects, detail, activeProjectId);
  $: counts = projectDetailCounts(visibleDetail);
  $: workItems = (visibleDetail?.workItems ?? []) as WorkItem[];
  $: sessions = (visibleDetail?.sessions ?? []) as Session[];
  $: runs = (visibleDetail?.runs ?? []) as WorkItemRun[];
  $: attachments = visibleDetail?.project.attachments ?? [];
  $: selectedPluginTemplate = pluginAttachmentTemplates.find(
    (template) => `${template.pluginId}:${template.id}` === attachmentMode,
  );
  $: recentWorkItems = workItems.slice(0, 5);
  $: recentSessions = sessions.slice(0, 5);
  $: sortedRuns = sortRunsRecent(runs);
  $: recentRuns = sortedRuns.slice(0, 5);
  $: if (visibleDetail?.project.id !== editProjectId) {
    editProjectId = visibleDetail?.project.id ?? "";
    editName = visibleDetail?.project.name ?? "";
    editDescription = visibleDetail?.project.description ?? "";
    activeTab = "overview";
    descOpen = false;
  }
  $: projectDirty = Boolean(
    visibleDetail &&
      (editName.trim() !== visibleDetail.project.name ||
        editDescription.trim() !== (visibleDetail.project.description ?? "")),
  );
  $: canSaveProject = Boolean(visibleDetail && editName.trim() && projectDirty && !loading);

  function runLabel(run: WorkItemRun) {
    return run.promptTemplateId || run.preset || "run";
  }

  function workItemTitle(workItemId: string) {
    return workItems.find((item) => item.id === workItemId)?.title ?? workItemId;
  }

  function formattedTime(value: unknown) {
    if (!value) return "";
    if (value instanceof Date) return value.toLocaleString();
    const parsed = new Date(String(value));
    return Number.isNaN(parsed.getTime()) ? String(value) : parsed.toLocaleString();
  }

  function runStatusDot(status: string) {
    if (status === "running" || status === "awaiting_input") return "text-green";
    if (status === "queued") return "text-blue";
    if (status === "failed" || status === "cancelled") return "text-red";
    return "text-text-muted";
  }

  function saveProject() {
    if (!visibleDetail || !canSaveProject) return;
    onUpdateProject(visibleDetail.project.id, {
      name: editName.trim(),
      description: editDescription.trim(),
    });
  }

  function deleteProject() {
    if (!visibleDetail) return;
    menuOpen = false;
    if (window.confirm(`Delete project ${visibleDetail.project.name}? Sessions and files will be kept.`)) {
      onDeleteProject(visibleDetail.project.id);
    }
  }

  function createCard() {
    if (!visibleDetail || !newCardTitle.trim() || loading) return;
    onCreateWorkItem({
      projectId: visibleDetail.project.id,
      title: newCardTitle.trim(),
      bodyMarkdown: newCardBody.trim(),
    });
    newCardTitle = "";
    newCardBody = "";
  }

  function createAttachment() {
    if (!visibleDetail || loading) return;
    if (selectedPluginTemplate && !attachmentEditId) {
      onRunPluginProjectAttachmentTemplate({
        pluginId: selectedPluginTemplate.pluginId,
        templateId: selectedPluginTemplate.id,
        projectId: visibleDetail.project.id,
        values: pluginFieldValues,
      });
      closeAttachmentForm();
      return;
    }
    const payload = buildProjectAttachmentUpdate(visibleDetail.project.id, attachmentKind, {
      title: attachmentTitle,
      target: attachmentTarget,
      note: attachmentNote,
      provider: attachmentProvider,
      includeInContext: attachmentInContext,
    });
    if (!payload) return;
    if (attachmentEditId) {
      onUpdateProjectAttachment(visibleDetail.project.id, attachmentEditId, payload);
      closeAttachmentForm();
      return;
    }
    onAddProjectAttachment({ ...payload, kind: attachmentKind });
    attachmentTitle = "";
    attachmentTarget = "";
    attachmentNote = "";
    closeAttachmentForm();
  }

  function openAttachmentForm(mode: string) {
    attachmentMode = mode;
    attachmentFormOpen = true;
    attachmentEditId = "";
    pluginFieldValues = {};
    if (mode === "file" || mode === "url" || mode === "note" || mode === "external") {
      attachmentKind = mode;
    }
  }

  function openAttachmentEditor(attachment: ProjectAttachmentLike) {
    const values = projectAttachmentEditValues(attachment);
    attachmentKind = attachment.kind;
    attachmentTitle = values.title;
    attachmentTarget = values.target;
    attachmentNote = values.note;
    attachmentProvider = values.provider;
    attachmentInContext = values.includeInContext;
    attachmentEditId = attachment.id;
    attachmentMode = attachment.kind;
    attachmentFormOpen = true;
    pluginFieldValues = {};
  }

  function closeAttachmentForm() {
    attachmentFormOpen = false;
    attachmentEditId = "";
    attachmentMode = "";
    pluginFieldValues = {};
  }

  function pluginFieldValue(id: string) {
    return pluginFieldValues[id] ?? "";
  }

  function setPluginFieldValue(id: string, value: string) {
    pluginFieldValues = { ...pluginFieldValues, [id]: value };
  }

  function attachmentTargetLabel(kind: string) {
    if (kind === "external" && attachmentProvider === "github") return "GitHub issue URL";
    if (kind === "url") return "URL";
    if (kind === "external") return "Target";
    return "Path";
  }

  function attachmentSummary(attachment: { path?: string; url?: string; note?: string; target?: string }) {
    return attachment.path || attachment.url || attachment.target || attachment.note || "";
  }

  function openAttachmentURL(attachment: { url?: string }) {
    void openExternalURL(externalAttachmentURL(attachment), Browser.OpenURL);
  }

  function deleteCard(item: WorkItem) {
    if (loading) return;
    onDeleteWorkItem(item.id);
  }

  const tabs: { id: ProjectTab; label: string; count: () => number }[] = [
    { id: "overview", label: "Overview", count: () => 0 },
    { id: "attachments", label: "Attachments", count: () => attachments.length },
    { id: "sessions", label: "Sessions", count: () => counts.sessions },
    { id: "cards", label: "Cards", count: () => counts.workItems },
    { id: "runs", label: "Runs", count: () => counts.runs },
  ];
</script>

{#snippet emptyState(text: string)}
  <div class="px-3 py-3 text-[12px] text-text-muted">
    {text}
  </div>
{/snippet}

<div class="h-full min-h-0 bg-bg-deep">
  <section class="flex h-full min-w-0 flex-col">
    {#if visibleDetail}
      <!-- Compact header -->
      <div class="shrink-0 border-b border-hairline bg-bg-deep px-4 py-2.5">
        <div class="flex min-w-0 items-center gap-2">
          <Folder size={14} class="shrink-0 text-accent" />
          <input
            class="min-w-0 rounded border border-transparent bg-transparent px-1 text-[15px] font-medium text-text-primary outline-none transition-colors focus:border-accent-dim focus:bg-bg-deep"
            bind:value={editName}
            disabled={loading}
            aria-label="Project name"
            on:blur={saveProject}
          />
          {#if canSaveProject}
            <button
              type="button"
              class="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded border border-border-subtle bg-bg-surface/60 text-text-secondary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-50"
              disabled={!canSaveProject}
              aria-label="Save project"
              title="Save project"
              on:click={saveProject}
            >
              <Save size={12} />
            </button>
          {/if}
          <span class="min-w-0 flex-1 truncate font-mono text-[11px] text-text-muted">
            {visibleDetail.project.rootDir}
          </span>
          <button
            type="button"
            class="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded border border-transparent text-text-muted transition-colors hover:border-border-subtle hover:text-text-primary {descOpen ? 'border-border-subtle text-text-primary' : ''}"
            aria-label="Toggle description"
            title="Toggle description"
            on:click={() => (descOpen = !descOpen)}
          >
            <Info size={13} />
          </button>
          <div class="relative">
            <button
              type="button"
              class="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded border border-transparent text-text-muted transition-colors hover:border-border-subtle hover:text-text-primary"
              aria-label="Project actions"
              title="Project actions"
              on:click={() => (menuOpen = !menuOpen)}
            >
              <Ellipsis size={13} />
            </button>
            {#if menuOpen}
              <!-- svelte-ignore a11y-click-events-have-key-events a11y-no-static-element-interactions -->
              <div class="fixed inset-0 z-10" on:click={() => (menuOpen = false)}></div>
              <div class="absolute right-0 top-full z-20 mt-1 min-w-36 rounded border border-border-subtle bg-bg-surface py-1 shadow-lg">
                <button
                  type="button"
                  class="flex w-full items-center gap-2 px-3 py-1.5 text-left text-[12px] text-text-secondary transition-colors hover:bg-bg-surface/80 hover:text-text-primary disabled:cursor-not-allowed disabled:opacity-50"
                  disabled={loading}
                  on:click={() => { onNewSession(visibleDetail.project.id); menuOpen = false; }}
                >
                  <Terminal size={13} />
                  New session
                </button>
                <button
                  type="button"
                  class="flex w-full items-center gap-2 px-3 py-1.5 text-left text-[12px] text-text-secondary transition-colors hover:bg-bg-surface/80 hover:text-text-primary disabled:cursor-not-allowed disabled:opacity-50"
                  disabled={loading}
                  on:click={() => { activeTab = "cards"; menuOpen = false; }}
                >
                  <Plus size={13} />
                  New card
                </button>
                <div class="my-1 border-t border-hairline"></div>
                <button
                  type="button"
                  class="flex w-full items-center gap-2 px-3 py-1.5 text-left text-[12px] text-red transition-colors hover:bg-red/10 disabled:cursor-not-allowed disabled:opacity-50"
                  disabled={loading}
                  on:click={deleteProject}
                >
                  <Trash2 size={13} />
                  Delete
                </button>
              </div>
            {/if}
          </div>
        </div>
        <div class="mt-1 flex items-center gap-1.5 text-[11px] text-text-muted">
          <span class="font-mono">{counts.workItems}</span><span>items</span>
          <span class="opacity-40">·</span>
          <span class="font-mono">{counts.sessions}</span><span>sessions</span>
          <span class="opacity-40">·</span>
          <span class="font-mono">{counts.runs}</span><span>runs</span>
        </div>
        {#if descOpen}
          <div class="mt-2 border-t border-hairline pt-2">
            <textarea
              class="min-h-16 w-full resize-y rounded border border-border-subtle bg-bg-surface/35 px-3 py-2 text-[13px] text-text-primary outline-none transition-colors placeholder:text-text-muted focus:border-accent-dim"
              bind:value={editDescription}
              disabled={loading}
              placeholder="Project description"
              aria-label="Project description"
            ></textarea>
            {#if canSaveProject}
              <div class="mt-1.5 flex justify-end">
                <button
                  type="button"
                  class="inline-flex h-7 items-center gap-1 rounded border border-accent-dim bg-accent-dim px-2.5 text-[12px] font-semibold text-text-primary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-50"
                  disabled={!canSaveProject}
                  on:click={saveProject}
                >
                  <Save size={13} />
                  Save
                </button>
              </div>
            {/if}
          </div>
        {/if}
      </div>

      <!-- Tab bar -->
      <div class="shrink-0 border-b border-hairline px-5">
        <div class="flex min-h-10 items-end gap-1 overflow-x-auto">
          {#each tabs as tab (tab.id)}
            <button
              type="button"
              class="inline-flex h-10 items-center gap-1.5 border-b px-3 text-[12px] font-medium transition-colors {activeTab ===
              tab.id
                ? 'border-accent text-text-primary'
                : 'border-transparent text-text-muted hover:text-text-primary'}"
              on:click={() => (activeTab = tab.id)}
            >
              {#if tab.id === "overview"}
                <LayoutDashboard size={14} />
              {:else if tab.id === "attachments"}
                <Paperclip size={14} />
              {:else if tab.id === "sessions"}
                <Terminal size={14} />
              {:else if tab.id === "cards"}
                <ListChecks size={14} />
              {:else}
                <PlayCircle size={14} />
              {/if}
              <span>{tab.label}</span>
              {#if tab.count() > 0}
                <span class="font-mono text-[10px] text-text-muted">{tab.count()}</span>
              {/if}
            </button>
          {/each}
        </div>
      </div>

      <!-- Content -->
      <div class="app-scrollbar min-h-0 flex-1 overflow-y-auto p-5">
        {#if activeTab === "overview"}
          <div class="grid gap-5 xl:grid-cols-3">
            <section>
              <div class="mb-2 flex items-center justify-between gap-2">
                <button
                  type="button"
                  class="text-left text-[11px] font-semibold uppercase text-text-muted transition-colors hover:text-text-primary"
                  on:click={() => (activeTab = "sessions")}
                >
                  Sessions <span class="font-mono">{counts.sessions}</span>
                </button>
                <button
                  type="button"
                  class="inline-flex h-7 items-center gap-1 rounded border border-border-subtle bg-bg-surface/60 px-2 text-[11px] text-text-secondary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-50"
                  disabled={loading}
                  on:click={() => onNewSession(visibleDetail.project.id)}
                >
                  <Plus size={13} />
                  <span>Add</span>
                </button>
              </div>
              <div class="divide-y divide-hairline">
                {#if recentSessions.length === 0}
                  {@render emptyState("No sessions.")}
                {:else}
                  {#each recentSessions as session (session.id)}
                    <button
                      type="button"
                      class="w-full min-w-0 px-3 py-2 text-left transition-colors hover:bg-bg-surface/40"
                      on:click={() => onOpenSession(session.id)}
                    >
                      <div class="truncate text-[13px] font-medium text-text-primary">
                        {session.name}
                        {#if sessionNameSuffix(session, sessions)}
                          <span class="font-mono text-[10px] text-text-muted">#{sessionNameSuffix(session, sessions)}</span>
                        {/if}
                      </div>
                      <div class="truncate font-mono text-[10px] text-text-muted">{session.rootDir}</div>
                    </button>
                  {/each}
                  {#if counts.sessions > recentSessions.length}
                    <button
                      type="button"
                      class="w-full px-3 py-2 text-left text-[12px] text-accent transition-colors hover:text-accent/80"
                      on:click={() => (activeTab = "sessions")}
                    >
                      View all ({counts.sessions}) →
                    </button>
                  {/if}
                {/if}
              </div>
            </section>

            <section>
              <div class="mb-2 flex items-center justify-between gap-2">
                <button
                  type="button"
                  class="text-left text-[11px] font-semibold uppercase text-text-muted transition-colors hover:text-text-primary"
                  on:click={() => (activeTab = "cards")}
                >
                  Recent cards <span class="font-mono">{counts.workItems}</span>
                </button>
                <button
                  type="button"
                  class="inline-flex h-7 items-center gap-1 rounded border border-border-subtle bg-bg-surface/60 px-2 text-[11px] text-text-secondary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-50"
                  disabled={loading}
                  on:click={() => (activeTab = "cards")}
                >
                  <Plus size={13} />
                  <span>Add</span>
                </button>
              </div>
              <div class="divide-y divide-hairline">
                {#if recentWorkItems.length === 0}
                  {@render emptyState("No cards.")}
                {:else}
                  {#each recentWorkItems as item (item.id)}
                    <button
                      type="button"
                      class="grid w-full min-w-0 grid-cols-[56px_minmax(0,1fr)] gap-2 px-3 py-2 text-left transition-colors hover:bg-bg-surface/40"
                      on:click={() => onOpenWorkItem(item.id)}
                    >
                      <span class="font-mono text-[11px] text-text-muted">#{item.number}</span>
                      <span class="min-w-0">
                        <span class="block truncate text-[13px] font-medium text-text-primary">{item.title}</span>
                        <span class="block truncate text-[11px] text-text-muted">{item.stageId}</span>
                      </span>
                    </button>
                  {/each}
                  {#if counts.workItems > recentWorkItems.length}
                    <button
                      type="button"
                      class="w-full px-3 py-2 text-left text-[12px] text-accent transition-colors hover:text-accent/80"
                      on:click={() => (activeTab = "cards")}
                    >
                      View all ({counts.workItems}) →
                    </button>
                  {/if}
                {/if}
              </div>
            </section>

            <section>
              <div class="mb-2 flex items-center justify-between gap-2">
                <button
                  type="button"
                  class="text-left text-[11px] font-semibold uppercase text-text-muted transition-colors hover:text-text-primary"
                  on:click={() => (activeTab = "runs")}
                >
                  Latest runs <span class="font-mono">{counts.runs}</span>
                </button>
              </div>
              <div class="divide-y divide-hairline">
                {#if recentRuns.length === 0}
                  {@render emptyState("No runs.")}
                {:else}
                  {#each recentRuns as run (run.id)}
                    <button
                      type="button"
                      class="w-full min-w-0 px-3 py-2 text-left transition-colors hover:bg-bg-surface/40 disabled:cursor-not-allowed disabled:opacity-50"
                      disabled={!run.sessionId && !run.ptyId}
                      on:click={() => onOpenRunTerminal(run)}
                    >
                      <div class="flex min-w-0 items-center gap-2">
                        <div class="min-w-0 flex-1 truncate text-[13px] font-medium text-text-primary">{runLabel(run)}</div>
                        <span class="shrink-0 text-[11px] {runStatusDot(run.status)}">●</span>
                        <span class="shrink-0 text-[11px] text-text-muted">{run.status}</span>
                      </div>
                      <div class="truncate text-[11px] text-text-muted">{workItemTitle(run.workItemId)}</div>
                    </button>
                  {/each}
                  {#if counts.runs > recentRuns.length}
                    <button
                      type="button"
                      class="w-full px-3 py-2 text-left text-[12px] text-accent transition-colors hover:text-accent/80"
                      on:click={() => (activeTab = "runs")}
                    >
                      View all ({counts.runs}) →
                    </button>
                  {/if}
                {/if}
              </div>
            </section>
          </div>

        {:else if activeTab === "attachments"}
          <section>
            <div class="mb-2 text-[11px] font-semibold uppercase text-text-muted">Attachments</div>
            <div class="mb-2 border border-border-subtle bg-bg-surface/25 p-2">
              <div class="flex flex-wrap gap-1.5">
                {#each ["file", "url", "note"] as mode}
                  <button
                    type="button"
                    class="inline-flex h-8 items-center gap-1 rounded border border-border-subtle bg-bg-surface/60 px-2.5 text-[12px] text-text-secondary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-50"
                    disabled={loading}
                    on:click={() => openAttachmentForm(mode)}
                  >
                    <Plus size={14} />
                    <span>{mode}</span>
                  </button>
                {/each}
                {#each pluginAttachmentTemplates as template (`${template.pluginId}:${template.id}`)}
                  <button
                    type="button"
                    class="inline-flex h-8 items-center gap-1 rounded border border-border-subtle bg-bg-surface/60 px-2.5 text-[12px] text-text-secondary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-50"
                    disabled={loading}
                    on:click={() => openAttachmentForm(`${template.pluginId}:${template.id}`)}
                  >
                    <Plus size={14} />
                    <span>{template.label || template.id}</span>
                  </button>
                {/each}
              </div>

              {#if attachmentFormOpen}
                <div class="mt-2 grid gap-2 border-t border-hairline pt-2">
                  {#if selectedPluginTemplate && !attachmentEditId}
                    <div class="text-[12px] font-medium text-text-primary">
                      {selectedPluginTemplate.label || selectedPluginTemplate.id}
                    </div>
                    <div class="grid gap-2 md:grid-cols-2">
                      {#each selectedPluginTemplate.fields ?? [] as field (field.id)}
                        <input
                          class="h-8 min-w-0 rounded border border-border bg-bg-deep px-2 text-[12px] text-text-primary outline-none placeholder:text-text-muted focus:border-accent-dim"
                          type="text"
                          value={pluginFieldValue(field.id)}
                          placeholder={field.placeholder || field.label || field.id}
                          disabled={loading}
                          on:input={(event) => setPluginFieldValue(field.id, event.currentTarget.value)}
                        />
                      {/each}
                    </div>
                  {:else}
                    <div class="grid gap-2 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
                      <input
                        class="h-8 min-w-0 rounded border border-border bg-bg-deep px-2 text-[12px] text-text-primary outline-none placeholder:text-text-muted focus:border-accent-dim"
                        type="text"
                        bind:value={attachmentTitle}
                        placeholder="Title"
                        disabled={loading}
                      />
                      {#if attachmentKind === "note"}
                        <input
                          class="h-8 min-w-0 rounded border border-border bg-bg-deep px-2 text-[12px] text-text-primary outline-none placeholder:text-text-muted focus:border-accent-dim"
                          type="text"
                          bind:value={attachmentNote}
                          placeholder="Note"
                          disabled={loading}
                        />
                      {:else}
                        <input
                          class="h-8 min-w-0 rounded border border-border bg-bg-deep px-2 text-[12px] text-text-primary outline-none placeholder:text-text-muted focus:border-accent-dim"
                          type="text"
                          bind:value={attachmentTarget}
                          placeholder={attachmentTargetLabel(attachmentKind)}
                          disabled={loading}
                        />
                      {/if}
                    </div>
                  {/if}

                  <div class="flex flex-wrap items-center justify-between gap-2">
                    <label class="inline-flex items-center gap-2 text-[12px] text-text-secondary">
                      <input type="checkbox" bind:checked={attachmentInContext} disabled={loading || Boolean(selectedPluginTemplate)} />
                      <span>Use as agent context</span>
                    </label>
                    <div class="flex gap-1.5">
                      <button
                        type="button"
                        class="inline-flex h-8 items-center rounded border border-border-subtle bg-bg-surface/60 px-2.5 text-[12px] text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary"
                        on:click={closeAttachmentForm}
                      >
                        Cancel
                      </button>
                      <button
                        type="button"
                        class="inline-flex h-8 items-center gap-1 rounded border border-accent-dim bg-accent-dim px-2.5 text-[12px] font-semibold text-text-primary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-50"
                        disabled={loading}
                        on:click={createAttachment}
                      >
                        {#if attachmentEditId}
                          <Save size={14} />
                          <span>Save</span>
                        {:else}
                          <Plus size={14} />
                          <span>Add</span>
                        {/if}
                      </button>
                    </div>
                  </div>
                </div>
              {/if}
            </div>
            <div class="divide-y divide-hairline">
              {#if attachments.length === 0}
                {@render emptyState("No attachments.")}
              {:else}
                {#each attachments as attachment (attachment.id)}
                  <div class="grid grid-cols-[96px_minmax(0,1fr)_72px_32px_32px_32px] items-center gap-3 px-3 py-2 transition-colors hover:bg-bg-surface/40">
                    <div class="font-mono text-[11px] text-text-muted">{attachment.kind}</div>
                    <div class="min-w-0">
                      <div class="truncate text-[13px] font-medium text-text-primary">
                        {attachment.title || attachmentSummary(attachment)}
                      </div>
                      <div class="truncate font-mono text-[10px] text-text-muted">
                        {attachmentSummary(attachment)}
                      </div>
                    </div>
                    <div class="text-[11px] text-text-muted">
                      {attachment.includeInContext ? "context" : ""}
                    </div>
                    {#if externalAttachmentURL(attachment)}
                      <button
                        type="button"
                        class="inline-flex h-7 w-7 items-center justify-center rounded border border-transparent text-text-muted transition-colors hover:border-accent/40 hover:bg-accent-dim/10 hover:text-accent"
                        aria-label="Open attachment URL"
                        title="Open attachment URL"
                        on:click={() => openAttachmentURL(attachment)}
                      >
                        <ExternalLink size={13} />
                      </button>
                    {:else}
                      <span></span>
                    {/if}
                    <button
                      type="button"
                      class="inline-flex h-7 w-7 items-center justify-center rounded border border-transparent text-text-muted transition-colors hover:border-accent/40 hover:bg-accent-dim/10 hover:text-accent disabled:cursor-not-allowed disabled:opacity-50"
                      disabled={loading}
                      aria-label="Edit attachment"
                      title="Edit attachment"
                      on:click={() => openAttachmentEditor(attachment)}
                    >
                      <Pencil size={13} />
                    </button>
                    <button
                      type="button"
                      class="inline-flex h-7 w-7 items-center justify-center rounded border border-transparent text-text-muted transition-colors hover:border-red/40 hover:bg-red/10 hover:text-red disabled:cursor-not-allowed disabled:opacity-50"
                      disabled={loading}
                      aria-label="Delete attachment"
                      title="Delete attachment"
                      on:click={() => onDeleteProjectAttachment(visibleDetail.project.id, attachment.id)}
                    >
                      <Trash2 size={13} />
                    </button>
                  </div>
                {/each}
              {/if}
            </div>
          </section>

        {:else if activeTab === "cards"}
          <section>
            <div class="mb-2 text-[11px] font-semibold uppercase text-text-muted">Cards</div>
            <div class="mb-2 grid gap-2 border border-border-subtle bg-bg-surface/25 p-2">
              <input
                class="h-8 min-w-0 rounded border border-border bg-bg-deep px-2 text-[12px] text-text-primary outline-none transition-colors placeholder:text-text-muted focus:border-accent-dim"
                type="text"
                bind:value={newCardTitle}
                placeholder="Card title"
                disabled={loading}
                on:keydown={(event) => {
                  if (event.key === "Enter" && !event.shiftKey) {
                    event.preventDefault();
                    createCard();
                  }
                }}
              />
              <textarea
                class="min-h-16 resize-y rounded border border-border bg-bg-deep px-2 py-1.5 text-[12px] text-text-primary outline-none transition-colors placeholder:text-text-muted focus:border-accent-dim"
                bind:value={newCardBody}
                placeholder="Notes"
                disabled={loading}
              ></textarea>
              <div class="flex justify-end">
                <button
                  type="button"
                  class="inline-flex h-8 items-center gap-1 rounded border border-accent-dim bg-accent-dim px-2.5 text-[12px] font-semibold text-text-primary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-50"
                  disabled={loading || !newCardTitle.trim()}
                  on:click={createCard}
                >
                  <Plus size={14} />
                  <span>Add card</span>
                </button>
              </div>
            </div>
            <div class="divide-y divide-hairline">
              {#if workItems.length === 0}
                {@render emptyState("No cards.")}
              {:else}
                {#each workItems as item (item.id)}
                  <div
                    class="grid grid-cols-[72px_minmax(0,1fr)_120px_32px] items-center gap-3 px-3 py-2 transition-colors hover:bg-bg-surface/40 cursor-pointer"
                    role="button"
                    tabindex="0"
                    on:click={() => onOpenWorkItem(item.id)}
                    on:keydown={(e) => e.key === "Enter" && onOpenWorkItem(item.id)}
                  >
                    <div class="font-mono text-[11px] text-text-muted">#{item.number}</div>
                    <div class="min-w-0">
                      <div class="truncate text-[13px] font-medium text-text-primary">
                        {item.title}
                      </div>
                      {#if item.bodyMarkdown}
                        <div class="truncate text-[11px] text-text-muted">{item.bodyMarkdown}</div>
                      {/if}
                    </div>
                    <div class="truncate text-right text-[11px] text-text-muted">{item.stageId}</div>
                    <button
                      type="button"
                      class="inline-flex h-7 w-7 items-center justify-center rounded border border-transparent text-text-muted transition-colors hover:border-red/40 hover:bg-red/10 hover:text-red disabled:cursor-not-allowed disabled:opacity-50"
                      disabled={loading}
                      aria-label="Delete card"
                      title="Delete card"
                      on:click|stopPropagation={() => deleteCard(item)}
                    >
                      <Trash2 size={13} />
                    </button>
                  </div>
                {/each}
              {/if}
            </div>
          </section>

        {:else if activeTab === "sessions"}
          <section>
            <div class="mb-2 flex items-center justify-between gap-2">
              <div class="text-[11px] font-semibold uppercase text-text-muted">Sessions</div>
              <button
                type="button"
                class="inline-flex h-7 items-center gap-1 rounded border border-border-subtle bg-bg-surface/60 px-2 text-[11px] text-text-secondary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-50"
                disabled={loading}
                on:click={() => onNewSession(visibleDetail.project.id)}
              >
                <Plus size={13} />
                <span>Add</span>
              </button>
            </div>
            <div class="divide-y divide-hairline">
              {#if sessions.length === 0}
                {@render emptyState("No sessions.")}
              {:else}
                {#each sessions as session (session.id)}
                  {@const sessionRun = sortedRuns.find((r) => r.sessionId === session.id) ?? null}
                  {@const sessionItem = sessionRun ? workItems.find((w) => w.id === sessionRun.workItemId) ?? null : null}
                  <div class="flex min-w-0 items-center gap-3 px-3 py-2">
                    <div class="min-w-0 flex-1">
                      <div class="truncate text-[13px] font-medium text-text-primary">
                        {session.name}
                        {#if sessionNameSuffix(session, sessions)}
                          <span class="font-mono text-[10px] text-text-muted">#{sessionNameSuffix(session, sessions)}</span>
                        {/if}
                      </div>
                      {#if sessionItem}
                        <div class="mt-0.5 flex min-w-0 items-center gap-1.5 text-[11px] text-text-muted">
                          <span class="truncate">{sessionItem.title}</span>
                          {#if sessionItem.stageId}
                            <span class="opacity-40">·</span>
                            <span class="shrink-0">{sessionItem.stageId}</span>
                          {/if}
                        </div>
                      {:else}
                        <div class="truncate font-mono text-[10px] text-text-muted">{session.rootDir}</div>
                      {/if}
                    </div>
                    {#if sessionRun}
                      <div class="flex shrink-0 items-center gap-1 text-[11px]">
                        <span class="{runStatusDot(sessionRun.status)}">●</span>
                        <span class="text-text-muted">{sessionRun.status}</span>
                      </div>
                    {/if}
                    <button
                      type="button"
                      class="h-7 shrink-0 rounded border border-border-subtle bg-bg-surface/60 px-2 text-[11px] text-text-secondary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-50"
                      disabled={loading}
                      on:click={() => onOpenSession(session.id)}
                    >
                      Open
                    </button>
                    <button
                      type="button"
                      class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded border border-transparent text-text-muted transition-colors hover:border-red/40 hover:bg-red/10 hover:text-red disabled:cursor-not-allowed disabled:opacity-50"
                      disabled={loading}
                      aria-label="Remove session"
                      title="Remove session"
                      on:click={() => onRemoveSession(session.id)}
                    >
                      <Trash2 size={13} />
                    </button>
                  </div>
                {/each}
              {/if}
            </div>
          </section>

        {:else if activeTab === "runs"}
          <section>
            <div class="mb-2 text-[11px] font-semibold uppercase text-text-muted">Runs</div>
            <div class="divide-y divide-hairline">
              {#if runs.length === 0}
                {@render emptyState("No runs.")}
              {:else}
                {#each sortedRuns as run (run.id)}
                  <div class="grid grid-cols-[minmax(0,1fr)_auto_88px] items-center gap-3 px-3 py-2 transition-colors hover:bg-bg-surface/40">
                    <div class="min-w-0">
                      <div class="truncate text-[13px] font-medium text-text-primary">
                        {runLabel(run)}
                      </div>
                      <div class="truncate font-mono text-[10px] text-text-muted">
                        {run.id}
                      </div>
                      <div class="truncate text-[11px] text-text-muted">
                        {workItemTitle(run.workItemId)}
                      </div>
                      {#if formattedTime(run.updatedAt || run.createdAt)}
                        <div class="truncate font-mono text-[10px] text-text-muted">
                          {formattedTime(run.updatedAt || run.createdAt)}
                        </div>
                      {/if}
                    </div>
                    <div class="flex shrink-0 items-center gap-1.5 text-[11px]">
                      <span class="{runStatusDot(run.status)}">●</span>
                      <span class="text-text-muted">{run.status}</span>
                    </div>
                    <button
                      type="button"
                      class="h-7 rounded border border-border-subtle bg-bg-surface/60 px-2 text-[11px] text-text-secondary transition-colors hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-50"
                      disabled={!run.sessionId && !run.ptyId}
                      on:click={() => onOpenRunTerminal(run)}
                    >
                      Open
                    </button>
                  </div>
                {/each}
              {/if}
            </div>
          </section>
        {/if}
      </div>
    {:else}
      <div class="flex h-full items-center justify-center text-[13px] text-text-muted">
        No project selected.
      </div>
    {/if}
  </section>
</div>
