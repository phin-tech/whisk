from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_response import ErrorResponse
from ...models.pty_history import PTYHistory
from ...types import Response


def _get_kwargs(
    pty_id: str,
) -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/pty-history/{pty_id}".format(
            pty_id=quote(str(pty_id), safe=""),
        ),
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorResponse | PTYHistory:
    if response.status_code == 200:
        response_200 = PTYHistory.from_dict(response.json())

        return response_200

    response_default = ErrorResponse.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorResponse | PTYHistory]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    pty_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[ErrorResponse | PTYHistory]:
    """Read one persisted PTY transcript

    Args:
        pty_id (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | PTYHistory]
    """

    kwargs = _get_kwargs(
        pty_id=pty_id,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    pty_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> ErrorResponse | PTYHistory | None:
    """Read one persisted PTY transcript

    Args:
        pty_id (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | PTYHistory
    """

    return sync_detailed(
        pty_id=pty_id,
        client=client,
    ).parsed


async def asyncio_detailed(
    pty_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[ErrorResponse | PTYHistory]:
    """Read one persisted PTY transcript

    Args:
        pty_id (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | PTYHistory]
    """

    kwargs = _get_kwargs(
        pty_id=pty_id,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    pty_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> ErrorResponse | PTYHistory | None:
    """Read one persisted PTY transcript

    Args:
        pty_id (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | PTYHistory
    """

    return (
        await asyncio_detailed(
            pty_id=pty_id,
            client=client,
        )
    ).parsed
