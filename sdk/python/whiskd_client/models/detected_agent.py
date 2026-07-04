from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

T = TypeVar("T", bound="DetectedAgent")


@_attrs_define
class DetectedAgent:
    """
    Attributes:
        detect_command (str):
        label (str):
        path (str):
        profile_id (str):
        provider (str):
    """

    detect_command: str
    label: str
    path: str
    profile_id: str
    provider: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        detect_command = self.detect_command

        label = self.label

        path = self.path

        profile_id = self.profile_id

        provider = self.provider

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "detectCommand": detect_command,
                "label": label,
                "path": path,
                "profileId": profile_id,
                "provider": provider,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        detect_command = d.pop("detectCommand")

        label = d.pop("label")

        path = d.pop("path")

        profile_id = d.pop("profileId")

        provider = d.pop("provider")

        detected_agent = cls(
            detect_command=detect_command,
            label=label,
            path=path,
            profile_id=profile_id,
            provider=provider,
        )

        detected_agent.additional_properties = d
        return detected_agent

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
