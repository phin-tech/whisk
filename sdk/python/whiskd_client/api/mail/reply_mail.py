from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_response import ErrorResponse
from ...models.mail_message import MailMessage
from ...models.reply_mail_request import ReplyMailRequest
from ...types import Response


def _get_kwargs(
    mail_id: str,
    *,
    body: ReplyMailRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/mail/{mail_id}/reply".format(
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
    if response.status_code == 201:
        response_201 = MailMessage.from_dict(response.json())

        return response_201

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
    body: ReplyMailRequest,
) -> Response[ErrorResponse | MailMessage]:
    """Reply to a mailbox message

    Args:
        mail_id (str):
        body (ReplyMailRequest):

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
    body: ReplyMailRequest,
) -> ErrorResponse | MailMessage | None:
    """Reply to a mailbox message

    Args:
        mail_id (str):
        body (ReplyMailRequest):

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
    body: ReplyMailRequest,
) -> Response[ErrorResponse | MailMessage]:
    """Reply to a mailbox message

    Args:
        mail_id (str):
        body (ReplyMailRequest):

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
    body: ReplyMailRequest,
) -> ErrorResponse | MailMessage | None:
    """Reply to a mailbox message

    Args:
        mail_id (str):
        body (ReplyMailRequest):

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
