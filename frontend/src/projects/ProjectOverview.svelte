<script lang="ts">
  import Check from "@lucide/svelte/icons/check";
  import ChevronDown from "@lucide/svelte/icons/chevron-down";
  import Plus from "@lucide/svelte/icons/plus";
  import type { Session } from "../../bindings/github.com/phin-tech/whisk/internal/domain/session/models";
  import type {
    Project,
    WorkflowDefinitionRecord,
    WorkflowMigrationPlan,
    WorkflowValidationReport,
    WorkItem,
    WorkItemRun,
  } from "../../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import Button from "../ui/Button.svelte";
  import EmptyState from "../ui/EmptyState.svelte";
  import List from "../ui/List.svelte";
  import ListRow from "../ui/ListRow.svelte";
  import Menu from "../ui/Menu.svelte";
  import MenuItem from "../ui/MenuItem.svelte";
  import Popover from "../ui/Popover.svelte";
  import StatusDot from "../ui/StatusDot.svelte";
  import TextField from "../ui/TextField.svelte";
  import WorkflowPreviewDialog from "./WorkflowPreviewDialog.svelte";

  type Counts = {
    sessions: number;
    workItems: number;
    runs: number;
  };

  export let project: Project;
  export let counts: Counts;
  export let workflowDefinitions: WorkflowDefinitionRecord[] = [];
  export let workflowMigrationPlan: WorkflowMigrationPlan | null = null;
  export let workflowValidationReport: WorkflowValidationReport | null = null;
  export let recentSessions: Session[] = [];
  export let recentWorkItems: WorkItem[] = [];
  export let recentRuns: WorkItemRun[] = [];
  export let sessions: Session[] = [];
  export let loading = false;
  export let onSelectTab: (tab: "sessions" | "cards" | "runs") => void;
  export let onSetWorkflowDefinition: (id: string, version: number) => void;
  export let onPlanWorkflowMigration: (id: string, version: number) => void;
  export let onValidateWorkflowFile: (path: string) => void;
  export let onImportWorkflowFile: (path: string) => void;
  export let onExportWorkflowFile: (id: string, version: number, path: string) => void;
  export let onDeleteWorkflowDefinition: (id: string, version: number) => void;
  export let onNewSession: () => void;
  export let onOpenSession: (sessionId: string) => void;
  export let onOpenWorkItem: (workItemId: string) => void;
  export let onOpenRunTerminal: (run: WorkItemRun) => void;
  export let sessionNameSuffix: (session: Session, sessions: Session[]) => string;
  export let runLabel: (run: WorkItemRun) => string;
  export let workItemTitle: (workItemId: string) => string;

  let workflowMenuOpen = false;
  let workflowPreviewOpen = false;
  let pendingWorkflowDefinition: WorkflowDefinitionRecord | null = null;
  let pendingWorkflowProjectId = "";
  let workflowFilePath = "";
  let workflowExportPath = "";

  $: activeWorkflow = activeWorkflowDefinition();
  $: selectedWorkflowDefinition = pendingWorkflowDefinition ?? activeWorkflow;
  $: deleteWorkflowTarget =
    pendingWorkflowDefinition && !isActiveWorkflowDefinition(pendingWorkflowDefinition)
      ? pendingWorkflowDefinition
      : null;
  $: migrationPlanMatchesPending =
    Boolean(
      pendingWorkflowDefinition &&
        workflowMigrationPlan &&
        workflowMigrationPlan.targetId === pendingWorkflowDefinition.id &&
        workflowMigrationPlan.targetVersion === pendingWorkflowDefinition.version,
    );
  $: validationErrors = workflowValidationReport?.errors ?? [];
  $: if (project.id !== pendingWorkflowProjectId) {
    pendingWorkflowProjectId = project.id;
    pendingWorkflowDefinition = null;
    workflowFilePath = "";
    workflowExportPath = "";
  }

  function workflowDefinitionLabel(id = project.workflow.definitionId, version = project.workflow.definitionVersion ?? 0) {
    const definition = workflowDefinitions.find((candidate) => candidate.id === id && candidate.version === version);
    const identityId = definition?.id || id || project.workflow.templateId || project.workflow.id;
    return version > 0 ? `${identityId}@${version}` : identityId;
  }

  function activeWorkflowDefinition() {
    return workflowDefinitions.find(
      (definition) =>
        definition.id === project.workflow.definitionId &&
        definition.version === project.workflow.definitionVersion,
    ) ?? null;
  }

  function isActiveWorkflowDefinition(definition: WorkflowDefinitionRecord) {
    return definition.id === project.workflow.definitionId && definition.version === project.workflow.definitionVersion;
  }

  function setWorkflowDefinition(definition: WorkflowDefinitionRecord) {
    workflowMenuOpen = false;
    if (isActiveWorkflowDefinition(definition)) {
      pendingWorkflowDefinition = null;
      return;
    }
    pendingWorkflowDefinition = definition;
    onPlanWorkflowMigration(definition.id, definition.version);
  }

  function applyPendingWorkflowDefinition() {
    if (!pendingWorkflowDefinition || loading) return;
    onSetWorkflowDefinition(pendingWorkflowDefinition.id, pendingWorkflowDefinition.version);
    pendingWorkflowDefinition = null;
  }

  function exportSelectedWorkflowDefinition() {
    const target = selectedWorkflowDefinition;
    if (!target || !workflowExportPath.trim() || loading) return;
    onExportWorkflowFile(target.id, target.version, workflowExportPath.trim());
  }

  function deleteSelectedWorkflowDefinition() {
    if (!deleteWorkflowTarget || loading) return;
    if (window.confirm(`Delete workflow ${workflowDefinitionLabel(deleteWorkflowTarget.id, deleteWorkflowTarget.version)}?`)) {
      onDeleteWorkflowDefinition(deleteWorkflowTarget.id, deleteWorkflowTarget.version);
      pendingWorkflowDefinition = null;
    }
  }
</script>

<div class="grid gap-5 xl:grid-cols-3">
  <section class="xl:col-span-3">
    <div class="grid gap-4 border-b border-hairline pb-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="min-w-0">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          align="start"
          class="h-auto border-transparent bg-transparent px-0 py-0 text-left"
          disabled={!activeWorkflowDefinition()}
          onclick={() => (workflowPreviewOpen = true)}
        >
          <span class="min-w-0">
            <span class="block text-[11px] font-semibold uppercase text-text-muted">Workflow</span>
            <span class="block truncate font-mono text-[12px] text-text-secondary">{workflowDefinitionLabel()}</span>
          </span>
        </Button>
      </div>
      <Popover bind:open={workflowMenuOpen} align="end">
        {#snippet trigger({ props })}
          <Button
            type="button"
            variant="outline"
            size="sm"
            class="max-w-[280px]"
            disabled={loading || workflowDefinitions.length === 0}
            {...props}
          >
            <span class="truncate font-mono">{workflowDefinitionLabel()}</span>
            <ChevronDown size={13} />
          </Button>
        {/snippet}
        <Menu class="min-w-64">
          {#each workflowDefinitions as definition (`${definition.id}:${definition.version}`)}
            <MenuItem active={isActiveWorkflowDefinition(definition)} onclick={() => setWorkflowDefinition(definition)}>
              <span class="min-w-0 flex-1 truncate font-mono">{workflowDefinitionLabel(definition.id, definition.version)}</span>
              {#if isActiveWorkflowDefinition(definition)}
                <Check size={13} class="shrink-0" />
              {/if}
            </MenuItem>
          {/each}
        </Menu>
      </Popover>
      </div>

      {#if pendingWorkflowDefinition}
        <div
          class="grid gap-3 rounded border border-accent/30 bg-accent-dim/10 p-3"
          data-testid="workflow-migration-plan"
        >
          <div class="flex flex-wrap items-center justify-between gap-2">
            <div class="min-w-0">
              <div class="text-[11px] font-semibold uppercase text-text-muted">Pending workflow</div>
              <div class="truncate font-mono text-[12px] text-text-primary">
                {workflowDefinitionLabel(pendingWorkflowDefinition.id, pendingWorkflowDefinition.version)}
              </div>
            </div>
            <div class="flex gap-2">
              <Button type="button" size="sm" variant="ghost" disabled={loading} onclick={() => (pendingWorkflowDefinition = null)}>
                Cancel
              </Button>
              <Button type="button" size="sm" variant="primary" disabled={loading || !migrationPlanMatchesPending} onclick={applyPendingWorkflowDefinition}>
                Apply
              </Button>
            </div>
          </div>
          {#if migrationPlanMatchesPending && workflowMigrationPlan}
            <div class="grid gap-2 text-[12px] text-text-secondary md:grid-cols-4">
              <div class="rounded border border-border-subtle bg-bg-surface/40 px-2 py-1.5">
                <div class="font-mono text-[13px] text-text-primary">{workflowMigrationPlan.existingItems}</div>
                <div>items</div>
              </div>
              <div class="rounded border border-border-subtle bg-bg-surface/40 px-2 py-1.5">
                <div class="font-mono text-[13px] text-text-primary">{workflowMigrationPlan.itemsPinnedToCurrentVersion}</div>
                <div>pinned</div>
              </div>
              <div class="rounded border border-border-subtle bg-bg-surface/40 px-2 py-1.5">
                <div class="font-mono text-[13px] text-green">{workflowMigrationPlan.compatibleItems}</div>
                <div>compatible</div>
              </div>
              <div class="rounded border border-border-subtle bg-bg-surface/40 px-2 py-1.5">
                <div class="font-mono text-[13px] text-amber">{workflowMigrationPlan.incompatibleItems}</div>
                <div>incompatible</div>
              </div>
            </div>
            {#if workflowMigrationPlan.incompatibleItems > 0}
              <div class="grid gap-1 rounded border border-amber/30 bg-amber/10 p-2 text-[11px] text-text-secondary">
                {#each workflowMigrationPlan.items.filter((planItem) => !planItem.compatible).slice(0, 4) as planItem (planItem.workItemId)}
                  <div class="flex min-w-0 gap-2">
                    <span class="font-mono text-amber">{planItem.currentStageId}</span>
                    <span class="min-w-0 flex-1 truncate">{planItem.reason}</span>
                  </div>
                {/each}
              </div>
            {/if}
          {:else}
            <div class="text-[12px] text-text-muted">Loading migration plan.</div>
          {/if}
        </div>
      {/if}

      <div
        class="grid gap-3 rounded border border-border-subtle bg-bg-surface/25 p-3"
        data-testid="workflow-file-controls"
      >
        <div class="grid gap-2 md:grid-cols-[minmax(0,1fr)_auto_auto]">
          <TextField
            bind:value={workflowFilePath}
            disabled={loading}
            placeholder="/path/to/workflow.json"
            aria-label="Workflow file path"
            class="font-mono"
          />
          <Button type="button" size="sm" variant="outline" disabled={loading || !workflowFilePath.trim()} onclick={() => onValidateWorkflowFile(workflowFilePath)}>
            Validate
          </Button>
          <Button type="button" size="sm" variant="outline" disabled={loading || !workflowFilePath.trim()} onclick={() => onImportWorkflowFile(workflowFilePath)}>
            Import
          </Button>
        </div>

        <div class="grid gap-2 md:grid-cols-[minmax(0,1fr)_auto_auto]">
          <TextField
            bind:value={workflowExportPath}
            disabled={loading || !selectedWorkflowDefinition}
            placeholder="/path/to/export-workflow.json"
            aria-label="Workflow export path"
            class="font-mono"
          />
          <Button type="button" size="sm" variant="outline" disabled={loading || !selectedWorkflowDefinition || !workflowExportPath.trim()} onclick={exportSelectedWorkflowDefinition}>
            Export
          </Button>
          <Button type="button" size="sm" variant="danger-ghost" disabled={loading || !deleteWorkflowTarget} onclick={deleteSelectedWorkflowDefinition}>
            Delete
          </Button>
        </div>

        {#if workflowValidationReport}
          <div class="rounded border border-border-subtle bg-bg-deep/50 p-2 text-[12px]">
            <div class="flex flex-wrap items-center gap-2">
              <span class={workflowValidationReport.valid ? "font-semibold text-green" : "font-semibold text-red"}>
                {workflowValidationReport.valid ? "valid" : "invalid"}
              </span>
              {#if workflowValidationReport.identity}
                <span class="font-mono text-text-secondary">{workflowValidationReport.identity}</span>
              {/if}
            </div>
            {#if validationErrors.length > 0}
              <div class="mt-2 grid gap-1 text-[11px] text-text-secondary">
                {#each validationErrors as validationError}
                  <div class="grid grid-cols-[minmax(0,96px)_minmax(0,1fr)] gap-2">
                    <span class="truncate font-mono text-text-muted">{validationError.path || "definition"}</span>
                    <span class="min-w-0">{validationError.message}</span>
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        {/if}
      </div>
    </div>
  </section>

  <section>
    <div class="mb-2 flex items-center justify-between gap-2">
      <Button variant="ghost" size="sm" align="start" class="border-transparent bg-transparent px-0 uppercase text-text-muted" onclick={() => onSelectTab("sessions")}>
        Sessions <span class="font-mono">{counts.sessions}</span>
      </Button>
      <Button size="sm" disabled={loading} onclick={onNewSession}>
        <Plus size={13} />
        <span>Add</span>
      </Button>
    </div>
    <List>
      {#if recentSessions.length === 0}
        <EmptyState message="No sessions." />
      {:else}
        {#each recentSessions as session (session.id)}
          <ListRow as="button" onclick={() => onOpenSession(session.id)}>
            <div class="truncate text-[13px] font-medium text-text-primary">
              {session.name}
              {#if sessionNameSuffix(session, sessions)}
                <span class="font-mono text-[10px] text-text-muted">#{sessionNameSuffix(session, sessions)}</span>
              {/if}
            </div>
            <div class="truncate font-mono text-[10px] text-text-muted">{session.rootDir}</div>
          </ListRow>
        {/each}
        {#if counts.sessions > recentSessions.length}
          <ListRow as="button" class="text-[12px] text-accent hover:text-accent/80" onclick={() => onSelectTab("sessions")}>
            View all ({counts.sessions}) →
          </ListRow>
        {/if}
      {/if}
    </List>
  </section>

  <section>
    <div class="mb-2 flex items-center justify-between gap-2">
      <Button variant="ghost" size="sm" align="start" class="border-transparent bg-transparent px-0 uppercase text-text-muted" onclick={() => onSelectTab("cards")}>
        Recent cards <span class="font-mono">{counts.workItems}</span>
      </Button>
      <Button size="sm" disabled={loading} onclick={() => onSelectTab("cards")}>
        <Plus size={13} />
        <span>Add</span>
      </Button>
    </div>
    <List>
      {#if recentWorkItems.length === 0}
        <EmptyState message="No cards." />
      {:else}
        {#each recentWorkItems as item (item.id)}
          <ListRow as="button" cols="grid-cols-[56px_minmax(0,1fr)]" onclick={() => onOpenWorkItem(item.id)}>
            <span class="font-mono text-[11px] text-text-muted">#{item.number}</span>
            <span class="min-w-0">
              <span class="block truncate text-[13px] font-medium text-text-primary">{item.title}</span>
              <span class="block truncate text-[11px] text-text-muted">{item.stageId}</span>
            </span>
          </ListRow>
        {/each}
        {#if counts.workItems > recentWorkItems.length}
          <ListRow as="button" class="text-[12px] text-accent hover:text-accent/80" onclick={() => onSelectTab("cards")}>
            View all ({counts.workItems}) →
          </ListRow>
        {/if}
      {/if}
    </List>
  </section>

  <section>
    <div class="mb-2 flex items-center justify-between gap-2">
      <Button variant="ghost" size="sm" align="start" class="border-transparent bg-transparent px-0 uppercase text-text-muted" onclick={() => onSelectTab("runs")}>
        Latest runs <span class="font-mono">{counts.runs}</span>
      </Button>
    </div>
    <List>
      {#if recentRuns.length === 0}
        <EmptyState message="No runs." />
      {:else}
        {#each recentRuns as run (run.id)}
          <ListRow as="button" disabled={!run.sessionId && !run.ptyId} onclick={() => onOpenRunTerminal(run)}>
            <div class="flex min-w-0 items-center gap-2">
              <div class="min-w-0 flex-1 truncate text-[13px] font-medium text-text-primary">{runLabel(run)}</div>
              <StatusDot status={run.status} showLabel class="shrink-0 text-[11px]" />
            </div>
            <div class="truncate text-[11px] text-text-muted">{workItemTitle(run.workItemId)}</div>
          </ListRow>
        {/each}
        {#if counts.runs > recentRuns.length}
          <ListRow as="button" class="text-[12px] text-accent hover:text-accent/80" onclick={() => onSelectTab("runs")}>
            View all ({counts.runs}) →
          </ListRow>
        {/if}
      {/if}
    </List>
  </section>
</div>

<WorkflowPreviewDialog
  visible={workflowPreviewOpen}
  workflow={activeWorkflowDefinition()}
  onclose={() => (workflowPreviewOpen = false)}
/>
