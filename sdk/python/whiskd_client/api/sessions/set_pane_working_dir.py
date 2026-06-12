from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_response import ErrorResponse
from ...models.session import Session
from ...models.set_pane_working_dir_request import SetPaneWorkingDirRequest
from ...types import Response


def _get_kwargs(
    session_id: str,
    pane_id: str,
    *,
    body: SetPaneWorkingDirRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/sessions/{session_id}/panes/{pane_id}/set-working-dir".format(
            session_id=quote(str(session_id), safe=""),
            pane_id=quote(str(pane_id), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorResponse | Session:
    if response.status_code == 200:
        response_200 = Session.from_dict(response.json())

        return response_200

    response_default = ErrorResponse.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorResponse | Session]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    session_id: str,
    pane_id: str,
    *,
    client: AuthenticatedClient | Client,
    body: SetPaneWorkingDirRequest,
) -> Response[ErrorResponse | Session]:
    """
    Args:
        session_id (str):
        pane_id (str):
        body (SetPaneWorkingDirRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | Session]
    """

    kwargs = _get_kwargs(
        session_id=session_id,
        pane_id=pane_id,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    session_id: str,
    pane_id: str,
    *,
    client: AuthenticatedClient | Client,
    body: SetPaneWorkingDirRequest,
) -> ErrorResponse | Session | None:
    """
    Args:
        session_id (str):
        pane_id (str):
        body (SetPaneWorkingDirRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | Session
    """

    return sync_detailed(
        session_id=session_id,
        pane_id=pane_id,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    session_id: str,
    pane_id: str,
    *,
    client: AuthenticatedClient | Client,
    body: SetPaneWorkingDirRequest,
) -> Response[ErrorResponse | Session]:
    """
    Args:
        session_id (str):
        pane_id (str):
        body (SetPaneWorkingDirRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | Session]
    """

    kwargs = _get_kwargs(
        session_id=session_id,
        pane_id=pane_id,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    session_id: str,
    pane_id: str,
    *,
    client: AuthenticatedClient | Client,
    body: SetPaneWorkingDirRequest,
) -> ErrorResponse | Session | None:
    """
    Args:
        session_id (str):
        pane_id (str):
        body (SetPaneWorkingDirRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | Session
    """

    return (
        await asyncio_detailed(
            session_id=session_id,
            pane_id=pane_id,
            client=client,
            body=body,
        )
    ).parsed
