from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.agent_bridge_approval import AgentBridgeApproval
from ...models.error_response import ErrorResponse
from ...models.resolve_agent_bridge_approval_request import (
    ResolveAgentBridgeApprovalRequest,
)
from ...types import Response


def _get_kwargs(
    approval_id: str,
    *,
    body: ResolveAgentBridgeApprovalRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/agent-bridge-approvals/{approval_id}/resolve".format(
            approval_id=quote(str(approval_id), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> AgentBridgeApproval | ErrorResponse:
    if response.status_code == 200:
        response_200 = AgentBridgeApproval.from_dict(response.json())

        return response_200

    response_default = ErrorResponse.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[AgentBridgeApproval | ErrorResponse]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    approval_id: str,
    *,
    client: AuthenticatedClient | Client,
    body: ResolveAgentBridgeApprovalRequest,
) -> Response[AgentBridgeApproval | ErrorResponse]:
    """Resolve a pending daemon-owned agent bridge approval

    Args:
        approval_id (str):
        body (ResolveAgentBridgeApprovalRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AgentBridgeApproval | ErrorResponse]
    """

    kwargs = _get_kwargs(
        approval_id=approval_id,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    approval_id: str,
    *,
    client: AuthenticatedClient | Client,
    body: ResolveAgentBridgeApprovalRequest,
) -> AgentBridgeApproval | ErrorResponse | None:
    """Resolve a pending daemon-owned agent bridge approval

    Args:
        approval_id (str):
        body (ResolveAgentBridgeApprovalRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AgentBridgeApproval | ErrorResponse
    """

    return sync_detailed(
        approval_id=approval_id,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    approval_id: str,
    *,
    client: AuthenticatedClient | Client,
    body: ResolveAgentBridgeApprovalRequest,
) -> Response[AgentBridgeApproval | ErrorResponse]:
    """Resolve a pending daemon-owned agent bridge approval

    Args:
        approval_id (str):
        body (ResolveAgentBridgeApprovalRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AgentBridgeApproval | ErrorResponse]
    """

    kwargs = _get_kwargs(
        approval_id=approval_id,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    approval_id: str,
    *,
    client: AuthenticatedClient | Client,
    body: ResolveAgentBridgeApprovalRequest,
) -> AgentBridgeApproval | ErrorResponse | None:
    """Resolve a pending daemon-owned agent bridge approval

    Args:
        approval_id (str):
        body (ResolveAgentBridgeApprovalRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AgentBridgeApproval | ErrorResponse
    """

    return (
        await asyncio_detailed(
            approval_id=approval_id,
            client=client,
            body=body,
        )
    ).parsed
