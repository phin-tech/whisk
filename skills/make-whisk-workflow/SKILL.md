---
name: make-whisk-workflow
description: Use when a user wants to design, create, or revise an editable Whisk workflow definition for project-level work item lifecycle stages, actions, gates, questions, artifacts, and agent run phases.
version: "1"
---

# Make Whisk Workflow

Create or revise Whisk workflow definition JSON. Keep workflows project-level: a project selects one workflow definition/version for new work items, and work items keep their stamped workflow version.

## Discovery

- If inside Whisk, prefer `WHISK_PROJECT_ROOT`, `WHISK_PROJECT_ID`, `WHISK_WORK_ITEM_ID`, and `WHISKD_URL`.
- If the user names an existing workflow file, read it first.
- If no file exists, use `plan-execute-review` as the baseline mental model.
- Read `README.md` in this skill before producing final JSON.

## Interview

Ask only questions that change the workflow definition:

- What type of work does this workflow manage?
- What stages should appear, in order?
- Which stage starts planning, and does it create an agent run?
- Is human plan approval required before execution?
- Which stage starts implementation, and should it require a worktree?
- What should happen when work is blocked?
- Which gates must pass before the item can be done?
- Which actions require a human?
- Should questions move the item to blocked, or only mark the run awaiting input?

Continue until stages, actions, gates, questions, and human approval points are clear.

## Output Rules

- Generate valid JSON using Whisk's `WorkflowDefinition` shape.
- Use only supported run presets: `reader`, `writer`, `reviewer`, `manager`.
- Use only supported prompt templates: `plan`, `implement`, `review`.
- Use only supported artifact kinds: `plan`, `feedback`, `gate_report`.
- Do not add shell commands, hooks, or arbitrary scripts to workflow files.
- Recommend `.whisk/workflows/<workflow-id>.json` for project-local workflow files.
- If the Whisk CLI supports validation, run `${WHISK_CLI:-whisk} workflow validate <path>` before import.
