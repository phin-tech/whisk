import { describe, expect, it } from "vitest";
import appSource from "./App.svelte?raw";
import mainRouterSource from "./MainRouter.svelte?raw";
import boardSource from "./WorkBoard.svelte?raw";
import detailSource from "./WorkItemDetail.svelte?raw";
import projectSource from "./ProjectsView.svelte?raw";
import projectAttachmentsSource from "./projects/ProjectAttachments.svelte?raw";
import projectCardsSource from "./projects/ProjectCards.svelte?raw";
import projectOverviewSource from "./projects/ProjectOverview.svelte?raw";
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
    expect(boardSource).toContain("deriveWorkItemCardIndicators");
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

  it("does not show plan-required attention for done cards", () => {
    expect(boardSource).toContain('return value.includes("execution") || value.includes("review");');
    expect(boardSource).not.toContain('value.includes("done")');
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
