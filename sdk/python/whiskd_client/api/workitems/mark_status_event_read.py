from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_response import ErrorResponse
from ...models.mark_status_event_read_request import MarkStatusEventReadRequest
from ...models.status_event import StatusEvent
from ...types import Response


def _get_kwargs(
    status_event_id: str,
    *,
    body: MarkStatusEventReadRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/status-events/{status_event_id}/read".format(
            status_event_id=quote(str(status_event_id), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorResponse | StatusEvent:
    if response.status_code == 200:
        response_200 = StatusEvent.from_dict(response.json())

        return response_200

    response_default = ErrorResponse.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorResponse | StatusEvent]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    status_event_id: str,
    *,
    client: AuthenticatedClient,
    body: MarkStatusEventReadRequest,
) -> Response[ErrorResponse | StatusEvent]:
    """
    Args:
        status_event_id (str):
        body (MarkStatusEventReadRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | StatusEvent]
    """

    kwargs = _get_kwargs(
        status_event_id=status_event_id,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    status_event_id: str,
    *,
    client: AuthenticatedClient,
    body: MarkStatusEventReadRequest,
) -> ErrorResponse | StatusEvent | None:
    """
    Args:
        status_event_id (str):
        body (MarkStatusEventReadRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | StatusEvent
    """

    return sync_detailed(
        status_event_id=status_event_id,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    status_event_id: str,
    *,
    client: AuthenticatedClient,
    body: MarkStatusEventReadRequest,
) -> Response[ErrorResponse | StatusEvent]:
    """
    Args:
        status_event_id (str):
        body (MarkStatusEventReadRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | StatusEvent]
    """

    kwargs = _get_kwargs(
        status_event_id=status_event_id,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    status_event_id: str,
    *,
    client: AuthenticatedClient,
    body: MarkStatusEventReadRequest,
) -> ErrorResponse | StatusEvent | None:
    """
    Args:
        status_event_id (str):
        body (MarkStatusEventReadRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | StatusEvent
    """

    return (
        await asyncio_detailed(
            status_event_id=status_event_id,
            client=client,
            body=body,
        )
    ).parsed
