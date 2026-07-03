from http import HTTPStatus
from typing import Any

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_response import ErrorResponse
from ...models.next_event_response import NextEventResponse
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    timeout_ms: int | Unset = UNSET,
    after_seq: int | Unset = UNSET,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["timeoutMs"] = timeout_ms

    params["afterSeq"] = after_seq

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/events/next",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorResponse | NextEventResponse:
    if response.status_code == 200:
        response_200 = NextEventResponse.from_dict(response.json())

        return response_200

    response_default = ErrorResponse.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorResponse | NextEventResponse]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    timeout_ms: int | Unset = UNSET,
    after_seq: int | Unset = UNSET,
) -> Response[ErrorResponse | NextEventResponse]:
    """
    Args:
        timeout_ms (int | Unset):
        after_seq (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | NextEventResponse]
    """

    kwargs = _get_kwargs(
        timeout_ms=timeout_ms,
        after_seq=after_seq,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient,
    timeout_ms: int | Unset = UNSET,
    after_seq: int | Unset = UNSET,
) -> ErrorResponse | NextEventResponse | None:
    """
    Args:
        timeout_ms (int | Unset):
        after_seq (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | NextEventResponse
    """

    return sync_detailed(
        client=client,
        timeout_ms=timeout_ms,
        after_seq=after_seq,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    timeout_ms: int | Unset = UNSET,
    after_seq: int | Unset = UNSET,
) -> Response[ErrorResponse | NextEventResponse]:
    """
    Args:
        timeout_ms (int | Unset):
        after_seq (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | NextEventResponse]
    """

    kwargs = _get_kwargs(
        timeout_ms=timeout_ms,
        after_seq=after_seq,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient,
    timeout_ms: int | Unset = UNSET,
    after_seq: int | Unset = UNSET,
) -> ErrorResponse | NextEventResponse | None:
    """
    Args:
        timeout_ms (int | Unset):
        after_seq (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | NextEventResponse
    """

    return (
        await asyncio_detailed(
            client=client,
            timeout_ms=timeout_ms,
            after_seq=after_seq,
        )
    ).parsed
