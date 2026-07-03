from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.complete_gate_request import CompleteGateRequest
from ...models.error_response import ErrorResponse
from ...models.gate_report import GateReport
from ...types import Response


def _get_kwargs(
    gate_report_id: str,
    *,
    body: CompleteGateRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/gate-reports/{gate_report_id}/complete".format(
            gate_report_id=quote(str(gate_report_id), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorResponse | GateReport:
    if response.status_code == 200:
        response_200 = GateReport.from_dict(response.json())

        return response_200

    response_default = ErrorResponse.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorResponse | GateReport]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    gate_report_id: str,
    *,
    client: AuthenticatedClient,
    body: CompleteGateRequest,
) -> Response[ErrorResponse | GateReport]:
    """
    Args:
        gate_report_id (str):
        body (CompleteGateRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | GateReport]
    """

    kwargs = _get_kwargs(
        gate_report_id=gate_report_id,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    gate_report_id: str,
    *,
    client: AuthenticatedClient,
    body: CompleteGateRequest,
) -> ErrorResponse | GateReport | None:
    """
    Args:
        gate_report_id (str):
        body (CompleteGateRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | GateReport
    """

    return sync_detailed(
        gate_report_id=gate_report_id,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    gate_report_id: str,
    *,
    client: AuthenticatedClient,
    body: CompleteGateRequest,
) -> Response[ErrorResponse | GateReport]:
    """
    Args:
        gate_report_id (str):
        body (CompleteGateRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorResponse | GateReport]
    """

    kwargs = _get_kwargs(
        gate_report_id=gate_report_id,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    gate_report_id: str,
    *,
    client: AuthenticatedClient,
    body: CompleteGateRequest,
) -> ErrorResponse | GateReport | None:
    """
    Args:
        gate_report_id (str):
        body (CompleteGateRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorResponse | GateReport
    """

    return (
        await asyncio_detailed(
            gate_report_id=gate_report_id,
            client=client,
            body=body,
        )
    ).parsed
