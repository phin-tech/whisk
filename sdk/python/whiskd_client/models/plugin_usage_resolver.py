from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="PluginUsageResolver")


@_attrs_define
class PluginUsageResolver:
    """
    Attributes:
        id (str):
        label (str):
        provider (str):
        min_refresh_ms (int | Unset):
        output_cap_bytes (int | Unset):
        profiles (list[str] | Unset):
        stale_after_ms (int | Unset):
        timeout_ms (int | Unset):
    """

    id: str
    label: str
    provider: str
    min_refresh_ms: int | Unset = UNSET
    output_cap_bytes: int | Unset = UNSET
    profiles: list[str] | Unset = UNSET
    stale_after_ms: int | Unset = UNSET
    timeout_ms: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id = self.id

        label = self.label

        provider = self.provider

        min_refresh_ms = self.min_refresh_ms

        output_cap_bytes = self.output_cap_bytes

        profiles: list[str] | Unset = UNSET
        if not isinstance(self.profiles, Unset):
            profiles = self.profiles

        stale_after_ms = self.stale_after_ms

        timeout_ms = self.timeout_ms

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "id": id,
                "label": label,
                "provider": provider,
            }
        )
        if min_refresh_ms is not UNSET:
            field_dict["minRefreshMs"] = min_refresh_ms
        if output_cap_bytes is not UNSET:
            field_dict["outputCapBytes"] = output_cap_bytes
        if profiles is not UNSET:
            field_dict["profiles"] = profiles
        if stale_after_ms is not UNSET:
            field_dict["staleAfterMs"] = stale_after_ms
        if timeout_ms is not UNSET:
            field_dict["timeoutMs"] = timeout_ms

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        label = d.pop("label")

        provider = d.pop("provider")

        min_refresh_ms = d.pop("minRefreshMs", UNSET)

        output_cap_bytes = d.pop("outputCapBytes", UNSET)

        profiles = cast(list[str], d.pop("profiles", UNSET))

        stale_after_ms = d.pop("staleAfterMs", UNSET)

        timeout_ms = d.pop("timeoutMs", UNSET)

        plugin_usage_resolver = cls(
            id=id,
            label=label,
            provider=provider,
            min_refresh_ms=min_refresh_ms,
            output_cap_bytes=output_cap_bytes,
            profiles=profiles,
            stale_after_ms=stale_after_ms,
            timeout_ms=timeout_ms,
        )

        plugin_usage_resolver.additional_properties = d
        return plugin_usage_resolver

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
