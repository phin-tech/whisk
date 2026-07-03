from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.agent_bridge_event import AgentBridgeEvent
from ...models.error_response import ErrorResponse
from ...models.mark_agent_bridge_event_read_request import (
    MarkAgentBridgeEventReadRequest,
)
from ...types import Response


def _get_kwargs(
    event_id: str,
    *,
    body: MarkAgentBridgeEventReadRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/agent-bridge-events/{event_id}/read".format(
            event_id=quote(str(event_id), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> AgentBridgeEvent | ErrorResponse:
    if response.status_code == 200:
        response_200 = AgentBridgeEvent.from_dict(response.json())

        return response_200

    response_default = ErrorResponse.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[AgentBridgeEvent | ErrorResponse]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    event_id: str,
    *,
    client: AuthenticatedClient,
    body: MarkAgentBridgeEventReadRequest,
) -> Response[AgentBridgeEvent | ErrorResponse]:
    """Mark a passive provider hook event read

    Args:
        event_id (str):
        body (MarkAgentBridgeEventReadRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AgentBridgeEvent | ErrorResponse]
    """

    kwargs = _get_kwargs(
        event_id=event_id,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    event_id: str,
    *,
    client: AuthenticatedClient,
    body: MarkAgentBridgeEventReadRequest,
) -> AgentBridgeEvent | ErrorResponse | None:
    """Mark a passive provider hook event read

    Args:
        event_id (str):
        body (MarkAgentBridgeEventReadRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AgentBridgeEvent | ErrorResponse
    """

    return sync_detailed(
        event_id=event_id,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    event_id: str,
    *,
    client: AuthenticatedClient,
    body: MarkAgentBridgeEventReadRequest,
) -> Response[AgentBridgeEvent | ErrorResponse]:
    """Mark a passive provider hook event read

    Args:
        event_id (str):
        body (MarkAgentBridgeEventReadRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AgentBridgeEvent | ErrorResponse]
    """

    kwargs = _get_kwargs(
        event_id=event_id,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    event_id: str,
    *,
    client: AuthenticatedClient,
    body: MarkAgentBridgeEventReadRequest,
) -> AgentBridgeEvent | ErrorResponse | None:
    """Mark a passive provider hook event read

    Args:
        event_id (str):
        body (MarkAgentBridgeEventReadRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AgentBridgeEvent | ErrorResponse
    """

    return (
        await asyncio_detailed(
            event_id=event_id,
            client=client,
            body=body,
        )
    ).parsed
