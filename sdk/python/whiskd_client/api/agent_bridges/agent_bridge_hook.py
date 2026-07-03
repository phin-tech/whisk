from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.agent_bridge_hook_request import AgentBridgeHookRequest
from ...models.agent_bridge_hook_response import AgentBridgeHookResponse
from ...models.error_response import ErrorResponse
from ...types import Response


def _get_kwargs(
    bridge_id: str,
    *,
    body: AgentBridgeHookRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/agent-bridges/{bridge_id}/hooks".format(
            bridge_id=quote(str(bridge_id), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> AgentBridgeHookResponse | ErrorResponse:
    if response.status_code == 200:
        response_200 = AgentBridgeHookResponse.from_dict(response.json())

        return response_200

    response_default = ErrorResponse.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[AgentBridgeHookResponse | ErrorResponse]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    bridge_id: str,
    *,
    client: AuthenticatedClient,
    body: AgentBridgeHookRequest,
) -> Response[AgentBridgeHookResponse | ErrorResponse]:
    """Handle provider hook callback for a daemon-owned agent bridge

    Args:
        bridge_id (str):
        body (AgentBridgeHookRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AgentBridgeHookResponse | ErrorResponse]
    """

    kwargs = _get_kwargs(
        bridge_id=bridge_id,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    bridge_id: str,
    *,
    client: AuthenticatedClient,
    body: AgentBridgeHookRequest,
) -> AgentBridgeHookResponse | ErrorResponse | None:
    """Handle provider hook callback for a daemon-owned agent bridge

    Args:
        bridge_id (str):
        body (AgentBridgeHookRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AgentBridgeHookResponse | ErrorResponse
    """

    return sync_detailed(
        bridge_id=bridge_id,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    bridge_id: str,
    *,
    client: AuthenticatedClient,
    body: AgentBridgeHookRequest,
) -> Response[AgentBridgeHookResponse | ErrorResponse]:
    """Handle provider hook callback for a daemon-owned agent bridge

    Args:
        bridge_id (str):
        body (AgentBridgeHookRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AgentBridgeHookResponse | ErrorResponse]
    """

    kwargs = _get_kwargs(
        bridge_id=bridge_id,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    bridge_id: str,
    *,
    client: AuthenticatedClient,
    body: AgentBridgeHookRequest,
) -> AgentBridgeHookResponse | ErrorResponse | None:
    """Handle provider hook callback for a daemon-owned agent bridge

    Args:
        bridge_id (str):
        body (AgentBridgeHookRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AgentBridgeHookResponse | ErrorResponse
    """

    return (
        await asyncio_detailed(
            bridge_id=bridge_id,
            client=client,
            body=body,
        )
    ).parsed
