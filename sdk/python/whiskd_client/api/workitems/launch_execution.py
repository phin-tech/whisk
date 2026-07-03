from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_response import ErrorResponse
from ...models.launch_execution_request import LaunchExecutionRequest
from ...models.work_item_run import WorkItemRun
from ...types import Response


def _get_kwargs(
    work_item_id: str,
    *,
    body: LaunchExecutionRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/work-items/{work_item_id}/launch-execution".format(
            work_item_id=quote(str(work_item_id), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorResponse | WorkItemRun:
    if response.status_code == 201:
        response_201 = WorkItemRun.from_dict(response.json())

        return response_201

    response_default = ErrorResponse.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorResponse | WorkItemRun]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    work_item_id: str,
    *,
    client: AuthenticatedClient,
    body: LaunchExecutionRequest,
) -> Response[ErrorResponse | WorkItemRun]:
    """
    Args:
        work_item_id (str):
        body (LaunchExecutionRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | WorkItemRun]
    """

    kwargs = _get_kwargs(
        work_item_id=work_item_id,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    work_item_id: str,
    *,
    client: AuthenticatedClient,
    body: LaunchExecutionRequest,
) -> ErrorResponse | WorkItemRun | None:
    """
    Args:
        work_item_id (str):
        body (LaunchExecutionRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | WorkItemRun
    """

    return sync_detailed(
        work_item_id=work_item_id,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    work_item_id: str,
    *,
    client: AuthenticatedClient,
    body: LaunchExecutionRequest,
) -> Response[ErrorResponse | WorkItemRun]:
    """
    Args:
        work_item_id (str):
        body (LaunchExecutionRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | WorkItemRun]
    """

    kwargs = _get_kwargs(
        work_item_id=work_item_id,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    work_item_id: str,
    *,
    client: AuthenticatedClient,
    body: LaunchExecutionRequest,
) -> ErrorResponse | WorkItemRun | None:
    """
    Args:
        work_item_id (str):
        body (LaunchExecutionRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | WorkItemRun
    """

    return (
        await asyncio_detailed(
            work_item_id=work_item_id,
            client=client,
            body=body,
        )
    ).parsed
