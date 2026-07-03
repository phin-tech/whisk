from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_response import ErrorResponse
from ...models.resize_pty_request import ResizePTYRequest
from ...types import Response


def _get_kwargs(
    pty_id: str,
    *,
    body: ResizePTYRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/ptys/{pty_id}/resize".format(
            pty_id=quote(str(pty_id), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Any | ErrorResponse:
    if response.status_code == 204:
        response_204 = cast(Any, None)
        return response_204

    response_default = ErrorResponse.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Any | ErrorResponse]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    pty_id: str,
    *,
    client: AuthenticatedClient,
    body: ResizePTYRequest,
) -> Response[Any | ErrorResponse]:
    """
    Args:
        pty_id (str):
        body (ResizePTYRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | ErrorResponse]
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
    client: AuthenticatedClient,
    body: ResizePTYRequest,
) -> Any | ErrorResponse | None:
    """
    Args:
        pty_id (str):
        body (ResizePTYRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | ErrorResponse
    """

    return sync_detailed(
        pty_id=pty_id,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    pty_id: str,
    *,
    client: AuthenticatedClient,
    body: ResizePTYRequest,
) -> Response[Any | ErrorResponse]:
    """
    Args:
        pty_id (str):
        body (ResizePTYRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | ErrorResponse]
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
    client: AuthenticatedClient,
    body: ResizePTYRequest,
) -> Any | ErrorResponse | None:
    """
    Args:
        pty_id (str):
        body (ResizePTYRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | ErrorResponse
    """

    return (
        await asyncio_detailed(
            pty_id=pty_id,
            client=client,
            body=body,
        )
    ).parsed
