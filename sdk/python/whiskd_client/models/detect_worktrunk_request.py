from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

T = TypeVar("T", bound="DetectWorktrunkRequest")


@_attrs_define
class DetectWorktrunkRequest:
    """
    Attributes:
        override_path (str):
        repo_path (str):
    """

    override_path: str
    repo_path: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        override_path = self.override_path

        repo_path = self.repo_path

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "overridePath": override_path,
                "repoPath": repo_path,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        override_path = d.pop("overridePath")

        repo_path = d.pop("repoPath")

        detect_worktrunk_request = cls(
            override_path=override_path,
            repo_path=repo_path,
        )

        detect_worktrunk_request.additional_properties = d
        return detect_worktrunk_request

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
