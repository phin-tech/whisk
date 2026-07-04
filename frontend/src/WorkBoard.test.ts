import { describe, expect, it } from "vitest";
import appSource from "./App.svelte?raw";
import mainRouterSource from "./MainRouter.svelte?raw";
import boardSource from "./WorkBoard.svelte?raw";
import boardStateSource from "./workboard/work-board-state.ts?raw";
import boardVirtualizationSource from "./workboard/work-board-virtualization.ts?raw";
import virtualWorkItemListSource from "./workboard/VirtualWorkItemList.svelte?raw";
import detailSource from "./WorkItemDetail.svelte?raw";
import projectSource from "./ProjectsView.svelte?raw";
import projectAttachmentsSource from "./projects/ProjectAttachments.svelte?raw";
import projectCardsSource from "./projects/ProjectCards.svelte?raw";
import projectOverviewSource from "./projects/ProjectOverview.svelte?raw";
import workflowPreviewDialogSource from "./projects/WorkflowPreviewDialog.svelte?raw";
import projectRunsSource from "./projects/ProjectRuns.svelte?raw";
import projectSessionsSource from "./projects/ProjectSessions.svelte?raw";
import listSource from "./ui/List.svelte?raw";
import cardIndicatorsSource from "./ui/CardIndicators.svelte?raw";
import boardColumnSource from "./workboard/WorkBoardColumn.svelte?raw";
import itemCardSource from "./workboard/WorkItemCard.svelte?raw";

describe("WorkBoard", () => {
  it("delegates the work item detail to the extracted component", () => {
    expect(boardSource).toContain("export let onUpdateWorkItem");
    expect(boardSource).toContain("function openDetail");
    expect(boardSource).toContain("import WorkItemDetail from");
    expect(boardSource).toContain("<WorkItemDetail");
    expect(boardSource).toContain("onClose={closeDetail}");
  });

  it("uses design-system tokens instead of raw surface color utilities", () => {
    const rawColorUtilities = Array.from(
      boardSource.matchAll(/\b(?:bg|text|border|shadow|ring)-\[#(?:[0-9a-fA-F]{3,8}|[^\]]+)\]/g),
      ([match]) => match,
    );
    const whiteBlackUtilities = Array.from(
      boardSource.matchAll(/\b(?:bg|border|text|divide)-(?:white|black)\/[0-9]+/g),
      ([match]) => match,
    );

    expect(rawColorUtilities).toEqual([]);
    expect(whiteBlackUtilities).toEqual([]);
  });

  it("uses explicit design-system type sizes instead of the default Tailwind scale", () => {
    const defaultTypeScale = Array.from(
      boardSource.matchAll(/\btext-(?:sm|base|lg|xl|2xl)\b/g),
      ([match]) => match,
    );

    expect(defaultTypeScale).toEqual([]);
  });

  it("renders board card status as dots and muted labels instead of filled badges", () => {
    expect(itemCardSource).toContain('from "../ui/StatusDot.svelte"');
    expect(detailSource).toContain('from "./ui/StatusDot.svelte"');
    expect(projectOverviewSource).toContain('from "../ui/StatusDot.svelte"');
    expect(projectRunsSource).toContain('from "../ui/StatusDot.svelte"');
    expect(projectSessionsSource).toContain('from "../ui/StatusDot.svelte"');
    expect(itemCardSource).toContain("●");
    expect(itemCardSource).toContain("text-text-muted");
    expect(boardSource).not.toContain("function runStatusDot");
    expect(detailSource).not.toContain("function runStatusDot");
    expect(projectSource).not.toContain("function runStatusDot");
    expect(boardSource).not.toContain("function attentionToneClass");
    expect(boardSource).not.toContain("rounded border px-1.5 py-0.5 text-[12px] font-medium");
  });

  it("renders workflow progress indicators on board and project cards", () => {
    expect(cardIndicatorsSource).toContain("text-green");
    expect(cardIndicatorsSource).toContain("text-blue");
    expect(cardIndicatorsSource).toContain("text-amber");
    expect(boardStateSource).toContain("deriveWorkItemCardIndicators");
    expect(itemCardSource).toContain('from "../ui/CardIndicators.svelte"');
    expect(itemCardSource).toContain("<CardIndicators");
    expect(projectCardsSource).toContain("deriveWorkItemCardIndicators");
    expect(projectCardsSource).toContain('from "../ui/CardIndicators.svelte"');
    expect(projectCardsSource).toContain("<CardIndicators");
    expect(projectSource).toContain("{artifacts}");
    expect(projectSource).toContain("{gateReports}");
    expect(mainRouterSource).toContain("{artifacts}");
    expect(mainRouterSource).toContain("{gateReports}");
  });

  it("loads workflow definitions from Wails and displays active project workflow identity", () => {
    expect(appSource).toContain("WorkflowDefinitionRecord");
    expect(appSource).toContain("WorkflowActionAvailability");
    expect(appSource).toContain("WorkflowMigrationPlan");
    expect(appSource).toContain("WorkflowValidationReport");
    expect(appSource).toContain("ListWorkflowDefinitions");
    expect(appSource).toContain("ListWorkItemWorkflowActions");
    expect(appSource).toContain("PlanProjectWorkflowMigration");
    expect(appSource).toContain("ValidateWorkflowDefinitionFile");
    expect(appSource).toContain("ImportWorkflowDefinitionFile");
    expect(appSource).toContain("ExportWorkflowDefinitionFile");
    expect(appSource).toContain("DeleteWorkflowDefinition");
    expect(appSource).toContain("SetProjectWorkflowDefinition");
    expect(appSource).toContain("let workflowDefinitions");
    expect(appSource).toContain("let workflowActionsByItem");
    expect(appSource).toContain("let workflowMigrationPlan");
    expect(appSource).toContain("let workflowValidationReport");
    expect(appSource).toContain("async function refreshWorkflowDefinitions");
    expect(appSource).toContain("async function refreshWorkItemWorkflowActions");
    expect(appSource).toContain("async function setProjectWorkflowDefinition");
    expect(appSource).toContain("async function planProjectWorkflowMigration");
    expect(appSource).toContain("async function validateWorkflowFile");
    expect(appSource).toContain("async function importWorkflowFile");
    expect(appSource).toContain("async function exportWorkflowFile");
    expect(appSource).toContain("async function deleteWorkflowDefinition");
    expect(appSource).toContain("workflowDefinitions = await ListWorkflowDefinitions()");
    expect(appSource).toContain("const updatedProject = await SetProjectWorkflowDefinition(projectId, { id, version })");
    expect(appSource).toContain("{workflowDefinitions}");
    expect(appSource).toContain("{workflowActionsByItem}");
    expect(appSource).toContain("{workflowMigrationPlan}");
    expect(appSource).toContain("{workflowValidationReport}");
    expect(mainRouterSource).toContain("export let workflowDefinitions");
    expect(mainRouterSource).toContain("export let workflowActionsByItem");
    expect(mainRouterSource).toContain("export let workflowMigrationPlan");
    expect(mainRouterSource).toContain("export let workflowValidationReport");
    expect(mainRouterSource).toContain("onSetProjectWorkflowDefinition");
    expect(mainRouterSource).toContain("onPlanProjectWorkflowMigration");
    expect(mainRouterSource).toContain("onValidateWorkflowFile");
    expect(mainRouterSource).toContain("onImportWorkflowFile");
    expect(mainRouterSource).toContain("onExportWorkflowFile");
    expect(mainRouterSource).toContain("onDeleteWorkflowDefinition");
    expect(mainRouterSource).toContain("{onSetProjectWorkflowDefinition}");
    expect(mainRouterSource).toContain("workflowDefinitions={workflowDefinitions}");
    expect(mainRouterSource).toContain("{workflowActionsByItem}");
    expect(projectSource).toContain("export let workflowDefinitions");
    expect(projectSource).toContain("export let workflowMigrationPlan");
    expect(projectSource).toContain("export let workflowValidationReport");
    expect(projectSource).toContain("export let onSetProjectWorkflowDefinition");
    expect(projectSource).toContain("export let onPlanProjectWorkflowMigration");
    expect(projectSource).toContain("workflowDefinitions={workflowDefinitions}");
    expect(projectSource).toContain("{workflowMigrationPlan}");
    expect(projectSource).toContain("{workflowValidationReport}");
    expect(projectSource).toContain("onSetWorkflowDefinition={(id, version) => onSetProjectWorkflowDefinition");
    expect(projectSource).toContain("onPlanWorkflowMigration={(id, version) => onPlanProjectWorkflowMigration");
    expect(projectOverviewSource).toContain("export let workflowDefinitions");
    expect(projectOverviewSource).toContain("export let workflowMigrationPlan");
    expect(projectOverviewSource).toContain("export let workflowValidationReport");
    expect(projectOverviewSource).toContain("let pendingWorkflowDefinition");
    expect(projectOverviewSource).toContain("function applyPendingWorkflowDefinition");
    expect(projectOverviewSource).toContain("function exportSelectedWorkflowDefinition");
    expect(projectOverviewSource).toContain("function deleteSelectedWorkflowDefinition");
    expect(projectOverviewSource).toContain("Workflow file path");
    expect(projectOverviewSource).toContain("Workflow export path");
    expect(projectOverviewSource).toContain("Pending workflow");
    expect(projectOverviewSource).toContain("Validate");
    expect(projectOverviewSource).toContain("Import");
    expect(projectOverviewSource).toContain("Export");
    expect(projectOverviewSource).toContain("Delete");
    expect(projectOverviewSource).toContain("function workflowDefinitionLabel");
    expect(projectOverviewSource).toContain("let workflowPreviewOpen");
    expect(projectOverviewSource).toContain("<WorkflowPreviewDialog");
    expect(projectOverviewSource).toContain("onclick={() => (workflowPreviewOpen = true)}");
    expect(projectOverviewSource).toContain("Workflow");
    expect(projectOverviewSource).toContain("onSetWorkflowDefinition");
    expect(workflowPreviewDialogSource).toContain('from "@xyflow/svelte"');
    expect(workflowPreviewDialogSource).toContain('import "@xyflow/svelte/dist/style.css"');
    expect(workflowPreviewDialogSource).toContain("<SvelteFlow");
    expect(workflowPreviewDialogSource).toContain("<Background");
    expect(workflowPreviewDialogSource).toContain("<Controls");
    expect(workflowPreviewDialogSource).toContain("max-w-[min(96vw,1440px)]");
    expect(workflowPreviewDialogSource).toContain("h-[min(70vh,720px)]");
    expect(workflowPreviewDialogSource).toContain("workflow-preview-node");
    expect(workflowPreviewDialogSource).toContain(".workflow-preview :global(.workflow-preview-node)");
    expect(workflowPreviewDialogSource).toContain(".workflow-preview :global(.svelte-flow__edge-label)");
    expect(workflowPreviewDialogSource).toContain("workflowPreviewStageConfig");
    expect(workflowPreviewDialogSource).toContain("workflowPreviewActionConfig");
    expect(workflowPreviewDialogSource).toContain("workflowPreviewGateConfig");
    expect(workflowPreviewDialogSource).toContain("selectedWorkflowConfig");
    expect(workflowPreviewDialogSource).toContain("selectWorkflowNode");
    expect(workflowPreviewDialogSource).toContain("selectWorkflowEdge");
    expect(workflowPreviewDialogSource).toContain("selectWorkflowGate");
    expect(workflowPreviewDialogSource).toContain("formatConfigValue");
    expect(workflowPreviewDialogSource).toContain("selectedWorkflowConfigEntries");
    expect(workflowPreviewDialogSource).toContain("Raw config");
    expect(workflowPreviewDialogSource).toContain("onnodeclick={selectWorkflowNode}");
    expect(workflowPreviewDialogSource).toContain("onedgeclick={selectWorkflowEdge}");
    expect(workflowPreviewDialogSource).toContain("Stage config");
    expect(workflowPreviewDialogSource).toContain("Action config");
    expect(workflowPreviewDialogSource).toContain("Gate config");
    expect(workflowPreviewDialogSource).toContain("Legend");
    expect(workflowPreviewDialogSource).toContain("proOptions={{ hideAttribution: true }}");
    expect(workflowPreviewDialogSource).toContain("workflowPreviewNodes");
    expect(workflowPreviewDialogSource).toContain("workflowPreviewEdges");
    expect(boardSource).toContain("export let workflowDefinitions");
    expect(boardSource).toContain("export let workflowActionsByItem");
    expect(boardSource).toContain("workflowActions={workflowActionsByItem[detailItem.id] ?? []}");
    expect(boardSource).toContain("deriveWorkBoardView");
    expect(boardSource).toContain("{workflowLabel}");
    expect(boardStateSource).toContain("function workflowDefinitionLabel");
    expect(boardStateSource).toContain("workflow.definitionId");
    expect(boardStateSource).toContain("workflow.definitionVersion");
    expect(detailSource).toContain("export let workflowActions");
    expect(detailSource).toContain("function runWorkflowAction");
    expect(detailSource).toContain("function workflowActionReason");
    expect(detailSource).toContain("workflowActions as availability");
  });

  it("does not show plan-required attention for done cards", () => {
    expect(boardStateSource).toContain('return value.includes("execution") || value.includes("review");');
    expect(boardStateSource).not.toContain('value.includes("done")');
  });

  it("derives WorkBoard view models from the beside-component state module", () => {
    expect(boardSource).toContain('from "./workboard/work-board-state"');
    expect(boardSource).toContain("deriveWorkBoardView");
    expect(boardSource).toContain("stageViews");
    expect(boardSource).toContain("cards={stageView.cards}");
    expect(boardStateSource).toContain("export function deriveWorkBoardView");
    expect(boardStateSource).toContain("export type WorkBoardStageView");
    expect(boardStateSource).toContain("export type WorkBoardCardView");
    expect(boardSource).not.toContain("function filterWorkItems");
    expect(boardSource).not.toContain("function groupRunsByItem");
    expect(boardSource).not.toContain("function attentionFor");
  });

  it("keeps WorkBoard virtual window math in a focused helper", () => {
    expect(boardVirtualizationSource).toContain("export function deriveWorkBoardCardWindow");
    expect(boardVirtualizationSource).toContain("WorkBoardCardView");
    expect(boardVirtualizationSource).toContain("visibleStartIndex");
    expect(boardVirtualizationSource).not.toContain("@tanstack");
    expect(boardVirtualizationSource).not.toContain("localStorage");
  });

  it("wires WorkBoard columns through the local virtualized card list", () => {
    expect(boardSource).toContain('from "./workboard/VirtualWorkItemList.svelte"');
    expect(boardSource).toContain("<VirtualWorkItemList");
    expect(boardSource).toContain("cards={stageView.cards}");
    expect(virtualWorkItemListSource).toContain("deriveWorkBoardCardWindow");
    expect(virtualWorkItemListSource).toContain("const ROW_HEIGHT = 128");
    expect(virtualWorkItemListSource).toContain("data-work-board-virtual-list");
    expect(virtualWorkItemListSource).toContain("data-work-card-virtual-row");
    expect(virtualWorkItemListSource).toContain("overflow-hidden bg-bg-base");
    expect(virtualWorkItemListSource).toContain("{#each virtualWindow.cards as virtualCard (virtualCard.key)}");
    expect(virtualWorkItemListSource).toContain("data-work-card-key={virtualCard.key}");
    expect(itemCardSource).toContain("relative h-full min-h-[76px] overflow-hidden");
    expect(virtualWorkItemListSource).not.toContain("@tanstack");
    expect(virtualWorkItemListSource).not.toContain("localStorage");
  });

  it("uses hairline-separated rows instead of bordered item cards", () => {
    expect(listSource).toContain("divide-y divide-hairline");
    expect(boardColumnSource).toContain("<List");
    expect(boardSource).not.toContain("grid min-w-0 content-start gap-2 p-2");
    expect(boardSource).not.toContain("rounded-md border border-border-subtle bg-bg-surface/60 transition-colors");
  });

  it("adopts display primitives for badges and empty states", () => {
    expect(boardSource).toContain('from "./ui/EmptyState.svelte"');
    expect(projectSource).toContain('from "./ui/EmptyState.svelte"');
    expect(boardSource).toContain("<EmptyState");
    expect(projectSource).toContain("<EmptyState");
    expect(projectSource).not.toContain("{#snippet emptyState");
    expect(projectSource).not.toContain("@render emptyState");
    expect(detailSource).toContain('from "./ui/Badge.svelte"');
    expect(detailSource).toContain("<Badge");
    expect(detailSource).not.toContain("rounded border border-border-subtle bg-bg-surface/60 px-1.5 py-0.5 text-text-secondary");
  });

  it("uses SectionHeader for reusable section labels without moving property-row labels", () => {
    expect(detailSource).toContain('from "./ui/SectionHeader.svelte"');
    expect(detailSource).toContain('<SectionHeader title="Description"');
    expect(detailSource).toContain('<SectionHeader title="Plan"');
    expect(detailSource).toContain('<SectionHeader title="Dependencies"');
    expect(detailSource).toContain('<SectionHeader title="Activity"');
    expect(projectAttachmentsSource).toContain('from "../ui/SectionHeader.svelte"');
    expect(projectRunsSource).toContain('from "../ui/SectionHeader.svelte"');
    expect(projectAttachmentsSource).toContain('<SectionHeader title="Attachments"');
    expect(projectRunsSource).toContain('<SectionHeader title="Runs"');
    expect(detailSource).toContain('const subHeader = "text-[11px] font-semibold uppercase text-text-muted"');
  });

  it("uses local button primitives instead of WorkItemDetail button recipe constants", () => {
    expect(detailSource).toContain('from "./ui/Button.svelte"');
    expect(detailSource).toContain('from "./ui/IconButton.svelte"');
    expect(detailSource).not.toContain("const ghostIcon =");
    expect(detailSource).not.toContain("const primaryBtn =");
    expect(detailSource).not.toContain("const outlineBtn =");
    expect(detailSource).not.toContain("class={ghostIcon}");
    expect(detailSource).not.toContain("class={primaryBtn}");
    expect(detailSource).not.toContain("class={outlineBtn}");
  });

  it("uses ModalShell for the WorkItemDetail dialog boundary", () => {
    expect(detailSource).toContain('from "./ui/ModalShell.svelte"');
    expect(detailSource).toContain("<ModalShell");
    expect(detailSource).toContain('titleId="work-item-detail-title"');
    expect(detailSource).not.toContain('role="dialog"');
    expect(detailSource).not.toContain('aria-modal="true"');
    expect(detailSource).not.toContain('<svelte:window on:keydown={handleKey} />');
  });

  it("uses DetailLayout for the WorkItemDetail content and properties rail", () => {
    expect(detailSource).toContain('from "./ui/DetailLayout.svelte"');
    expect(detailSource).toContain("<DetailLayout");
    expect(detailSource).toContain("{#snippet main()}");
    expect(detailSource).toContain("{#snippet aside()}");
    expect(detailSource).not.toContain('xl:grid-cols-[minmax(0,1fr)_280px]');
    expect(detailSource).not.toContain('<aside class="min-w-0">');
  });

  it("threads work item links and ready explanations into the detail drawer", () => {
    expect(boardSource).toContain("export let workItemLinks");
    expect(boardSource).toContain("export let readyWork");
    expect(boardSource).toContain("export let onAddWorkItemLink");
    expect(boardSource).toContain("workItemLinks={workItemLinks}");
    expect(boardSource).toContain("readyWork={readyWork}");
    expect(boardSource).toContain("onAddWorkItemLink={onAddWorkItemLink}");
  });
});

describe("WorkItemDetail", () => {
  it("edits work item text with explicit save and cancel callbacks", () => {
    expect(detailSource).toContain("export let onUpdateWorkItem");
    expect(detailSource).toContain("function saveDetail");
    expect(detailSource).toContain("function resetDraft");
    expect(detailSource).toContain('aria-label="Work item title"');
    expect(detailSource).toContain('aria-label="Work item description"');
    expect(detailSource).toContain("onUpdateWorkItem({");
    expect(detailSource).toContain("Cancel");
    expect(detailSource).toContain("Save");
  });

  it("shows dependency controls and the daemon ready-work explanation", () => {
    expect(detailSource).toContain("export let workItemLinks");
    expect(detailSource).toContain("export let readyWork");
    expect(detailSource).toContain("export let onAddWorkItemLink");
    expect(detailSource).toContain("Blocked by");
    expect(detailSource).toContain("Add blocker");
    expect(detailSource).toContain("No blocking dependencies");
    expect(detailSource).toContain("Ready because");
    expect(detailSource).toContain("onAddWorkItemLink({");
    expect(detailSource).toContain('type: "blocks"');
  });
});

describe("App work item dependencies", () => {
  it("loads daemon links and ready work through Wails and passes them to the board", () => {
    expect(appSource).toContain("WorkItemLink");
    expect(appSource).toContain("ReadyWorkExplanation");
    expect(appSource).toContain("AddWorkItemLink");
    expect(appSource).toContain("ListWorkItemLinks");
    expect(appSource).toContain("ReadyWork");
    expect(appSource).toContain("let workItemLinks");
    expect(appSource).toContain("let readyWork");
    expect(appSource).toContain("{workItemLinks}");
    expect(appSource).toContain("{readyWork}");
    expect(appSource).toContain("onAddWorkItemLink={addWorkItemLink}");
    expect(mainRouterSource).toContain('from "./WorkBoard.svelte"');
    expect(mainRouterSource).toContain("{workItemLinks}");
    expect(mainRouterSource).toContain("{readyWork}");
    expect(mainRouterSource).toContain("{onAddWorkItemLink}");
  });
});
