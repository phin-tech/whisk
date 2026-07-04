from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_response import ErrorResponse
from ...models.refresh_usage_resolver_request import RefreshUsageResolverRequest
from ...models.usage_resolver_read_model import UsageResolverReadModel
from ...types import Response


def _get_kwargs(
    plugin_id: str,
    resolver_id: str,
    *,
    body: RefreshUsageResolverRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/plugins/{plugin_id}/usage-resolvers/{resolver_id}/refresh".format(
            plugin_id=quote(str(plugin_id), safe=""),
            resolver_id=quote(str(resolver_id), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorResponse | UsageResolverReadModel:
    if response.status_code == 200:
        response_200 = UsageResolverReadModel.from_dict(response.json())

        return response_200

    response_default = ErrorResponse.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorResponse | UsageResolverReadModel]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    plugin_id: str,
    resolver_id: str,
    *,
    client: AuthenticatedClient,
    body: RefreshUsageResolverRequest,
) -> Response[ErrorResponse | UsageResolverReadModel]:
    """Refresh one trusted plugin usage resolver

    Args:
        plugin_id (str):
        resolver_id (str):
        body (RefreshUsageResolverRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | UsageResolverReadModel]
    """

    kwargs = _get_kwargs(
        plugin_id=plugin_id,
        resolver_id=resolver_id,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    plugin_id: str,
    resolver_id: str,
    *,
    client: AuthenticatedClient,
    body: RefreshUsageResolverRequest,
) -> ErrorResponse | UsageResolverReadModel | None:
    """Refresh one trusted plugin usage resolver

    Args:
        plugin_id (str):
        resolver_id (str):
        body (RefreshUsageResolverRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | UsageResolverReadModel
    """

    return sync_detailed(
        plugin_id=plugin_id,
        resolver_id=resolver_id,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    plugin_id: str,
    resolver_id: str,
    *,
    client: AuthenticatedClient,
    body: RefreshUsageResolverRequest,
) -> Response[ErrorResponse | UsageResolverReadModel]:
    """Refresh one trusted plugin usage resolver

    Args:
        plugin_id (str):
        resolver_id (str):
        body (RefreshUsageResolverRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | UsageResolverReadModel]
    """

    kwargs = _get_kwargs(
        plugin_id=plugin_id,
        resolver_id=resolver_id,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    plugin_id: str,
    resolver_id: str,
    *,
    client: AuthenticatedClient,
    body: RefreshUsageResolverRequest,
) -> ErrorResponse | UsageResolverReadModel | None:
    """Refresh one trusted plugin usage resolver

    Args:
        plugin_id (str):
        resolver_id (str):
        body (RefreshUsageResolverRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | UsageResolverReadModel
    """

    return (
        await asyncio_detailed(
            plugin_id=plugin_id,
            resolver_id=resolver_id,
            client=client,
            body=body,
        )
    ).parsed
