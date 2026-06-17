from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_response import ErrorResponse
from ...models.project import Project
from ...models.run_plugin_project_attachment_template_request import (
    RunPluginProjectAttachmentTemplateRequest,
)
from ...types import Response


def _get_kwargs(
    plugin_id: str,
    template_id: str,
    *,
    body: RunPluginProjectAttachmentTemplateRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/plugins/{plugin_id}/project-attachment-templates/{template_id}".format(
            plugin_id=quote(str(plugin_id), safe=""),
            template_id=quote(str(template_id), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorResponse | Project:
    if response.status_code == 201:
        response_201 = Project.from_dict(response.json())

        return response_201

    response_default = ErrorResponse.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorResponse | Project]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    plugin_id: str,
    template_id: str,
    *,
    client: AuthenticatedClient | Client,
    body: RunPluginProjectAttachmentTemplateRequest,
) -> Response[ErrorResponse | Project]:
    """Run a trusted plugin project attachment template

    Args:
        plugin_id (str):
        template_id (str):
        body (RunPluginProjectAttachmentTemplateRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | Project]
    """

    kwargs = _get_kwargs(
        plugin_id=plugin_id,
        template_id=template_id,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    plugin_id: str,
    template_id: str,
    *,
    client: AuthenticatedClient | Client,
    body: RunPluginProjectAttachmentTemplateRequest,
) -> ErrorResponse | Project | None:
    """Run a trusted plugin project attachment template

    Args:
        plugin_id (str):
        template_id (str):
        body (RunPluginProjectAttachmentTemplateRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | Project
    """

    return sync_detailed(
        plugin_id=plugin_id,
        template_id=template_id,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    plugin_id: str,
    template_id: str,
    *,
    client: AuthenticatedClient | Client,
    body: RunPluginProjectAttachmentTemplateRequest,
) -> Response[ErrorResponse | Project]:
    """Run a trusted plugin project attachment template

    Args:
        plugin_id (str):
        template_id (str):
        body (RunPluginProjectAttachmentTemplateRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | Project]
    """

    kwargs = _get_kwargs(
        plugin_id=plugin_id,
        template_id=template_id,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    plugin_id: str,
    template_id: str,
    *,
    client: AuthenticatedClient | Client,
    body: RunPluginProjectAttachmentTemplateRequest,
) -> ErrorResponse | Project | None:
    """Run a trusted plugin project attachment template

    Args:
        plugin_id (str):
        template_id (str):
        body (RunPluginProjectAttachmentTemplateRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | Project
    """

    return (
        await asyncio_detailed(
            plugin_id=plugin_id,
            template_id=template_id,
            client=client,
            body=body,
        )
    ).parsed
