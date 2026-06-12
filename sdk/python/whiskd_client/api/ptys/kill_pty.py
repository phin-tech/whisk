from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_response import ErrorResponse
from ...models.kill_pty_request import KillPTYRequest
from ...models.pty_info import PTYInfo
from ...types import Response


def _get_kwargs(
    pty_id: str,
    *,
    body: KillPTYRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/ptys/{pty_id}/kill".format(
            pty_id=quote(str(pty_id), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorResponse | PTYInfo:
    if response.status_code == 200:
        response_200 = PTYInfo.from_dict(response.json())

        return response_200

    response_default = ErrorResponse.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorResponse | PTYInfo]:
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
    body: KillPTYRequest,
) -> Response[ErrorResponse | PTYInfo]:
    """
    Args:
        pty_id (str):
        body (KillPTYRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | PTYInfo]
    """

    kwargs = _get_kwargs(
        pty_id=pty_id,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    pty_id: str,
    *,
    client: AuthenticatedClient | Client,
    body: KillPTYRequest,
) -> ErrorResponse | PTYInfo | None:
    """
    Args:
        pty_id (str):
        body (KillPTYRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | PTYInfo
    """

    return sync_detailed(
        pty_id=pty_id,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    pty_id: str,
    *,
    client: AuthenticatedClient | Client,
    body: KillPTYRequest,
) -> Response[ErrorResponse | PTYInfo]:
    """
    Args:
        pty_id (str):
        body (KillPTYRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | PTYInfo]
    """

    kwargs = _get_kwargs(
        pty_id=pty_id,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    pty_id: str,
    *,
    client: AuthenticatedClient | Client,
    body: KillPTYRequest,
) -> ErrorResponse | PTYInfo | None:
    """
    Args:
        pty_id (str):
        body (KillPTYRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | PTYInfo
    """

    return (
        await asyncio_detailed(
            pty_id=pty_id,
            client=client,
            body=body,
        )
    ).parsed
