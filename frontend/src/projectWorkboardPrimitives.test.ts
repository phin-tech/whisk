import { describe, expect, it } from "vitest";
import projectsViewSource from "./ProjectsView.svelte?raw";
import workBoardSource from "./WorkBoard.svelte?raw";

const svelteSources = import.meta.glob("./**/*.svelte", {
  eager: true,
  import: "default",
  query: "?raw",
});

const sourceFor = (path: string) => String(svelteSources[path] ?? "");

const projectFeaturePaths = [
  "./projects/ProjectOverview.svelte",
  "./projects/ProjectAttachments.svelte",
  "./projects/ProjectCards.svelte",
  "./projects/ProjectSessions.svelte",
  "./projects/ProjectRuns.svelte",
];

const workBoardFeaturePaths = [
  "./workboard/WorkBoardColumn.svelte",
  "./workboard/WorkItemCard.svelte",
];

function expectImports(source: string, imports: string[]) {
  for (const importPath of imports) {
    expect(source).toContain(`from "${importPath}"`);
  }
}

function expectNoNativeControls(source: string, path: string) {
  expect(source, path).not.toMatch(/<(button|input|textarea|select)\b/);
  expect(source, path).not.toMatch(/\son:[a-z]/);
}

describe("ProjectsView and WorkBoard primitive adoption", () => {
  it("provides List, ListRow, Tabs, and PanelHeader through the local UI layer", () => {
    expect(Object.keys(svelteSources)).toEqual(
      expect.arrayContaining([
        "./ui/List.svelte",
        "./ui/ListRow.svelte",
        "./ui/Tabs.svelte",
        "./ui/PanelHeader.svelte",
      ]),
    );

    expect(sourceFor("./ui/List.svelte")).toContain("divide-y divide-hairline");
    expect(sourceFor("./ui/List.svelte")).toContain("$props()");
    expect(sourceFor("./ui/List.svelte")).toContain("{@render");

    expect(sourceFor("./ui/ListRow.svelte")).toContain("hover:bg-bg-surface/40");
    expect(sourceFor("./ui/ListRow.svelte")).toContain("cols");
    expect(sourceFor("./ui/ListRow.svelte")).toContain("$props()");

    expect(sourceFor("./ui/Tabs.svelte")).toContain("border-b border-hairline");
    expect(sourceFor("./ui/Tabs.svelte")).toContain("$bindable");
    expect(sourceFor("./ui/Tabs.svelte")).toContain("count");

    expect(sourceFor("./ui/PanelHeader.svelte")).toContain("border-b border-hairline bg-bg-deep");
    expect(sourceFor("./ui/PanelHeader.svelte")).toContain("meta");
    expect(sourceFor("./ui/PanelHeader.svelte")).toContain("$props()");
  });

  it("keeps ProjectsView as a shell over tabs, list rows, panel header, and feature sections", () => {
    expectImports(projectsViewSource, [
      "./ui/Tabs.svelte",
      "./ui/PanelHeader.svelte",
      "./ui/Button.svelte",
      "./ui/IconButton.svelte",
      "./ui/TextArea.svelte",
      "./projects/ProjectOverview.svelte",
      "./projects/ProjectAttachments.svelte",
      "./projects/ProjectCards.svelte",
      "./projects/ProjectSessions.svelte",
      "./projects/ProjectRuns.svelte",
    ]);

    expect(projectsViewSource).not.toContain("<!-- Compact header -->");
    expect(projectsViewSource).not.toContain("<!-- Tab bar -->");
    expect(projectsViewSource).not.toContain('class="divide-y divide-hairline"');
    expect(projectsViewSource).not.toContain('class="w-full min-w-0 px-3 py-2');
    expectNoNativeControls(projectsViewSource, "ProjectsView.svelte");
  });

  it("keeps ProjectsView tab sections on local primitives instead of native controls", () => {
    for (const path of projectFeaturePaths) {
      const source = sourceFor(path);
      expect(source, path).not.toBe("");
      expect(source, path).toContain('from "../ui/List.svelte"');
      expect(source, path).toContain('from "../ui/ListRow.svelte"');
      expect(source, path).toMatch(/from "\.\.\/ui\/(?:Button|IconButton)\.svelte"/);
      expectNoNativeControls(source, path);
    }

    expect(sourceFor("./projects/ProjectAttachments.svelte")).toContain('from "../ui/TextField.svelte"');
    expect(sourceFor("./projects/ProjectCards.svelte")).toContain('from "../ui/TextField.svelte"');
    expect(sourceFor("./projects/ProjectCards.svelte")).toContain('from "../ui/TextArea.svelte"');
  });

  it("splits WorkBoard into column and card feature components using local UI primitives", () => {
    expectImports(workBoardSource, [
      "./ui/Button.svelte",
      "./ui/IconButton.svelte",
      "./ui/TextArea.svelte",
      "./ui/TextField.svelte",
      "./workboard/WorkBoardColumn.svelte",
      "./workboard/WorkItemCard.svelte",
    ]);

    expect(workBoardSource).not.toContain("<article");
    expect(workBoardSource).not.toContain("writing-vertical");
    expect(workBoardSource).not.toContain("work-card-title");
    expect(workBoardSource).not.toContain('class="min-w-0 flex-1 divide-y divide-hairline"');
    expectNoNativeControls(workBoardSource, "WorkBoard.svelte");
  });

  it("keeps extracted WorkBoard feature components on local primitives", () => {
    for (const path of workBoardFeaturePaths) {
      const source = sourceFor(path);
      expect(source, path).not.toBe("");
      expect(source, path).toMatch(/from "\.\.\/ui\/(?:Button|IconButton|List|ListRow|StatusDot)\.svelte"/);
      expectNoNativeControls(source, path);
    }

    expect(sourceFor("./workboard/WorkBoardColumn.svelte")).toContain('from "../ui/List.svelte"');
    expect(sourceFor("./workboard/WorkBoardColumn.svelte")).toContain('from "../ui/ListRow.svelte"');
    expect(sourceFor("./workboard/WorkItemCard.svelte")).toContain('from "../ui/IconButton.svelte"');
  });
});
