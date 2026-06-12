from conftest import run_json


def test_workflow_cli_smoke(tmp_path, daemon):
    project = run_json(
        daemon,
        ["project", "create", "-name", "App", "-root", str(tmp_path), "-json"],
    )
    item = run_json(
        daemon,
        ["work-item", "create", "-project", project["id"], "-title", "Task", "-json"],
    )

    planning_run = run_json(
        daemon,
        ["workflow", "start-planning", "-work-item", item["id"], "-json"],
    )
    assert planning_run["workItemId"] == item["id"]
    assert planning_run["promptTemplateId"] == "plan"

    draft_plan = run_json(
        daemon,
        [
            "workflow",
            "submit-plan",
            "-work-item",
            item["id"],
            "-run",
            planning_run["id"],
            "-body",
            "1. Add a smoke test\n2. Keep the daemon contract stable",
            "-json",
        ],
    )
    assert draft_plan["kind"] == "plan"
    assert draft_plan["status"] == "draft"

    ready_item = run_json(
        daemon,
        [
            "workflow",
            "approve-plan",
            "-work-item",
            item["id"],
            "-artifact",
            draft_plan["id"],
            "-json",
        ],
    )
    assert ready_item["stageId"] == "ready"

    execution_run = run_json(
        daemon,
        ["workflow", "start-execution", "-work-item", item["id"], "-json"],
    )
    assert execution_run["workItemId"] == item["id"]
    assert execution_run["promptTemplateId"] == "implement"
