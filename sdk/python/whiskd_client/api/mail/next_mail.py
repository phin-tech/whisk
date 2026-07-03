from http import HTTPStatus
from typing import Any

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_response import ErrorResponse
from ...models.next_mail_response import NextMailResponse
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    to: str | Unset = UNSET,
    types: str | Unset = UNSET,
    timeout_ms: int | Unset = UNSET,
    project_id: str | Unset = UNSET,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["to"] = to

    params["types"] = types

    params["timeoutMs"] = timeout_ms

    params["projectId"] = project_id

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/mail/next",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorResponse | NextMailResponse:
    if response.status_code == 200:
        response_200 = NextMailResponse.from_dict(response.json())

        return response_200

    response_default = ErrorResponse.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorResponse | NextMailResponse]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    to: str | Unset = UNSET,
    types: str | Unset = UNSET,
    timeout_ms: int | Unset = UNSET,
    project_id: str | Unset = UNSET,
) -> Response[ErrorResponse | NextMailResponse]:
    """Get the next unread mailbox message, optionally waiting

    Args:
        to (str | Unset):
        types (str | Unset):
        timeout_ms (int | Unset):
        project_id (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | NextMailResponse]
    """

    kwargs = _get_kwargs(
        to=to,
        types=types,
        timeout_ms=timeout_ms,
        project_id=project_id,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient,
    to: str | Unset = UNSET,
    types: str | Unset = UNSET,
    timeout_ms: int | Unset = UNSET,
    project_id: str | Unset = UNSET,
) -> ErrorResponse | NextMailResponse | None:
    """Get the next unread mailbox message, optionally waiting

    Args:
        to (str | Unset):
        types (str | Unset):
        timeout_ms (int | Unset):
        project_id (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | NextMailResponse
    """

    return sync_detailed(
        client=client,
        to=to,
        types=types,
        timeout_ms=timeout_ms,
        project_id=project_id,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    to: str | Unset = UNSET,
    types: str | Unset = UNSET,
    timeout_ms: int | Unset = UNSET,
    project_id: str | Unset = UNSET,
) -> Response[ErrorResponse | NextMailResponse]:
    """Get the next unread mailbox message, optionally waiting

    Args:
        to (str | Unset):
        types (str | Unset):
        timeout_ms (int | Unset):
        project_id (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | NextMailResponse]
    """

    kwargs = _get_kwargs(
        to=to,
        types=types,
        timeout_ms=timeout_ms,
        project_id=project_id,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient,
    to: str | Unset = UNSET,
    types: str | Unset = UNSET,
    timeout_ms: int | Unset = UNSET,
    project_id: str | Unset = UNSET,
) -> ErrorResponse | NextMailResponse | None:
    """Get the next unread mailbox message, optionally waiting

    Args:
        to (str | Unset):
        types (str | Unset):
        timeout_ms (int | Unset):
        project_id (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | NextMailResponse
    """

    return (
        await asyncio_detailed(
            client=client,
            to=to,
            types=types,
            timeout_ms=timeout_ms,
            project_id=project_id,
        )
    ).parsed
