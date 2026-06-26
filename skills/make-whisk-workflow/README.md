# Make Whisk Workflow Reference

Use this reference when creating editable Whisk workflow JSON.

## WorkflowDefinition

Shape:

```json
{
  "id": "custom-workflow",
  "version": 1,
  "stages": ["backlog", "planning", "ready", "execution", "review", "done"],
  "actions": [],
  "questions": {
    "enabled": true,
    "moveToBlocked": false,
    "setsRunState": "awaiting_input",
    "answerClearsAwaitingInputWhenNoOpenQuestionsRemain": true
  },
  "gates": []
}
```

The built-in baseline is `plan-execute-review`.

## Actions

An action declares where a work item can move and what side effects Whisk performs through the daemon.

```json
{
  "id": "start_execution",
  "from": ["ready"],
  "to": "execution",
  "requires": [{ "kind": "plan", "status": "approved" }],
  "createsRun": {
    "phase": "execution",
    "preset": "writer",
    "promptTemplateId": "implement",
    "workingDir": "worktree",
    "autoProvisionWorktree": true
  }
}
```

Supported action effect fields:

- `requires`: artifact requirements before action can run.
- `createsArtifact`: artifact kind/status to create.
- `updatesArtifact`: artifact kind/status update.
- `createsRun`: agent run to create.
- `completesRun`: marks the active run complete.
- `createsGates`: gate IDs to create after action.
- `resumesRun`: currently use `existing_execution` for review feedback loops.
- `requiresPassingBlockingGates`: require blocking gates before moving on.
- `requiresHuman`: action should be human approved.
- `sideStage`: temporary side path such as blocked.

Supported run presets: `reader`, `writer`, `reviewer`, `manager`.

Supported `promptTemplateId` values: `plan`, `implement`, `review`.

Supported artifact kinds: `plan`, `feedback`, `gate_report`.

Supported artifact statuses: `draft`, `approved`.

## Gates

Use gates for checks before done:

```json
{
  "id": "review",
  "phase": "review",
  "blocking": true
}
```

Pair blocking gates with an action that uses:

```json
{
  "requiresPassingBlockingGates": true
}
```

## Questions

Default question policy:

```json
{
  "enabled": true,
  "moveToBlocked": false,
  "setsRunState": "awaiting_input",
  "answerClearsAwaitingInputWhenNoOpenQuestionsRemain": true
}
```

Prefer `moveToBlocked: false` unless the user explicitly wants questions to move cards into a blocked stage. Questions should usually pause the run without changing the workflow path.

## Interview Checklist

- Stages, in order.
- Planning start action and run preset.
- Plan artifact and approval requirement.
- Execution start action and worktree requirement.
- Block and unblock behavior.
- Review feedback loop.
- Blocking gates before done.
- Human-only actions via `requiresHuman`.

## Validation

If available, validate with:

```sh
${WHISK_CLI:-whisk} workflow validate .whisk/workflows/<workflow-id>.json
```

If import is available:

```sh
${WHISK_CLI:-whisk} workflow import .whisk/workflows/<workflow-id>.json
```

Do not bypass the daemon by writing runtime state directly.
