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
        protocol_version (int):
        supported_previous_protocol_versions (list[int] | None):
        dirty (bool | Unset):
        version (str | Unset):
    """

    api_version: int
    git_sha: str
    protocol_version: int
    supported_previous_protocol_versions: list[int] | None
    dirty: bool | Unset = UNSET
    version: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        api_version = self.api_version

        git_sha = self.git_sha

        protocol_version = self.protocol_version

        supported_previous_protocol_versions: list[int] | None
        if isinstance(self.supported_previous_protocol_versions, list):
            supported_previous_protocol_versions = (
                self.supported_previous_protocol_versions
            )

        else:
            supported_previous_protocol_versions = (
                self.supported_previous_protocol_versions
            )

        dirty = self.dirty

        version = self.version

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "apiVersion": api_version,
                "gitSha": git_sha,
                "protocolVersion": protocol_version,
                "supportedPreviousProtocolVersions": supported_previous_protocol_versions,
            }
        )
        if dirty is not UNSET:
            field_dict["dirty"] = dirty
        if version is not UNSET:
            field_dict["version"] = version

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        api_version = d.pop("apiVersion")

        git_sha = d.pop("gitSha")

        protocol_version = d.pop("protocolVersion")

        def _parse_supported_previous_protocol_versions(
            data: object,
        ) -> list[int] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                supported_previous_protocol_versions_type_0 = cast(list[int], data)

                return supported_previous_protocol_versions_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[int] | None, data)

        supported_previous_protocol_versions = (
            _parse_supported_previous_protocol_versions(
                d.pop("supportedPreviousProtocolVersions")
            )
        )

        dirty = d.pop("dirty", UNSET)

        version = d.pop("version", UNSET)

        compatibility_response = cls(
            api_version=api_version,
            git_sha=git_sha,
            protocol_version=protocol_version,
            supported_previous_protocol_versions=supported_previous_protocol_versions,
            dirty=dirty,
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
