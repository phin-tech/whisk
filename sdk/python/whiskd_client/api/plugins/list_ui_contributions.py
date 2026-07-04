from http import HTTPStatus
from typing import Any

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_response import ErrorResponse
from ...models.ui_contributions_response import UIContributionsResponse
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    project_id: str | Unset = UNSET,
    work_item_id: str | Unset = UNSET,
    run_id: str | Unset = UNSET,
    session_id: str | Unset = UNSET,
    pane_id: str | Unset = UNSET,
    pty_id: str | Unset = UNSET,
    gate_report_id: str | Unset = UNSET,
    phase: str | Unset = UNSET,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["projectId"] = project_id

    params["workItemId"] = work_item_id

    params["runId"] = run_id

    params["sessionId"] = session_id

    params["paneId"] = pane_id

    params["ptyId"] = pty_id

    params["gateReportId"] = gate_report_id

    params["phase"] = phase

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/ui-contributions",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorResponse | UIContributionsResponse:
    if response.status_code == 200:
        response_200 = UIContributionsResponse.from_dict(response.json())

        return response_200

    response_default = ErrorResponse.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorResponse | UIContributionsResponse]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    project_id: str | Unset = UNSET,
    work_item_id: str | Unset = UNSET,
    run_id: str | Unset = UNSET,
    session_id: str | Unset = UNSET,
    pane_id: str | Unset = UNSET,
    pty_id: str | Unset = UNSET,
    gate_report_id: str | Unset = UNSET,
    phase: str | Unset = UNSET,
) -> Response[ErrorResponse | UIContributionsResponse]:
    """Get aggregated UI contributions scoped to an entity

    Args:
        project_id (str | Unset):
        work_item_id (str | Unset):
        run_id (str | Unset):
        session_id (str | Unset):
        pane_id (str | Unset):
        pty_id (str | Unset):
        gate_report_id (str | Unset):
        phase (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | UIContributionsResponse]
    """

    kwargs = _get_kwargs(
        project_id=project_id,
        work_item_id=work_item_id,
        run_id=run_id,
        session_id=session_id,
        pane_id=pane_id,
        pty_id=pty_id,
        gate_report_id=gate_report_id,
        phase=phase,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient,
    project_id: str | Unset = UNSET,
    work_item_id: str | Unset = UNSET,
    run_id: str | Unset = UNSET,
    session_id: str | Unset = UNSET,
    pane_id: str | Unset = UNSET,
    pty_id: str | Unset = UNSET,
    gate_report_id: str | Unset = UNSET,
    phase: str | Unset = UNSET,
) -> ErrorResponse | UIContributionsResponse | None:
    """Get aggregated UI contributions scoped to an entity

    Args:
        project_id (str | Unset):
        work_item_id (str | Unset):
        run_id (str | Unset):
        session_id (str | Unset):
        pane_id (str | Unset):
        pty_id (str | Unset):
        gate_report_id (str | Unset):
        phase (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | UIContributionsResponse
    """

    return sync_detailed(
        client=client,
        project_id=project_id,
        work_item_id=work_item_id,
        run_id=run_id,
        session_id=session_id,
        pane_id=pane_id,
        pty_id=pty_id,
        gate_report_id=gate_report_id,
        phase=phase,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    project_id: str | Unset = UNSET,
    work_item_id: str | Unset = UNSET,
    run_id: str | Unset = UNSET,
    session_id: str | Unset = UNSET,
    pane_id: str | Unset = UNSET,
    pty_id: str | Unset = UNSET,
    gate_report_id: str | Unset = UNSET,
    phase: str | Unset = UNSET,
) -> Response[ErrorResponse | UIContributionsResponse]:
    """Get aggregated UI contributions scoped to an entity

    Args:
        project_id (str | Unset):
        work_item_id (str | Unset):
        run_id (str | Unset):
        session_id (str | Unset):
        pane_id (str | Unset):
        pty_id (str | Unset):
        gate_report_id (str | Unset):
        phase (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | UIContributionsResponse]
    """

    kwargs = _get_kwargs(
        project_id=project_id,
        work_item_id=work_item_id,
        run_id=run_id,
        session_id=session_id,
        pane_id=pane_id,
        pty_id=pty_id,
        gate_report_id=gate_report_id,
        phase=phase,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient,
    project_id: str | Unset = UNSET,
    work_item_id: str | Unset = UNSET,
    run_id: str | Unset = UNSET,
    session_id: str | Unset = UNSET,
    pane_id: str | Unset = UNSET,
    pty_id: str | Unset = UNSET,
    gate_report_id: str | Unset = UNSET,
    phase: str | Unset = UNSET,
) -> ErrorResponse | UIContributionsResponse | None:
    """Get aggregated UI contributions scoped to an entity

    Args:
        project_id (str | Unset):
        work_item_id (str | Unset):
        run_id (str | Unset):
        session_id (str | Unset):
        pane_id (str | Unset):
        pty_id (str | Unset):
        gate_report_id (str | Unset):
        phase (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | UIContributionsResponse
    """

    return (
        await asyncio_detailed(
            client=client,
            project_id=project_id,
            work_item_id=work_item_id,
            run_id=run_id,
            session_id=session_id,
            pane_id=pane_id,
            pty_id=pty_id,
            gate_report_id=gate_report_id,
            phase=phase,
        )
    ).parsed
