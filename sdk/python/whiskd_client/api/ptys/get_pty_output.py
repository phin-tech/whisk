from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_response import ErrorResponse
from ...models.output_snapshot import OutputSnapshot
from ...types import UNSET, Response, Unset


def _get_kwargs(
    pty_id: str,
    *,
    from_: int | Unset = UNSET,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["from"] = from_

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/ptys/{pty_id}/output".format(
            pty_id=quote(str(pty_id), safe=""),
        ),
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorResponse | OutputSnapshot:
    if response.status_code == 200:
        response_200 = OutputSnapshot.from_dict(response.json())

        return response_200

    response_default = ErrorResponse.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorResponse | OutputSnapshot]:
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
    from_: int | Unset = UNSET,
) -> Response[ErrorResponse | OutputSnapshot]:
    """
    Args:
        pty_id (str):
        from_ (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | OutputSnapshot]
    """

    kwargs = _get_kwargs(
        pty_id=pty_id,
        from_=from_,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    pty_id: str,
    *,
    client: AuthenticatedClient | Client,
    from_: int | Unset = UNSET,
) -> ErrorResponse | OutputSnapshot | None:
    """
    Args:
        pty_id (str):
        from_ (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | OutputSnapshot
    """

    return sync_detailed(
        pty_id=pty_id,
        client=client,
        from_=from_,
    ).parsed


async def asyncio_detailed(
    pty_id: str,
    *,
    client: AuthenticatedClient | Client,
    from_: int | Unset = UNSET,
) -> Response[ErrorResponse | OutputSnapshot]:
    """
    Args:
        pty_id (str):
        from_ (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | OutputSnapshot]
    """

    kwargs = _get_kwargs(
        pty_id=pty_id,
        from_=from_,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    pty_id: str,
    *,
    client: AuthenticatedClient | Client,
    from_: int | Unset = UNSET,
) -> ErrorResponse | OutputSnapshot | None:
    """
    Args:
        pty_id (str):
        from_ (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | OutputSnapshot
    """

    return (
        await asyncio_detailed(
            pty_id=pty_id,
            client=client,
            from_=from_,
        )
    ).parsed
