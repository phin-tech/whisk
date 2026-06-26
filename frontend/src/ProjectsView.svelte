<script lang="ts">
  import { Browser } from "@wailsio/runtime";
  import Ellipsis from "@lucide/svelte/icons/ellipsis";
  import Folder from "@lucide/svelte/icons/folder";
  import Info from "@lucide/svelte/icons/info";
  import LayoutDashboard from "@lucide/svelte/icons/layout-dashboard";
  import ListChecks from "@lucide/svelte/icons/list-checks";
  import Paperclip from "@lucide/svelte/icons/paperclip";
  import PlayCircle from "@lucide/svelte/icons/play-circle";
  import Plus from "@lucide/svelte/icons/plus";
  import Save from "@lucide/svelte/icons/save";
  import Terminal from "@lucide/svelte/icons/terminal";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import type { Session } from "../bindings/github.com/phin-tech/whisk/internal/domain/session/models";
  import type {
    Artifact,
    GateReport,
    MetadataValue,
    Project,
    ProjectAttachmentTemplate,
    ProjectDetail,
    WorkflowDefinitionRecord,
    WorkItem,
    WorkItemRun,
  } from "../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import ProjectAttachments from "./projects/ProjectAttachments.svelte";
  import ProjectCards from "./projects/ProjectCards.svelte";
  import ProjectOverview from "./projects/ProjectOverview.svelte";
  import ProjectRuns from "./projects/ProjectRuns.svelte";
  import ProjectSessions from "./projects/ProjectSessions.svelte";
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
  import Button from "./ui/Button.svelte";
  import EmptyState from "./ui/EmptyState.svelte";
  import IconButton from "./ui/IconButton.svelte";
  import Menu from "./ui/Menu.svelte";
  import MenuItem from "./ui/MenuItem.svelte";
  import PanelHeader from "./ui/PanelHeader.svelte";
  import Popover from "./ui/Popover.svelte";
  import Tabs from "./ui/Tabs.svelte";
  import TextArea from "./ui/TextArea.svelte";

  export let projects: Project[] = [];
  export let activeProjectId = "";
  export let detail: ProjectDetail | null = null;
  export let workflowDefinitions: WorkflowDefinitionRecord[] = [];
  export let artifacts: Artifact[] = [];
  export let gateReports: GateReport[] = [];
  export let loading = false;
  export let onUpdateProject: (
    projectId: string,
    request: { name: string; description: string },
  ) => void;
  export let onSetProjectWorkflowDefinition: (
    projectId: string,
    id: string,
    version: number,
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
  $: tabs = [
    { id: "overview", label: "Overview", count: 0, icon: LayoutDashboard },
    { id: "attachments", label: "Attachments", count: attachments.length, icon: Paperclip },
    { id: "sessions", label: "Sessions", count: counts.sessions, icon: Terminal },
    { id: "cards", label: "Cards", count: counts.workItems, icon: ListChecks },
    { id: "runs", label: "Runs", count: counts.runs, icon: PlayCircle },
  ];
  $: headerMeta = [
    { value: counts.workItems, label: "items" },
    { value: counts.sessions, label: "sessions" },
    { value: counts.runs, label: "runs" },
  ];

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

  function openAttachment(attachment: { url?: string }) {
    void openExternalURL(externalAttachmentURL(attachment), Browser.OpenURL);
  }

  function deleteAttachment(attachmentId: string) {
    if (!visibleDetail) return;
    onDeleteProjectAttachment(visibleDetail.project.id, attachmentId);
  }

  function deleteCard(item: WorkItem) {
    if (loading) return;
    onDeleteWorkItem(item.id);
  }

  function selectTab(tab: string) {
    if (tab === "overview" || tab === "attachments" || tab === "sessions" || tab === "cards" || tab === "runs") {
      activeTab = tab;
    }
  }
</script>

<div class="h-full min-h-0 bg-bg-deep">
  <section class="flex h-full min-w-0 flex-col">
    {#if visibleDetail}
      <PanelHeader
        bind:value={editName}
        meta={headerMeta}
        disabled={loading}
        ariaLabel="Project name"
        oncommit={saveProject}
      >
        {#snippet icon()}
          <Folder size={14} class="shrink-0 text-accent" />
        {/snippet}
        {#snippet actions()}
          <span class="min-w-0 flex-1 truncate font-mono text-[11px] text-text-muted">
            {visibleDetail.project.rootDir}
          </span>
          {#if canSaveProject}
            <IconButton label="Save project" disabled={!canSaveProject} onclick={saveProject}>
              <Save size={12} />
            </IconButton>
          {/if}
          <IconButton
            label="Toggle description"
            class={descOpen ? "border-border-subtle text-text-primary" : ""}
            onclick={() => (descOpen = !descOpen)}
          >
            <Info size={13} />
          </IconButton>
          <Popover bind:open={menuOpen} class="min-w-36">
            {#snippet trigger({ props })}
              <IconButton {...props} label="Project actions">
                <Ellipsis size={13} />
              </IconButton>
            {/snippet}
            <Menu class="min-w-36">
              <MenuItem disabled={loading} onclick={() => { onNewSession(visibleDetail.project.id); menuOpen = false; }}>
                <Terminal size={13} />
                New session
              </MenuItem>
              <MenuItem disabled={loading} onclick={() => { activeTab = "cards"; menuOpen = false; }}>
                <Plus size={13} />
                New card
              </MenuItem>
              <div class="my-1 border-t border-hairline"></div>
              <MenuItem tone="danger" disabled={loading} onclick={deleteProject}>
                <Trash2 size={13} />
                Delete
              </MenuItem>
            </Menu>
          </Popover>
        {/snippet}
        {#if descOpen}
          <div class="mt-2 border-t border-hairline pt-2">
            <TextArea
              bind:value={editDescription}
              disabled={loading}
              placeholder="Project description"
              aria-label="Project description"
              class="border-border-subtle bg-bg-surface/35 text-[13px]"
            />
            {#if canSaveProject}
              <div class="mt-1.5 flex justify-end">
                <Button variant="primary" size="sm" disabled={!canSaveProject} onclick={saveProject}>
                  <Save size={13} />
                  Save
                </Button>
              </div>
            {/if}
          </div>
        {/if}
      </PanelHeader>

      <Tabs bind:active={activeTab} {tabs} onchange={selectTab} />

      <div class="app-scrollbar min-h-0 flex-1 overflow-y-auto p-5">
        {#if activeTab === "overview"}
          <ProjectOverview
            project={visibleDetail.project}
            {counts}
            workflowDefinitions={workflowDefinitions}
            {recentSessions}
            {recentWorkItems}
            {recentRuns}
            {sessions}
            {loading}
            onSelectTab={selectTab}
            onSetWorkflowDefinition={(id, version) => onSetProjectWorkflowDefinition(visibleDetail.project.id, id, version)}
            onNewSession={() => onNewSession(visibleDetail.project.id)}
            {onOpenSession}
            {onOpenWorkItem}
            {onOpenRunTerminal}
            {sessionNameSuffix}
            {runLabel}
            {workItemTitle}
          />
        {:else if activeTab === "attachments"}
          <ProjectAttachments
            {attachments}
            {pluginAttachmentTemplates}
            {selectedPluginTemplate}
            {loading}
            bind:attachmentFormOpen
            bind:attachmentKind
            bind:attachmentTitle
            bind:attachmentTarget
            bind:attachmentNote
            bind:attachmentInContext
            bind:attachmentEditId
            {openAttachmentForm}
            {closeAttachmentForm}
            {createAttachment}
            openAttachmentEditor={(attachment) => openAttachmentEditor(attachment)}
            deleteAttachment={deleteAttachment}
            openAttachmentURL={openAttachment}
            {externalAttachmentURL}
            {attachmentSummary}
            {attachmentTargetLabel}
            {pluginFieldValue}
            {setPluginFieldValue}
          />
        {:else if activeTab === "cards"}
          <ProjectCards
            {workItems}
            runs={sortedRuns}
            {artifacts}
            {gateReports}
            bind:newCardTitle
            bind:newCardBody
            {loading}
            {createCard}
            {deleteCard}
            {onOpenWorkItem}
          />
        {:else if activeTab === "sessions"}
          <ProjectSessions
            {sessions}
            {sortedRuns}
            {workItems}
            {loading}
            onNewSession={() => onNewSession(visibleDetail.project.id)}
            {onOpenSession}
            {onRemoveSession}
            {sessionNameSuffix}
          />
        {:else if activeTab === "runs"}
          <ProjectRuns
            {sortedRuns}
            {runLabel}
            {workItemTitle}
            {formattedTime}
            {onOpenRunTerminal}
          />
        {/if}
      </div>
    {:else}
      <div class="flex h-full items-center justify-center text-[13px] text-text-muted">
        <EmptyState message="No project selected." />
      </div>
    {/if}
  </section>
</div>
