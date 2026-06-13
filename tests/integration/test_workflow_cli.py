from conftest import run_failure, run_json


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

    artifacts = run_json(daemon, ["workflow", "artifacts", "-work-item", item["id"], "-json"])
    assert [artifact["kind"] for artifact in artifacts] == ["plan"]

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

    question = run_json(
        daemon,
        [
            "question",
            "ask",
            "-work-item",
            item["id"],
            "-run",
            execution_run["id"],
            "-prompt",
            "Which branch?",
            "-json",
        ],
    )
    assert question["status"] == "open"

    questions = run_json(daemon, ["question", "list", "-work-item", item["id"], "-json"])
    assert len(questions) == 1
    assert questions[0]["id"] == question["id"]

    review_item = run_json(
        daemon,
        [
            "workflow",
            "complete-execution",
            "-run",
            execution_run["id"],
            "-message",
            "ready",
            "-json",
        ],
    )
    assert review_item["stageId"] == "review"

    gates = run_json(daemon, ["gate", "list", "-work-item", item["id"], "-json"])
    assert len(gates) == 1
    assert gates[0]["blocking"] is True
    assert gates[0]["status"] == "pending"

    failed_done = run_failure(
        daemon,
        ["workflow", "approve-done", "-work-item", item["id"], "-json"],
    )
    assert "blocking gates" in failed_done.stderr

    passed_gate = run_json(
        daemon,
        ["gate", "complete", gates[0]["id"], "-status", "passed", "-json"],
    )
    assert passed_gate["status"] == "passed"

    done_item = run_json(
        daemon,
        [
            "workflow",
            "approve-done",
            "-work-item",
            item["id"],
            "-reason",
            "review gate passed",
            "-json",
        ],
    )
    assert done_item["stageId"] == "done"

    events = run_json(daemon, ["workflow", "events", "-work-item", item["id"], "-json"])
    assert events[-1]["type"] == "done_approved"
