from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.agent_prompt import AgentPrompt
from ...models.error_response import ErrorResponse
from ...models.resolve_agent_prompt_request import ResolveAgentPromptRequest
from ...types import Response


def _get_kwargs(
    prompt_id: str,
    *,
    body: ResolveAgentPromptRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/agent-prompts/{prompt_id}/resolve".format(
            prompt_id=quote(str(prompt_id), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> AgentPrompt | ErrorResponse:
    if response.status_code == 200:
        response_200 = AgentPrompt.from_dict(response.json())

        return response_200

    response_default = ErrorResponse.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[AgentPrompt | ErrorResponse]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    prompt_id: str,
    *,
    client: AuthenticatedClient,
    body: ResolveAgentPromptRequest,
) -> Response[AgentPrompt | ErrorResponse]:
    """Resolve a pending daemon-owned agent prompt

    Args:
        prompt_id (str):
        body (ResolveAgentPromptRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AgentPrompt | ErrorResponse]
    """

    kwargs = _get_kwargs(
        prompt_id=prompt_id,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    prompt_id: str,
    *,
    client: AuthenticatedClient,
    body: ResolveAgentPromptRequest,
) -> AgentPrompt | ErrorResponse | None:
    """Resolve a pending daemon-owned agent prompt

    Args:
        prompt_id (str):
        body (ResolveAgentPromptRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AgentPrompt | ErrorResponse
    """

    return sync_detailed(
        prompt_id=prompt_id,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    prompt_id: str,
    *,
    client: AuthenticatedClient,
    body: ResolveAgentPromptRequest,
) -> Response[AgentPrompt | ErrorResponse]:
    """Resolve a pending daemon-owned agent prompt

    Args:
        prompt_id (str):
        body (ResolveAgentPromptRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AgentPrompt | ErrorResponse]
    """

    kwargs = _get_kwargs(
        prompt_id=prompt_id,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    prompt_id: str,
    *,
    client: AuthenticatedClient,
    body: ResolveAgentPromptRequest,
) -> AgentPrompt | ErrorResponse | None:
    """Resolve a pending daemon-owned agent prompt

    Args:
        prompt_id (str):
        body (ResolveAgentPromptRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AgentPrompt | ErrorResponse
    """

    return (
        await asyncio_detailed(
            prompt_id=prompt_id,
            client=client,
            body=body,
        )
    ).parsed
