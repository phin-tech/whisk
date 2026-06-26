from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_response import ErrorResponse
from ...models.workflow_definition_record import WorkflowDefinitionRecord
from ...types import Response


def _get_kwargs(
    workflow_id: str,
    version: str,
) -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/workflow-definitions/{workflow_id}/{version}/delete".format(
            workflow_id=quote(str(workflow_id), safe=""),
            version=quote(str(version), safe=""),
        ),
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorResponse | WorkflowDefinitionRecord:
    if response.status_code == 200:
        response_200 = WorkflowDefinitionRecord.from_dict(response.json())

        return response_200

    response_default = ErrorResponse.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorResponse | WorkflowDefinitionRecord]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workflow_id: str,
    version: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[ErrorResponse | WorkflowDefinitionRecord]:
    """
    Args:
        workflow_id (str):
        version (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | WorkflowDefinitionRecord]
    """

    kwargs = _get_kwargs(
        workflow_id=workflow_id,
        version=version,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    workflow_id: str,
    version: str,
    *,
    client: AuthenticatedClient | Client,
) -> ErrorResponse | WorkflowDefinitionRecord | None:
    """
    Args:
        workflow_id (str):
        version (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | WorkflowDefinitionRecord
    """

    return sync_detailed(
        workflow_id=workflow_id,
        version=version,
        client=client,
    ).parsed


async def asyncio_detailed(
    workflow_id: str,
    version: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[ErrorResponse | WorkflowDefinitionRecord]:
    """
    Args:
        workflow_id (str):
        version (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | WorkflowDefinitionRecord]
    """

    kwargs = _get_kwargs(
        workflow_id=workflow_id,
        version=version,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    workflow_id: str,
    version: str,
    *,
    client: AuthenticatedClient | Client,
) -> ErrorResponse | WorkflowDefinitionRecord | None:
    """
    Args:
        workflow_id (str):
        version (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | WorkflowDefinitionRecord
    """

    return (
        await asyncio_detailed(
            workflow_id=workflow_id,
            version=version,
            client=client,
        )
    ).parsed
