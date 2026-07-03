from http import HTTPStatus
from typing import Any

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_response import ErrorResponse
from ...models.mail_message import MailMessage
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    to: str | Unset = UNSET,
    unread: bool | Unset = UNSET,
    types: str | Unset = UNSET,
    project_id: str | Unset = UNSET,
    work_item_id: str | Unset = UNSET,
    run_id: str | Unset = UNSET,
    thread_id: str | Unset = UNSET,
    limit: int | Unset = UNSET,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["to"] = to

    params["unread"] = unread

    params["types"] = types

    params["projectId"] = project_id

    params["workItemId"] = work_item_id

    params["runId"] = run_id

    params["threadId"] = thread_id

    params["limit"] = limit

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/mail",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorResponse | list[MailMessage]:
    if response.status_code == 200:
        response_200 = []
        _response_200 = response.json()
        for response_200_item_data in _response_200:
            response_200_item = MailMessage.from_dict(response_200_item_data)

            response_200.append(response_200_item)

        return response_200

    response_default = ErrorResponse.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorResponse | list[MailMessage]]:
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
    unread: bool | Unset = UNSET,
    types: str | Unset = UNSET,
    project_id: str | Unset = UNSET,
    work_item_id: str | Unset = UNSET,
    run_id: str | Unset = UNSET,
    thread_id: str | Unset = UNSET,
    limit: int | Unset = UNSET,
) -> Response[ErrorResponse | list[MailMessage]]:
    """List daemon-owned mailbox messages

    Args:
        to (str | Unset):
        unread (bool | Unset):
        types (str | Unset):
        project_id (str | Unset):
        work_item_id (str | Unset):
        run_id (str | Unset):
        thread_id (str | Unset):
        limit (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | list[MailMessage]]
    """

    kwargs = _get_kwargs(
        to=to,
        unread=unread,
        types=types,
        project_id=project_id,
        work_item_id=work_item_id,
        run_id=run_id,
        thread_id=thread_id,
        limit=limit,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient,
    to: str | Unset = UNSET,
    unread: bool | Unset = UNSET,
    types: str | Unset = UNSET,
    project_id: str | Unset = UNSET,
    work_item_id: str | Unset = UNSET,
    run_id: str | Unset = UNSET,
    thread_id: str | Unset = UNSET,
    limit: int | Unset = UNSET,
) -> ErrorResponse | list[MailMessage] | None:
    """List daemon-owned mailbox messages

    Args:
        to (str | Unset):
        unread (bool | Unset):
        types (str | Unset):
        project_id (str | Unset):
        work_item_id (str | Unset):
        run_id (str | Unset):
        thread_id (str | Unset):
        limit (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | list[MailMessage]
    """

    return sync_detailed(
        client=client,
        to=to,
        unread=unread,
        types=types,
        project_id=project_id,
        work_item_id=work_item_id,
        run_id=run_id,
        thread_id=thread_id,
        limit=limit,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    to: str | Unset = UNSET,
    unread: bool | Unset = UNSET,
    types: str | Unset = UNSET,
    project_id: str | Unset = UNSET,
    work_item_id: str | Unset = UNSET,
    run_id: str | Unset = UNSET,
    thread_id: str | Unset = UNSET,
    limit: int | Unset = UNSET,
) -> Response[ErrorResponse | list[MailMessage]]:
    """List daemon-owned mailbox messages

    Args:
        to (str | Unset):
        unread (bool | Unset):
        types (str | Unset):
        project_id (str | Unset):
        work_item_id (str | Unset):
        run_id (str | Unset):
        thread_id (str | Unset):
        limit (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | list[MailMessage]]
    """

    kwargs = _get_kwargs(
        to=to,
        unread=unread,
        types=types,
        project_id=project_id,
        work_item_id=work_item_id,
        run_id=run_id,
        thread_id=thread_id,
        limit=limit,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient,
    to: str | Unset = UNSET,
    unread: bool | Unset = UNSET,
    types: str | Unset = UNSET,
    project_id: str | Unset = UNSET,
    work_item_id: str | Unset = UNSET,
    run_id: str | Unset = UNSET,
    thread_id: str | Unset = UNSET,
    limit: int | Unset = UNSET,
) -> ErrorResponse | list[MailMessage] | None:
    """List daemon-owned mailbox messages

    Args:
        to (str | Unset):
        unread (bool | Unset):
        types (str | Unset):
        project_id (str | Unset):
        work_item_id (str | Unset):
        run_id (str | Unset):
        thread_id (str | Unset):
        limit (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | list[MailMessage]
    """

    return (
        await asyncio_detailed(
            client=client,
            to=to,
            unread=unread,
            types=types,
            project_id=project_id,
            work_item_id=work_item_id,
            run_id=run_id,
            thread_id=thread_id,
            limit=limit,
        )
    ).parsed
