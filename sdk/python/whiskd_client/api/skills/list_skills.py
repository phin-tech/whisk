from http import HTTPStatus
from typing import Any

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_response import ErrorResponse
from ...models.skill_catalog import SkillCatalog
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    project_id: str | Unset = UNSET,
    session_id: str | Unset = UNSET,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["projectId"] = project_id

    params["sessionId"] = session_id

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/skills",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorResponse | SkillCatalog:
    if response.status_code == 200:
        response_200 = SkillCatalog.from_dict(response.json())

        return response_200

    response_default = ErrorResponse.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorResponse | SkillCatalog]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    project_id: str | Unset = UNSET,
    session_id: str | Unset = UNSET,
) -> Response[ErrorResponse | SkillCatalog]:
    """List daemon-discovered agent skills

    Args:
        project_id (str | Unset):
        session_id (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | SkillCatalog]
    """

    kwargs = _get_kwargs(
        project_id=project_id,
        session_id=session_id,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient,
    project_id: str | Unset = UNSET,
    session_id: str | Unset = UNSET,
) -> ErrorResponse | SkillCatalog | None:
    """List daemon-discovered agent skills

    Args:
        project_id (str | Unset):
        session_id (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | SkillCatalog
    """

    return sync_detailed(
        client=client,
        project_id=project_id,
        session_id=session_id,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    project_id: str | Unset = UNSET,
    session_id: str | Unset = UNSET,
) -> Response[ErrorResponse | SkillCatalog]:
    """List daemon-discovered agent skills

    Args:
        project_id (str | Unset):
        session_id (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | SkillCatalog]
    """

    kwargs = _get_kwargs(
        project_id=project_id,
        session_id=session_id,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient,
    project_id: str | Unset = UNSET,
    session_id: str | Unset = UNSET,
) -> ErrorResponse | SkillCatalog | None:
    """List daemon-discovered agent skills

    Args:
        project_id (str | Unset):
        session_id (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | SkillCatalog
    """

    return (
        await asyncio_detailed(
            client=client,
            project_id=project_id,
            session_id=session_id,
        )
    ).parsed
