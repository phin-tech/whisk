from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="CompatibilityResponse")


@_attrs_define
class CompatibilityResponse:
    """
    Attributes:
        api_version (int):
        git_sha (str):
        dirty (bool | Unset):
        protocol_version (int | Unset):
        supported_previous_protocol_versions (list[int] | Unset):
        version (str | Unset):
    """

    api_version: int
    git_sha: str
    dirty: bool | Unset = UNSET
    protocol_version: int | Unset = UNSET
    supported_previous_protocol_versions: list[int] | Unset = UNSET
    version: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        api_version = self.api_version

        git_sha = self.git_sha

        dirty = self.dirty

        protocol_version = self.protocol_version

        supported_previous_protocol_versions: list[int] | Unset = UNSET
        if not isinstance(self.supported_previous_protocol_versions, Unset):
            supported_previous_protocol_versions = (
                self.supported_previous_protocol_versions
            )

        version = self.version

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "apiVersion": api_version,
                "gitSha": git_sha,
            }
        )
        if dirty is not UNSET:
            field_dict["dirty"] = dirty
        if protocol_version is not UNSET:
            field_dict["protocolVersion"] = protocol_version
        if supported_previous_protocol_versions is not UNSET:
            field_dict["supportedPreviousProtocolVersions"] = (
                supported_previous_protocol_versions
            )
        if version is not UNSET:
            field_dict["version"] = version

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        api_version = d.pop("apiVersion")

        git_sha = d.pop("gitSha")

        dirty = d.pop("dirty", UNSET)

        protocol_version = d.pop("protocolVersion", UNSET)

        supported_previous_protocol_versions = cast(
            list[int], d.pop("supportedPreviousProtocolVersions", UNSET)
        )

        version = d.pop("version", UNSET)

        compatibility_response = cls(
            api_version=api_version,
            git_sha=git_sha,
            dirty=dirty,
            protocol_version=protocol_version,
            supported_previous_protocol_versions=supported_previous_protocol_versions,
            version=version,
        )

        compatibility_response.additional_properties = d
        return compatibility_response

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
