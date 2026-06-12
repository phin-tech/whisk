"""End-to-end tests: the generated Python client against a live whiskd daemon.

These verify the spec matches real wire behavior — status codes, camelCase
field mapping, RFC3339 time parsing, nested arrays, and query parameters — which
the offline byte-compile and the Go route-parity guard cannot.
"""

import datetime as dt

from whiskd_client.api.sessions import list_sessions
from whiskd_client.api.system import get_compatibility
from whiskd_client.api.workitems import create_project, create_work_item, list_work_items
from whiskd_client.client import Client
from whiskd_client.models import (
    CreateProjectRequest,
    CreateWorkItemRequest,
    Project,
    WorkItem,
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

    # POST with a JSON body, 201 response decoded into a typed model.
    project = create_project.sync(
        client=client,
        body=CreateProjectRequest(name="SDK Integration", root_dir=str(tmp_path)),
    )
    assert isinstance(project, Project), f"unexpected: {project!r}"
    assert project.id
    assert project.slug  # server-derived field round-trips back

    item = create_work_item.sync(
        client=client,
        body=CreateWorkItemRequest(project_id=project.id, title="hello from python"),
    )
    assert isinstance(item, WorkItem), f"unexpected: {item!r}"
    assert item.project_id == project.id
    assert item.number >= 1
    # time.Time -> RFC3339 string -> datetime, and nested arrays deserialize.
    assert isinstance(item.created_at, dt.datetime)
    # A nil Go slice serializes as null, so an empty collection is None, not [].
    assert item.attachments is None or isinstance(item.attachments, list)
    # History always carries the "created" event, so it round-trips as a list.
    assert isinstance(item.history, list) and len(item.history) >= 1

    # Query parameter (?projectId=) round-trips and filters.
    items = list_work_items.sync(client=client, project_id=project.id)
    assert isinstance(items, list)
    assert any(i.id == item.id for i in items)
