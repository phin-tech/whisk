<script lang="ts">
  import X from "@lucide/svelte/icons/x";
  import { Background, Controls, MarkerType, SvelteFlow } from "@xyflow/svelte";
  import type { Edge, Node } from "@xyflow/svelte";
  import "@xyflow/svelte/dist/style.css";
  import type { WorkflowDefinitionRecord } from "../../bindings/github.com/phin-tech/whisk/internal/protocol/models";
  import Button from "../ui/Button.svelte";
  import IconButton from "../ui/IconButton.svelte";
  import ModalShell from "../ui/ModalShell.svelte";

  type WorkflowPreviewSelection = {
    kind: "stage" | "action" | "gate";
    label: string;
    config: Record<string, unknown>;
  };
  type WorkflowActionDefinition = WorkflowDefinitionRecord["definition"]["actions"][number];
  type WorkflowGateDefinition = WorkflowDefinitionRecord["definition"]["gates"][number];

  export let visible = false;
  export let workflow: WorkflowDefinitionRecord | null = null;
  export let onclose: () => void;

  let selectedWorkflowConfig: WorkflowPreviewSelection | null = null;
  let selectedWorkflowIdentity = "";

  const edgeLabelStyle = [
    "background: rgb(24 24 27)",
    "border: 1px solid rgb(82 82 91)",
    "border-radius: 4px",
    "color: rgb(244 244 245)",
    "font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
    "font-size: 11px",
    "font-weight: 700",
    "line-height: 1",
    "padding: 4px 7px",
  ].join(";");

  function workflowIdentity() {
    if (!workflow) return "Workflow";
    return `${workflow.id}@${workflow.version}`;
  }

  function isRecord(value: unknown): value is Record<string, unknown> {
    return Boolean(value) && typeof value === "object" && !Array.isArray(value);
  }

  function prettyStage(stage: string) {
    return stage.replace(/_/g, " ");
  }

  function workflowPreviewGateConfig(gate: WorkflowGateDefinition): Record<string, unknown> {
    return {
      gateId: gate.id,
      phase: gate.phase,
      blocking: gate.blocking,
    };
  }

  function workflowPreviewActionConfig(
    action: WorkflowActionDefinition,
    source?: string,
  ): Record<string, unknown> {
    return {
      actionId: action.id,
      edge: source ? `${source} -> ${action.to}` : `${(action.from ?? []).join(", ")} -> ${action.to}`,
      from: action.from ?? [],
      to: action.to,
      requires: action.requires ?? [],
      createsArtifact: action.createsArtifact ?? null,
      updatesArtifact: action.updatesArtifact ?? null,
      createsRun: action.createsRun ?? null,
      completesRun: Boolean(action.completesRun),
      createsGates: action.createsGates ?? [],
      resumesRun: action.resumesRun ?? "",
      requiresPassingBlockingGates: Boolean(action.requiresPassingBlockingGates),
      requiresHuman: Boolean(action.requiresHuman),
      sideStage: Boolean(action.sideStage),
    };
  }

  function workflowPreviewStageConfig(record: WorkflowDefinitionRecord, stage: string): Record<string, unknown> {
    const stages = record.definition.stages ?? [];
    const actions = record.definition.actions ?? [];
    const gates = record.definition.gates ?? [];
    const relatedGateIds = new Set(
      actions
        .filter((action) => action.to === stage || (action.from ?? []).includes(stage))
        .flatMap((action) => action.createsGates ?? []),
    );

    return {
      stageId: stage,
      position: stages.indexOf(stage) + 1,
      initial: stages[0] === stage,
      terminal: stages[stages.length - 1] === stage,
      incomingActions: actions.filter((action) => action.to === stage).map((action) => action.id),
      outgoingActions: actions.filter((action) => (action.from ?? []).includes(stage)).map((action) => action.id),
      gates: gates
        .filter((gate) => gate.phase === stage || relatedGateIds.has(gate.id))
        .map((gate) => workflowPreviewGateConfig(gate)),
    };
  }

  function workflowPreviewStageSelection(record: WorkflowDefinitionRecord, stage: string): WorkflowPreviewSelection {
    return {
      kind: "stage",
      label: prettyStage(stage),
      config: workflowPreviewStageConfig(record, stage),
    };
  }

  function workflowPreviewNodes(record: WorkflowDefinitionRecord | null): Node[] {
    const stages = record?.definition.stages ?? [];
    return stages.map((stage, index) => ({
      id: stage,
      type: index === 0 ? "input" : index === stages.length - 1 ? "output" : "default",
      position: { x: index * 210, y: index % 2 === 0 ? 70 : 155 },
      data: {
        label: prettyStage(stage),
        config: record ? workflowPreviewStageConfig(record, stage) : { stageId: stage },
      },
      draggable: false,
      selectable: true,
      class: "workflow-preview-node",
    }));
  }

  function workflowPreviewEdgeClass(action: WorkflowActionDefinition) {
    return [
      "workflow-preview-edge",
      action.createsRun ? "workflow-preview-edge-run" : "",
      action.requiresHuman ? "workflow-preview-edge-human" : "",
      (action.createsGates?.length ?? 0) > 0 || action.requiresPassingBlockingGates ? "workflow-preview-edge-gate" : "",
    ]
      .filter(Boolean)
      .join(" ");
  }

  function workflowPreviewEdges(record: WorkflowDefinitionRecord | null): Edge[] {
    const stages = new Set(record?.definition.stages ?? []);
    return (record?.definition.actions ?? []).flatMap((action) =>
      (action.from ?? [])
        .filter((source) => stages.has(source) && stages.has(action.to))
        .map((source) => ({
          id: `${action.id}:${source}:${action.to}`,
          source,
          target: action.to,
          type: "smoothstep",
          label: action.id,
          labelStyle: edgeLabelStyle,
          data: { config: workflowPreviewActionConfig(action, source) },
          animated: Boolean(action.createsRun || action.requiresHuman),
          interactionWidth: 24,
          markerEnd: { type: MarkerType.ArrowClosed },
          class: workflowPreviewEdgeClass(action),
        })),
    );
  }

  $: nodes = workflowPreviewNodes(workflow);
  $: edges = workflowPreviewEdges(workflow);
  $: workflowActions = workflow?.definition.actions ?? [];
  $: workflowGates = workflow?.definition.gates ?? [];
  $: actionCount = workflow?.definition.actions.length ?? 0;
  $: gateCount = workflow?.definition.gates.length ?? 0;
  $: questionMode = workflow?.definition.questions.enabled ? "questions on" : "questions off";
  $: if (visible && workflow && selectedWorkflowIdentity !== workflowIdentity()) {
    selectedWorkflowIdentity = workflowIdentity();
    selectedWorkflowConfig = workflow.definition.stages[0] ? workflowPreviewStageSelection(workflow, workflow.definition.stages[0]) : null;
  }
  $: if (!visible && selectedWorkflowIdentity) {
    selectedWorkflowIdentity = "";
    selectedWorkflowConfig = null;
  }
  $: selectedWorkflowConfigTitle =
    selectedWorkflowConfig?.kind === "action"
      ? "Action config"
      : selectedWorkflowConfig?.kind === "gate"
        ? "Gate config"
        : "Stage config";
  $: selectedWorkflowConfigEntries = Object.entries(selectedWorkflowConfig?.config ?? {});
  $: selectedWorkflowConfigJson = JSON.stringify(selectedWorkflowConfig?.config ?? {}, null, 2);

  function formatConfigValue(value: unknown): string {
    if (value === null || value === undefined || value === "") return "none";
    if (typeof value === "boolean") return value ? "yes" : "no";
    if (Array.isArray(value)) {
      if (value.length === 0) return "none";
      return value.map((entry) => formatConfigValue(entry)).join(", ");
    }
    if (isRecord(value)) {
      const entries = Object.entries(value)
        .filter(([, entry]) => entry !== undefined && entry !== null && !(Array.isArray(entry) && entry.length === 0));
      if (entries.length === 0) return "none";
      return entries.map(([key, entry]) => `${key}: ${formatConfigValue(entry)}`).join("; ");
    }
    return String(value);
  }

  function selectWorkflowNode({ node }: { node: Node }) {
    const config = node.data?.config;
    selectedWorkflowConfig = {
      kind: "stage",
      label: String(node.data?.label ?? node.id),
      config: isRecord(config) ? config : { stageId: node.id },
    };
  }

  function selectWorkflowEdge({ edge }: { edge: Edge }) {
    const config = edge.data?.config;
    selectedWorkflowConfig = {
      kind: "action",
      label: String(edge.label ?? edge.id),
      config: isRecord(config) ? config : { actionId: edge.id },
    };
  }

  function selectWorkflowAction(action: WorkflowActionDefinition) {
    selectedWorkflowConfig = {
      kind: "action",
      label: action.id,
      config: workflowPreviewActionConfig(action),
    };
  }

  function selectWorkflowGate(gate: WorkflowGateDefinition) {
    selectedWorkflowConfig = {
      kind: "gate",
      label: gate.id,
      config: workflowPreviewGateConfig(gate),
    };
  }

  function handleEscape(event: KeyboardEvent) {
    event.preventDefault();
    onclose();
  }

  function handleOpenChange(open: boolean) {
    if (!open && visible) onclose();
  }
</script>

<ModalShell
  open={visible}
  titleId="workflow-preview-title"
  class="max-w-[min(96vw,1440px)] overflow-hidden bg-bg-base shadow-[0_24px_80px_rgba(0,0,0,0.45)]"
  onOpenChange={handleOpenChange}
  onEscapeKeydown={handleEscape}
>
  {#snippet heading()}
    Workflow preview
  {/snippet}

  <div class="flex h-11 items-center justify-between border-b border-hairline px-4">
    <div class="min-w-0">
      <div class="text-[13px] font-semibold text-text-primary">Workflow preview</div>
      <div class="truncate font-mono text-[10px] text-text-muted">{workflowIdentity()}</div>
    </div>
    <IconButton label="Close workflow preview" onclick={onclose}>
      <X size={14} />
    </IconButton>
  </div>

  <div class="grid gap-3 px-4 py-4">
    <div class="flex flex-wrap gap-2 text-[11px] text-text-secondary">
      <span class="rounded border border-border-subtle bg-bg-surface/50 px-2 py-1">{nodes.length} stages</span>
      <span class="rounded border border-border-subtle bg-bg-surface/50 px-2 py-1">{actionCount} actions</span>
      <span class="rounded border border-border-subtle bg-bg-surface/50 px-2 py-1">{gateCount} gates</span>
      <span class="rounded border border-border-subtle bg-bg-surface/50 px-2 py-1">{questionMode}</span>
    </div>

    <div class="flex flex-wrap items-center gap-x-4 gap-y-2 text-[11px] text-text-secondary">
      <span class="font-semibold uppercase text-text-muted">Legend</span>
      <span class="inline-flex items-center gap-1.5">
        <span class="h-3 w-6 rounded border border-zinc-500 bg-zinc-900"></span>
        stage
      </span>
      <span class="inline-flex items-center gap-1.5">
        <span class="h-px w-6 bg-slate-400"></span>
        action
      </span>
      <span class="inline-flex items-center gap-1.5">
        <span class="h-px w-6 border-t-2 border-sky-300"></span>
        run or human
      </span>
      <span class="inline-flex items-center gap-1.5">
        <span class="h-px w-6 border-t-2 border-amber"></span>
        gate-related
      </span>
    </div>

    <div class="grid min-h-0 gap-3 xl:grid-cols-[minmax(0,1fr)_360px]">
      <div class="workflow-preview h-[min(70vh,720px)] overflow-hidden rounded border border-border-subtle bg-bg-deep">
        <SvelteFlow
          {nodes}
          {edges}
          fitView
          fitViewOptions={{ padding: 0.18 }}
          nodesDraggable={false}
          nodesConnectable={false}
          elementsSelectable={true}
          onnodeclick={selectWorkflowNode}
          onedgeclick={selectWorkflowEdge}
          proOptions={{ hideAttribution: true }}
        >
          <Background patternColor="rgba(226,232,240,0.34)" gap={24} />
          <Controls />
        </SvelteFlow>
      </div>

      <aside class="flex max-h-[min(70vh,720px)] min-h-0 flex-col overflow-hidden rounded border border-border-subtle bg-bg-deep">
        <div class="border-b border-hairline px-3 py-2">
          <div class="text-[11px] font-semibold uppercase text-text-muted">{selectedWorkflowConfigTitle}</div>
          <div class="truncate font-mono text-[13px] font-semibold text-text-primary">{selectedWorkflowConfig?.label}</div>
        </div>

        <div class="app-scrollbar min-h-0 flex-1 overflow-auto p-3">
          <div class="grid gap-1.5">
            {#each selectedWorkflowConfigEntries as [key, value]}
              <div class="grid grid-cols-[108px_minmax(0,1fr)] gap-2 rounded border border-hairline bg-bg-surface/30 px-2 py-1.5 text-[11px]">
                <dt class="truncate font-mono text-text-muted">{key}</dt>
                <dd class="min-w-0 break-words text-text-secondary">{formatConfigValue(value)}</dd>
              </div>
            {:else}
              <div class="rounded border border-hairline bg-bg-surface/30 px-2 py-1.5 text-[11px] text-text-muted">
                No config.
              </div>
            {/each}
          </div>

          <details class="mt-3 rounded border border-hairline bg-bg-base/50">
            <summary class="cursor-pointer px-2 py-1.5 text-[11px] font-semibold uppercase text-text-muted">
              Raw config
            </summary>
            <pre class="app-scrollbar max-h-52 overflow-auto whitespace-pre-wrap break-words border-t border-hairline p-2 font-mono text-[11px] leading-5 text-text-secondary">{selectedWorkflowConfigJson}</pre>
          </details>

          <div class="mt-3 border-t border-hairline pt-3">
            <div class="mb-2 text-[11px] font-semibold uppercase text-text-muted">Gates</div>
            <div class="flex flex-wrap gap-1.5">
              {#each workflowGates as gate (gate.id)}
                <button
                  type="button"
                  class="rounded border border-amber/30 bg-amber/10 px-2 py-1 font-mono text-[10px] text-amber hover:border-amber"
                  onclick={() => selectWorkflowGate(gate)}
                >
                  {gate.id}
                </button>
              {:else}
                <span class="text-[11px] text-text-muted">none</span>
              {/each}
            </div>
          </div>

          <div class="mt-3 border-t border-hairline pt-3">
            <div class="mb-2 text-[11px] font-semibold uppercase text-text-muted">Actions</div>
            <div class="grid gap-1.5">
              {#each workflowActions as action (action.id)}
                <button
                  type="button"
                  class="min-w-0 rounded border border-border-subtle bg-bg-surface/45 px-2 py-1.5 text-left text-[11px] text-text-secondary hover:border-accent hover:text-accent"
                  onclick={() => selectWorkflowAction(action)}
                >
                  <span class="block truncate font-mono font-semibold text-text-primary">{action.id}</span>
                  <span class="block truncate font-mono text-[10px]">{(action.from ?? []).join(", ")} -> {action.to}</span>
                </button>
              {/each}
            </div>
          </div>
        </div>
      </aside>
    </div>
  </div>

  <div class="flex justify-end border-t border-hairline px-4 py-3">
    <Button type="button" variant="outline" onclick={onclose}>Close</Button>
  </div>
</ModalShell>

<style>
  .workflow-preview :global(.svelte-flow) {
    --xy-node-background-color-default: rgb(24 24 27);
    --xy-node-border-default: 1px solid rgb(82 82 91);
    --xy-node-color-default: rgb(244 244 245);
    --xy-edge-stroke-default: rgb(148 163 184);
    --xy-edge-stroke-width-default: 2;
    --xy-controls-button-background-color-default: rgb(24 24 27);
    --xy-controls-button-color-default: rgb(244 244 245);
    --xy-controls-button-border-color-default: rgb(63 63 70);
  }

  .workflow-preview :global(.workflow-preview-node) {
    min-width: 170px;
    border-color: rgb(82 82 91);
    background: rgb(24 24 27);
    color: rgb(244 244 245);
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 13px;
    font-weight: 700;
    box-shadow: 0 18px 40px rgba(0, 0, 0, 0.38);
  }

  .workflow-preview :global(.workflow-preview-edge .svelte-flow__edge-path) {
    stroke: rgb(148 163 184);
    stroke-width: 2;
  }

  .workflow-preview :global(.workflow-preview-edge.animated .svelte-flow__edge-path) {
    stroke: rgb(125 211 252);
  }

  .workflow-preview :global(.workflow-preview-edge-gate .svelte-flow__edge-path) {
    stroke: rgb(251 191 36);
  }

  .workflow-preview :global(.workflow-preview-edge-run .svelte-flow__edge-path),
  .workflow-preview :global(.workflow-preview-edge-human .svelte-flow__edge-path) {
    stroke: rgb(125 211 252);
  }

  .workflow-preview :global(.svelte-flow__edge-label) {
    background: rgb(24 24 27);
    border: 1px solid rgb(82 82 91);
    border-radius: 4px;
    color: rgb(244 244 245);
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 11px;
    font-weight: 700;
    line-height: 1;
    padding: 4px 7px;
    pointer-events: none !important;
  }
</style>
