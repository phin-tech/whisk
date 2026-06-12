from http import HTTPStatus
from typing import Any

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_response import ErrorResponse
from ...models.status_event import StatusEvent
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    project_id: str | Unset = UNSET,
    work_item_id: str | Unset = UNSET,
    run_id: str | Unset = UNSET,
    session_id: str | Unset = UNSET,
    pty_id: str | Unset = UNSET,
    unread_only: bool | Unset = UNSET,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["projectId"] = project_id

    params["workItemId"] = work_item_id

    params["runId"] = run_id

    params["sessionId"] = session_id

    params["ptyId"] = pty_id

    params["unreadOnly"] = unread_only

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/status-events",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorResponse | list[StatusEvent]:
    if response.status_code == 200:
        response_200 = []
        _response_200 = response.json()
        for response_200_item_data in _response_200:
            response_200_item = StatusEvent.from_dict(response_200_item_data)

            response_200.append(response_200_item)

        return response_200

    response_default = ErrorResponse.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorResponse | list[StatusEvent]]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    project_id: str | Unset = UNSET,
    work_item_id: str | Unset = UNSET,
    run_id: str | Unset = UNSET,
    session_id: str | Unset = UNSET,
    pty_id: str | Unset = UNSET,
    unread_only: bool | Unset = UNSET,
) -> Response[ErrorResponse | list[StatusEvent]]:
    """
    Args:
        project_id (str | Unset):
        work_item_id (str | Unset):
        run_id (str | Unset):
        session_id (str | Unset):
        pty_id (str | Unset):
        unread_only (bool | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | list[StatusEvent]]
    """

    kwargs = _get_kwargs(
        project_id=project_id,
        work_item_id=work_item_id,
        run_id=run_id,
        session_id=session_id,
        pty_id=pty_id,
        unread_only=unread_only,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    project_id: str | Unset = UNSET,
    work_item_id: str | Unset = UNSET,
    run_id: str | Unset = UNSET,
    session_id: str | Unset = UNSET,
    pty_id: str | Unset = UNSET,
    unread_only: bool | Unset = UNSET,
) -> ErrorResponse | list[StatusEvent] | None:
    """
    Args:
        project_id (str | Unset):
        work_item_id (str | Unset):
        run_id (str | Unset):
        session_id (str | Unset):
        pty_id (str | Unset):
        unread_only (bool | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | list[StatusEvent]
    """

    return sync_detailed(
        client=client,
        project_id=project_id,
        work_item_id=work_item_id,
        run_id=run_id,
        session_id=session_id,
        pty_id=pty_id,
        unread_only=unread_only,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    project_id: str | Unset = UNSET,
    work_item_id: str | Unset = UNSET,
    run_id: str | Unset = UNSET,
    session_id: str | Unset = UNSET,
    pty_id: str | Unset = UNSET,
    unread_only: bool | Unset = UNSET,
) -> Response[ErrorResponse | list[StatusEvent]]:
    """
    Args:
        project_id (str | Unset):
        work_item_id (str | Unset):
        run_id (str | Unset):
        session_id (str | Unset):
        pty_id (str | Unset):
        unread_only (bool | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | list[StatusEvent]]
    """

    kwargs = _get_kwargs(
        project_id=project_id,
        work_item_id=work_item_id,
        run_id=run_id,
        session_id=session_id,
        pty_id=pty_id,
        unread_only=unread_only,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    project_id: str | Unset = UNSET,
    work_item_id: str | Unset = UNSET,
    run_id: str | Unset = UNSET,
    session_id: str | Unset = UNSET,
    pty_id: str | Unset = UNSET,
    unread_only: bool | Unset = UNSET,
) -> ErrorResponse | list[StatusEvent] | None:
    """
    Args:
        project_id (str | Unset):
        work_item_id (str | Unset):
        run_id (str | Unset):
        session_id (str | Unset):
        pty_id (str | Unset):
        unread_only (bool | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | list[StatusEvent]
    """

    return (
        await asyncio_detailed(
            client=client,
            project_id=project_id,
            work_item_id=work_item_id,
            run_id=run_id,
            session_id=session_id,
            pty_id=pty_id,
            unread_only=unread_only,
        )
    ).parsed
