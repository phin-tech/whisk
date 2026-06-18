from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="RegistryPlugin")


@_attrs_define
class RegistryPlugin:
    """
    Attributes:
        id (str):
        installed (bool):
        registry (str):
        source_type (str):
        trusted (bool):
        description (str | Unset):
        name (str | Unset):
    """

    id: str
    installed: bool
    registry: str
    source_type: str
    trusted: bool
    description: str | Unset = UNSET
    name: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id = self.id

        installed = self.installed

        registry = self.registry

        source_type = self.source_type

        trusted = self.trusted

        description = self.description

        name = self.name

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "id": id,
                "installed": installed,
                "registry": registry,
                "sourceType": source_type,
                "trusted": trusted,
            }
        )
        if description is not UNSET:
            field_dict["description"] = description
        if name is not UNSET:
            field_dict["name"] = name

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        installed = d.pop("installed")

        registry = d.pop("registry")

        source_type = d.pop("sourceType")

        trusted = d.pop("trusted")

        description = d.pop("description", UNSET)

        name = d.pop("name", UNSET)

        registry_plugin = cls(
            id=id,
            installed=installed,
            registry=registry,
            source_type=source_type,
            trusted=trusted,
            description=description,
            name=name,
        )

        registry_plugin.additional_properties = d
        return registry_plugin

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
