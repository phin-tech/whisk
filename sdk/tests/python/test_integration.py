"""End-to-end tests: the generated Python client against a live whiskd daemon.

These verify the spec matches real wire behavior: status codes, camelCase field
mapping, RFC3339 time parsing, nested arrays, and query parameters.
"""

import datetime as dt

from whiskd_client.api.sessions import list_sessions
from whiskd_client.api.system import get_compatibility
from whiskd_client.api.workitems import (
    answer_question,
    approve_plan,
    ask_question,
    complete_execution_for_work_item,
    create_project,
    create_work_item,
    list_work_items,
    start_execution,
    start_planning,
    submit_draft_plan,
    submit_review_feedback,
)
from whiskd_client.client import Client
from whiskd_client.models import (
    AnswerQuestionRequest,
    ApprovePlanRequest,
    Artifact,
    AskQuestionRequest,
    CompleteExecutionRequest,
    CreateProjectRequest,
    CreateWorkItemRequest,
    Project,
    Question,
    StartExecutionRequest,
    StartPlanningRequest,
    SubmitDraftPlanRequest,
    SubmitReviewFeedbackRequest,
    WorkItem,
    WorkItemRun,
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


def test_work_item_round_trip(base_url, tmp_path):
    client = Client(base_url=base_url)

    project = create_project.sync(
        client=client,
        body=CreateProjectRequest(name="SDK Integration", root_dir=str(tmp_path)),
    )
    assert isinstance(project, Project), f"unexpected: {project!r}"
    assert project.id
    assert project.slug

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
