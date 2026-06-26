from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_response import ErrorResponse
from ...models.workflow_action_availability import WorkflowActionAvailability
from ...types import Response


def _get_kwargs(
    work_item_id: str,
) -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/work-items/{work_item_id}/workflow-actions".format(
            work_item_id=quote(str(work_item_id), safe=""),
        ),
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorResponse | list[WorkflowActionAvailability]:
    if response.status_code == 200:
        response_200 = []
        _response_200 = response.json()
        for response_200_item_data in _response_200:
            response_200_item = WorkflowActionAvailability.from_dict(
                response_200_item_data
            )

            response_200.append(response_200_item)

        return response_200

    response_default = ErrorResponse.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorResponse | list[WorkflowActionAvailability]]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    work_item_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[ErrorResponse | list[WorkflowActionAvailability]]:
    """
    Args:
        work_item_id (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | list[WorkflowActionAvailability]]
    """

    kwargs = _get_kwargs(
        work_item_id=work_item_id,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    work_item_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> ErrorResponse | list[WorkflowActionAvailability] | None:
    """
    Args:
        work_item_id (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | list[WorkflowActionAvailability]
    """

    return sync_detailed(
        work_item_id=work_item_id,
        client=client,
    ).parsed


async def asyncio_detailed(
    work_item_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[ErrorResponse | list[WorkflowActionAvailability]]:
    """
    Args:
        work_item_id (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | list[WorkflowActionAvailability]]
    """

    kwargs = _get_kwargs(
        work_item_id=work_item_id,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    work_item_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> ErrorResponse | list[WorkflowActionAvailability] | None:
    """
    Args:
        work_item_id (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | list[WorkflowActionAvailability]
    """

    return (
        await asyncio_detailed(
            work_item_id=work_item_id,
            client=client,
        )
    ).parsed
