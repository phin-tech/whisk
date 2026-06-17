"""End-to-end tests: the generated Python client against a live whiskd daemon.

These verify the spec matches real wire behavior: status codes, camelCase field
mapping, RFC3339 time parsing, nested arrays, and query parameters.
"""

import datetime as dt

from whiskd_client.api.sessions import list_sessions
from whiskd_client.api.system import clear_daemon, get_compatibility
from whiskd_client.api.workitems import (
    answer_question,
    approve_done,
    approve_plan,
    ask_question,
    complete_gate,
    complete_execution_for_work_item,
    create_project,
    create_work_item,
    add_project_attachment,
    get_project_context,
    list_artifacts,
    list_gate_reports,
    list_questions,
    list_work_items,
    list_workflow_events,
    start_execution,
    start_planning,
    submit_draft_plan,
    submit_review_feedback,
)
from whiskd_client.client import Client
from whiskd_client.models import (
    AnswerQuestionRequest,
    ApproveDoneRequest,
    ApprovePlanRequest,
    Artifact,
    AskQuestionRequest,
    ClearDaemonRequest,
    ClearDaemonResponse,
    CompleteGateRequest,
    CompleteExecutionRequest,
    CreateProjectRequest,
    CreateWorkItemRequest,
    AddProjectAttachmentRequest,
    GateReport,
    Project,
    ProjectContext,
    Question,
    StartExecutionRequest,
    StartPlanningRequest,
    SubmitDraftPlanRequest,
    SubmitReviewFeedbackRequest,
    WorkItem,
    WorkItemRun,
    WorkflowEvent,
)


def test_compatibility_handshake(base_url):
    client = Client(base_url=base_url)
    compat = get_compatibility.sync(client=client)
    assert compat is not None
    assert compat.api_version >= 1
    assert compat.git_sha != ""


def test_sessions_start_empty(base_url):
    client = Client(base_url=base_url)
    sessions = list_sessions.sync(client=client)
    assert sessions == []


def test_daemon_clear_resets_work_item_state(base_url, tmp_path):
    client = Client(base_url=base_url)

    project = create_project.sync(
        client=client,
        body=CreateProjectRequest(name="SDK Clear", root_dir=str(tmp_path)),
    )
    assert isinstance(project, Project), f"unexpected: {project!r}"

    item = create_work_item.sync(
        client=client,
        body=CreateWorkItemRequest(project_id=project.id, title="clear me"),
    )
    assert isinstance(item, WorkItem), f"unexpected: {item!r}"

    cleared = clear_daemon.sync(client=client, body=ClearDaemonRequest())
    assert isinstance(cleared, ClearDaemonResponse), f"unexpected: {cleared!r}"
    assert cleared.projects_cleared >= 1
    assert cleared.work_items_cleared >= 1

    assert list_work_items.sync(client=client) == []


def test_work_item_round_trip(base_url, tmp_path):
    client = Client(base_url=base_url)

    project = create_project.sync(
        client=client,
        body=CreateProjectRequest(name="SDK Integration", root_dir=str(tmp_path)),
    )
    assert isinstance(project, Project), f"unexpected: {project!r}"
    assert project.id
    assert project.slug

    project = add_project_attachment.sync(
        project.id,
        client=client,
        body=AddProjectAttachmentRequest(
            project_id=project.id,
            kind="note",
            title="Context note",
            note="remember this",
            include_in_context=True,
        ),
    )
    assert isinstance(project, Project), f"unexpected: {project!r}"
    assert project.attachments is not None and len(project.attachments) == 1
    context = get_project_context.sync(project.id, client=client)
    assert isinstance(context, ProjectContext), f"unexpected: {context!r}"
    assert context.items is not None and context.items[0].content == "remember this"

    item = create_work_item.sync(
        client=client,
        body=CreateWorkItemRequest(project_id=project.id, title="hello from python"),
    )
    assert isinstance(item, WorkItem), f"unexpected: {item!r}"
    assert item.project_id == project.id
    assert item.number >= 1
    assert isinstance(item.created_at, dt.datetime)
    assert item.attachments is None or isinstance(item.attachments, list)
    assert isinstance(item.history, list) and len(item.history) >= 1

    items = list_work_items.sync(client=client, project_id=project.id)
    assert isinstance(items, list)
    assert any(i.id == item.id for i in items)


def test_workflow_round_trip(base_url, tmp_path):
    client = Client(base_url=base_url)

    project = create_project.sync(
        client=client,
        body=CreateProjectRequest(name="SDK Workflow", root_dir=str(tmp_path)),
    )
    assert isinstance(project, Project), f"unexpected: {project!r}"

    item = create_work_item.sync(
        client=client,
        body=CreateWorkItemRequest(project_id=project.id, title="workflow from python"),
    )
    assert isinstance(item, WorkItem), f"unexpected: {item!r}"

    planning = start_planning.sync(
        item.id,
        client=client,
        body=StartPlanningRequest(work_item_id=item.id, actor="pytest"),
    )
    assert isinstance(planning, WorkItemRun), f"unexpected: {planning!r}"
    assert planning.work_item_id == item.id

    draft = submit_draft_plan.sync(
        item.id,
        client=client,
        body=SubmitDraftPlanRequest(
            work_item_id=item.id,
            run_id=planning.id,
            title="Test plan",
            body="1. Change the code\n2. Run tests",
            actor="pytest",
        ),
    )
    assert isinstance(draft, Artifact), f"unexpected: {draft!r}"
    assert draft.kind == "plan"
    assert draft.status == "draft"

    ready = approve_plan.sync(
        item.id,
        client=client,
        body=ApprovePlanRequest(work_item_id=item.id, artifact_id=draft.id, actor="human"),
    )
    assert isinstance(ready, WorkItem), f"unexpected: {ready!r}"
    assert ready.stage_id == "ready"

    execution = start_execution.sync(
        item.id,
        client=client,
        body=StartExecutionRequest(work_item_id=item.id, actor="pytest"),
    )
    assert isinstance(execution, WorkItemRun), f"unexpected: {execution!r}"
    assert execution.work_item_id == item.id

    question = ask_question.sync(
        client=client,
        body=AskQuestionRequest(
            work_item_id=item.id,
            run_id=execution.id,
            prompt="Which branch should I use?",
            actor="agent",
        ),
    )
    assert isinstance(question, Question), f"unexpected: {question!r}"
    assert question.status == "open"

    answered = answer_question.sync(
        question.id,
        client=client,
        body=AnswerQuestionRequest(id=question.id, answer="Use the current branch.", actor="human"),
    )
    assert isinstance(answered, Question), f"unexpected: {answered!r}"
    assert answered.status == "answered"
    assert answered.answer == "Use the current branch."

    questions = list_questions.sync(client=client, work_item_id=item.id)
    assert isinstance(questions, list), f"unexpected: {questions!r}"
    assert len(questions) == 1
    assert questions[0].id == question.id

    review = complete_execution_for_work_item.sync(
        item.id,
        client=client,
        body=CompleteExecutionRequest(
            work_item_id=item.id,
            run_id=execution.id,
            message="ready for review",
            actor="pytest",
        ),
    )
    assert isinstance(review, WorkItem), f"unexpected: {review!r}"
    assert review.stage_id == "review"

    feedback = submit_review_feedback.sync(
        item.id,
        client=client,
        body=SubmitReviewFeedbackRequest(
            work_item_id=item.id,
            run_id=execution.id,
            body="Please tighten the assertions.",
            actor="reviewer",
        ),
    )
    assert isinstance(feedback, Artifact), f"unexpected: {feedback!r}"
    assert feedback.kind == "feedback"

    artifacts = list_artifacts.sync(client=client, work_item_id=item.id)
    assert isinstance(artifacts, list), f"unexpected: {artifacts!r}"
    assert sorted(a.kind for a in artifacts) == ["feedback", "plan"]

    review = complete_execution_for_work_item.sync(
        item.id,
        client=client,
        body=CompleteExecutionRequest(
            work_item_id=item.id,
            run_id=execution.id,
            message="ready after feedback",
            actor="pytest",
        ),
    )
    assert isinstance(review, WorkItem), f"unexpected: {review!r}"
    assert review.stage_id == "review"

    gates = list_gate_reports.sync(client=client, work_item_id=item.id)
    assert isinstance(gates, list), f"unexpected: {gates!r}"
    assert len(gates) == 1
    gate = gates[0]
    assert isinstance(gate, GateReport)
    assert gate.blocking is True
    assert gate.status == "pending"

    blocked_done = approve_done.sync_detailed(
        item.id,
        client=client,
        body=ApproveDoneRequest(work_item_id=item.id, actor="human"),
    )
    assert blocked_done.status_code == 400

    passed_gate = complete_gate.sync(
        gate.id,
        client=client,
        body=CompleteGateRequest(id=gate.id, status="passed", actor="pytest"),
    )
    assert isinstance(passed_gate, GateReport), f"unexpected: {passed_gate!r}"
    assert passed_gate.status == "passed"

    done = approve_done.sync(
        item.id,
        client=client,
        body=ApproveDoneRequest(work_item_id=item.id, reason="review gate passed", actor="human"),
    )
    assert isinstance(done, WorkItem), f"unexpected: {done!r}"
    assert done.stage_id == "done"

    events = list_workflow_events.sync(client=client, work_item_id=item.id)
    assert isinstance(events, list), f"unexpected: {events!r}"
    assert all(isinstance(event, WorkflowEvent) for event in events)
    assert events[-1].type_ == "done_approved"
