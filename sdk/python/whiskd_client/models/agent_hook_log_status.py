from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

T = TypeVar("T", bound="AgentHookLogStatus")


@_attrs_define
class AgentHookLogStatus:
    """
    Attributes:
        clear_after_session (bool):
        enabled (bool):
        path (str):
        size_bytes (int):
    """

    clear_after_session: bool
    enabled: bool
    path: str
    size_bytes: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        clear_after_session = self.clear_after_session

        enabled = self.enabled

        path = self.path

        size_bytes = self.size_bytes

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "clearAfterSession": clear_after_session,
                "enabled": enabled,
                "path": path,
                "sizeBytes": size_bytes,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        clear_after_session = d.pop("clearAfterSession")

        enabled = d.pop("enabled")

        path = d.pop("path")

        size_bytes = d.pop("sizeBytes")

        agent_hook_log_status = cls(
            clear_after_session=clear_after_session,
            enabled=enabled,
            path=path,
            size_bytes=size_bytes,
        )

        agent_hook_log_status.additional_properties = d
        return agent_hook_log_status

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
