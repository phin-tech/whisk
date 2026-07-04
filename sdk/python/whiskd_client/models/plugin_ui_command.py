from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="PluginUICommand")


@_attrs_define
class PluginUICommand:
    """
    Attributes:
        id (str):
        label (str):
        scope (str):
        output_cap_bytes (int | Unset):
        timeout_ms (int | Unset):
    """

    id: str
    label: str
    scope: str
    output_cap_bytes: int | Unset = UNSET
    timeout_ms: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id = self.id

        label = self.label

        scope = self.scope

        output_cap_bytes = self.output_cap_bytes

        timeout_ms = self.timeout_ms

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "id": id,
                "label": label,
                "scope": scope,
            }
        )
        if output_cap_bytes is not UNSET:
            field_dict["outputCapBytes"] = output_cap_bytes
        if timeout_ms is not UNSET:
            field_dict["timeoutMs"] = timeout_ms

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        label = d.pop("label")

        scope = d.pop("scope")

        output_cap_bytes = d.pop("outputCapBytes", UNSET)

        timeout_ms = d.pop("timeoutMs", UNSET)

        plugin_ui_command = cls(
            id=id,
            label=label,
            scope=scope,
            output_cap_bytes=output_cap_bytes,
            timeout_ms=timeout_ms,
        )

        plugin_ui_command.additional_properties = d
        return plugin_ui_command

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
