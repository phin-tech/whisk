from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_response import ErrorResponse
from ...models.mail_message import MailMessage
from ...models.mark_mail_read_request import MarkMailReadRequest
from ...types import Response


def _get_kwargs(
    mail_id: str,
    *,
    body: MarkMailReadRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/mail/{mail_id}/read".format(
            mail_id=quote(str(mail_id), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorResponse | MailMessage:
    if response.status_code == 200:
        response_200 = MailMessage.from_dict(response.json())

        return response_200

    response_default = ErrorResponse.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorResponse | MailMessage]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    mail_id: str,
    *,
    client: AuthenticatedClient,
    body: MarkMailReadRequest,
) -> Response[ErrorResponse | MailMessage]:
    """Mark a mailbox message read

    Args:
        mail_id (str):
        body (MarkMailReadRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | MailMessage]
    """

    kwargs = _get_kwargs(
        mail_id=mail_id,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    mail_id: str,
    *,
    client: AuthenticatedClient,
    body: MarkMailReadRequest,
) -> ErrorResponse | MailMessage | None:
    """Mark a mailbox message read

    Args:
        mail_id (str):
        body (MarkMailReadRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | MailMessage
    """

    return sync_detailed(
        mail_id=mail_id,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    mail_id: str,
    *,
    client: AuthenticatedClient,
    body: MarkMailReadRequest,
) -> Response[ErrorResponse | MailMessage]:
    """Mark a mailbox message read

    Args:
        mail_id (str):
        body (MarkMailReadRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | MailMessage]
    """

    kwargs = _get_kwargs(
        mail_id=mail_id,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    mail_id: str,
    *,
    client: AuthenticatedClient,
    body: MarkMailReadRequest,
) -> ErrorResponse | MailMessage | None:
    """Mark a mailbox message read

    Args:
        mail_id (str):
        body (MarkMailReadRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | MailMessage
    """

    return (
        await asyncio_detailed(
            mail_id=mail_id,
            client=client,
            body=body,
        )
    ).parsed
