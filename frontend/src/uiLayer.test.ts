import { describe, expect, it } from "vitest";
import designSystemDoc from "../../agents/DESIGN-SYSTEM.md?raw";
import commandPaletteSource from "./CommandPalette.svelte?raw";
import confirmDialogSource from "./ConfirmDialog.svelte?raw";
import newProjectDialogSource from "./NewProjectDialog.svelte?raw";
import newSessionDialogSource from "./NewSessionDialog.svelte?raw";
import onboardingPanelSource from "./OnboardingPanel.svelte?raw";
import workItemDetailSource from "./WorkItemDetail.svelte?raw";

const svelteSources = import.meta.glob("./**/*.svelte", {
  eager: true,
  import: "default",
  query: "?raw",
});

const sourceEntries = Object.entries(svelteSources).map(([path, source]) => ({
  path,
  source: String(source),
}));
const uiEntries = sourceEntries.filter(({ path }) => path.startsWith("./ui/"));

describe("local UI layer", () => {
  it("provides the first Bits-backed primitives through frontend/src/ui", () => {
    expect(Object.keys(svelteSources)).toEqual(
      expect.arrayContaining([
        "./ui/Button.svelte",
        "./ui/IconButton.svelte",
        "./ui/ModalShell.svelte",
        "./ui/Popover.svelte",
        "./ui/Menu.svelte",
        "./ui/MenuItem.svelte",
        "./ui/SelectField.svelte",
        "./ui/Switch.svelte",
      ]),
    );
    expect(String(svelteSources["./ui/Button.svelte"])).toContain('from "bits-ui"');
    expect(String(svelteSources["./ui/ModalShell.svelte"])).toContain('from "bits-ui"');
    expect(String(svelteSources["./ui/SelectField.svelte"])).toContain('from "bits-ui"');
    expect(String(svelteSources["./ui/Switch.svelte"])).toContain('from "bits-ui"');
  });

  it("provides Tier-1 display primitives through frontend/src/ui", () => {
    expect(Object.keys(svelteSources)).toEqual(
      expect.arrayContaining([
        "./ui/StatusDot.svelte",
        "./ui/Badge.svelte",
        "./ui/SectionHeader.svelte",
        "./ui/EmptyState.svelte",
        "./ui/PropertyRow.svelte",
        "./ui/NextActionBar.svelte",
      ]),
    );
  });

  it("keeps new UI primitives on Svelte 5 runes", () => {
    expect(uiEntries.map(({ path }) => path)).toEqual(
      expect.arrayContaining([
        "./ui/Button.svelte",
        "./ui/IconButton.svelte",
        "./ui/Checkbox.svelte",
        "./ui/ModalShell.svelte",
        "./ui/Popover.svelte",
        "./ui/Menu.svelte",
        "./ui/MenuItem.svelte",
        "./ui/SelectField.svelte",
        "./ui/Switch.svelte",
        "./ui/TextArea.svelte",
        "./ui/TextField.svelte",
        "./ui/DetailLayout.svelte",
        "./ui/StatusDot.svelte",
        "./ui/Badge.svelte",
        "./ui/SectionHeader.svelte",
        "./ui/EmptyState.svelte",
      ]),
    );

    for (const { path, source } of uiEntries) {
      expect(source, path).toContain("$props()");
      expect(source, path).not.toMatch(/\bexport let\b/);
      expect(source, path).not.toMatch(/\$:/);
      expect(source, path).not.toContain("<slot");
      expect(source, path).not.toContain("$$restProps");
      expect(source, path).not.toMatch(/\son:[a-z]/);
    }

    expect(String(svelteSources["./ui/Button.svelte"])).toContain("$derived(");
    expect(String(svelteSources["./ui/IconButton.svelte"])).toContain("$derived(");
    expect(String(svelteSources["./ui/Checkbox.svelte"])).toContain("$bindable(false)");
    expect(String(svelteSources["./ui/Checkbox.svelte"])).toContain("bind:checked");
    expect(String(svelteSources["./ui/ModalShell.svelte"])).toContain("$state<");
    expect(String(svelteSources["./ui/ModalShell.svelte"])).toContain("heading?: Snippet");
    expect(String(svelteSources["./ui/Popover.svelte"])).toContain("Popover.Root");
    expect(String(svelteSources["./ui/MenuItem.svelte"])).toContain('from "./Button.svelte"');
    expect(String(svelteSources["./ui/SelectField.svelte"])).toContain("Select.Root");
    expect(String(svelteSources["./ui/StatusDot.svelte"])).toContain("showLabel");
    expect(String(svelteSources["./ui/StatusDot.svelte"])).toContain("status === \"running\"");
    expect(String(svelteSources["./ui/StatusDot.svelte"])).toContain("status === \"awaiting_input\"");
    expect(String(svelteSources["./ui/StatusDot.svelte"])).toContain("status === \"queued\"");
    expect(String(svelteSources["./ui/StatusDot.svelte"])).toContain("status === \"failed\"");
    expect(String(svelteSources["./ui/Switch.svelte"])).toContain("$bindable(false)");
    expect(String(svelteSources["./ui/TextArea.svelte"])).toContain('$bindable("")');
    expect(String(svelteSources["./ui/TextField.svelte"])).toContain('$bindable("")');
    expect(String(svelteSources["./ui/PropertyRow.svelte"])).toContain("label");
    expect(String(svelteSources["./ui/NextActionBar.svelte"])).toContain("NextStepView");
  });

  it("uses snippets where primitives expose child content", () => {
    for (const path of [
      "./ui/Button.svelte",
      "./ui/IconButton.svelte",
      "./ui/Checkbox.svelte",
      "./ui/Menu.svelte",
      "./ui/MenuItem.svelte",
      "./ui/ModalShell.svelte",
      "./ui/Popover.svelte",
      "./ui/DetailLayout.svelte",
      "./ui/SectionHeader.svelte",
      "./ui/EmptyState.svelte",
      "./ui/Badge.svelte",
      "./ui/PropertyRow.svelte",
      "./ui/NextActionBar.svelte",
    ]) {
      expect(String(svelteSources[path]), path).toContain("{@render");
    }
    expect(String(svelteSources["./ui/SelectField.svelte"])).toContain("{#snippet children");
  });

  it("keeps feature components from importing bits-ui directly", () => {
    const directFeatureImports = sourceEntries
      .filter(({ path }) => !path.startsWith("./ui/"))
      .filter(({ source }) => /from\s+["']bits-ui["']/.test(source))
      .map(({ path }) => path);

    expect(directFeatureImports).toEqual([]);
  });

  it("migrates ConfirmDialog onto the local UI layer", () => {
    expect(confirmDialogSource).toContain('from "./ui/ModalShell.svelte"');
    expect(confirmDialogSource).toContain('from "./ui/Button.svelte"');
    expect(confirmDialogSource).not.toMatch(/<button\b/);
    expect(confirmDialogSource).toContain("$props()");
    expect(confirmDialogSource).toContain("$state(false)");
    expect(confirmDialogSource).toContain("$effect(");
    expect(confirmDialogSource).toContain("{#snippet heading()}");
    expect(confirmDialogSource).not.toMatch(/\bexport let\b/);
    expect(confirmDialogSource).not.toMatch(/\$:/);
    expect(confirmDialogSource).not.toMatch(/\son:[a-z]/);
  });

  it("migrates creation dialogs onto the local UI layer", () => {
    for (const source of [newProjectDialogSource, newSessionDialogSource]) {
      expect(source).toContain('from "./ui/ModalShell.svelte"');
      expect(source).toContain('from "./ui/Button.svelte"');
      expect(source).toContain('from "./ui/TextField.svelte"');
      expect(source).not.toMatch(/<button\b/);
      expect(source).not.toMatch(/<input\b/);
      expect(source).not.toMatch(/<textarea\b/);
      expect(source).not.toMatch(/<select\b/);
      expect(source).toContain("$props()");
      expect(source).toContain("$state(");
      expect(source).toContain("$derived(");
      expect(source).toContain("$effect(");
      expect(source).toContain("{#snippet heading()}");
      expect(source).not.toMatch(/\bexport let\b/);
      expect(source).not.toMatch(/\$:/);
      expect(source).not.toMatch(/\son:[a-z]/);
    }
    expect(newProjectDialogSource).toContain('from "./ui/TextArea.svelte"');
    expect(newSessionDialogSource).toContain('from "./ui/SelectField.svelte"');
    expect(newSessionDialogSource).toContain('from "./ui/Switch.svelte"');
  });

  it("migrates small overlay surfaces onto the local UI layer", () => {
    for (const source of [commandPaletteSource, onboardingPanelSource]) {
      expect(source).toContain('from "./ui/ModalShell.svelte"');
      expect(source).toContain('from "./ui/Button.svelte"');
      expect(source).not.toMatch(/<button\b/);
      expect(source).not.toMatch(/<input\b/);
      expect(source).not.toMatch(/<textarea\b/);
      expect(source).not.toMatch(/<select\b/);
      expect(source).toContain("$props()");
      expect(source).toContain("$state(");
      expect(source).toContain("$derived(");
      expect(source).toContain("$effect(");
      expect(source).toContain("{#snippet heading()}");
      expect(source).not.toMatch(/\bexport let\b/);
      expect(source).not.toMatch(/\$:/);
      expect(source).not.toMatch(/\son:[a-z]/);
    }
    expect(commandPaletteSource).toContain('from "./ui/TextField.svelte"');
    expect(onboardingPanelSource).toContain('from "./ui/Checkbox.svelte"');
    expect(onboardingPanelSource).toContain('from "./ui/IconButton.svelte"');
  });

  it("migrates WorkItemDetail property menus onto Popover/Menu primitives", () => {
    expect(workItemDetailSource).toContain('from "./ui/Popover.svelte"');
    expect(workItemDetailSource).toContain('from "./ui/Menu.svelte"');
    expect(workItemDetailSource).toContain('from "./ui/MenuItem.svelte"');
    expect(workItemDetailSource).toContain('from "./ui/TextField.svelte"');
    expect(workItemDetailSource).toContain('from "./ui/Checkbox.svelte"');
    expect(workItemDetailSource).not.toContain('class="fixed inset-0 z-10"');
    expect(workItemDetailSource).not.toContain("const menuItem =");
    expect(workItemDetailSource).toContain("<Popover");
    expect(workItemDetailSource).toContain("<Menu");
    expect(workItemDetailSource).toContain("<MenuItem");
  });

  it("migrates WorkItemDetail rail rows and next action bar onto local layout primitives", () => {
    expect(workItemDetailSource).toContain('from "./ui/PropertyRow.svelte"');
    expect(workItemDetailSource).toContain('from "./ui/NextActionBar.svelte"');
    expect(workItemDetailSource).toContain('from "./ui/SelectField.svelte"');
    expect(workItemDetailSource).toContain("<NextActionBar");
    expect(workItemDetailSource).toContain("<PropertyRow");
    expect(workItemDetailSource).not.toContain("function nextStepToneClass");
    expect(workItemDetailSource).not.toContain('class="flex items-center justify-between gap-2 px-3 py-2"');
    expect(workItemDetailSource).not.toContain('<!-- Next action bar -->');
  });

  it("keeps WorkItemDetail off raw native controls once local wrappers exist", () => {
    expect(workItemDetailSource).not.toMatch(/<(button|input|textarea|select)\b/);
    expect(workItemDetailSource).toContain('from "./ui/TextArea.svelte"');
    expect(workItemDetailSource).toContain('from "./ui/SelectField.svelte"');
    expect(workItemDetailSource).toContain("<TextField");
    expect(workItemDetailSource).toContain("<TextArea");
    expect(workItemDetailSource).toContain("<SelectField");
    expect(workItemDetailSource).toContain("<Button");
    expect(workItemDetailSource).toContain("<IconButton");
  });

  it("documents the Bits UI boundary as a design-system rule", () => {
    expect(designSystemDoc).toContain("`bits-ui` is the behavior foundation");
    expect(designSystemDoc).toContain("Feature components must not import from `bits-ui` directly");
    expect(designSystemDoc).toMatch(/New reusable controls\s+live in `frontend\/src\/ui\/`/);
    expect(designSystemDoc).toContain("New `frontend/src/ui/` primitives use Svelte 5 runes");
    expect(designSystemDoc).toContain("Form controls use local wrappers (`TextField`, `TextArea`, `Switch`, `SelectField`)");
    expect(designSystemDoc).toContain("Popover/menu controls use local wrappers (`Popover`, `Menu`, `MenuItem`)");
    expect(designSystemDoc).toContain("Display primitives use local wrappers (`StatusDot`, `Badge`, `SectionHeader`,");
    expect(designSystemDoc).toContain("Detail-view content/rail layouts use `DetailLayout`");
    expect(designSystemDoc).toContain("Primary action bars use `NextActionBar`");
    expect(designSystemDoc).toContain("Properties rails use `PropertyRow`");
  });
});
