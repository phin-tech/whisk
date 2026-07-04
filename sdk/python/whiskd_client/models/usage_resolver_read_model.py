from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.usage_resolver_result import UsageResolverResult


T = TypeVar("T", bound="UsageResolverReadModel")


@_attrs_define
class UsageResolverReadModel:
    """
    Attributes:
        label (str):
        plugin_id (str):
        provider (str):
        resolver_id (str):
        status (str):
        trusted (bool):
        valid (bool):
        error (str | Unset):
        min_refresh_ms (int | Unset):
        profile (str | Unset):
        refreshed_at (datetime.datetime | None | Unset):
        result (None | Unset | UsageResolverResult):
        stale (bool | Unset):
        stale_after_ms (int | Unset):
    """

    label: str
    plugin_id: str
    provider: str
    resolver_id: str
    status: str
    trusted: bool
    valid: bool
    error: str | Unset = UNSET
    min_refresh_ms: int | Unset = UNSET
    profile: str | Unset = UNSET
    refreshed_at: datetime.datetime | None | Unset = UNSET
    result: None | Unset | UsageResolverResult = UNSET
    stale: bool | Unset = UNSET
    stale_after_ms: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.usage_resolver_result import UsageResolverResult

        label = self.label

        plugin_id = self.plugin_id

        provider = self.provider

        resolver_id = self.resolver_id

        status = self.status

        trusted = self.trusted

        valid = self.valid

        error = self.error

        min_refresh_ms = self.min_refresh_ms

        profile = self.profile

        refreshed_at: None | str | Unset
        if isinstance(self.refreshed_at, Unset):
            refreshed_at = UNSET
        elif isinstance(self.refreshed_at, datetime.datetime):
            refreshed_at = self.refreshed_at.isoformat()
        else:
            refreshed_at = self.refreshed_at

        result: dict[str, Any] | None | Unset
        if isinstance(self.result, Unset):
            result = UNSET
        elif isinstance(self.result, UsageResolverResult):
            result = self.result.to_dict()
        else:
            result = self.result

        stale = self.stale

        stale_after_ms = self.stale_after_ms

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "label": label,
                "pluginId": plugin_id,
                "provider": provider,
                "resolverId": resolver_id,
                "status": status,
                "trusted": trusted,
                "valid": valid,
            }
        )
        if error is not UNSET:
            field_dict["error"] = error
        if min_refresh_ms is not UNSET:
            field_dict["minRefreshMs"] = min_refresh_ms
        if profile is not UNSET:
            field_dict["profile"] = profile
        if refreshed_at is not UNSET:
            field_dict["refreshedAt"] = refreshed_at
        if result is not UNSET:
            field_dict["result"] = result
        if stale is not UNSET:
            field_dict["stale"] = stale
        if stale_after_ms is not UNSET:
            field_dict["staleAfterMs"] = stale_after_ms

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.usage_resolver_result import UsageResolverResult

        d = dict(src_dict)
        label = d.pop("label")

        plugin_id = d.pop("pluginId")

        provider = d.pop("provider")

        resolver_id = d.pop("resolverId")

        status = d.pop("status")

        trusted = d.pop("trusted")

        valid = d.pop("valid")

        error = d.pop("error", UNSET)

        min_refresh_ms = d.pop("minRefreshMs", UNSET)

        profile = d.pop("profile", UNSET)

        def _parse_refreshed_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                refreshed_at_type_0 = datetime.datetime.fromisoformat(data)

                return refreshed_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        refreshed_at = _parse_refreshed_at(d.pop("refreshedAt", UNSET))

        def _parse_result(data: object) -> None | Unset | UsageResolverResult:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                result_type_1 = UsageResolverResult.from_dict(data)

                return result_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | Unset | UsageResolverResult, data)

        result = _parse_result(d.pop("result", UNSET))

        stale = d.pop("stale", UNSET)

        stale_after_ms = d.pop("staleAfterMs", UNSET)

        usage_resolver_read_model = cls(
            label=label,
            plugin_id=plugin_id,
            provider=provider,
            resolver_id=resolver_id,
            status=status,
            trusted=trusted,
            valid=valid,
            error=error,
            min_refresh_ms=min_refresh_ms,
            profile=profile,
            refreshed_at=refreshed_at,
            result=result,
            stale=stale,
            stale_after_ms=stale_after_ms,
        )

        usage_resolver_read_model.additional_properties = d
        return usage_resolver_read_model

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties
